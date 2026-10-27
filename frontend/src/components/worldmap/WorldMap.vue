<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import { MapLibreProvider } from '../worldmap/MapLibreProvider'
import { useTravelStore } from '../stores/store'
import { distanceKm } from '../geo/geo'
import { language, t } from '../../i18n'
import { countryLabels } from './countryLabels'
import { formatDistance, formatNumber } from '../../i18n/format'
import { useLayout } from '../ui/useLayout'
import { sheetInsets } from '../ui/mapInsets'
import type { MapInsets } from './MapProvider'
import { PLACES_MIN_ZOOM, placesOf } from './places'
import { PIN_HEIGHT, PIN_SVG, PIN_WIDTH } from './pin'
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
    tripMap,
    tripFocus,
    pinMode,
    pin,
} = storeToRefs(store)

const EARTH_CIRCUMFERENCE_KM = 40075

const totalKm = computed(() =>
    flights.value.reduce((sum, f) => sum + distanceKm(f.from, f.to), 0)
)
const timesAroundEarth = computed(() =>
    formatNumber(totalKm.value / EARTH_CIRCUMFERENCE_KM, {
        minimumFractionDigits: 1,
        maximumFractionDigits: 1,
    })
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
    // A trip opened by URL asks for its focus before the map has loaded
    provider.onReady(() => {
        syncMap()
        if (tripMap.value) fitTrip()
    })
    provider.mount(mapContainer.value)

    // While a trip is open, clicks belong to the trip editor
    provider.onCountryClick((iso) => {
        if (props.view !== 'countries' || tripMap.value) return
        store.selectCountry(iso)
    })
    // A click away from all routes clears the selection
    provider.onFlightClick((id) => {
        if (props.view === 'flights' && !tripMap.value) store.selectFlight(id)
    })
    // A click on a place name sets the pin exactly on that place
    provider.onMapClick((point, place) => {
        if (pinMode.value !== 'click') return
        if (place) {
            const iso = provider.countryAt(place.lng, place.lat) ?? point.iso
            store.setPin({ lng: place.lng, lat: place.lat, iso }, place.name)
        } else {
            store.setPin(point)
        }
    })
    provider.onMoveEnd(() => {
        updatePlaces()
        if (pinMode.value === 'center') readCenterPin()
    })
    window.addEventListener('resize', placeCenterPin)
})

onUnmounted(() => {
    window.removeEventListener('resize', placeCenterPin)
    provider.destroy()
})

// --- Trip --------------------------------------------------------------

// Names of the places in the countries in view, while a trip is edited and
// the map is zoomed in far enough
let placesShown = false
let placesRequest = 0
async function updatePlaces() {
    const request = ++placesRequest
    if (!tripMap.value?.editing || provider.getZoom() < PLACES_MIN_ZOOM) {
        if (placesShown) provider.showPlaces(null)
        placesShown = false
        return
    }
    const places = await placesOf(provider.countriesInView())
    // The map moved on, or the editor closed, while the files loaded
    if (request !== placesRequest) return
    provider.showPlaces(places)
    placesShown = true
}

watch(
    tripMap,
    (trip) => {
        provider.showTrip(trip?.stops ?? null)
        updatePlaces()
    },
    { deep: true }
)

function fitTrip() {
    const stops = tripMap.value?.stops ?? []
    if (stops.length) provider.fitPoints(stops, insets())
    else if (tripFocus.value.iso) {
        provider.fitCountry(tripFocus.value.iso, insets())
    }
}

// After the sheet had a moment to change
watch(
    () => tripFocus.value.count,
    () => setTimeout(fitTrip, 50)
)

// The pin set by click is a map marker; the centre pin is an overlay
watch([pin, pinMode], ([point, mode]) =>
    provider.setPin(mode === 'click' ? point : null)
)

// Screen point of the centre pin's tip: the middle of the free map area
const centerPin = ref<{ x: number; y: number } | null>(null)

function placeCenterPin() {
    const el = mapContainer.value
    if (pinMode.value !== 'center' || !el) {
        centerPin.value = null
        return
    }
    const free = insets()
    centerPin.value = {
        x: free.left + (el.clientWidth - free.left - free.right) / 2,
        y: free.top + (el.clientHeight - free.top - free.bottom) / 2,
    }
}

function readCenterPin() {
    if (!centerPin.value) return
    const point = provider.pointAt(centerPin.value.x, centerPin.value.y)
    if (point) store.setPin(point)
}

watch(pinMode, (mode) => {
    if (mode !== 'center') return placeCenterPin()
    // After the sheet collapsed. A pin set before (moving a stop) is brought
    // under the centre pin; otherwise the pin points where the map is.
    setTimeout(() => {
        placeCenterPin()
        if (!centerPin.value) return
        if (pin.value) {
            const { x, y } = centerPin.value
            provider.panTo(pin.value, x, y, 8)
        } else {
            readCenterPin()
        }
    }, 300)
})

watch(
    language,
    (lang) => provider.setCountryNames(lang === 'en' ? null : countryLabels()),
    { immediate: true }
)

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
    provider.showTrip(tripMap.value?.stops ?? null)
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
        :class="{
            'no-pointer': view !== 'countries' || !!tripMap,
            placing: pinMode === 'click',
        }"
    />
    <!-- The map moves underneath; the tip marks the place -->
    <div
        v-if="pinMode === 'center' && centerPin"
        class="center-pin"
        :style="{
            left: `${centerPin.x - PIN_WIDTH / 2}px`,
            top: `${centerPin.y - PIN_HEIGHT}px`,
        }"
        v-html="PIN_SVG"
    />
    <!-- Stats -->
    <div class="stats">
        <template v-if="view === 'countries'">
            <span>
                <span class="dot visited" />{{
                    t('map.visited', { count: visitedCount })
                }}
            </span>
            <span>
                <span class="dot wishlist" />{{
                    t('map.wishlist', { count: wishlist.size })
                }}
            </span>
        </template>
        <template v-else>
            <span>
                <span class="dot flight" />{{
                    t('map.flown', {
                        distance: formatDistance(
                            totalKm,
                            store.settings.distanceUnit
                        ),
                    })
                }}
            </span>
            <span>{{ t('map.aroundWorld', { times: timesAroundEarth }) }}</span>
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

.map.placing :deep(.maplibregl-canvas) {
    cursor: crosshair !important;
}

.center-pin {
    position: absolute;
    z-index: 12;
    line-height: 0;
    pointer-events: none;
    filter: drop-shadow(0 2px 3px rgb(0 0 0 / 0.35));
}

:deep(.trip-pin) {
    line-height: 0;
    filter: drop-shadow(0 2px 3px rgb(0 0 0 / 0.35));
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
