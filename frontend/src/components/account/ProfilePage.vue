<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import PageWrapper from '../ui/PageWrapper.vue'
import LegalFooter from '../ui/LegalFooter.vue'
import ConfirmModal from '../ui/ConfirmModal.vue'
import TabBar from '../ui/TabBar.vue'
import SegmentedControl from '../ui/SegmentedControl.vue'
import ProfileHeader from './ProfileHeader.vue'
import { vEnterNext } from '../ui/enterNext'
import { useLeaveGuard } from '../ui/useLeaveGuard'
import { useTimedMessage } from '../ui/useTimedMessage'
import { visibilityOptions } from '../ui/visibility'
import { Countries, byCountryName, countryName } from '../countries/countries'
import { useTravelStore, type Settings } from '../stores/store'
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
const { message, say } = useTimedMessage()

// --- Profile -----------------------------------------------------------

const form = ref({
    username: '',
    email: '',
    firstName: '',
    lastName: '',
    displayName: '',
    bio: '',
    homeCountry: '',
})
const profileError = ref('')

function stored() {
    const p = store.profile
    return {
        username: p?.username ?? '',
        email: p?.email ?? '',
        firstName: p?.firstName ?? '',
        lastName: p?.lastName ?? '',
        displayName: p?.displayName ?? '',
        bio: p?.bio ?? '',
        homeCountry: p?.homeCountry ?? '',
    }
}

function reset() {
    if (store.profile) form.value = stored()
}

const profileDirty = computed(
    () => JSON.stringify(form.value) !== JSON.stringify(stored())
)

// Like the editors: leaving with unsaved profile edits asks first
const { pendingLeave, allowLeave, confirmLeave, cancelLeave } = useLeaveGuard(
    () => profileDirty.value
)
// Not on every profile change: the colour is saved on its own and must not
// wipe what is typed into the form
watch(
    () => !!store.profile,
    (loaded) => loaded && reset(),
    { immediate: true }
)

const countries = computed(() => [...Countries].sort(byCountryName))

async function saveProfile() {
    if (!profileDirty.value) return
    profileError.value = ''
    const failed = await store.updateProfile(form.value)
    if (failed) profileError.value = failed
    else {
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

async function setProfileLanguage(language: Language) {
    const failed = await store.updateProfile({ language })
    if (failed) return say(failed, true)
    // Confirm in the language just picked
    await setLanguage(language)
    say(t('profile.settingsSaved'))
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
    allowLeave()
    router.push({ name: 'home' })
}

async function deleteAccount() {
    confirmDelete.value = false
    const failed = await store.deleteAccount()
    if (failed) {
        say(failed, true)
        return
    }
    allowLeave()
    router.push({ name: 'home' })
}
</script>

<template>
    <PageWrapper
        :title="t('profile.title')"
        :subtitle="store.profile ? `@${store.profile.username}` : undefined"
    >
        <template #actions>
            <button
                class="btn secondary"
                :disabled="!store.canWrite"
                @click="logout"
            >
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
                    <form
                        v-enter-next
                        class="fields"
                        @submit.prevent="saveProfile"
                    >
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
                            :disabled="!store.canWrite || !profileDirty"
                        >
                            {{ t('profile.saveProfile') }}
                        </button>
                    </form>
                </section>
            </template>
            <template v-else-if="tab === 'preferences'">
                <section class="panel">
                    <h2>{{ t('profile.appearance') }}</h2>
                    <div class="pref">
                        <span class="pref-label">{{ t('profile.theme') }}</span>
                        <SegmentedControl
                            :model-value="store.settings.theme"
                            :label="t('profile.theme')"
                            :disabled="!store.canWrite"
                            :options="[
                                {
                                    value: 'system',
                                    label: '💻 ' + t('profile.themeSystem'),
                                },
                                {
                                    value: 'light',
                                    label: '☀️ ' + t('profile.themeLight'),
                                },
                                {
                                    value: 'dark',
                                    label: '🌙 ' + t('profile.themeDark'),
                                },
                            ]"
                            @update:model-value="setSetting('theme', $event)"
                        />
                    </div>
                    <div class="pref">
                        <label class="pref-label" for="language">{{
                            t('common.language')
                        }}</label>
                        <select
                            id="language"
                            :disabled="!store.canWrite"
                            :value="store.profile.language"
                            @change="
                                setProfileLanguage(
                                    ($event.target as HTMLSelectElement)
                                        .value as Language
                                )
                            "
                        >
                            <option
                                v-for="l in LANGUAGES"
                                :key="l.code"
                                :value="l.code"
                            >
                                {{ l.name }}
                            </option>
                        </select>
                    </div>
                    <div class="pref">
                        <span class="pref-label">{{
                            t('profile.distances')
                        }}</span>
                        <SegmentedControl
                            :model-value="store.settings.distanceUnit"
                            :label="t('profile.distances')"
                            :disabled="!store.canWrite"
                            :options="[
                                { value: 'km', label: t('profile.kilometers') },
                                { value: 'mi', label: t('profile.miles') },
                            ]"
                            @update:model-value="
                                setSetting('distanceUnit', $event)
                            "
                        />
                    </div>
                </section>
                <section class="panel">
                    <h2>{{ t('profile.defaults') }}</h2>
                    <div class="pref">
                        <span class="pref-label">{{
                            t('profile.startWith')
                        }}</span>
                        <SegmentedControl
                            :model-value="store.settings.defaultView"
                            :label="t('profile.startWith')"
                            :disabled="!store.canWrite"
                            :options="[
                                {
                                    value: 'countries',
                                    label: '🌍 ' + t('app.countries'),
                                },
                                {
                                    value: 'flights',
                                    label: '✈️ ' + t('app.flights'),
                                },
                            ]"
                            @update:model-value="
                                setSetting('defaultView', $event)
                            "
                        />
                    </div>
                    <div class="pref">
                        <span class="pref-label">{{
                            t('profile.newItems')
                        }}</span>
                        <SegmentedControl
                            :model-value="store.settings.defaultVisibility"
                            :label="t('profile.newItems')"
                            :disabled="!store.canWrite"
                            :options="visibilityOptions()"
                            @update:model-value="
                                setSetting('defaultVisibility', $event)
                            "
                        />
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

        <LegalFooter />

        <ConfirmModal
            v-if="confirmDelete"
            :title="t('profile.deleteAccount')"
            :message="t('profile.deleteConfirm')"
            :confirm-label="t('common.delete')"
            danger
            @confirm="deleteAccount"
            @cancel="confirmDelete = false"
        />
        <ConfirmModal
            v-if="pendingLeave"
            :title="t('country.discardTitle')"
            :message="t('country.discardMessage')"
            :confirm-label="t('country.discard')"
            danger
            @confirm="confirmLeave"
            @cancel="cancelLeave"
        />
    </PageWrapper>
</template>

<style src="../ui/panel.css"></style>
<style scoped>
.pref {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem 1rem;
    padding: 0.7rem 0;
    border-top: 1px solid var(--line);
}

.pref:first-of-type {
    padding-top: 0;
    border-top: none;
}

.panel .pref-label {
    margin: 0;
    font-size: 0.85rem;
    color: var(--text);
}

.panel .pref select {
    width: auto;
    min-width: 10rem;
}
</style>
