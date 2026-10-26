<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { useTravelStore } from '../stores/store'
import AirportSelect from './AirportSelect.vue'
import ConfirmModal from '../ui/ConfirmModal.vue'
import AppSheet from '../ui/AppSheet.vue'
import { useLayout } from '../ui/useLayout'
import { OFFLINE_HINT } from '../ui/offline'
import { vEnterNext } from '../ui/enterNext'
import { distanceKm, formatDistance } from './distance'
import type { Airport } from './airports'
import type { Flight } from '../stores/store'

const format = (km: number) => formatDistance(km, store.settings.distanceUnit)

const store = useTravelStore()
const { layout } = useLayout()

onUnmounted(() => store.highlightFlight(null))

const pendingRemoval = ref<Flight | null>(null)

function removeFlight() {
    if (pendingRemoval.value) store.removeFlight(pendingRemoval.value.id)
    pendingRemoval.value = null
}

const from = ref<Airport | null>(null)
const to = ref<Airport | null>(null)

const sameAirport = computed(
    () => !!from.value && !!to.value && from.value.iata === to.value.iata
)

const duplicate = computed(() => {
    const [a, b] = [from.value, to.value]
    if (!a || !b) return false
    return store.flights.some((f) => f.label === `${a.iata} → ${b.iata}`)
})

const canAdd = computed(
    () =>
        store.canWrite &&
        !!from.value &&
        !!to.value &&
        !sameAirport.value &&
        !duplicate.value
)

const totalKm = computed(() =>
    store.flights.reduce((sum, f) => sum + distanceKm(f.from, f.to), 0)
)

// Phones start with the map; side sheets start open
const expanded = ref(layout.value !== 'phone')

// Touch screens have no hover: a tap highlights a flight, a second tap clears
function toggleHighlight(id: string) {
    store.highlightFlight(store.highlightedFlightId === id ? null : id)
}

async function addFlight() {
    if (!from.value || !to.value || !canAdd.value) return

    // The server also marks both countries as visited
    if (!(await store.addFlight(from.value, to.value))) return

    from.value = null
    to.value = null
}
</script>

<template>
    <AppSheet v-model:expanded="expanded" side="right" label="Flights">
        <form v-enter-next class="flight-form" @submit.prevent="addFlight">
            <div>
                <h2>Add flight</h2>
                <p class="sub">Draw a route on the map</p>
            </div>

            <div>
                <label>From</label>
                <AirportSelect
                    v-model="from"
                    placeholder="Search airport or IATA"
                    @focus="expanded = true"
                />
            </div>

            <div>
                <label>To</label>
                <AirportSelect
                    v-model="to"
                    placeholder="Search airport or IATA"
                    @focus="expanded = true"
                />
            </div>

            <button
                class="btn btn-primary"
                type="submit"
                :disabled="!canAdd"
                :title="store.canWrite ? undefined : OFFLINE_HINT"
            >
                Add route
            </button>
            <p v-if="!store.canWrite" class="hint offline">
                {{ OFFLINE_HINT }}
            </p>
            <p v-else-if="duplicate" class="hint">
                This flight was already added.
            </p>
            <p v-else-if="sameAirport" class="hint">
                Choose two different airports.
            </p>

            <div v-if="store.flights.length">
                <label class="list-head">
                    <span>Flights</span>
                    <span>{{ format(totalKm) }} total</span>
                </label>
                <ul class="flights">
                    <li
                        v-for="f in store.flights"
                        :key="f.id"
                        :class="{ active: store.highlightedFlightId === f.id }"
                        @mouseenter="store.highlightFlight(f.id)"
                        @mouseleave="store.highlightFlight(null)"
                    >
                        <button
                            type="button"
                            class="route"
                            :aria-pressed="store.highlightedFlightId === f.id"
                            @click="toggleHighlight(f.id)"
                        >
                            <span>{{ f.label }}</span>
                            <span class="km">{{
                                format(distanceKm(f.from, f.to))
                            }}</span>
                        </button>
                        <button
                            type="button"
                            class="icon visibility"
                            :disabled="!store.canWrite"
                            :aria-label="
                                f.visibility === 'public'
                                    ? `${f.label} is public; make it private`
                                    : `${f.label} is private; make it public`
                            "
                            :title="
                                !store.canWrite
                                    ? OFFLINE_HINT
                                    : f.visibility === 'public'
                                      ? 'Public - click to make private'
                                      : 'Only you - click to make public'
                            "
                            @click="
                                store.setFlightVisibility(
                                    f.id,
                                    f.visibility === 'public'
                                        ? 'private'
                                        : 'public'
                                )
                            "
                        >
                            {{ f.visibility === 'public' ? '🌐' : '🔒' }}
                        </button>
                        <button
                            type="button"
                            class="icon remove"
                            :disabled="!store.canWrite"
                            :title="store.canWrite ? undefined : OFFLINE_HINT"
                            :aria-label="`Remove ${f.label}`"
                            @click="pendingRemoval = f"
                        >
                            ×
                        </button>
                    </li>
                </ul>
            </div>
        </form>
    </AppSheet>

    <ConfirmModal
        v-if="pendingRemoval"
        title="Remove flight"
        :message="`Remove ${pendingRemoval.label} from your flights?`"
        confirm-label="Remove"
        @confirm="removeFlight"
        @cancel="pendingRemoval = null"
    />
</template>

<style scoped>
.flight-form {
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
}

h2 {
    font-family: 'DM Serif Display', serif;
    font-size: 1rem;
    font-weight: 400;
}

.sub {
    font-size: 0.75rem;
    color: var(--muted);
    margin-top: 0.1rem;
}

label {
    display: block;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--muted);
    margin-bottom: 0.25rem;
}

.flights {
    margin: 0;
    padding: 0;
    list-style: none;
}

/* Desktop: a short list, the panel floats over the map */
@media (min-width: 1024px) and (min-height: 501px) {
    .flights {
        max-height: 9rem;
        overflow-y: auto;
    }
}

.list-head {
    display: flex;
    justify-content: space-between;
}

.hint {
    margin: -0.4rem 0 0;
    font-size: 0.72rem;
    color: #c0392b;
}

.hint.offline {
    color: #8a5a12;
}

.flights li {
    display: flex;
    align-items: center;
    border-radius: 0.4rem;
    font-size: 0.85rem;
}

.flights li.active {
    background: var(--line-faint);
}

.route {
    display: flex;
    flex: 1;
    align-items: center;
    min-width: 0;
    min-height: var(--touch);
    padding: 0 0.1rem;
    border: none;
    background: none;
    color: var(--text);
    font: inherit;
    text-align: left;
    cursor: pointer;
}

.km {
    margin-left: 0.6rem;
    color: var(--muted);
    font-size: 0.72rem;
}

.icon {
    flex: none;
    width: var(--touch);
    height: var(--touch);
    padding: 0;
    border: none;
    background: none;
    line-height: 1;
    cursor: pointer;
}

/* Mouse: denser rows, as before */
@media (pointer: fine) {
    .flights li {
        font-size: 0.8rem;
    }

    .route,
    .icon {
        min-height: 1.8rem;
        height: auto;
    }

    .icon {
        width: 1.6rem;
    }
}

.visibility {
    font-size: 0.8rem;
    opacity: 0.55;
}

.visibility:hover:not(:disabled) {
    opacity: 1;
}

.remove {
    color: var(--muted);
    font-size: 1.1rem;
}

.remove:hover:not(:disabled) {
    color: #c0392b;
}

.icon:disabled {
    opacity: 0.3;
}

.btn {
    box-sizing: border-box;
    width: 100%;
    min-height: var(--touch);
    padding: 0.5rem;
    border: none;
    border-radius: 0.5rem;
    font: inherit;
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
}

@media (pointer: fine) {
    .btn {
        min-height: 0;
        font-size: 0.82rem;
    }
}

.btn-primary {
    background: #1a1c1e;
    color: white;
}

.btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}
</style>
