<script setup lang="ts">
import { ref, watch, watchEffect } from 'vue'
import { RouterView, RouterLink, useRoute, useRouter } from 'vue-router'
import { useTravelStore } from './components/stores/store'
import FlightPanel from './components/flights/FlightPanel.vue'
import 'maplibre-gl/dist/maplibre-gl.css'
import CountryPanel from './components/countries/CountryPanel.vue'
import WorldMap from './components/worldmap/WorldMap.vue'

const store = useTravelStore()
const route = useRoute()
const router = useRouter()

const view = ref<'countries' | 'flights'>('countries')

// Each user starts in the view picked in their settings
watch(
    () => store.profile?.username,
    () => (view.value = store.settings.defaultView)
)

// Without a session (backend without -local-user) only the login page works
watch(
    () => [store.authState, route.name],
    ([state, name]) => {
        if (state === 'anonymous' && name !== 'login') {
            router.replace({ name: 'login' })
        }
    },
    { immediate: true }
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

// Theme colours are defined per data-theme in style.css
watchEffect(() => {
    document.documentElement.dataset.theme = store.settings.theme
})
</script>

<template>
    <div class="wrapper">
        <!-- Map-->
        <WorldMap :view="view" />

        <!-- Top bar -->
        <div class="topbar">
            <span class="logo">Travel Blog</span>
            <div class="toggle-pill">
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
            <RouterLink
                v-if="store.profile"
                class="account avatar"
                :to="{ name: 'profile' }"
                :title="`Profile of ${store.profile.displayName}`"
                :aria-label="`Profile of ${store.profile.displayName}`"
            >
                {{
                    (store.profile.displayName || store.profile.username)
                        .charAt(0)
                        .toUpperCase()
                }}
            </RouterLink>
        </div>

        <!-- Country panel -->
        <Transition name="slide">
            <div v-if="view === 'countries'">
                <CountryPanel />
            </div>
        </Transition>

        <!-- Flight panel -->
        <Transition name="slide">
            <div v-if="view === 'flights'">
                <FlightPanel />
            </div>
        </Transition>

        <!-- Full-page routes (e.g. new trip) -->
        <RouterView />
    </div>
</template>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500&family=DM+Serif+Display&display=swap');

.wrapper {
    position: relative;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    font-family: 'DM Sans', sans-serif;
}

/* Top bar */
.topbar {
    position: absolute;
    top: 1.2rem;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 1rem;
    background: var(--panel);
    backdrop-filter: blur(8px);
    border: 1px solid var(--line);
    border-radius: 2rem;
    padding: 0.45rem 0.45rem 0.45rem 1.2rem;
    z-index: 10;
}

.toggle-pill {
    display: flex;
    background: var(--line-faint);
    border-radius: 2rem;
    padding: 3px;
}

.toggle-pill button {
    border: none;
    background: transparent;
    border-radius: 2rem;
    padding: 0.3rem 0.85rem;
    font-family: 'DM Sans', sans-serif;
    font-size: 0.78rem;
    font-weight: 500;
    color: var(--muted);
    cursor: pointer;
    transition:
        background 0.15s,
        color 0.15s;
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

.avatar {
    display: grid;
    place-items: center;
    width: 1.9rem;
    height: 1.9rem;
    border-radius: 50%;
    background: #3a7d5c;
    color: #fff;
}

.avatar:hover {
    color: #fff;
    background: #2f6a4e;
}

.slide-enter-active,
.slide-leave-active {
    transition:
        opacity 0.2s,
        transform 0.2s;
}
.slide-enter-from,
.slide-leave-to {
    opacity: 0;
    transform: translateY(-6px);
}
</style>
