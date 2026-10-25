<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import PageWrapper from '../ui/PageWrapper.vue'
import ConfirmModal from '../ui/ConfirmModal.vue'
import { vEnterNext } from '../ui/enterNext'
import AirportSelect from '../flights/AirportSelect.vue'
import { getAirport, type Airport } from '../flights/airports'
import { Countries } from '../countries/countries'
import { useTravelStore, type Settings } from '../stores/store'
import { LANGUAGES, TIMEZONES } from './options'

const store = useTravelStore()
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
const homeAirport = ref<Airport | null>(null)
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
    homeAirport.value = getAirport(p.homeAirport) ?? null
}
watch(() => store.profile, reset, { immediate: true })

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
    const { email, ...rest } = form.value
    const failed = await store.updateProfile({
        ...rest,
        // The local user has no email
        ...(store.isLocalUser ? {} : { email }),
        homeAirport: homeAirport.value?.iata ?? '',
    })
    if (failed) profileError.value = failed
    else say('Profile saved')
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
        title="Profile"
        :subtitle="store.profile ? `@${store.profile.username}` : undefined"
    >
        <p
            v-if="message"
            class="message"
            :class="{ error: message.error }"
            role="status"
        >
            {{ message.text }}
        </p>
        <div v-if="store.profile" class="columns">
            <section>
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
                    <template v-if="!store.isLocalUser">
                        <label for="email">Email</label>
                        <input id="email" v-model="form.email" type="email" />
                    </template>
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
                    <label>Home airport</label>
                    <AirportSelect
                        v-model="homeAirport"
                        placeholder="Search airport or IATA"
                    />
                    <p v-if="profileError" class="error" role="alert">
                        {{ profileError }}
                    </p>
                    <button class="btn" type="submit">Save profile</button>
                </form>
            </section>

            <section>
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
                    <label for="profile-visibility">Public profile</label>
                    <select
                        id="profile-visibility"
                        :value="store.settings.profileVisibility"
                        @change="
                            setSetting(
                                'profileVisibility',
                                ($event.target as HTMLSelectElement)
                                    .value as Settings['profileVisibility']
                            )
                        "
                    >
                        <option value="private">Off</option>
                        <option value="public">On</option>
                    </select>
                    <p class="note">
                        <template
                            v-if="store.settings.profileVisibility === 'public'"
                        >
                            Anyone can see your display name, bio, home country
                            and everything marked 🌐 Public.
                        </template>
                        <template v-else>
                            Nobody else can see anything, even items marked 🌐
                            Public.
                        </template>
                    </p>
                </div>
            </section>

            <section v-if="store.isAdmin">
                <h2>Administration</h2>
                <p class="note">
                    Manage users and decide whether visitors can register.
                </p>
                <RouterLink class="btn secondary" :to="{ name: 'admin' }">
                    Open administration
                </RouterLink>
            </section>

            <section v-if="store.isLocalUser">
                <h2>Account</h2>
                <p class="note">
                    You are using the local development user. Create an account
                    to keep your own travel data.
                </p>
                <RouterLink class="btn secondary" :to="{ name: 'login' }">
                    Log in or register
                </RouterLink>
            </section>

            <section v-else>
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
                    <button class="btn" type="submit">Change password</button>
                </form>

                <h2>Account</h2>
                <div class="fields">
                    <button class="btn secondary" @click="logout">
                        Log out
                    </button>
                    <button class="btn danger" @click="confirmDelete = true">
                        Delete account
                    </button>
                </div>
            </section>
        </div>

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

<style scoped>
.message {
    position: sticky;
    top: 0;
    z-index: 1;
    max-width: 1000px;
    margin: 0 auto 1rem;
    padding: 0.55rem 0.8rem;
    border-radius: 0.5rem;
    background: var(--accent-soft);
    color: var(--text);
    font-size: 0.8rem;
}

.message.error {
    background: #c0392b;
    color: #fff;
}

.columns {
    display: flex;
    flex-wrap: wrap;
    gap: 2.5rem;
    max-width: 1000px;
    margin: 0 auto;
}

section {
    flex: 1 1 280px;
    min-width: 0;
}

h2 {
    margin: 0 0 0.6rem;
    font-family: 'DM Serif Display', serif;
    font-size: 1rem;
    font-weight: 400;
}

section h2 + .fields,
.fields + h2 {
    margin-top: 0;
}

.fields + h2 {
    margin-top: 1.6rem;
}

.fields {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
}

label {
    margin-top: 0.4rem;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--muted);
}

input,
select,
textarea {
    box-sizing: border-box;
    width: 100%;
    padding: 0.5rem 0.6rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    background: var(--raised);
    color: var(--text);
    font: inherit;
    font-size: 0.85rem;
}

textarea {
    resize: vertical;
}

.row {
    display: flex;
    gap: 0.6rem;
}

.row > div {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
}

.note {
    margin: 0.4rem 0 0;
    font-size: 0.75rem;
    line-height: 1.4;
    color: var(--muted);
}

.error {
    margin: 0.4rem 0 0;
    font-size: 0.78rem;
    color: #c0392b;
}

.btn {
    display: block;
    box-sizing: border-box;
    margin-top: 0.8rem;
    padding: 0.55rem;
    border: none;
    border-radius: 0.5rem;
    background: #3a7d5c;
    color: #fff;
    font: inherit;
    font-size: 0.82rem;
    font-weight: 500;
    text-align: center;
    text-decoration: none;
    cursor: pointer;
}

.btn.secondary {
    border: 1px solid var(--line-strong);
    background: var(--raised);
    color: var(--text);
}

.btn.danger {
    margin-top: 0.4rem;
    background: #c0392b;
}
</style>
