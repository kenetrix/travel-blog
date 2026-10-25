<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { useTravelStore } from '../stores/store'
import AirportSelect from './AirportSelect.vue'
import ConfirmModal from '../ui/ConfirmModal.vue'
import { vEnterNext } from '../ui/enterNext'
import { distanceKm, formatDistance } from './distance'
import type { Airport } from './airports'
import type { Flight } from '../stores/store'

const format = (km: number) => formatDistance(km, store.settings.distanceUnit)

const store = useTravelStore()

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
    () => !!from.value && !!to.value && !sameAirport.value && !duplicate.value
)

const totalKm = computed(() =>
    store.flights.reduce((sum, f) => sum + distanceKm(f.from, f.to), 0)
)

async function addFlight() {
    if (!from.value || !to.value || !canAdd.value) return

    // The server also marks both countries as visited
    if (!(await store.addFlight(from.value, to.value))) return

    from.value = null
    to.value = null
}
</script>

<template>
    <form v-enter-next class="panel" @submit.prevent="addFlight">
        <div>
            <h2>Add flight</h2>
            <p class="sub">Draw a route on the map</p>
        </div>

        <div>
            <label>From</label>
            <AirportSelect
                v-model="from"
                placeholder="Search airport or IATA"
            />
        </div>

        <div>
            <label>To</label>
            <AirportSelect v-model="to" placeholder="Search airport or IATA" />
        </div>

        <button class="btn btn-primary" type="submit" :disabled="!canAdd">
            Add route
        </button>
        <p v-if="duplicate" class="hint">This flight was already added.</p>
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
                    @mouseenter="store.highlightFlight(f.id)"
                    @mouseleave="store.highlightFlight(null)"
                >
                    <span>{{ f.label }}</span>
                    <span class="km">{{
                        format(distanceKm(f.from, f.to))
                    }}</span>
                    <button
                        type="button"
                        class="visibility"
                        :aria-label="
                            f.visibility === 'public'
                                ? `${f.label} is public; make it private`
                                : `${f.label} is private; make it public`
                        "
                        :title="
                            f.visibility === 'public'
                                ? 'Public - click to make private'
                                : 'Only you - click to make public'
                        "
                        @click="
                            store.setFlightVisibility(
                                f.id,
                                f.visibility === 'public' ? 'private' : 'public'
                            )
                        "
                    >
                        {{ f.visibility === 'public' ? '🌐' : '🔒' }}
                    </button>
                    <button
                        type="button"
                        class="remove"
                        :aria-label="`Remove ${f.label}`"
                        @click="pendingRemoval = f"
                    >
                        ×
                    </button>
                </li>
            </ul>
        </div>
    </form>

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
.panel {
    top: 5.2rem;
    right: 1.2rem;
    z-index: 10;
    position: absolute;
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
    width: 240px;
    background: var(--panel);
    backdrop-filter: blur(8px);
    border: 1px solid var(--line);
    border-radius: 1rem;
    padding: 1.1rem;
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
    max-height: 9rem;
    overflow-y: auto;
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

.km {
    margin: 0 auto 0 0.6rem;
    color: var(--muted);
    font-size: 0.72rem;
}

.flights li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.25rem 0.1rem;
    font-size: 0.8rem;
}

.visibility {
    border: none;
    background: none;
    font-size: 0.75rem;
    line-height: 1;
    cursor: pointer;
    opacity: 0.55;
}

.visibility:hover {
    opacity: 1;
}

.remove {
    border: none;
    background: none;
    color: var(--muted);
    font-size: 1rem;
    line-height: 1;
    cursor: pointer;
}

.remove:hover {
    color: #c0392b;
}

.btn {
    box-sizing: border-box;
    width: 100%;
    padding: 0.5rem;
    border: none;
    border-radius: 0.5rem;
    font-size: 0.82rem;
    font-weight: 500;
    cursor: pointer;
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
