<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api, problemMessage, type Schemas } from '../../api/client'
import { useTravelStore } from '../stores/store'
import { t } from '../../i18n'

type OidcConfig = Schemas['OidcAdminConfig']

const store = useTravelStore()
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
        say(t('oidc.copyFailed'), true)
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
    const source = t(
        oidc.value?.source === 'env' ? 'oidc.fromEnv' : 'oidc.fromSettings'
    )
    if (!oidc.value || oidc.value.source === 'none') {
        return { kind: 'off', text: t('oidc.notConfigured') }
    }
    return oidc.value.enabled
        ? { kind: 'ok', text: t('oidc.active', { source }) }
        : { kind: 'bad', text: t('oidc.broken', { source }) }
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
        say(problemMessage(error, 'oidc.unreachable'), true)
        return
    }
    say(t('oidc.answered'))
}

async function saveOidc() {
    oidcBusy.value = true
    const { error } = await api.PUT('/admin/oidc', { body: oidcBody() })
    oidcBusy.value = false
    if (error) {
        say(problemMessage(error, 'oidc.saveFailed'), true)
        return
    }
    try {
        await reloadOidc()
        say(t('oidc.saved'))
    } catch (e) {
        say(problemMessage(e, 'oidc.savedReloadFailed'), true)
    }
}

async function removeOidc() {
    oidcBusy.value = true
    const { error } = await api.DELETE('/admin/oidc')
    oidcBusy.value = false
    if (error) {
        say(problemMessage(error, 'oidc.removeFailed'), true)
        return
    }
    try {
        await reloadOidc()
        say(t('oidc.removed'))
    } catch (e) {
        say(problemMessage(e, 'oidc.removedReloadFailed'), true)
    }
}

onMounted(async () => {
    try {
        await reloadOidc()
    } catch (e) {
        say(problemMessage(e, 'oidc.loadFailed'), true)
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
        <p class="note">
            {{ oidcReadonly ? t('oidc.envNote') : t('oidc.callbackNote') }}
        </p>
        <button
            type="button"
            class="callback"
            :title="t('oidc.copyTitle')"
            @click="copyCallback"
        >
            <code>{{ shownCallback }}</code>
            <span class="copy">{{
                copied ? t('oidc.copied') : t('oidc.copy')
            }}</span>
        </button>
        <form class="fields" @submit.prevent="saveOidc">
            <label for="oidc-issuer">{{ t('oidc.issuer') }}</label>
            <input
                id="oidc-issuer"
                v-model="oidcForm.issuer"
                type="text"
                placeholder="https://id.example.com"
                :disabled="oidcReadonly"
                required
            />
            <label for="oidc-client">{{ t('oidc.clientId') }}</label>
            <input
                id="oidc-client"
                v-model="oidcForm.clientId"
                type="text"
                autocomplete="off"
                :disabled="oidcReadonly"
                required
            />
            <label for="oidc-secret">{{ t('oidc.clientSecret') }}</label>
            <input
                id="oidc-secret"
                v-model="oidcForm.clientSecret"
                type="password"
                autocomplete="new-password"
                :placeholder="oidc.secretSet ? t('oidc.secretSet') : ''"
                :disabled="oidcReadonly"
                :required="!oidc.secretSet"
            />
            <label for="oidc-name">{{ t('oidc.buttonText') }}</label>
            <input
                id="oidc-name"
                v-model="oidcForm.name"
                type="text"
                maxlength="40"
                placeholder="SSO"
                :disabled="oidcReadonly"
            />
            <label for="oidc-scopes">{{ t('oidc.scopes') }}</label>
            <input
                id="oidc-scopes"
                v-model="oidcForm.scopes"
                type="text"
                placeholder="openid profile email"
                :disabled="oidcReadonly"
            />
            <label class="check toggle">
                <span>{{ t('oidc.disablePassword') }}</span>
                <input
                    v-model="oidcForm.disablePassword"
                    type="checkbox"
                    role="switch"
                    class="switch"
                    :disabled="oidcReadonly"
                />
            </label>
            <p class="note">{{ t('oidc.saveNote') }}</p>
            <div v-if="!oidcReadonly" class="actions">
                <button
                    type="submit"
                    class="btn"
                    :disabled="oidcBusy || !store.canWrite"
                >
                    {{ t('common.save') }}
                </button>
                <button
                    type="button"
                    class="btn secondary"
                    :disabled="oidcBusy || !store.canWrite"
                    @click="testOidc"
                >
                    {{ t('oidc.test') }}
                </button>
                <button
                    v-if="oidc.source === 'settings'"
                    type="button"
                    class="btn danger"
                    :disabled="oidcBusy || !store.canWrite"
                    @click="removeOidc"
                >
                    {{ t('common.remove') }}
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
