<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import { MapLibreProvider } from '../worldmap/MapLibreProvider'
import { useTravelStore } from '../stores/store'
import { distanceKm, formatDistance } from '../flights/distance'
import 'maplibre-gl/dist/maplibre-gl.css'

const store = useTravelStore()
const {
    visited,
    wishlist,
    selectedIso,
    preview,
    flights,
    highlightedFlightId,
    visitedCount,
} = storeToRefs(store)

const EARTH_CIRCUMFERENCE_KM = 40075

const totalKm = computed(() =>
    flights.value.reduce((sum, f) => sum + distanceKm(f.from, f.to), 0)
)
const timesAroundEarth = computed(() =>
    (totalKm.value / EARTH_CIRCUMFERENCE_KM).toFixed(1)
)

const mapContainer = ref<HTMLElement | null>(null)
const props = defineProps<{ view: 'countries' | 'flights' }>()
const provider = new MapLibreProvider()

onMounted(() => {
    if (!mapContainer.value) return
    provider.onReady(syncMap)
    provider.mount(mapContainer.value)

    provider.onCountryClick((iso) => {
        if (props.view !== 'countries') return
        store.selectCountry(iso)
    })
})

onUnmounted(() => provider.destroy())

function syncMap() {
    const shown = new Set(visited.value)
    const wished = new Set(wishlist.value)
    if (preview.value) {
        const { iso, status } = preview.value
        shown.delete(iso)
        wished.delete(iso)
        if (status === 'visited') shown.add(iso)
        else if (status === 'wishlist') wished.add(iso)
    }
    provider.highlightCountries([...shown])
    provider.highlightWishlist([...wished])
    provider.showFlightRoutes(
        flights.value.map((f) => ({
            id: f.id,
            from: f.from,
            to: f.to,
            label: f.label,
        })),
        props.view === 'flights'
    )
    provider.highlightFlight(highlightedFlightId.value)
}

// Re-sync whenever store or view changes
// Zoom to the selected country, whether picked on the map or via search
watch(selectedIso, (iso) => {
    // Defer so the colour change is painted first; computing the bounds of a
    // huge country (e.g. Russia) would otherwise block the main thread
    if (iso) setTimeout(() => provider.focusCountry(iso), 50)
    else provider.focusCountry(null)
})

watch(
    [
        visited,
        wishlist,
        preview,
        flights,
        highlightedFlightId,
        () => props.view,
    ],
    syncMap,
    { deep: true }
)
</script>

<template>
    <div
        ref="mapContainer"
        class="map"
        :class="{ 'no-pointer': view !== 'countries' }"
    />
    <!-- Stats -->
    <div class="stats">
        <template v-if="view === 'countries'">
            <span>
                <span class="dot visited" />{{ visitedCount }} visited
            </span>
            <span>
                <span class="dot wishlist" />{{ wishlist.size }} want to go
            </span>
        </template>
        <template v-else>
            <span>
                <span class="dot flight" />{{
                    formatDistance(totalKm, store.settings.distanceUnit)
                }}
                flown
            </span>
            <span>✈ {{ timesAroundEarth }}× around the world</span>
        </template>
    </div>
</template>

<style scoped>
.map {
    width: 100%;
    height: 100%;
}

/* Countries are only clickable in the countries view */
.map.no-pointer :deep(.maplibregl-canvas) {
    cursor: default !important;
}

/* Stats */
.stats {
    position: absolute;
    bottom: 1.5rem;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    gap: 1.2rem;
    background: var(--panel);
    backdrop-filter: blur(8px);
    border: 1px solid var(--line);
    border-radius: 2rem;
    padding: 0.45rem 1.1rem;
    font-size: 0.78rem;
    color: var(--muted);
    pointer-events: none;
    z-index: 10;
}

.dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    margin-right: 0.3rem;
    vertical-align: middle;
}

.dot.visited {
    background: var(--visited);
}

.dot.wishlist {
    background: var(--wishlist);
}

.dot.flight {
    background: var(--flight);
}
</style>
