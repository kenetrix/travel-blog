<script setup lang="ts">
import { ref, computed, nextTick, onUnmounted, watch } from 'vue'
import { useTravelStore } from '../stores/store'
import AirportSelect from './AirportSelect.vue'
import FlightDetails from './FlightDetails.vue'
import ConfirmModal from '../ui/ConfirmModal.vue'
import AppSheet from '../ui/AppSheet.vue'
import { useLayout } from '../ui/useLayout'
import { t } from '../../i18n'
import { vEnterNext } from '../ui/enterNext'
import { distanceKm } from '../geo/geo'
import { formatDistance } from '../../i18n/format'
import { formatFlightNumber, normalizeFlightNumber } from './flightNumber'
import type { Airport } from './airports'
import type { Flight } from '../stores/store'

const format = (km: number) => formatDistance(km, store.settings.distanceUnit)

const store = useTravelStore()
const { layout } = useLayout()

onUnmounted(() => store.selectFlight(null))

const pendingRemoval = ref<Flight | null>(null)

function removeFlight() {
    const flight = pendingRemoval.value
    pendingRemoval.value = null
    if (!flight) return
    if (added.value?.id === flight.id) skipNumber()
    store.removeFlight(flight.id)
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
const peekSelected = computed(() => layout.value === 'phone' && !expanded.value)

// Flight whose details are open, picked here or on the map; it stays
// highlighted and the map shows it
const openId = computed(() => store.selectedFlightId)
const selected = computed(() =>
    store.flights.find((f) => f.id === openId.value)
)
const list = ref<HTMLElement | null>(null)

// Touch screens have no hover: a tap opens a flight, a second tap closes it
function toggleOpen(id: string) {
    store.selectFlight(openId.value === id ? null : id)
}

// A phone sheet stays collapsed when a route is tapped on the map, so the
// map keeps showing it; the details are revealed once the sheet opens
watch([openId, expanded], () => {
    if (openId.value && expanded.value) revealOpen()
})

// The phone sheet covers most of the map when open: fit the route into
// whatever is left after each change
watch(expanded, () => {
    if (layout.value === 'phone') store.focusSelectedFlight()
})

// Scrolls only the list or the sheet; scrollIntoView would also shift the
// map page
function revealOpen() {
    nextTick(() => {
        const row = list.value?.querySelector<HTMLElement>('li.open')
        const box = [list.value, row?.closest<HTMLElement>('[data-scroll]')]
            .filter((el): el is HTMLElement => !!el)
            .find((el) => el.scrollHeight > el.clientHeight)
        if (!row || !box) return
        const rowBox = row.getBoundingClientRect()
        const view = box.getBoundingClientRect()
        if (rowBox.bottom <= view.bottom) return
        box.scrollBy({
            top: Math.min(rowBox.bottom - view.bottom, rowBox.top - view.top),
            behavior: 'smooth',
        })
    })
}

// With flights, the list comes first and the route form opens on request;
// without any, the form is all there is to show
const adding = ref(false)
const showRouteForm = computed(() => adding.value || !store.flights.length)
const routeForm = ref<HTMLFormElement | null>(null)

function startAdd() {
    adding.value = true
    expanded.value = true
    nextTick(() => routeForm.value?.querySelector('input')?.focus())
}

function cancelAdd() {
    if (!store.flights.length) return
    adding.value = false
    from.value = null
    to.value = null
}

// After a route is added, the form asks for its flight number
const added = ref<Flight | null>(null)
const number = ref('')
const numberInput = ref<HTMLInputElement | null>(null)
const normalizedNumber = computed(() =>
    number.value.trim() ? normalizeFlightNumber(number.value) : ''
)

async function addFlight() {
    if (!from.value || !to.value || !canAdd.value) return

    // The server also marks both countries as visited
    const id = await store.addFlight(from.value, to.value)
    if (!id) return

    from.value = null
    to.value = null
    adding.value = false
    added.value = store.flights.find((f) => f.id === id) ?? null
    number.value = ''
    nextTick(() => numberInput.value?.focus())
}

async function saveNumber() {
    const flight = added.value
    if (!flight || normalizedNumber.value === null) return
    if (!normalizedNumber.value) return skipNumber()
    const saved = await store.updateFlight(flight.id, {
        flightNumber: normalizedNumber.value,
    })
    if (!saved) return
    skipNumber()
    store.selectFlight(flight.id)
}

function skipNumber() {
    added.value = null
    number.value = ''
}
</script>

<template>
    <AppSheet
        v-model:expanded="expanded"
        side="right"
        :label="t('flights.title')"
    >
        <form
            v-if="added"
            class="flight-form"
            @submit.prevent="saveNumber"
            @keydown.esc="skipNumber"
        >
            <div>
                <h2>{{ t('flights.add') }}</h2>
                <p class="sub">
                    {{ t('flights.wasAdded', { route: added.label }) }}
                </p>
            </div>

            <div>
                <label for="new-flight-number">{{
                    t('flights.numberOptional')
                }}</label>
                <input
                    id="new-flight-number"
                    ref="numberInput"
                    v-model="number"
                    class="field"
                    :placeholder="t('flights.numberPlaceholder')"
                    autocapitalize="characters"
                    autocomplete="off"
                    spellcheck="false"
                    enterkeyhint="done"
                    maxlength="16"
                    :disabled="!store.canWrite"
                />
                <p v-if="normalizedNumber === null" class="hint below">
                    {{ t('flights.numberRule') }}
                </p>
            </div>

            <div class="actions">
                <button
                    class="btn btn-primary"
                    type="submit"
                    :disabled="!store.canWrite || normalizedNumber === null"
                    :title="
                        store.canWrite ? undefined : t('common.offlineHint')
                    "
                >
                    {{ t('common.save') }}
                </button>
                <button
                    class="btn btn-secondary"
                    type="button"
                    @click="skipNumber"
                >
                    {{ t('flights.skip') }}
                </button>
            </div>
            <p v-if="!store.canWrite" class="hint offline">
                {{ t('common.offlineHint') }}
            </p>
        </form>

        <form
            v-else-if="showRouteForm"
            ref="routeForm"
            v-enter-next
            class="flight-form"
            @submit.prevent="addFlight"
            @keydown.esc="cancelAdd"
        >
            <div>
                <h2>{{ t('flights.add') }}</h2>
                <p class="sub">
                    {{
                        store.flights.length
                            ? t('flights.drawRoute')
                            : t('flights.firstRoute')
                    }}
                </p>
            </div>

            <div>
                <label>{{ t('flights.from') }}</label>
                <AirportSelect
                    v-model="from"
                    :placeholder="t('flights.searchAirport')"
                    @focus="expanded = true"
                />
            </div>

            <div>
                <label>{{ t('flights.to') }}</label>
                <AirportSelect
                    v-model="to"
                    :placeholder="t('flights.searchAirport')"
                    @focus="expanded = true"
                />
            </div>

            <div class="actions">
                <button
                    class="btn btn-primary"
                    type="submit"
                    :disabled="!canAdd"
                    :title="
                        store.canWrite ? undefined : t('common.offlineHint')
                    "
                >
                    {{ t('flights.addRoute') }}
                </button>
                <button
                    v-if="store.flights.length"
                    class="btn btn-secondary"
                    type="button"
                    @click="cancelAdd"
                >
                    {{ t('common.cancel') }}
                </button>
            </div>
            <p v-if="!store.canWrite" class="hint offline">
                {{ t('common.offlineHint') }}
            </p>
            <p v-else-if="duplicate" class="hint">
                {{ t('flights.duplicate') }}
            </p>
            <p v-else-if="sameAirport" class="hint">
                {{ t('flights.sameAirport') }}
            </p>
        </form>

        <template v-else>
            <div class="head">
                <!-- The collapsed phone sheet names the selected route; a tap
                     opens its details, like the handle -->
                <div
                    v-if="peekSelected && selected"
                    class="peek"
                    @click="expanded = true"
                >
                    <h2>{{ selected.label }}</h2>
                    <p class="sub">
                        <template v-if="selected.flightNumber">
                            {{ formatFlightNumber(selected.flightNumber) }} ·
                        </template>
                        {{ format(distanceKm(selected.from, selected.to)) }}
                        · {{ t('flights.tapForDetails') }}
                    </p>
                </div>
                <div v-else>
                    <h2>{{ t('flights.title') }}</h2>
                    <p class="sub">
                        {{
                            t('flights.count', { count: store.flights.length })
                        }}
                        · {{ format(totalKm) }}
                    </p>
                </div>
                <button
                    class="btn btn-primary add"
                    type="button"
                    :disabled="!store.canWrite"
                    :aria-label="t('flights.add')"
                    :title="
                        store.canWrite
                            ? t('flights.add')
                            : t('common.offlineHint')
                    "
                    @click="startAdd"
                >
                    <svg
                        viewBox="0 0 24 24"
                        width="16"
                        height="16"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2.4"
                        stroke-linecap="round"
                        aria-hidden="true"
                    >
                        <path d="M12 5v14M5 12h14" />
                    </svg>
                    <span class="add-label">{{ t('flights.add') }}</span>
                </button>
            </div>
            <ul ref="list" class="flights">
                <li
                    v-for="f in store.flights"
                    :key="f.id"
                    :class="{
                        active: store.highlightedFlightId === f.id,
                        open: openId === f.id,
                    }"
                    @mouseenter="store.highlightFlight(f.id)"
                    @mouseleave="store.highlightFlight(openId)"
                >
                    <div class="row">
                        <button
                            type="button"
                            class="route"
                            :aria-expanded="openId === f.id"
                            :aria-controls="`details-${f.id}`"
                            @click="toggleOpen(f.id)"
                        >
                            <span class="name">
                                <span>{{ f.label }}</span>
                                <span v-if="f.flightNumber" class="no">{{
                                    formatFlightNumber(f.flightNumber)
                                }}</span>
                            </span>
                            <span class="km">{{
                                format(distanceKm(f.from, f.to))
                            }}</span>
                        </button>
                        <button
                            type="button"
                            class="icon visibility"
                            :disabled="!store.canWrite"
                            :aria-label="
                                t(
                                    f.visibility === 'public'
                                        ? 'flights.makePrivate'
                                        : 'flights.makePublic',
                                    { route: f.label }
                                )
                            "
                            :title="
                                !store.canWrite
                                    ? t('common.offlineHint')
                                    : f.visibility === 'public'
                                      ? t('flights.publicTitle')
                                      : t('flights.privateTitle')
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
                            :title="
                                store.canWrite
                                    ? undefined
                                    : t('common.offlineHint')
                            "
                            :aria-label="
                                t('flights.remove', { route: f.label })
                            "
                            @click="pendingRemoval = f"
                        >
                            ×
                        </button>
                    </div>
                    <FlightDetails
                        v-if="openId === f.id"
                        :id="`details-${f.id}`"
                        :flight="f"
                    />
                </li>
            </ul>
        </template>
    </AppSheet>

    <ConfirmModal
        v-if="pendingRemoval"
        :title="t('flights.removeTitle')"
        :message="t('flights.removeMessage', { route: pendingRemoval.label })"
        :confirm-label="t('common.remove')"
        @confirm="removeFlight"
        @cancel="pendingRemoval = null"
    />
</template>

<style scoped>
.flight-form {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
}

h2 {
    font-size: 1rem;
    font-weight: 600;
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

.field {
    box-sizing: border-box;
    width: 100%;
    min-height: var(--touch);
    padding: 0.5rem 0.6rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    background: var(--raised);
    color: var(--text);
    font: inherit;
    font-size: var(--field-font);
    outline: none;
}

.field:focus {
    border-color: #3a7d5c;
}

.actions {
    display: flex;
    gap: 0.5rem;
}

/* Also the peek of the collapsed phone sheet: title, total and the button */
.head {
    display: flex;
    flex: none;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    container-type: inline-size;
}

.head > div {
    min-width: 0;
}

.peek {
    flex: 1;
    cursor: pointer;
}

.peek .sub {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.btn.add {
    display: inline-flex;
    flex: none;
    align-items: center;
    justify-content: center;
    gap: 0.35rem;
    width: auto;
    min-width: var(--touch);
    padding: 0.45rem 1rem;
    border-radius: 999px;
}

/* The narrow desktop panel: icon only, so the total stays on one line */
@container (max-width: 15rem) {
    .add-label {
        display: none;
    }

    .btn.add {
        min-width: 2rem;
        padding: 0.45rem;
    }
}

/* Only the list scrolls, so the head and its button stay in view */
.flights {
    flex: 0 1 auto;
    min-height: 5.5rem;
    margin: 0;
    padding: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
    list-style: none;
}

.hint {
    margin: -0.4rem 0 0;
    font-size: 0.72rem;
    color: #c0392b;
}

.hint.offline {
    color: #8a5a12;
}

.hint.below {
    margin-top: 0.3rem;
}

.flights li {
    border-radius: 0.4rem;
    font-size: 0.85rem;
}

.flights li.active,
.flights li.open {
    background: var(--line-faint);
}

.row {
    display: flex;
    align-items: center;
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

.name {
    display: flex;
    flex-direction: column;
    padding: 0.2rem 0;
    white-space: nowrap;
}

.no {
    color: var(--muted);
    font-size: 0.68rem;
}

.km {
    flex: none;
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

    .field {
        min-height: 0;
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

.btn-secondary {
    background: var(--line-faint);
    color: var(--text);
}

.btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}
</style>
