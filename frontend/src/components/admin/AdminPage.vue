<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import PageWrapper from '../ui/PageWrapper.vue'
import LegalFooter from '../ui/LegalFooter.vue'
import ConfirmModal from '../ui/ConfirmModal.vue'
import TabBar from '../ui/TabBar.vue'
import OidcSettings from './OidcSettings.vue'
import { api, problemMessage, type Schemas } from '../../api/client'
import { useTravelStore } from '../stores/store'
import { t } from '../../i18n'

type AdminUser = Schemas['AdminUser']

const store = useTravelStore()
const tab = ref('users')
const tabs = computed(() => [
    {
        id: 'users',
        label: t('admin.usersCount', { count: users.value.length }),
    },
    { id: 'registration', label: t('admin.registration') },
])

const users = ref<AdminUser[]>([])
const registrationOpen = ref(true)
const loaded = ref(false)
const message = ref<{ text: string; error: boolean } | null>(null)

// User with an open password form, and the password typed into it
const resetting = ref<string | null>(null)
const newPassword = ref('')
const toDelete = ref<AdminUser | null>(null)

function say(text: string, error = false) {
    message.value = { text, error }
}

async function load() {
    try {
        const [list, settings] = await Promise.all([
            api.GET('/admin/users'),
            api.GET('/admin/settings'),
        ])
        if (list.error || !list.data) throw list.error
        if (settings.error || !settings.data) throw settings.error
        users.value = list.data
        registrationOpen.value = settings.data.registrationOpen
        loaded.value = true
    } catch (e) {
        say(problemMessage(e, 'admin.loadFailed'), true)
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
        say(problemMessage(error, 'admin.settingFailed'), true)
        return
    }
    say(t(open ? 'admin.registrationOpen' : 'admin.registrationClosed'))
}

async function setRole(user: AdminUser, role: AdminUser['role']) {
    const { data, error } = await api.PATCH('/admin/users/{id}', {
        params: { path: { id: user.id } },
        body: { role },
    })
    if (error || !data) {
        say(problemMessage(error, 'admin.roleFailed'), true)
        // Show the role the server still has
        await load()
        return
    }
    users.value = users.value.map((u) => (u.id === data.id ? data : u))
    say(
        t(data.role === 'admin' ? 'admin.nowAdmin' : 'admin.nowUser', {
            name: data.username,
        })
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
        say(problemMessage(error, 'admin.resetFailed'), true)
        return
    }
    resetting.value = null
    newPassword.value = ''
    say(t('admin.passwordReset', { name: user.username }))
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
        say(problemMessage(error, 'admin.deleteFailed'), true)
        return
    }
    users.value = users.value.filter((u) => u.id !== user.id)
    say(t('admin.deleted', { name: user.username }))
}
</script>

<template>
    <PageWrapper :title="t('admin.title')" :subtitle="t('admin.subtitle')">
        <p
            v-if="!store.isAdmin && store.authState === 'ready'"
            class="page-error"
        >
            {{ t('admin.onlyAdmins') }}
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

            <p v-if="!store.canWrite" class="toast offline-note">
                {{ t('common.offlineHint') }}. {{ t('admin.offline') }}
            </p>
            <TabBar v-model="tab" :tabs="tabs" />
            <template v-if="tab === 'users'">
                <section class="panel">
                    <h2>{{ t('admin.users') }}</h2>
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
                                        {{ t('admin.mustChange') }}
                                    </span>
                                </div>
                                <select
                                    :value="user.role"
                                    :disabled="!store.canWrite"
                                    :aria-label="
                                        t('admin.roleOf', {
                                            name: user.username,
                                        })
                                    "
                                    @change="
                                        setRole(
                                            user,
                                            ($event.target as HTMLSelectElement)
                                                .value as AdminUser['role']
                                        )
                                    "
                                >
                                    <option value="user">
                                        {{ t('admin.roleUser') }}
                                    </option>
                                    <option value="admin">
                                        {{ t('admin.roleAdmin') }}
                                    </option>
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
                                        :disabled="!store.canWrite"
                                        @click="startReset(user)"
                                    >
                                        {{ t('admin.resetPassword') }}
                                    </button>
                                    <button
                                        type="button"
                                        class="btn danger"
                                        :disabled="!store.canWrite"
                                        @click="toDelete = user"
                                    >
                                        {{ t('common.delete') }}
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
                                    :placeholder="t('admin.tempPassword')"
                                    :aria-label="
                                        t('admin.tempPasswordFor', {
                                            name: user.username,
                                        })
                                    "
                                    required
                                />
                                <button
                                    type="submit"
                                    class="btn"
                                    :disabled="!store.canWrite"
                                >
                                    {{ t('admin.set') }}
                                </button>
                            </form>
                        </li>
                    </ul>
                </section>
            </template>
            <template v-else>
                <section class="panel">
                    <h2>{{ t('admin.registration') }}</h2>
                    <label class="check">
                        <input
                            type="checkbox"
                            :checked="registrationOpen"
                            :disabled="!loaded || !store.canWrite"
                            @change="
                                setRegistration(
                                    ($event.target as HTMLInputElement).checked
                                )
                            "
                        />
                        {{ t('admin.letVisitors') }}
                    </label>
                    <p class="note">{{ t('admin.closedNote') }}</p>
                </section>
                <OidcSettings @say="say" />
            </template>
        </template>

        <LegalFooter />

        <ConfirmModal
            v-if="toDelete"
            :title="t('profile.deleteAccount')"
            :message="t('admin.deleteConfirm', { name: toDelete.username })"
            :confirm-label="t('common.delete')"
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
    flex-direction: row;
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
