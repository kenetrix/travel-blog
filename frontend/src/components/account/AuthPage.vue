<script setup lang="ts">
import { ref, computed, nextTick, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTravelStore } from '../stores/store'
import { LANGUAGES } from './options'
import { vEnterNext } from '../ui/enterNext'

const store = useTravelStore()
const router = useRouter()
const route = useRoute()

const MIN_PASSWORD = 10

// One form for both cases: the username decides whether to log in or register
const step = ref<'username' | 'login' | 'register'>('username')
const username = ref('')
const password = ref('')
const email = ref('')
const firstName = ref('')
const lastName = ref('')
// Preselect the browser's language if it is one we offer
const browserLanguage = navigator.language.split('-')[0]
const language = ref(
    LANGUAGES.some((l) => l.code === browserLanguage) ? browserLanguage : 'en'
)

const showPassword = ref(false)
const capsLock = ref(false)

// ?error= comes from a failed single sign-on; the codes are set by the backend
const OIDC_ERRORS: Record<string, string> = {
    oidc_failed: 'Single sign-on failed. Please try again.',
    registration_closed:
        'There is no account for this login, and registration is closed. Ask an administrator for an account.',
    email_unverified:
        'An account with this email already exists, but the provider has not verified your email address.',
}
const oidcError = route.query.error
const error = ref(
    typeof oidcError === 'string'
        ? (OIDC_ERRORS[oidcError] ?? OIDC_ERRORS.oidc_failed)
        : ''
)
const busy = ref(false)

// Single sign-on button and, if the server turned passwords off, no form
const sso = ref<{ name: string } | null>(null)
const passwordLogin = ref(true)

// Each step moves the cursor on to its first new field
const usernameInput = ref<HTMLInputElement>()
const emailInput = ref<HTMLInputElement>()
const passwordInput = ref<HTMLInputElement>()

// On touch screens the keyboard would cover the single sign-on button
const focusFirst = !matchMedia('(pointer: coarse)').matches

onMounted(async () => {
    if (focusFirst) usernameInput.value?.focus()
    const config = await store.oidcConfig()
    if (!config) return
    sso.value = config.enabled ? { name: config.name } : null
    passwordLogin.value = config.passwordLogin || !config.enabled
})

const title = computed(() => {
    if (step.value === 'login') return 'Welcome back'
    if (step.value === 'register') return 'Create your account'
    return 'Log in or sign up'
})

const subtitle = computed(() => {
    if (step.value === 'login') return 'Enter your password to continue.'
    if (step.value === 'register') return 'This username is still free.'
    return 'Your countries and flights, on every device.'
})

const buttonLabel = computed(() => {
    if (busy.value) {
        if (step.value === 'login') return 'Logging in…'
        if (step.value === 'register') return 'Creating account…'
        return 'Checking…'
    }
    if (step.value === 'login') return 'Log in'
    if (step.value === 'register') return 'Create account'
    return 'Continue'
})

const passwordLongEnough = computed(() => password.value.length >= MIN_PASSWORD)

function trackCapsLock(e: KeyboardEvent) {
    capsLock.value = e.getModifierState?.('CapsLock') ?? false
}

async function checkUsername() {
    const name = username.value.trim().toLowerCase()
    const check = await store.checkUsername(name)
    if (typeof check === 'string') {
        error.value = check
        return
    }
    username.value = name
    if (check.exists) {
        step.value = 'login'
        await nextTick()
        passwordInput.value?.focus()
    } else if (check.registrationOpen) {
        step.value = 'register'
        await nextTick()
        emailInput.value?.focus()
    } else {
        error.value =
            'There is no account with this username, and registration is closed. Ask an administrator for an account.'
    }
}

async function submit() {
    busy.value = true
    error.value = ''
    if (step.value === 'username') {
        await checkUsername()
        busy.value = false
        return
    }
    const failed =
        step.value === 'login'
            ? await store.login(username.value, password.value)
            : await store.register({
                  username: username.value,
                  password: password.value,
                  email: email.value.trim() || undefined,
                  firstName: firstName.value.trim() || undefined,
                  lastName: lastName.value.trim() || undefined,
                  language: language.value,
              })
    busy.value = false
    if (failed) {
        error.value = failed
        password.value = ''
        await nextTick()
        passwordInput.value?.focus()
        return
    }
    password.value = ''
    router.push({ name: 'home' })
}

async function changeUsername() {
    step.value = 'username'
    password.value = ''
    showPassword.value = false
    error.value = ''
    await nextTick()
    usernameInput.value?.focus()
}
</script>

<template>
    <div class="auth">
        <button
            class="back"
            type="button"
            aria-label="Back to the map"
            @click="router.push({ name: 'home' })"
        >
            <svg
                viewBox="0 0 24 24"
                width="18"
                height="18"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
            >
                <path d="M15 6l-6 6 6 6" />
            </svg>
            <span class="back-text">Map</span>
        </button>

        <main class="card">
            <header>
                <img
                    class="logo"
                    src="/favicon.svg"
                    alt=""
                    width="56"
                    height="56"
                />
                <h1>{{ title }}</h1>
                <p class="subtitle">{{ subtitle }}</p>
            </header>

            <p v-if="!store.canWrite" class="notice" role="status">
                You are offline. Logging in needs a connection.
            </p>

            <p v-if="error" class="error" role="alert">
                <svg
                    viewBox="0 0 24 24"
                    width="18"
                    height="18"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    aria-hidden="true"
                >
                    <circle cx="12" cy="12" r="9" />
                    <path d="M12 7.5v5.5M12 16.5v.01" />
                </svg>
                <span>{{ error }}</span>
            </p>

            <template v-if="sso && step === 'username'">
                <a
                    class="btn sso"
                    :class="{ disabled: !store.canWrite }"
                    href="/api/auth/oidc/login"
                    :aria-disabled="!store.canWrite"
                    @click="!store.canWrite && $event.preventDefault()"
                >
                    <svg
                        viewBox="0 0 24 24"
                        width="20"
                        height="20"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        aria-hidden="true"
                    >
                        <circle cx="8" cy="15" r="4" />
                        <path d="M10.8 12.2L20 3M16 7l3 3M14 9l2 2" />
                    </svg>
                    Continue with {{ sso.name }}
                </a>
                <div v-if="passwordLogin" class="divider">
                    <span>or with your username</span>
                </div>
            </template>

            <form
                v-if="passwordLogin"
                v-enter-next
                class="form"
                @submit.prevent="submit"
            >
                <!-- Stays in the form for password managers; the chip below
                     shows it once it is fixed -->
                <div
                    class="field"
                    :class="{ 'visually-hidden': step !== 'username' }"
                >
                    <label for="username">Username</label>
                    <input
                        id="username"
                        ref="usernameInput"
                        v-model="username"
                        autocomplete="username"
                        autocapitalize="none"
                        autocorrect="off"
                        spellcheck="false"
                        enterkeyhint="next"
                        pattern="[a-zA-Z0-9_]{3,30}"
                        title="3–30 letters, digits or _"
                        :readonly="step !== 'username'"
                        :tabindex="step === 'username' ? undefined : -1"
                        required
                    />
                </div>

                <div v-if="step !== 'username'" class="chip">
                    <span class="initial" aria-hidden="true">
                        {{ username.charAt(0).toUpperCase() }}
                    </span>
                    <span class="who">@{{ username }}</span>
                    <button type="button" class="link" @click="changeUsername">
                        {{ step === 'login' ? 'Not you?' : 'Change' }}
                    </button>
                </div>

                <template v-if="step === 'register'">
                    <div class="field">
                        <label for="email">
                            Email <span class="optional">optional</span>
                        </label>
                        <input
                            id="email"
                            ref="emailInput"
                            v-model="email"
                            type="email"
                            inputmode="email"
                            autocomplete="email"
                            autocapitalize="none"
                        />
                    </div>
                    <div class="row">
                        <div class="field">
                            <label for="first">First name</label>
                            <input
                                id="first"
                                v-model="firstName"
                                autocomplete="given-name"
                            />
                        </div>
                        <div class="field">
                            <label for="last">Last name</label>
                            <input
                                id="last"
                                v-model="lastName"
                                autocomplete="family-name"
                            />
                        </div>
                    </div>
                    <div class="field">
                        <label for="language">Language</label>
                        <select id="language" v-model="language">
                            <option
                                v-for="l in LANGUAGES"
                                :key="l.code"
                                :value="l.code"
                            >
                                {{ l.name }}
                            </option>
                        </select>
                    </div>
                </template>

                <div v-if="step !== 'username'" class="field">
                    <label for="password">Password</label>
                    <div class="password">
                        <input
                            id="password"
                            ref="passwordInput"
                            v-model="password"
                            :type="showPassword ? 'text' : 'password'"
                            :autocomplete="
                                step === 'login'
                                    ? 'current-password'
                                    : 'new-password'
                            "
                            :minlength="
                                step === 'register' ? MIN_PASSWORD : undefined
                            "
                            enterkeyhint="go"
                            :aria-invalid="!!error && step === 'login'"
                            aria-describedby="password-hints"
                            required
                            @keydown="trackCapsLock"
                            @keyup="trackCapsLock"
                            @blur="capsLock = false"
                        />
                        <button
                            type="button"
                            class="reveal"
                            :aria-label="
                                showPassword ? 'Hide password' : 'Show password'
                            "
                            :aria-pressed="showPassword"
                            @click="showPassword = !showPassword"
                        >
                            <svg
                                viewBox="0 0 24 24"
                                width="20"
                                height="20"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                aria-hidden="true"
                            >
                                <path
                                    d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12z"
                                />
                                <circle cx="12" cy="12" r="3" />
                                <path v-if="showPassword" d="M4 4l16 16" />
                            </svg>
                        </button>
                    </div>
                    <div id="password-hints" class="hints">
                        <p v-if="capsLock" class="hint warn">Caps Lock is on</p>
                        <p
                            v-if="step === 'register'"
                            class="hint"
                            :class="{ ok: passwordLongEnough }"
                        >
                            {{ passwordLongEnough ? '✓' : '•' }} At least
                            {{ MIN_PASSWORD }} characters
                        </p>
                    </div>
                </div>

                <button
                    class="btn primary"
                    type="submit"
                    :disabled="busy || !store.canWrite"
                    :aria-busy="busy"
                >
                    <span v-if="busy" class="spinner" aria-hidden="true" />
                    {{ buttonLabel }}
                </button>
            </form>
        </main>
    </div>
</template>

<style scoped>
/* Own full-screen layout over the map, like PageWrapper but centred */
.auth {
    position: absolute;
    inset: 0;
    z-index: 30;
    display: flex;
    flex-direction: column;
    align-items: center;
    overflow-y: auto;
    overscroll-behavior: contain;
    box-sizing: border-box;
    padding: calc(var(--safe-top) + 4rem) calc(var(--safe-right) + 1rem)
        calc(var(--safe-bottom) + 2rem) calc(var(--safe-left) + 1rem);
    /* A hint of the visited (gold) and flight (cyan) colours */
    background:
        radial-gradient(
            circle at 15% 0%,
            rgb(217 160 63 / 0.16),
            transparent 45%
        ),
        radial-gradient(
            circle at 100% 100%,
            rgb(95 212 228 / 0.12),
            transparent 45%
        ),
        var(--page);
    font-family: 'DM Sans', sans-serif;
    color: var(--text);
}

/* Centred vertically when there is room */
@media (min-height: 700px) {
    .auth {
        justify-content: center;
        padding-top: calc(var(--safe-top) + 2rem);
    }
}

.back {
    position: absolute;
    top: calc(var(--safe-top) + 0.75rem);
    left: calc(var(--safe-left) + 0.75rem);
    display: flex;
    align-items: center;
    gap: 0.2rem;
    min-width: var(--touch);
    min-height: var(--touch);
    padding: 0 0.8rem 0 0.6rem;
    border: 1px solid var(--line);
    border-radius: 2rem;
    background: var(--panel);
    color: var(--muted);
    font: inherit;
    font-size: 0.85rem;
    cursor: pointer;
}

.back:hover {
    color: var(--text);
}

.back-text {
    display: none;
}

@media (min-width: 600px) {
    .back-text {
        display: inline;
    }
}

.card {
    box-sizing: border-box;
    width: 100%;
    max-width: 400px;
    padding: 1.75rem 1.25rem;
    border: 1px solid var(--line);
    border-radius: 1.25rem;
    background: var(--raised);
    box-shadow: 0 12px 40px rgb(0 0 0 / 0.18);
}

@media (min-width: 600px) {
    .card {
        padding: 2.25rem 2rem;
    }
}

header {
    margin-bottom: 1.5rem;
    text-align: center;
}

.logo {
    display: block;
    margin: 0 auto 0.9rem;
}

h1 {
    margin: 0;
    font-family: 'DM Serif Display', serif;
    font-size: 1.65rem;
    font-weight: 400;
    line-height: 1.2;
}

.subtitle {
    margin: 0.4rem 0 0;
    font-size: 0.92rem;
    color: var(--muted);
}

.notice {
    margin: 0 0 1rem;
    padding: 0.7rem 0.8rem;
    border-radius: 0.6rem;
    background: rgb(183 121 31 / 0.14);
    color: #b7791f;
    font-size: 0.85rem;
}

.error {
    display: flex;
    gap: 0.5rem;
    align-items: flex-start;
    margin: 0 0 1rem;
    padding: 0.7rem 0.8rem;
    border: 1px solid rgb(192 57 43 / 0.35);
    border-radius: 0.6rem;
    background: rgb(192 57 43 / 0.1);
    color: #c0392b;
    font-size: 0.85rem;
    line-height: 1.4;
}

:root[data-theme='dark'] .error {
    color: #f08a7e;
}

@media (prefers-color-scheme: dark) {
    :root[data-theme='system'] .error {
        color: #f08a7e;
    }
}

.error svg {
    flex: none;
    margin-top: 0.05rem;
}

.btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.6rem;
    box-sizing: border-box;
    width: 100%;
    min-height: 48px;
    padding: 0 1rem;
    border-radius: 0.7rem;
    font: inherit;
    font-size: 0.95rem;
    font-weight: 500;
    text-decoration: none;
    cursor: pointer;
    transition:
        background 0.15s,
        border-color 0.15s;
}

.btn.primary {
    margin-top: 0.5rem;
    border: none;
    background: #3a7d5c;
    color: #fff;
}

.btn.primary:hover:not(:disabled) {
    background: #2f6a4e;
}

.btn.primary:disabled {
    opacity: 0.6;
    cursor: not-allowed;
}

.btn.primary[aria-busy='true'] {
    cursor: progress;
}

/* Secondary: the provider is another way in, not the page's main action */
.btn.sso {
    border: 1px solid var(--line-strong);
    background: var(--page);
    color: var(--text);
}

.btn.sso:hover {
    border-color: #3a7d5c;
}

.btn.sso.disabled {
    opacity: 0.6;
    cursor: not-allowed;
}

.divider {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin: 1.25rem 0;
    font-size: 0.8rem;
    color: var(--muted);
}

.divider::before,
.divider::after {
    content: '';
    flex: 1;
    height: 1px;
    background: var(--line-strong);
}

.form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.field {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    min-width: 0;
}

label {
    font-size: 0.85rem;
    font-weight: 500;
    color: var(--text-soft);
}

.optional {
    margin-left: 0.25rem;
    font-weight: 400;
    color: var(--muted);
}

input,
select {
    box-sizing: border-box;
    width: 100%;
    min-height: 48px;
    padding: 0 0.85rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.7rem;
    background: var(--page);
    color: var(--text);
    font: inherit;
    /* 16px: below that, iOS zooms into the field */
    font-size: 1rem;
    transition:
        border-color 0.15s,
        box-shadow 0.15s;
}

input:focus,
select:focus {
    outline: none;
    border-color: #3a7d5c;
    box-shadow: 0 0 0 3px rgb(58 125 92 / 0.25);
}

input[aria-invalid='true'] {
    border-color: #c0392b;
}

.row {
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

@media (min-width: 600px),
    (orientation: landscape) and (max-height: 500px) and (min-width: 480px) {
    .row {
        flex-direction: row;
        gap: 0.75rem;
    }

    .row > .field {
        flex: 1;
    }
}

.chip {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.4rem 0.4rem 0.4rem 0.45rem;
    border: 1px solid var(--line);
    border-radius: 2rem;
    background: var(--page);
}

.initial {
    display: grid;
    flex: none;
    place-items: center;
    width: 2rem;
    height: 2rem;
    border-radius: 50%;
    background: #3a7d5c;
    color: #fff;
    font-weight: 500;
}

.who {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 500;
}

.link {
    flex: none;
    min-height: 2.25rem;
    padding: 0 0.75rem;
    border: none;
    border-radius: 2rem;
    background: none;
    color: #3a7d5c;
    font: inherit;
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
}

.link:hover {
    background: var(--accent-soft);
}

.password {
    position: relative;
}

.password input {
    padding-right: 3.25rem;
}

.reveal {
    position: absolute;
    top: 0;
    right: 0;
    display: grid;
    place-items: center;
    width: 48px;
    height: 48px;
    padding: 0;
    border: none;
    background: none;
    color: var(--muted);
    cursor: pointer;
}

.reveal:hover {
    color: var(--text);
}

.hints:empty {
    display: none;
}

.hint {
    margin: 0.15rem 0 0;
    font-size: 0.8rem;
    color: var(--muted);
}

.hint.ok {
    color: #3a7d5c;
}

.hint.warn {
    color: #b7791f;
}

.spinner {
    width: 1rem;
    height: 1rem;
    border: 2px solid rgb(255 255 255 / 0.4);
    border-top-color: #fff;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
}

@keyframes spin {
    to {
        transform: rotate(360deg);
    }
}

@media (prefers-reduced-motion: reduce) {
    .spinner {
        animation-duration: 2s;
    }
}

/* Out of sight but still read by password managers and screen readers */
.visually-hidden {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
}
</style>
