<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api, problemMessage, type Schemas } from '../../api/client'

type OidcConfig = Schemas['OidcAdminConfig']

const emit = defineEmits<{ say: [text: string, error?: boolean] }>()
const say = (text: string, error = false) => emit('say', text, error)

// The form mirrors what the server has; the secret is never sent back, so its
// field only ever holds a replacement
const oidc = ref<OidcConfig | null>(null)
const oidcForm = ref({
    issuer: '',
    clientId: '',
    clientSecret: '',
    name: '',
    scopes: '',
    disablePassword: false,
})
const oidcBusy = ref(false)
const callbackUrl = `${window.location.origin}/api/auth/oidc/callback`
const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | undefined

async function copyCallback() {
    try {
        await navigator.clipboard.writeText(shownCallback.value)
    } catch {
        say('Could not copy; select the URL and copy it by hand', true)
        return
    }
    copied.value = true
    clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => (copied.value = false), 2000)
}
// The callback the app answers on, built from the address the admin uses; with
// environment variables the registered one is shown instead
const shownCallback = computed(() =>
    oidcReadonly.value && oidc.value?.redirectUrl
        ? oidc.value.redirectUrl
        : callbackUrl
)
const status = computed(() => {
    const source =
        oidc.value?.source === 'env' ? 'environment variables' : 'settings'
    if (!oidc.value || oidc.value.source === 'none') {
        return { kind: 'off', text: 'Not configured' }
    }
    return oidc.value.enabled
        ? { kind: 'ok', text: `Active: the provider answers (from ${source})` }
        : {
              kind: 'bad',
              text: `Not working: the provider cannot be reached (from ${source})`,
          }
})
const oidcReadonly = computed(() => oidc.value?.source === 'env')

function applyOidc(config: OidcConfig) {
    oidc.value = config
    oidcForm.value = {
        issuer: config.issuer,
        clientId: config.clientId,
        clientSecret: '',
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
        redirectUrl: callbackUrl,
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

onMounted(async () => {
    try {
        await reloadOidc()
    } catch (e) {
        say(problemMessage(e, 'Could not load single sign-on'), true)
    }
})
</script>

<template>
    <section v-if="oidc" class="panel">
        <h2>
            OpenID Connect
            <span
                class="light"
                :class="status.kind"
                role="img"
                :title="status.text"
                :aria-label="status.text"
            />
        </h2>
        <p v-if="oidcReadonly" class="note">
            Set by environment variables (OIDC_*) on the server. Remove them
            there to change it here.
        </p>
        <p v-else class="note">
            Works with any OpenID Connect provider, for example Pocket ID.
            Register this callback URL there:
        </p>
        <button
            type="button"
            class="callback"
            title="Copy to clipboard"
            @click="copyCallback"
        >
            <code>{{ shownCallback }}</code>
            <span class="copy">{{ copied ? 'Copied' : 'Copy' }}</span>
        </button>
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
                    oidc.secretSet ? 'Set. Leave empty to keep it' : ''
                "
                :disabled="oidcReadonly"
                :required="!oidc.secretSet"
            />
            <label for="oidc-name">Button text (provider name)</label>
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
                Saving checks that the provider can be reached first, so you
                cannot lock yourself out.
            </p>
            <div v-if="!oidcReadonly" class="actions">
                <button type="submit" class="btn" :disabled="oidcBusy">
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

<style src="../ui/panel.css"></style>
<style scoped>
.light {
    display: inline-block;
    width: 0.65rem;
    height: 0.65rem;
    margin-left: 0.3rem;
    border-radius: 50%;
    vertical-align: middle;
    background: var(--muted);
}

.light.ok {
    background: #2e8b57;
    box-shadow: 0 0 0 3px #2e8b5733;
}

.light.bad {
    background: #c0392b;
    box-shadow: 0 0 0 3px #c0392b33;
}

.callback {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.8rem;
    width: 100%;
    margin: 0.6rem 0 0.2rem;
    padding: 0.7rem 0.9rem;
    border: 1px dashed var(--line-strong);
    border-radius: 0.6rem;
    background: var(--accent-soft);
    color: var(--text);
    font: inherit;
    text-align: left;
    cursor: pointer;
}

.callback:hover {
    border-style: solid;
}

.callback code {
    min-width: 0;
    overflow-wrap: anywhere;
    font-size: 0.9rem;
    font-weight: 500;
}

.copy {
    flex: none;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--muted);
}
</style>
