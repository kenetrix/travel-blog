<script setup lang="ts">
import { ref, watch, watchEffect } from 'vue'
import { RouterView, RouterLink, useRoute, useRouter } from 'vue-router'
import { useTravelStore } from './components/stores/store'
import FlightPanel from './components/flights/FlightPanel.vue'
import CountryPanel from './components/countries/CountryPanel.vue'
import WorldMap from './components/worldmap/WorldMap.vue'
import OfflineBanner from './components/ui/OfflineBanner.vue'
import { OFFLINE_HINT } from './components/ui/offline'

const store = useTravelStore()
const route = useRoute()
const router = useRouter()

const view = ref<'countries' | 'flights'>('countries')

// Each user starts in the view picked in their settings
watch(
    () => store.profile?.username,
    () => (view.value = store.settings.defaultView)
)

// Without a session visitors see the bare map; every other page needs a login
watch(
    () => [store.authState, route.name],
    ([state, name]) => {
        if (state === 'anonymous' && name !== 'home' && name !== 'login') {
            router.replace({ name: 'login' })
        }
    },
    { immediate: true }
)

// A session that expired (e.g. while offline) ends on the login page; a
// deliberate logout moves on to the map afterwards
watch(
    () => store.authState,
    (state, before) => {
        if (before === 'ready' && state === 'anonymous') {
            router.replace({ name: 'login' })
        }
    }
)

// A generated or reset password has to be replaced before anything else
watch(
    () => [store.mustChangePassword, route.name],
    ([must, name]) => {
        if (must && name !== 'password') {
            router.replace({ name: 'password' })
        }
    },
    { immediate: true }
)

async function logout() {
    if (await store.logout()) return
    router.push({ name: 'home' })
}

// Theme colours are defined per data-theme in style.css
watchEffect(() => {
    document.documentElement.dataset.theme = store.settings.theme
})
</script>

<template>
    <div class="wrapper">
        <!-- Map-->
        <WorldMap :view="view" />

        <!-- One bar on phones and tablets; on desktops its parts float apart -->
        <header class="appbar">
            <div class="topbar">
                <span class="logo">
                    <span class="logo-icon" aria-hidden="true">✈</span>
                    <span class="logo-text">Travel Blog</span>
                </span>
                <div v-if="store.profile" class="toggle-pill">
                    <button
                        :class="{ active: view === 'countries' }"
                        @click="view = 'countries'"
                    >
                        Countries
                    </button>
                    <button
                        :class="{ active: view === 'flights' }"
                        @click="view = 'flights'"
                    >
                        Flights
                    </button>
                </div>
            </div>

            <!-- Account, administration and log out -->
            <div class="useractions">
                <RouterLink
                    v-if="store.authState === 'anonymous'"
                    class="account login"
                    :to="{ name: 'login' }"
                >
                    Log in
                </RouterLink>
                <RouterLink
                    v-if="store.profile"
                    class="account avatar"
                    :to="{ name: 'account' }"
                    :title="`Account of ${store.profile.displayName}`"
                    :aria-label="`Account of ${store.profile.displayName}`"
                >
                    <span
                        class="circle"
                        :style="
                            store.profile.color
                                ? { background: store.profile.color }
                                : undefined
                        "
                    >
                        {{
                            (
                                store.profile.displayName ||
                                store.profile.username
                            )
                                .charAt(0)
                                .toUpperCase()
                        }}
                    </span>
                </RouterLink>
                <RouterLink
                    v-if="store.isAdmin"
                    class="account avatar admin"
                    :to="{ name: 'admin' }"
                    title="Administration"
                    aria-label="Administration"
                >
                    <span class="circle">
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
                            <path
                                d="M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6l8-3z"
                            />
                            <path d="M9 12l2 2 4-4" />
                        </svg>
                    </span>
                </RouterLink>
                <button
                    v-if="store.profile"
                    class="account avatar admin"
                    :title="store.canWrite ? 'Log out' : OFFLINE_HINT"
                    aria-label="Log out"
                    :disabled="!store.canWrite"
                    @click="logout"
                >
                    <span class="circle">
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
                            <path d="M9 4H5a1 1 0 0 0-1 1v14a1 1 0 0 0 1 1h4" />
                            <path d="M16 8l4 4-4 4M20 12H9" />
                        </svg>
                    </span>
                </button>
            </div>
        </header>

        <OfflineBanner />

        <div v-if="store.toast" class="toast" role="status">
            {{ store.toast }}
        </div>

        <!-- Country panel -->
        <Transition name="fade">
            <CountryPanel v-if="store.profile && view === 'countries'" />
        </Transition>

        <!-- Flight panel -->
        <Transition name="fade">
            <FlightPanel v-if="store.profile && view === 'flights'" />
        </Transition>

        <!-- Full-page routes (e.g. new trip) -->
        <RouterView />
    </div>
</template>

<style scoped>
.wrapper {
    position: relative;
    width: 100vw;
    height: 100vh;
    height: 100dvh;
    overflow: hidden;
    font-family: 'DM Sans', sans-serif;
}

/* Phones and tablets: one bar across the top */
.appbar {
    position: absolute;
    top: calc(var(--safe-top) + 0.5rem);
    left: calc(var(--safe-left) + var(--edge));
    right: calc(var(--safe-right) + var(--edge));
    z-index: 20;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.4rem;
    box-sizing: border-box;
    height: var(--appbar-h);
    padding: 0 0.1rem 0 0.3rem;
    background: var(--panel);
    backdrop-filter: blur(8px);
    border: 1px solid var(--line);
    border-radius: 2rem;
}

.topbar,
.useractions {
    display: flex;
    align-items: center;
    min-width: 0;
}

.topbar {
    gap: 0.5rem;
}

.logo {
    display: flex;
    align-items: center;
    font-family: 'DM Serif Display', serif;
    white-space: nowrap;
    color: var(--text);
}

.logo-icon {
    display: none;
    padding-left: 0.4rem;
    font-size: 1.1rem;
}

.logo-text {
    display: none;
}

@media (min-width: 380px) {
    .logo-icon {
        display: inline;
    }
}

@media (min-width: 600px) {
    .appbar {
        top: calc(var(--safe-top) + 1.2rem);
        padding: 0 0.2rem 0 1rem;
    }

    .topbar {
        gap: 1rem;
    }

    .logo-icon {
        display: none;
    }

    .logo-text {
        display: inline;
    }
}

/* Desktop: centred pill and actions on the right, as separate islands */
@media (min-width: 1024px) {
    .appbar {
        display: contents;
    }

    .topbar,
    .useractions {
        position: absolute;
        top: 1.2rem;
        z-index: 20;
        background: var(--panel);
        backdrop-filter: blur(8px);
        border: 1px solid var(--line);
        border-radius: 2rem;
    }

    .topbar {
        left: 50%;
        transform: translateX(-50%);
        padding: 0.3rem 0.3rem 0.3rem 1.2rem;
    }

    .useractions {
        right: calc(var(--safe-right) + 1.2rem);
        padding: 0 0.1rem;
    }
}

.toggle-pill {
    display: flex;
    background: var(--line-faint);
    border-radius: 2rem;
    padding: 3px;
}

.toggle-pill button {
    min-height: 36px;
    border: none;
    background: transparent;
    border-radius: 2rem;
    padding: 0 0.7rem;
    font-family: 'DM Sans', sans-serif;
    font-size: 0.78rem;
    font-weight: 500;
    color: var(--muted);
    cursor: pointer;
    transition:
        background 0.15s,
        color 0.15s;
}

@media (min-width: 380px) {
    .toggle-pill button {
        padding: 0 0.85rem;
    }
}

.toggle-pill button.active {
    background: var(--raised);
    color: var(--text);
    box-shadow: 0 1px 3px var(--line-strong);
}

.account {
    font-size: 0.78rem;
    font-weight: 500;
    color: var(--muted);
    text-decoration: none;
}

.account:hover {
    color: var(--text);
}

.login {
    display: flex;
    align-items: center;
    min-height: 36px;
    margin: 0 0.2rem;
    padding: 0 0.9rem;
    border-radius: 1.2rem;
    background: #3a7d5c;
    color: #fff;
}

.login:hover {
    color: #fff;
    background: #2f6a4e;
}

/* The button is the touch target; the circle inside is what shows */
.avatar {
    display: grid;
    place-items: center;
    width: var(--touch);
    height: var(--touch);
    padding: 0;
    border: none;
    background: none;
    color: #fff;
    cursor: pointer;
}

.circle {
    display: grid;
    place-items: center;
    width: 2rem;
    height: 2rem;
    border-radius: 50%;
    background: #3a7d5c;
}

.avatar:hover {
    color: #fff;
}

.avatar:hover .circle {
    filter: brightness(0.9);
}

.avatar.admin {
    color: var(--muted);
}

.avatar.admin .circle {
    background: var(--line-faint);
}

.avatar.admin:hover {
    color: var(--text);
}

.avatar.admin:hover .circle {
    background: var(--raised);
    filter: none;
}

.avatar:disabled {
    opacity: 0.4;
}

.toast {
    position: absolute;
    top: calc(var(--safe-top) + var(--appbar-h) + 3.2rem);
    left: 50%;
    z-index: 25;
    transform: translateX(-50%);
    width: max-content;
    max-width: calc(100vw - 2 * var(--edge));
    box-sizing: border-box;
    padding: 0.5rem 0.9rem;
    border-radius: 0.5rem;
    background: #1a1c1e;
    color: #fff;
    font-size: 0.8rem;
}

@media (min-width: 600px) {
    .toast {
        top: calc(var(--safe-top) + var(--appbar-h) + 4rem);
        left: calc(var(--safe-left) + var(--edge) + min(360px, 45vw) + 1rem);
        transform: none;
    }
}

/* Desktop: in the bottom-left corner, where it always was */
@media (min-width: 1024px) {
    .toast {
        top: auto;
        bottom: 1.5rem;
        left: 1.2rem;
        transform: none;
    }
}

.fade-enter-active,
.fade-leave-active {
    transition: opacity 0.2s;
}
.fade-enter-from,
.fade-leave-to {
    opacity: 0;
}
</style>
