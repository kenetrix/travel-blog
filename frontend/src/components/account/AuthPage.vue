<script setup lang="ts">
import { ref, computed, nextTick, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageWrapper from '../ui/PageWrapper.vue'
import { useTravelStore } from '../stores/store'
import { LANGUAGES } from './options'
import { vEnterNext } from '../ui/enterNext'
import { OFFLINE_HINT } from '../ui/offline'

const store = useTravelStore()
const router = useRouter()
const route = useRoute()

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
onMounted(async () => {
    usernameInput.value?.focus()
    const config = await store.oidcConfig()
    if (!config) return
    sso.value = config.enabled ? { name: config.name } : null
    passwordLogin.value = config.passwordLogin || !config.enabled
})

const title = computed(() => {
    if (step.value === 'login') return 'Log in'
    if (step.value === 'register') return 'Create account'
    return 'Log in or sign up'
})

const buttonLabel = computed(() => {
    if (step.value === 'login') return 'Log in'
    if (step.value === 'register') return 'Create account'
    return 'Continue'
})

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
        return
    }
    password.value = ''
    router.push({ name: 'home' })
}

async function changeUsername() {
    step.value = 'username'
    password.value = ''
    error.value = ''
    await nextTick()
    usernameInput.value?.focus()
}
</script>

<template>
    <PageWrapper
        :title="title"
        subtitle="Your countries and flights, on every device"
    >
        <p v-if="!store.canWrite" class="card note offline">
            Logging in needs a connection.
        </p>
        <div v-if="sso" class="card">
            <a
                class="btn sso"
                :class="{ disabled: !store.canWrite }"
                href="/api/auth/oidc/login"
                :aria-disabled="!store.canWrite"
                @click="!store.canWrite && $event.preventDefault()"
            >
                Log in with {{ sso.name }}
            </a>
            <p v-if="!passwordLogin && error" class="error" role="alert">
                {{ error }}
            </p>
            <p v-if="passwordLogin" class="note or">or with a password</p>
        </div>

        <form
            v-if="passwordLogin"
            v-enter-next
            class="card"
            @submit.prevent="submit"
        >
            <label for="username">Username</label>
            <input
                id="username"
                ref="usernameInput"
                v-model="username"
                autocomplete="username"
                pattern="[a-zA-Z0-9_]{3,30}"
                title="3–30 letters, digits or _"
                :readonly="step !== 'username'"
                required
            />
            <p v-if="step === 'login'" class="note">
                Welcome back.
                <button type="button" class="link" @click="changeUsername">
                    Not you?
                </button>
            </p>
            <p v-else-if="step === 'register'" class="note">
                This username is free, so we'll create a new account.
                <button type="button" class="link" @click="changeUsername">
                    Change it
                </button>
            </p>

            <template v-if="step === 'register'">
                <label for="email">Email (optional)</label>
                <input
                    id="email"
                    ref="emailInput"
                    v-model="email"
                    type="email"
                    autocomplete="email"
                />
                <div class="row">
                    <div>
                        <label for="first">First name</label>
                        <input
                            id="first"
                            v-model="firstName"
                            autocomplete="given-name"
                        />
                    </div>
                    <div>
                        <label for="last">Last name</label>
                        <input
                            id="last"
                            v-model="lastName"
                            autocomplete="family-name"
                        />
                    </div>
                </div>
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
            </template>

            <template v-if="step !== 'username'">
                <label for="password">Password</label>
                <input
                    id="password"
                    ref="passwordInput"
                    v-model="password"
                    type="password"
                    :autocomplete="
                        step === 'login' ? 'current-password' : 'new-password'
                    "
                    :minlength="step === 'register' ? 10 : undefined"
                    required
                />
                <p v-if="step === 'register'" class="note">
                    At least 10 characters.
                </p>
            </template>

            <p v-if="error" class="error" role="alert">{{ error }}</p>

            <button
                class="btn"
                type="submit"
                :disabled="busy || !store.canWrite"
                :title="store.canWrite ? undefined : OFFLINE_HINT"
            >
                {{ buttonLabel }}
            </button>
        </form>
    </PageWrapper>
</template>

<style scoped>
.card {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    max-width: 360px;
    margin: 1rem auto;
}

label {
    margin-top: 0.4rem;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--muted);
}

input,
select {
    box-sizing: border-box;
    width: 100%;
    padding: 0.5rem 0.6rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    background: var(--raised);
    color: var(--text);
    font: inherit;
    font-size: var(--field-font);
}

.row {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
}

@media (min-width: 600px) {
    .row {
        flex-direction: row;
        gap: 0.6rem;
    }
}

.row > div {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
}

.note {
    margin: 0;
    font-size: 0.72rem;
    color: var(--muted);
}

.link {
    padding: 0;
    border: none;
    background: none;
    color: inherit;
    font: inherit;
    text-decoration: underline;
    cursor: pointer;
}

.error {
    margin: 0.4rem 0 0;
    font-size: 0.78rem;
    color: #c0392b;
}

.btn {
    min-height: var(--touch);
    margin-top: 0.9rem;
    padding: 0.6rem;
    border: none;
    border-radius: 0.5rem;
    background: #3a7d5c;
    color: #fff;
    font: inherit;
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
}

.sso {
    margin-top: 0;
    text-align: center;
    text-decoration: none;
}

.or {
    text-align: center;
}

.offline {
    color: #8a5a12;
}

.sso.disabled {
    opacity: 0.6;
    cursor: not-allowed;
}

.link {
    min-height: 2rem;
}

.btn:disabled {
    opacity: 0.6;
    cursor: wait;
}
</style>
