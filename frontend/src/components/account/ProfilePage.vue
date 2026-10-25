<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import PageWrapper from '../ui/PageWrapper.vue'
import ConfirmModal from '../ui/ConfirmModal.vue'
import TabBar from '../ui/TabBar.vue'
import ProfileHeader from './ProfileHeader.vue'
import { vEnterNext } from '../ui/enterNext'
import { Countries } from '../countries/countries'
import { useTravelStore, type Settings } from '../stores/store'
import { LANGUAGES, TIMEZONES } from './options'

const store = useTravelStore()
const tab = ref('profile')
const tabs = [
    { id: 'profile', label: 'Profile' },
    { id: 'preferences', label: 'Preferences' },
    { id: 'security', label: 'Security' },
]
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
    language: 'en',
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

// Keep a language or time zone set elsewhere selectable
const languages = computed(() =>
    LANGUAGES.some((l) => l.code === form.value.language)
        ? LANGUAGES
        : [
              ...LANGUAGES,
              { code: form.value.language, name: form.value.language },
          ]
)
const timezones = computed(() =>
    TIMEZONES.includes(form.value.timezone)
        ? TIMEZONES
        : [form.value.timezone, ...TIMEZONES]
)
const countries = [...Countries].sort((a, b) => a.name.localeCompare(b.name))

async function saveProfile() {
    profileError.value = ''
    const failed = await store.updateProfile(form.value)
    if (failed) profileError.value = failed
    else {
        reset()
        say('Profile saved')
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
    else say('Settings saved')
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
    say('Password changed; other devices are logged out')
}

// --- Account -----------------------------------------------------------

const confirmDelete = ref(false)

async function logout() {
    await store.logout()
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
        title="Account"
        :subtitle="store.profile ? `@${store.profile.username}` : undefined"
    >
        <template #actions>
            <button class="logout" @click="logout">Log out</button>
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
                    <h2>About you</h2>
                    <form class="fields" @submit.prevent="saveProfile">
                        <div class="row">
                            <div>
                                <label for="first">First name</label>
                                <input id="first" v-model="form.firstName" />
                            </div>
                            <div>
                                <label for="last">Last name</label>
                                <input id="last" v-model="form.lastName" />
                            </div>
                        </div>
                        <label for="display">Display name</label>
                        <input
                            id="display"
                            v-model="form.displayName"
                            placeholder="Shown on your public profile"
                        />
                        <label for="username">Username</label>
                        <input
                            id="username"
                            v-model="form.username"
                            pattern="[a-z0-9_]{3,30}"
                            title="3–30 lowercase letters, digits or _"
                        />
                        <label for="email">Email</label>
                        <input id="email" v-model="form.email" type="email" />

                        <label for="bio">About me</label>
                        <textarea id="bio" v-model="form.bio" rows="3" />
                        <div class="row">
                            <div>
                                <label for="language">Language</label>
                                <select id="language" v-model="form.language">
                                    <option
                                        v-for="l in languages"
                                        :key="l.code"
                                        :value="l.code"
                                    >
                                        {{ l.name }}
                                    </option>
                                </select>
                            </div>
                            <div>
                                <label for="timezone">Time zone</label>
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
                        <label for="home-country">Home country</label>
                        <select id="home-country" v-model="form.homeCountry">
                            <option value="">Not set</option>
                            <option
                                v-for="c in countries"
                                :key="c.alpha3"
                                :value="c.alpha3"
                            >
                                {{ c.name }}
                            </option>
                        </select>
                        <p v-if="profileError" class="error" role="alert">
                            {{ profileError }}
                        </p>
                        <button class="btn" type="submit">Save profile</button>
                    </form>
                </section>
            </template>
            <template v-else-if="tab === 'preferences'">
                <section class="panel">
                    <h2>Settings</h2>
                    <div class="fields">
                        <label for="unit">Distances in</label>
                        <select
                            id="unit"
                            :value="store.settings.distanceUnit"
                            @change="
                                setSetting(
                                    'distanceUnit',
                                    ($event.target as HTMLSelectElement)
                                        .value as Settings['distanceUnit']
                                )
                            "
                        >
                            <option value="km">Kilometers</option>
                            <option value="mi">Miles</option>
                        </select>
                        <label for="theme">Theme</label>
                        <select
                            id="theme"
                            :value="store.settings.theme"
                            @change="
                                setSetting(
                                    'theme',
                                    ($event.target as HTMLSelectElement)
                                        .value as Settings['theme']
                                )
                            "
                        >
                            <option value="system">Like the system</option>
                            <option value="light">Light</option>
                            <option value="dark">Dark</option>
                        </select>
                        <label for="view">Start with</label>
                        <select
                            id="view"
                            :value="store.settings.defaultView"
                            @change="
                                setSetting(
                                    'defaultView',
                                    ($event.target as HTMLSelectElement)
                                        .value as Settings['defaultView']
                                )
                            "
                        >
                            <option value="countries">Countries</option>
                            <option value="flights">Flights</option>
                        </select>
                        <label for="default-visibility"
                            >New countries and flights</label
                        >
                        <select
                            id="default-visibility"
                            :value="store.settings.defaultVisibility"
                            @change="
                                setSetting(
                                    'defaultVisibility',
                                    ($event.target as HTMLSelectElement)
                                        .value as Settings['defaultVisibility']
                                )
                            "
                        >
                            <option value="private">🔒 Only me</option>
                            <option value="public">🌐 Public</option>
                        </select>
                    </div>
                </section>
            </template>
            <template v-else>
                <section class="panel">
                    <h2>Password</h2>
                    <form
                        v-enter-next
                        class="fields"
                        @submit.prevent="changePassword"
                    >
                        <label for="current">Current password</label>
                        <input
                            id="current"
                            v-model="currentPassword"
                            type="password"
                            autocomplete="current-password"
                            required
                        />
                        <label for="new">New password</label>
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
                        <button class="btn" type="submit">
                            Change password
                        </button>
                    </form>
                </section>
                <section class="panel danger-zone">
                    <h2>Delete account</h2>
                    <p class="note">
                        Removes your account with all countries, flights and
                        photos.
                    </p>
                    <div class="actions">
                        <button
                            class="btn danger"
                            @click="confirmDelete = true"
                        >
                            Delete account
                        </button>
                    </div>
                </section>
            </template>
        </template>

        <ConfirmModal
            v-if="confirmDelete"
            title="Delete account"
            message="This deletes your account with all countries, flights and photos. It cannot be undone."
            confirm-label="Delete"
            @confirm="deleteAccount"
            @cancel="confirmDelete = false"
        />
    </PageWrapper>
</template>

<style src="../ui/panel.css"></style>
<style scoped>
.logout {
    padding: 0.4rem 0.9rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    background: transparent;
    font: inherit;
    font-size: 0.8rem;
    color: var(--text);
    cursor: pointer;
}

.logout:hover {
    background: var(--chip);
}
</style>
