<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import PageWrapper from '../ui/PageWrapper.vue'
import ConfirmModal from '../ui/ConfirmModal.vue'
import TabBar from '../ui/TabBar.vue'
import { api, problemMessage, type Schemas } from '../../api/client'
import { useTravelStore } from '../stores/store'

type AdminUser = Schemas['AdminUser']
type OidcConfig = Schemas['OidcAdminConfig']

const store = useTravelStore()
const tab = ref('users')
const tabs = computed(() => [
    { id: 'users', label: `Users (${users.value.length})` },
    { id: 'registration', label: 'Registration' },
    { id: 'sso', label: 'Single sign-on' },
])

const users = ref<AdminUser[]>([])
const registrationOpen = ref(true)
const loaded = ref(false)
const message = ref<{ text: string; error: boolean } | null>(null)

// Single sign-on: the form mirrors what the server has; the secret is never
// sent back, so its field only ever holds a replacement
const oidc = ref<OidcConfig | null>(null)
const oidcForm = ref({
    issuer: '',
    clientId: '',
    clientSecret: '',
    redirectUrl: '',
    name: '',
    scopes: '',
    disablePassword: false,
})
const oidcBusy = ref(false)
const callbackUrl = `${window.location.origin}/api/auth/oidc/callback`
const oidcReadonly = computed(() => oidc.value?.source === 'env')

function applyOidc(config: OidcConfig) {
    oidc.value = config
    oidcForm.value = {
        issuer: config.issuer,
        clientId: config.clientId,
        clientSecret: '',
        redirectUrl: config.redirectUrl,
        name: config.name,
        scopes: config.scopes,
        disablePassword: config.disablePassword,
    }
}

function oidcBody() {
    const f = oidcForm.value
    return {
        issuer: f.issuer.trim(),
        clientId: f.clientId.trim(),
        clientSecret: f.clientSecret || undefined,
        redirectUrl: f.redirectUrl.trim() || callbackUrl,
        name: f.name.trim() || undefined,
        scopes: f.scopes.trim() || undefined,
        disablePassword: f.disablePassword,
    }
}

async function reloadOidc() {
    const { data, error } = await api.GET('/admin/oidc')
    if (error || !data) throw error
    applyOidc(data)
}

async function testOidc() {
    oidcBusy.value = true
    const { error } = await api.POST('/admin/oidc/test', { body: oidcBody() })
    oidcBusy.value = false
    if (error) {
        say(problemMessage(error, 'The provider could not be reached'), true)
        return
    }
    say('The provider answered')
}

async function saveOidc() {
    oidcBusy.value = true
    const { error } = await api.PUT('/admin/oidc', { body: oidcBody() })
    oidcBusy.value = false
    if (error) {
        say(problemMessage(error, 'Could not save the configuration'), true)
        return
    }
    try {
        await reloadOidc()
        say('Single sign-on saved and active')
    } catch (e) {
        say(problemMessage(e, 'Saved, but could not reload'), true)
    }
}

async function removeOidc() {
    oidcBusy.value = true
    const { error } = await api.DELETE('/admin/oidc')
    oidcBusy.value = false
    if (error) {
        say(problemMessage(error, 'Could not remove the configuration'), true)
        return
    }
    try {
        await reloadOidc()
        say('Single sign-on is off')
    } catch (e) {
        say(problemMessage(e, 'Removed, but could not reload'), true)
    }
}

// User with an open password form, and the password typed into it
const resetting = ref<string | null>(null)
const newPassword = ref('')
const toDelete = ref<AdminUser | null>(null)

function say(text: string, error = false) {
    message.value = { text, error }
}

async function load() {
    try {
        const [list, settings, oidcConfig] = await Promise.all([
            api.GET('/admin/users'),
            api.GET('/admin/settings'),
            api.GET('/admin/oidc'),
        ])
        if (list.error || !list.data) throw list.error
        if (settings.error || !settings.data) throw settings.error
        if (oidcConfig.error || !oidcConfig.data) throw oidcConfig.error
        applyOidc(oidcConfig.data)
        users.value = list.data
        registrationOpen.value = settings.data.registrationOpen
        loaded.value = true
    } catch (e) {
        say(problemMessage(e, 'Could not load the administration'), true)
    }
}

// Wait for the account: only administrators get this page
watch(
    () => [store.authState, store.isAdmin],
    () => {
        if (store.authState === 'ready' && store.isAdmin && !loaded.value) {
            load()
        }
    },
    { immediate: true }
)

async function setRegistration(open: boolean) {
    const before = registrationOpen.value
    registrationOpen.value = open
    const { error } = await api.PUT('/admin/settings', {
        body: { registrationOpen: open },
    })
    if (error) {
        registrationOpen.value = before
        say(problemMessage(error, 'Could not save the setting'), true)
        return
    }
    say(open ? 'Registration is open' : 'Registration is closed')
}

async function setRole(user: AdminUser, role: AdminUser['role']) {
    const { data, error } = await api.PATCH('/admin/users/{id}', {
        params: { path: { id: user.id } },
        body: { role },
    })
    if (error || !data) {
        say(problemMessage(error, 'Could not change the role'), true)
        // Show the role the server still has
        await load()
        return
    }
    users.value = users.value.map((u) => (u.id === data.id ? data : u))
    say(
        `@${data.username} is now ${data.role === 'admin' ? 'an administrator' : 'a user'}`
    )
}

function startReset(user: AdminUser) {
    resetting.value = resetting.value === user.id ? null : user.id
    newPassword.value = ''
}

async function resetPassword(user: AdminUser) {
    const { error } = await api.PUT('/admin/users/{id}/password', {
        params: { path: { id: user.id } },
        body: { newPassword: newPassword.value },
    })
    if (error) {
        say(problemMessage(error, 'Could not reset the password'), true)
        return
    }
    resetting.value = null
    newPassword.value = ''
    say(
        `@${user.username} is logged out everywhere and has to choose a new password at the next login`
    )
    await load()
}

async function deleteUser() {
    const user = toDelete.value
    toDelete.value = null
    if (!user) return
    const { error } = await api.DELETE('/admin/users/{id}', {
        params: { path: { id: user.id } },
    })
    if (error) {
        say(problemMessage(error, 'Could not delete the account'), true)
        return
    }
    users.value = users.value.filter((u) => u.id !== user.id)
    say(`Deleted @${user.username}`)
}
</script>

<template>
    <PageWrapper
        title="Administration"
        subtitle="Users, registration and single sign-on"
    >
        <p
            v-if="!store.isAdmin && store.authState === 'ready'"
            class="page-error"
        >
            Only administrators can open this page.
        </p>
        <template v-else>
            <p
                v-if="message"
                class="toast"
                :class="{ error: message.error }"
                role="status"
            >
                {{ message.text }}
            </p>

            <TabBar v-model="tab" :tabs="tabs" />
            <template v-if="tab === 'users'">
                <section class="panel">
                    <h2>Users</h2>
                    <ul class="users">
                        <li v-for="user in users" :key="user.id">
                            <div class="row">
                                <div class="who">
                                    <strong>@{{ user.username }}</strong>
                                    <span class="muted">
                                        {{ user.displayName }}
                                        <template v-if="user.email">
                                            · {{ user.email }}
                                        </template>
                                    </span>
                                    <span
                                        v-if="user.mustChangePassword"
                                        class="tag"
                                    >
                                        must change password
                                    </span>
                                </div>
                                <select
                                    :value="user.role"
                                    :aria-label="`Role of ${user.username}`"
                                    @change="
                                        setRole(
                                            user,
                                            ($event.target as HTMLSelectElement)
                                                .value as AdminUser['role']
                                        )
                                    "
                                >
                                    <option value="user">User</option>
                                    <option value="admin">Administrator</option>
                                </select>
                                <div
                                    v-if="
                                        user.username !==
                                        store.profile?.username
                                    "
                                    class="actions"
                                >
                                    <button
                                        type="button"
                                        class="btn secondary"
                                        @click="startReset(user)"
                                    >
                                        Reset password
                                    </button>
                                    <button
                                        type="button"
                                        class="btn danger"
                                        @click="toDelete = user"
                                    >
                                        Delete
                                    </button>
                                </div>
                            </div>
                            <form
                                v-if="resetting === user.id"
                                class="reset"
                                @submit.prevent="resetPassword(user)"
                            >
                                <input
                                    v-model="newPassword"
                                    type="text"
                                    minlength="10"
                                    maxlength="72"
                                    autocomplete="off"
                                    placeholder="Temporary password, at least 10 characters"
                                    :aria-label="`Temporary password for ${user.username}`"
                                    required
                                />
                                <button type="submit" class="btn">Set</button>
                            </form>
                        </li>
                    </ul>
                </section>
            </template>
            <template v-else-if="tab === 'registration'">
                <section class="panel">
                    <h2>Registration</h2>
                    <label class="check">
                        <input
                            type="checkbox"
                            :checked="registrationOpen"
                            :disabled="!loaded"
                            @change="
                                setRegistration(
                                    ($event.target as HTMLInputElement).checked
                                )
                            "
                        />
                        Let visitors create their own account
                    </label>
                    <p class="note">
                        When closed, only administrators can add people, and
                        only existing accounts can log in.
                    </p>
                </section>
            </template>
            <template v-else>
                <section v-if="oidc" class="panel">
                    <h2>OpenID Connect</h2>
                    <p v-if="oidcReadonly" class="note">
                        Set by environment variables (OIDC_*) on the server.
                        Remove them there to change it here.
                    </p>
                    <p v-else class="note">
                        Works with any OpenID Connect provider, for example
                        Pocket ID. Register
                        <code>{{ callbackUrl }}</code> as the callback URL
                        there.
                    </p>
                    <p v-if="oidc.source !== 'none'" class="note">
                        Status:
                        {{
                            oidc.enabled
                                ? 'active'
                                : 'configured, but the provider cannot be reached'
                        }}
                    </p>
                    <form class="fields" @submit.prevent="saveOidc">
                        <label for="oidc-issuer">Issuer URL</label>
                        <input
                            id="oidc-issuer"
                            v-model="oidcForm.issuer"
                            type="text"
                            placeholder="https://id.example.com"
                            :disabled="oidcReadonly"
                            required
                        />
                        <label for="oidc-client">Client ID</label>
                        <input
                            id="oidc-client"
                            v-model="oidcForm.clientId"
                            type="text"
                            autocomplete="off"
                            :disabled="oidcReadonly"
                            required
                        />
                        <label for="oidc-secret">Client secret</label>
                        <input
                            id="oidc-secret"
                            v-model="oidcForm.clientSecret"
                            type="password"
                            autocomplete="new-password"
                            :placeholder="
                                oidc.secretSet
                                    ? 'Set. Leave empty to keep it'
                                    : ''
                            "
                            :disabled="oidcReadonly"
                            :required="!oidc.secretSet"
                        />
                        <label for="oidc-redirect">Callback URL</label>
                        <input
                            id="oidc-redirect"
                            v-model="oidcForm.redirectUrl"
                            type="text"
                            :placeholder="callbackUrl"
                            :disabled="oidcReadonly"
                        />
                        <label for="oidc-name"
                            >Button text (provider name)</label
                        >
                        <input
                            id="oidc-name"
                            v-model="oidcForm.name"
                            type="text"
                            maxlength="40"
                            placeholder="SSO"
                            :disabled="oidcReadonly"
                        />
                        <label for="oidc-scopes">Scopes</label>
                        <input
                            id="oidc-scopes"
                            v-model="oidcForm.scopes"
                            type="text"
                            placeholder="openid profile email"
                            :disabled="oidcReadonly"
                        />
                        <label class="check">
                            <input
                                v-model="oidcForm.disablePassword"
                                type="checkbox"
                                :disabled="oidcReadonly"
                            />
                            Turn off password login and registration
                        </label>
                        <p class="note">
                            Saving checks that the provider can be reached
                            first, so you cannot lock yourself out.
                        </p>
                        <div v-if="!oidcReadonly" class="actions">
                            <button
                                type="submit"
                                class="btn"
                                :disabled="oidcBusy"
                            >
                                Save
                            </button>
                            <button
                                type="button"
                                class="btn secondary"
                                :disabled="oidcBusy"
                                @click="testOidc"
                            >
                                Test
                            </button>
                            <button
                                v-if="oidc.source === 'settings'"
                                type="button"
                                class="btn danger"
                                :disabled="oidcBusy"
                                @click="removeOidc"
                            >
                                Remove
                            </button>
                        </div>
                    </form>
                </section>
            </template>
        </template>

        <ConfirmModal
            v-if="toDelete"
            title="Delete account"
            :message="`This deletes @${toDelete.username} with all countries, flights and photos. It cannot be undone.`"
            confirm-label="Delete"
            @confirm="deleteUser"
            @cancel="toDelete = null"
        />
    </PageWrapper>
</template>

<style src="../ui/panel.css"></style>
<style scoped>
.users {
    list-style: none;
    margin: 0;
    padding: 0;
}

.users li {
    padding: 0.8rem 0;
    border-bottom: 1px solid var(--line);
}

.users li:last-child {
    border-bottom: none;
    padding-bottom: 0;
}

.users li:first-child {
    padding-top: 0;
}

.panel .users .row {
    flex-wrap: wrap;
    align-items: center;
    gap: 0.6rem 1rem;
}

.who {
    flex: 1 1 14rem;
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    min-width: 0;
    font-size: 0.85rem;
}

.muted {
    font-size: 0.75rem;
    color: var(--muted);
    overflow-wrap: anywhere;
}

.tag {
    align-self: flex-start;
    padding: 0.05rem 0.45rem;
    border-radius: 0.6rem;
    background: var(--chip);
    font-size: 0.68rem;
    color: var(--muted);
}

.panel .users select {
    width: auto;
}

.panel .users .actions {
    margin-top: 0;
}

.reset {
    display: flex;
    gap: 0.5rem;
    margin-top: 0.6rem;
}

.reset input {
    flex: 1;
    min-width: 0;
}
</style>
