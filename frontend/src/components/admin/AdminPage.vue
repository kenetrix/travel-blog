<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import PageWrapper from '../ui/PageWrapper.vue'
import LegalFooter from '../ui/LegalFooter.vue'
import ConfirmModal from '../ui/ConfirmModal.vue'
import TabBar from '../ui/TabBar.vue'
import OidcSettings from './OidcSettings.vue'
import { api, problemMessage, type Schemas } from '../../api/client'
import { useTravelStore } from '../stores/store'
import { t, type MessageKey } from '../../i18n'
import { collator, formatDate } from '../../i18n/format'

type AdminUser = Schemas['AdminUser']
type Role = Schemas['Role']

// A new role in the API fails to compile until it is listed here
const ROLES: Record<Role, { label: MessageKey; note: MessageKey }> = {
    user: { label: 'admin.roleUser', note: 'admin.roleUserNote' },
    admin: { label: 'admin.roleAdmin', note: 'admin.roleAdminNote' },
}

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
// User whose role is being changed, and the role picked for them
const toChangeRole = ref<AdminUser | null>(null)
const newRole = ref<Role>('user')
const query = ref('')

// Administrators first, then by name
const shown = computed(() => {
    const q = query.value.trim().toLowerCase()
    return users.value
        .filter(
            (u) =>
                !q ||
                [u.username, u.displayName, u.email ?? ''].some((f) =>
                    f.toLowerCase().includes(q)
                )
        )
        .sort(
            (a, b) =>
                Number(b.role === 'admin') - Number(a.role === 'admin') ||
                collator().compare(a.displayName, b.displayName)
        )
})

function isMe(user: AdminUser) {
    return user.username === store.profile?.username
}

function initial(user: AdminUser) {
    return (user.displayName || user.username).charAt(0).toUpperCase()
}

// A stable colour per user, so avatars are told apart at a glance
function hue(user: AdminUser) {
    let h = 0
    for (const c of user.username) h = (h * 31 + c.charCodeAt(0)) % 360
    return h
}

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

// User whose action menu is open
const menuFor = ref<string | null>(null)

function toggleMenu(user: AdminUser) {
    menuFor.value = menuFor.value === user.id ? null : user.id
}

// Closes the menu, then runs the chosen action
function pick(action: () => void) {
    menuFor.value = null
    action()
}

function closeMenu(event: Event) {
    if (event instanceof KeyboardEvent && event.key !== 'Escape') return
    if (event.target instanceof Element && event.target.closest('.menu-wrap')) {
        return
    }
    menuFor.value = null
}

onMounted(() => {
    window.addEventListener('click', closeMenu)
    window.addEventListener('keydown', closeMenu)
})
onBeforeUnmount(() => {
    window.removeEventListener('click', closeMenu)
    window.removeEventListener('keydown', closeMenu)
})

function startRoleChange(user: AdminUser) {
    toChangeRole.value = user
    newRole.value = user.role
}

async function changeRole() {
    const user = toChangeRole.value
    const role = newRole.value
    toChangeRole.value = null
    if (!user || role === user.role) return
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
        t('admin.roleChanged', {
            name: data.username,
            role: t(ROLES[data.role].label),
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
                    <div class="head">
                        <h2>{{ t('admin.users') }}</h2>
                        <span v-if="loaded" class="count">{{
                            users.length
                        }}</span>
                    </div>
                    <input
                        v-if="users.length > 5"
                        v-model="query"
                        type="search"
                        class="search"
                        :placeholder="t('admin.search')"
                        :aria-label="t('admin.search')"
                    />
                    <ul class="users">
                        <li
                            v-for="user in shown"
                            :key="user.id"
                            :class="{ open: resetting === user.id }"
                        >
                            <div class="user">
                                <span
                                    class="avatar"
                                    :style="{ '--hue': hue(user) }"
                                    aria-hidden="true"
                                >
                                    {{ initial(user) }}
                                </span>
                                <div class="who">
                                    <div class="name">
                                        <strong>{{ user.displayName }}</strong>
                                        <span
                                            class="badge role"
                                            :class="`role-${user.role}`"
                                        >
                                            {{ t(ROLES[user.role].label) }}
                                        </span>
                                        <span
                                            v-if="isMe(user)"
                                            class="badge me"
                                        >
                                            {{ t('admin.you') }}
                                        </span>
                                        <span
                                            v-if="user.mustChangePassword"
                                            class="badge warn"
                                        >
                                            {{ t('admin.mustChange') }}
                                        </span>
                                    </div>
                                    <span class="muted">
                                        @{{ user.username }}
                                        <template v-if="user.email">
                                            · {{ user.email }}
                                        </template>
                                    </span>
                                    <span class="muted joined">
                                        {{
                                            t('admin.joined', {
                                                date: formatDate(
                                                    user.createdAt
                                                ),
                                            })
                                        }}
                                    </span>
                                </div>
                                <div v-if="!isMe(user)" class="menu-wrap">
                                    <button
                                        type="button"
                                        class="menu-button"
                                        aria-haspopup="menu"
                                        :aria-expanded="menuFor === user.id"
                                        :aria-label="
                                            t('admin.actionsFor', {
                                                name: user.username,
                                            })
                                        "
                                        :title="
                                            store.canWrite
                                                ? t('admin.actionsFor', {
                                                      name: user.username,
                                                  })
                                                : t('common.offlineHint')
                                        "
                                        :disabled="!store.canWrite"
                                        @click="toggleMenu(user)"
                                    >
                                        <svg
                                            viewBox="0 0 24 24"
                                            width="16"
                                            height="16"
                                            fill="none"
                                            stroke="currentColor"
                                            stroke-width="2"
                                            stroke-linecap="round"
                                            stroke-linejoin="round"
                                            aria-hidden="true"
                                        >
                                            <path d="M12 20h9" />
                                            <path
                                                d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"
                                            />
                                        </svg>
                                    </button>
                                    <div
                                        v-if="menuFor === user.id"
                                        class="menu"
                                        role="menu"
                                    >
                                        <button
                                            type="button"
                                            role="menuitem"
                                            @click="
                                                pick(() =>
                                                    startRoleChange(user)
                                                )
                                            "
                                        >
                                            {{ t('admin.changeRole') }}
                                        </button>
                                        <button
                                            type="button"
                                            role="menuitem"
                                            @click="
                                                pick(() => startReset(user))
                                            "
                                        >
                                            {{ t('admin.resetPassword') }}
                                        </button>
                                        <button
                                            type="button"
                                            role="menuitem"
                                            class="danger"
                                            @click="
                                                pick(() => (toDelete = user))
                                            "
                                        >
                                            {{ t('common.delete') }}
                                        </button>
                                    </div>
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
                                    type="button"
                                    class="btn secondary"
                                    @click="startReset(user)"
                                >
                                    {{ t('common.cancel') }}
                                </button>
                                <button
                                    type="submit"
                                    class="btn"
                                    :disabled="!store.canWrite"
                                >
                                    {{ t('admin.set') }}
                                </button>
                            </form>
                        </li>
                        <li v-if="loaded && !shown.length" class="empty">
                            {{ t('admin.noMatch') }}
                        </li>
                    </ul>
                </section>
            </template>
            <template v-else>
                <section class="panel">
                    <h2>{{ t('admin.registration') }}</h2>
                    <label class="check toggle">
                        <span>{{ t('admin.letVisitors') }}</span>
                        <input
                            type="checkbox"
                            role="switch"
                            class="switch"
                            :checked="registrationOpen"
                            :disabled="!loaded || !store.canWrite"
                            @change="
                                setRegistration(
                                    ($event.target as HTMLInputElement).checked
                                )
                            "
                        />
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
        <ConfirmModal
            v-if="toChangeRole"
            :title="t('admin.changeRole')"
            :message="t('admin.changeRoleOf', { name: toChangeRole.username })"
            :confirm-label="t('admin.changeRole')"
            :confirm-disabled="newRole === toChangeRole.role"
            @confirm="changeRole"
            @cancel="toChangeRole = null"
        >
            <div
                class="roles"
                role="radiogroup"
                :aria-label="t('admin.changeRole')"
            >
                <label
                    v-for="(info, role) in ROLES"
                    :key="role"
                    class="role-option"
                    :class="{ on: newRole === role }"
                >
                    <input v-model="newRole" type="radio" :value="role" />
                    <span>
                        <strong>
                            {{ t(info.label) }}
                            <span
                                v-if="role === toChangeRole.role"
                                class="current"
                            >
                                · {{ t('admin.currentRole') }}
                            </span>
                        </strong>
                        <small>{{ t(info.note) }}</small>
                    </span>
                </label>
            </div>
        </ConfirmModal>
    </PageWrapper>
</template>

<style src="../ui/panel.css"></style>
<style scoped>
.head {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.8rem;
}

.panel .head h2 {
    margin: 0;
}

.count {
    padding: 0.05rem 0.5rem;
    border-radius: 1rem;
    background: var(--chip);
    font-size: 0.72rem;
    font-weight: 600;
    color: var(--muted);
}

.panel input.search {
    margin-bottom: 0.8rem;
}

.users {
    list-style: none;
    margin: 0 -0.5rem;
    padding: 0;
}

.users li {
    padding: 0.75rem 0.5rem;
    border-radius: 0.6rem;
    transition: background 0.15s;
}

.users li + li {
    border-top: 1px solid var(--line);
}

.users li:hover,
.users li.open {
    background: var(--line-faint);
}

.users li.empty {
    font-size: 0.8rem;
    color: var(--muted);
}

.user {
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: 0.6rem 0.8rem;
}

.avatar {
    flex: none;
    display: grid;
    place-items: center;
    width: 2.4rem;
    height: 2.4rem;
    border-radius: 50%;
    background: hsl(var(--hue) 40% 38%);
    color: #fff;
    font-weight: 600;
}

.who {
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
    min-width: 0;
}

.name {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.35rem;
    font-size: 0.9rem;
}

.name strong {
    font-weight: 600;
    overflow-wrap: anywhere;
}

.badge {
    padding: 0.05rem 0.45rem;
    border-radius: 0.6rem;
    font-size: 0.66rem;
    font-weight: 600;
}

.badge.role {
    background: var(--chip);
    color: var(--muted);
}

.badge.role-admin {
    background: #6b5bd233;
    color: #7a6ae0;
}

.badge.me {
    background: #3a7d5c33;
    color: #4a9a72;
}

.badge.warn {
    background: #c9871f33;
    color: #c4851c;
}

.muted {
    font-size: 0.75rem;
    color: var(--muted);
    overflow-wrap: anywhere;
}

.joined {
    font-size: 0.7rem;
}

.menu-wrap {
    position: relative;
}

.menu-button {
    display: grid;
    place-items: center;
    width: var(--touch);
    height: var(--touch);
    padding: 0;
    border: none;
    border-radius: 0.5rem;
    background: transparent;
    color: var(--muted);
    cursor: pointer;
}

@media (pointer: fine) {
    .menu-button {
        width: 32px;
        height: 32px;
    }
}

.menu-button:hover:not(:disabled),
.menu-button[aria-expanded='true'] {
    background: var(--line-faint);
    color: var(--text);
}

.menu-button:disabled {
    opacity: 0.5;
    cursor: default;
}

.menu {
    position: absolute;
    top: calc(100% + 0.25rem);
    right: 0;
    z-index: 5;
    display: flex;
    flex-direction: column;
    min-width: 11rem;
    padding: 0.3rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.6rem;
    background: var(--raised);
    box-shadow: 0 8px 24px var(--line-strong);
}

.menu button {
    min-height: var(--touch);
    padding: 0.4rem 0.7rem;
    border: none;
    border-radius: 0.4rem;
    background: transparent;
    color: var(--text);
    font: inherit;
    font-size: 0.82rem;
    text-align: left;
    white-space: nowrap;
    cursor: pointer;
}

@media (pointer: fine) {
    .menu button {
        min-height: 0;
    }
}

.menu button:hover {
    background: var(--line-faint);
}

.menu button.danger {
    color: #e0675a;
}

.menu button.danger:hover {
    background: #c0392b22;
}

/* The dialog is teleported to body, outside the panel styles */
.roles {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
}

.role-option {
    display: flex;
    align-items: flex-start;
    gap: 0.6rem;
    padding: 0.6rem 0.7rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.6rem;
    cursor: pointer;
}

.role-option.on {
    border-color: #3a7d5c;
    background: #3a7d5c1a;
}

.role-option input {
    margin: 0.15rem 0 0;
    accent-color: #3a7d5c;
}

.role-option > span {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
}

.role-option strong {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text);
}

.role-option .current {
    font-weight: 400;
    color: var(--muted);
}

.role-option small {
    font-size: 0.75rem;
    line-height: 1.35;
    color: var(--muted);
}

.reset {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-top: 0.7rem;
    padding-left: 3.2rem;
}

.panel .reset input {
    flex: 1 1 12rem;
    width: auto;
    min-width: 0;
}
</style>
