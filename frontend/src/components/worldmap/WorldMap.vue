<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import { MapLibreProvider } from '../worldmap/MapLibreProvider'
import { useTravelStore } from '../stores/store'
import { distanceKm, formatDistance } from '../flights/distance'
import { useLayout } from '../ui/useLayout'
import { sheetInsets } from '../ui/mapInsets'
import type { MapInsets } from './MapProvider'
import 'maplibre-gl/dist/maplibre-gl.css'

const store = useTravelStore()
const {
    visited,
    wishlist,
    selectedIso,
    preview,
    flights,
    highlightedFlightId,
    selectedFlightId,
    flightFocus,
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
const { layout } = useLayout()

// Free edges around the map per layout: the app bar (and on phones the
// stats) on top, plus whatever the open sheet covers
const BASE: Record<string, MapInsets> = {
    phone: { top: 120, right: 24, bottom: 24, left: 24 },
    tablet: { top: 100, right: 40, bottom: 40, left: 40 },
    desktop: { top: 80, right: 60, bottom: 80, left: 40 },
}

function insets(): MapInsets {
    const base = BASE[layout.value]
    const sheet = sheetInsets()
    return {
        top: Math.max(base.top, sheet.top ?? 0),
        right: Math.max(base.right, sheet.right ?? 0),
        bottom: Math.max(base.bottom, sheet.bottom ?? 0),
        left: Math.max(base.left, sheet.left ?? 0),
    }
}

onMounted(() => {
    if (!mapContainer.value) return
    provider.onReady(syncMap)
    provider.mount(mapContainer.value)

    provider.onCountryClick((iso) => {
        if (props.view !== 'countries') return
        store.selectCountry(iso)
    })
    // A click away from all routes clears the selection
    provider.onFlightClick((id) => {
        if (props.view === 'flights') store.selectFlight(id)
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
    if (iso) setTimeout(() => provider.focusCountry(iso, insets()), 50)
    else provider.focusCountry(null)
})

// Fit the selected route into the free part of the map, after the sheet
// had a moment to change
watch(flightFocus, () => {
    setTimeout(() => {
        if (selectedFlightId.value)
            provider.focusFlight(selectedFlightId.value, insets())
    }, 50)
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

/* Countries are only clickable in the countries view, routes in the flights view */
.map.no-pointer :deep(.maplibregl-canvas) {
    cursor: default !important;
}

.map.no-pointer :deep(.maplibregl-canvas.route-hover) {
    cursor: pointer !important;
}

/* Stats: phones show them under the app bar, the sheet covers the bottom */
.stats {
    position: absolute;
    top: calc(var(--safe-top) + var(--appbar-h) + 1rem);
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    gap: 1rem;
    width: max-content;
    max-width: calc(100vw - 2 * var(--edge));
    box-sizing: border-box;
    background: var(--panel);
    backdrop-filter: blur(8px);
    border: 1px solid var(--line);
    border-radius: 2rem;
    padding: 0.35rem 0.9rem;
    font-size: 0.75rem;
    color: var(--muted);
    white-space: nowrap;
    pointer-events: none;
    z-index: 10;
}

/* Tablet: bottom right, beside the side sheet */
@media (min-width: 600px),
    (orientation: landscape) and (max-height: 500px) and (min-width: 480px) {
    .stats {
        top: auto;
        bottom: calc(var(--safe-bottom) + 1.2rem);
        left: calc(var(--safe-left) + var(--sheet-space) + 1rem);
        transform: none;
        gap: 1.2rem;
        padding: 0.45rem 1.1rem;
        font-size: 0.78rem;
    }
}

@media (orientation: landscape) and (max-height: 500px) and (min-width: 480px) {
    .stats {
        bottom: calc(var(--safe-bottom) + var(--edge));
        left: calc(var(--safe-left) + var(--edge) + var(--sheet-space));
        padding: 0.3rem 0.8rem;
        font-size: 0.72rem;
    }
}

/* Desktop: centred at the bottom */
@media (min-width: 1024px) and (min-height: 501px) {
    .stats {
        bottom: 1.5rem;
        left: 50%;
        transform: translateX(-50%);
    }
}

/* Phones: zoom and attribution move to the top right, away from the sheet */
@media (max-width: 479px), (max-width: 599px) and (min-height: 501px) {
    .map :deep(.maplibregl-ctrl-bottom-right) {
        top: calc(var(--safe-top) + var(--appbar-h) + 5.4rem);
        bottom: auto;
        right: var(--safe-right);
    }

    .map :deep(.maplibregl-ctrl-group) {
        display: none;
    }
}

.map :deep(.maplibregl-ctrl-bottom-right) {
    margin-bottom: var(--safe-bottom);
    margin-right: var(--safe-right);
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
