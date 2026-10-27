<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import PageWrapper from '../ui/PageWrapper.vue'
import ConfirmModal from '../ui/ConfirmModal.vue'
import TabBar from '../ui/TabBar.vue'
import ProfileHeader from './ProfileHeader.vue'
import { vEnterNext } from '../ui/enterNext'
import { Countries, byCountryName, countryName } from '../countries/countries'
import { useTravelStore, type Settings } from '../stores/store'
import { TIMEZONES } from './options'
import { LANGUAGES, setLanguage, t, type Language } from '../../i18n'

const store = useTravelStore()
const tab = ref('profile')
const tabs = computed(() => [
    { id: 'profile', label: t('profile.tabProfile') },
    { id: 'preferences', label: t('profile.tabPreferences') },
    { id: 'security', label: t('profile.tabSecurity') },
])
const router = useRouter()

// The store's toast lives in the map panels, under this page; confirm here
const message = ref<{ text: string; error: boolean } | null>(null)
let messageTimer: ReturnType<typeof setTimeout> | undefined

function say(text: string, error = false) {
    message.value = { text, error }
    clearTimeout(messageTimer)
    messageTimer = setTimeout(() => (message.value = null), 3000)
}

// --- Profile -----------------------------------------------------------

const form = ref({
    username: '',
    email: '',
    firstName: '',
    lastName: '',
    displayName: '',
    bio: '',
    language: 'en' as Language,
    timezone: 'UTC',
    homeCountry: '',
})
const profileError = ref('')

function reset() {
    const p = store.profile
    if (!p) return
    form.value = {
        username: p.username,
        email: p.email ?? '',
        firstName: p.firstName,
        lastName: p.lastName,
        displayName: p.displayName,
        bio: p.bio,
        language: p.language,
        timezone: p.timezone,
        homeCountry: p.homeCountry,
    }
}
// Not on every profile change: the colour is saved on its own and must not
// wipe what is typed into the form
watch(
    () => !!store.profile,
    (loaded) => loaded && reset(),
    { immediate: true }
)

// Keep a time zone set elsewhere selectable
const timezones = computed(() =>
    TIMEZONES.includes(form.value.timezone)
        ? TIMEZONES
        : [form.value.timezone, ...TIMEZONES]
)
const countries = computed(() => [...Countries].sort(byCountryName))

async function saveProfile() {
    profileError.value = ''
    const failed = await store.updateProfile(form.value)
    if (failed) profileError.value = failed
    else {
        // Confirm in the language just picked
        await setLanguage(form.value.language)
        reset()
        say(t('profile.saved'))
    }
}

// --- Settings ----------------------------------------------------------

// Saved on every change; the store rolls back if the server rejects it
async function setSetting<K extends keyof Settings>(
    key: K,
    value: Settings[K]
) {
    const failed = await store.saveSettings({ ...store.settings, [key]: value })
    if (failed) say(failed, true)
    else say(t('profile.settingsSaved'))
}

// --- Password ----------------------------------------------------------

const currentPassword = ref('')
const newPassword = ref('')
const passwordError = ref('')

async function changePassword() {
    passwordError.value = ''
    const failed = await store.changePassword(
        currentPassword.value,
        newPassword.value
    )
    if (failed) {
        passwordError.value = failed
        return
    }
    currentPassword.value = ''
    newPassword.value = ''
    say(t('profile.passwordChanged'))
}

// --- Account -----------------------------------------------------------

const confirmDelete = ref(false)

async function logout() {
    const failed = await store.logout()
    if (failed) return say(failed, true)
    router.push({ name: 'home' })
}

async function deleteAccount() {
    confirmDelete.value = false
    const failed = await store.deleteAccount()
    if (failed) {
        say(failed, true)
        return
    }
    router.push({ name: 'home' })
}
</script>

<template>
    <PageWrapper
        :title="t('profile.title')"
        :subtitle="store.profile ? `@${store.profile.username}` : undefined"
    >
        <template #actions>
            <button class="logout" :disabled="!store.canWrite" @click="logout">
                {{ t('app.logout') }}
            </button>
        </template>
        <p
            v-if="message"
            class="toast"
            :class="{ error: message.error }"
            role="status"
        >
            {{ message.text }}
        </p>
        <TabBar v-if="store.profile" v-model="tab" :tabs="tabs" />
        <template v-if="store.profile">
            <template v-if="tab === 'profile'">
                <ProfileHeader @say="say" />
                <section class="panel">
                    <h2>{{ t('profile.aboutYou') }}</h2>
                    <form class="fields" @submit.prevent="saveProfile">
                        <div class="row">
                            <div>
                                <label for="first">{{
                                    t('auth.firstName')
                                }}</label>
                                <input id="first" v-model="form.firstName" />
                            </div>
                            <div>
                                <label for="last">{{
                                    t('auth.lastName')
                                }}</label>
                                <input id="last" v-model="form.lastName" />
                            </div>
                        </div>
                        <label for="display">{{
                            t('profile.displayName')
                        }}</label>
                        <input
                            id="display"
                            v-model="form.displayName"
                            :placeholder="t('profile.displayNameHint')"
                        />
                        <label for="username">{{ t('auth.username') }}</label>
                        <input
                            id="username"
                            v-model="form.username"
                            pattern="[a-z0-9_]{3,30}"
                            :title="t('profile.usernameRule')"
                        />
                        <label for="email">{{ t('auth.email') }}</label>
                        <input id="email" v-model="form.email" type="email" />

                        <label for="bio">{{ t('profile.aboutMe') }}</label>
                        <textarea id="bio" v-model="form.bio" rows="3" />
                        <div class="row">
                            <div>
                                <label for="language">{{
                                    t('common.language')
                                }}</label>
                                <select id="language" v-model="form.language">
                                    <option
                                        v-for="l in LANGUAGES"
                                        :key="l.code"
                                        :value="l.code"
                                    >
                                        {{ l.name }}
                                    </option>
                                </select>
                            </div>
                            <div>
                                <label for="timezone">{{
                                    t('profile.timezone')
                                }}</label>
                                <select id="timezone" v-model="form.timezone">
                                    <option
                                        v-for="tz in timezones"
                                        :key="tz"
                                        :value="tz"
                                    >
                                        {{ tz.replaceAll('_', ' ') }}
                                    </option>
                                </select>
                            </div>
                        </div>
                        <label for="home-country">{{
                            t('profile.homeCountry')
                        }}</label>
                        <select id="home-country" v-model="form.homeCountry">
                            <option value="">{{ t('common.notSet') }}</option>
                            <option
                                v-for="c in countries"
                                :key="c.alpha3"
                                :value="c.alpha3"
                            >
                                {{ countryName(c) }}
                            </option>
                        </select>
                        <p v-if="profileError" class="error" role="alert">
                            {{ profileError }}
                        </p>
                        <button
                            class="btn"
                            type="submit"
                            :disabled="!store.canWrite"
                        >
                            {{ t('profile.saveProfile') }}
                        </button>
                    </form>
                </section>
            </template>
            <template v-else-if="tab === 'preferences'">
                <section class="panel">
                    <h2>{{ t('profile.settings') }}</h2>
                    <div class="fields">
                        <label for="unit">{{ t('profile.distances') }}</label>
                        <select
                            id="unit"
                            :disabled="!store.canWrite"
                            :value="store.settings.distanceUnit"
                            @change="
                                setSetting(
                                    'distanceUnit',
                                    ($event.target as HTMLSelectElement)
                                        .value as Settings['distanceUnit']
                                )
                            "
                        >
                            <option value="km">
                                {{ t('profile.kilometers') }}
                            </option>
                            <option value="mi">{{ t('profile.miles') }}</option>
                        </select>
                        <label for="theme">{{ t('profile.theme') }}</label>
                        <select
                            id="theme"
                            :disabled="!store.canWrite"
                            :value="store.settings.theme"
                            @change="
                                setSetting(
                                    'theme',
                                    ($event.target as HTMLSelectElement)
                                        .value as Settings['theme']
                                )
                            "
                        >
                            <option value="system">
                                {{ t('profile.themeSystem') }}
                            </option>
                            <option value="light">
                                {{ t('profile.themeLight') }}
                            </option>
                            <option value="dark">
                                {{ t('profile.themeDark') }}
                            </option>
                        </select>
                        <label for="view">{{ t('profile.startWith') }}</label>
                        <select
                            id="view"
                            :disabled="!store.canWrite"
                            :value="store.settings.defaultView"
                            @change="
                                setSetting(
                                    'defaultView',
                                    ($event.target as HTMLSelectElement)
                                        .value as Settings['defaultView']
                                )
                            "
                        >
                            <option value="countries">
                                {{ t('app.countries') }}
                            </option>
                            <option value="flights">
                                {{ t('app.flights') }}
                            </option>
                        </select>
                        <label for="default-visibility">{{
                            t('profile.newItems')
                        }}</label>
                        <select
                            id="default-visibility"
                            :disabled="!store.canWrite"
                            :value="store.settings.defaultVisibility"
                            @change="
                                setSetting(
                                    'defaultVisibility',
                                    ($event.target as HTMLSelectElement)
                                        .value as Settings['defaultVisibility']
                                )
                            "
                        >
                            <option value="private">
                                {{ t('common.onlyMe') }}
                            </option>
                            <option value="public">
                                {{ t('common.public') }}
                            </option>
                        </select>
                    </div>
                </section>
            </template>
            <template v-else>
                <section class="panel">
                    <h2>{{ t('profile.password') }}</h2>
                    <form
                        v-enter-next
                        class="fields"
                        @submit.prevent="changePassword"
                    >
                        <label for="current">{{ t('password.current') }}</label>
                        <input
                            id="current"
                            v-model="currentPassword"
                            type="password"
                            autocomplete="current-password"
                            required
                        />
                        <label for="new">{{ t('password.new') }}</label>
                        <input
                            id="new"
                            v-model="newPassword"
                            type="password"
                            autocomplete="new-password"
                            minlength="10"
                            required
                        />
                        <p v-if="passwordError" class="error" role="alert">
                            {{ passwordError }}
                        </p>
                        <button
                            class="btn"
                            type="submit"
                            :disabled="!store.canWrite"
                        >
                            {{ t('password.change') }}
                        </button>
                    </form>
                </section>
                <section class="panel danger-zone">
                    <h2>{{ t('profile.deleteAccount') }}</h2>
                    <p class="note">{{ t('profile.deleteNote') }}</p>
                    <div class="actions">
                        <button
                            class="btn danger"
                            :disabled="!store.canWrite"
                            @click="confirmDelete = true"
                        >
                            {{ t('profile.deleteAccount') }}
                        </button>
                    </div>
                </section>
            </template>
        </template>

        <!-- Written by the build (licenses.ts); a plain link, so it opens
             outside the app and also works offline from the precache -->
        <p class="legal">
            <a href="/third-party-licenses.txt" target="_blank" rel="noopener">
                {{ t('profile.licenses') }}
            </a>
        </p>

        <ConfirmModal
            v-if="confirmDelete"
            :title="t('profile.deleteAccount')"
            :message="t('profile.deleteConfirm')"
            :confirm-label="t('common.delete')"
            @confirm="deleteAccount"
            @cancel="confirmDelete = false"
        />
    </PageWrapper>
</template>

<style src="../ui/panel.css"></style>
<style scoped>
.legal {
    max-width: 720px;
    margin: 1.5rem auto 0;
    font-size: 0.78rem;
    text-align: center;
}

.legal a {
    display: inline-block;
    padding: 0.6rem 0.4rem;
    color: var(--muted);
}

.logout {
    min-height: var(--touch);
    padding: 0.4rem 0.9rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    background: transparent;
    font: inherit;
    font-size: 0.8rem;
    color: var(--text);
    cursor: pointer;
}

.logout:disabled {
    opacity: 0.5;
}

@media (pointer: fine) {
    .logout {
        min-height: 0;
    }
}

.logout:hover {
    background: var(--chip);
}
</style>
