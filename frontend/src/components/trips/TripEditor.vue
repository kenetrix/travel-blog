<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import AppSheet from '../ui/AppSheet.vue'
import ConfirmModal from '../ui/ConfirmModal.vue'
import SegmentedControl from '../ui/SegmentedControl.vue'
import { useLayout } from '../ui/useLayout'
import {
    useTravelStore,
    type Trip,
    type TripInput,
    type MapPoint,
    type Visibility,
} from '../stores/store'
import { countryName, getCountryByIso } from '../countries/countries'
import { suggestName } from '../worldmap/places'
import type { StopCategory } from './categories'
import { t } from '../../i18n'

const MAX_STOPS = 200
const MAX_TITLE = 120

interface DraftStop {
    // Local key for the list; id is the server's, unset for new stops
    key: number
    id?: string
    name: string
    country: string
    lng: number
    lat: number
    arrival: string
    notes: string
    rating: number
    category: string
    links: { url: string; label: string }[]
}

interface Draft {
    title: string
    startsOn: string
    endsOn: string
    notes: string
    visibility: Visibility
    stops: DraftStop[]
}

const store = useTravelStore()
const route = useRoute()
const router = useRouter()
const { layout, coarsePointer } = useLayout()

const id = computed(() =>
    route.name === 'trip-edit' ? String(route.params.id) : null
)
// The country the new trip is started from
const startCountry = computed(() =>
    typeof route.query.country === 'string' ? route.query.country : null
)

let nextKey = 1
const draft = ref<Draft>(emptyDraft())
const saved = ref('')
const loading = ref(false)
const notFound = ref(false)
const saving = ref(false)
const expanded = ref(layout.value !== 'phone')

function emptyDraft(): Draft {
    return {
        title: '',
        startsOn: '',
        endsOn: '',
        notes: '',
        visibility: store.settings.defaultVisibility,
        stops: [],
    }
}

function draftOf(trip: Trip): Draft {
    return {
        title: trip.title,
        startsOn: trip.startsOn ?? '',
        endsOn: trip.endsOn ?? '',
        notes: trip.notes,
        visibility: trip.visibility,
        stops: trip.stops.map((s) => ({
            key: nextKey++,
            id: s.id,
            name: s.name,
            country: s.country,
            lng: s.lng,
            lat: s.lat,
            arrival: s.arrival ?? '',
            notes: s.notes,
            rating: s.rating,
            category: s.category,
            links: s.links.map((l) => ({ url: l.url, label: l.label ?? '' })),
        })),
    }
}

// Keys are local, so they do not count as a change
const comparable = (d: Draft) =>
    JSON.stringify({ ...d, stops: d.stops.map(({ key: _key, ...s }) => s) })

const dirty = computed(() => comparable(draft.value) !== saved.value)

function fill(next: Draft) {
    draft.value = next
    saved.value = comparable(next)
}

async function load() {
    notFound.value = false
    if (!id.value) {
        fill(emptyDraft())
        store.focusTrip(startCountry.value)
        return
    }
    const tripId = id.value
    const known = store.tripDetails[tripId]
    if (known) fill(draftOf(known))
    loading.value = !known
    const trip = await store.loadTrip(tripId)
    if (tripId !== id.value) return
    loading.value = false
    if (!trip) {
        notFound.value = true
        return
    }
    // Keep edits the user already started on the stored copy
    if (!dirty.value) fill(draftOf(trip))
    store.focusTrip()
}

onMounted(load)
watch(id, load)

// --- Map -----------------------------------------------------------------

watch(
    () =>
        draft.value.stops.map((s) => ({
            lng: s.lng,
            lat: s.lat,
            name: s.name,
        })),
    (stops) => store.showTrip({ stops, editing: true }),
    { immediate: true, deep: true }
)

onBeforeUnmount(() => store.showTrip(null))

// --- Placing a stop with the pin -------------------------------------------

// Adding a new stop, or moving the stop with this key
const placing = ref<{ key: number | null } | null>(null)
const placeName = ref('')
// Once the user typed a name, suggestions no longer replace it
let nameTyped = false

const placingStop = computed(() =>
    placing.value?.key
        ? draft.value.stops.find((s) => s.key === placing.value!.key)
        : undefined
)

function startPlacing(stop?: DraftStop) {
    placing.value = { key: stop?.key ?? null }
    placeName.value = ''
    nameTyped = false
    store.setPinMode('off')
    if (stop) store.setPin({ lng: stop.lng, lat: stop.lat, iso: stop.country })
    // Phones and tablets: the sheet makes room, the map moves under the pin
    if (coarsePointer.value) {
        expanded.value = false
        store.setPinMode('center')
    } else {
        store.setPinMode('click')
    }
}

function onNameInput() {
    nameTyped = true
}

function stopPlacing() {
    placing.value = null
    naming.value = null
    store.setPinMode('off')
}

// Touch: the place picked with the centre pin, named on a card of its own
// before the stop is added
const naming = ref<MapPoint | null>(null)
const namingInput = ref<HTMLInputElement | null>(null)

async function startNaming(point: MapPoint) {
    naming.value = { ...point }
    store.setPinMode('off')
    expanded.value = true
    await nextTick()
    namingInput.value?.focus()
}

function addNamedStop() {
    const point = naming.value
    const name = placeName.value.trim()
    if (!point?.iso || !name) return
    draft.value.stops.push({
        key: nextKey++,
        name,
        country: point.iso,
        lng: round(point.lng),
        lat: round(point.lat),
        arrival: '',
        notes: '',
        rating: 0,
        category: '',
        links: [],
    })
    naming.value = null
    // Straight on to the next stop
    startPlacing()
}

// Back to the centre pin, which starts where it was
function changePlace() {
    const point = naming.value
    naming.value = null
    expanded.value = false
    store.setPin(point)
    store.setPinMode('center')
}

let suggestRequest = 0
watch(
    () => store.pin,
    async (point) => {
        if (!placing.value || placing.value.key || !point) return
        if (store.pinPlace) {
            placeName.value = store.pinPlace
            nameTyped = false
            return
        }
        if (nameTyped) return
        const request = ++suggestRequest
        const name = point.iso
            ? await suggestName([point.lng, point.lat], point.iso)
            : null
        if (request === suggestRequest && !nameTyped) {
            placeName.value = name ?? ''
        }
    }
)

const pinCountry = computed(() => {
    const iso = store.pin?.iso
    if (!iso) return null
    const country = getCountryByIso(iso)
    return country ? countryName(country) : iso
})

const nameInputs = ref<Record<number, HTMLInputElement | null>>({})

// Six decimals are about 10 cm
const round = (n: number) => Math.round(n * 1e6) / 1e6

async function confirmPlace() {
    const point = store.pin
    if (!placing.value || !point?.iso) return
    const target = placingStop.value
    if (target) {
        target.lng = round(point.lng)
        target.lat = round(point.lat)
        target.country = point.iso
        stopPlacing()
        return
    }
    if (coarsePointer.value) return startNaming(point)
    const stop: DraftStop = {
        key: nextKey++,
        name: placeName.value.trim(),
        country: point.iso,
        lng: round(point.lng),
        lat: round(point.lat),
        arrival: '',
        notes: '',
        rating: 0,
        category: '',
        links: [],
    }
    draft.value.stops.push(stop)
    // Named stops go straight on to the next one; an unnamed one is named first
    if (stop.name) return startPlacing()
    stopPlacing()
    expanded.value = true
    await nextTick()
    nameInputs.value[stop.key]?.focus()
}

// --- Stops ---------------------------------------------------------------

function nameOf(iso: string): string {
    const country = getCountryByIso(iso)
    return country ? countryName(country) : iso
}

function flagOf(iso: string): string {
    const country = getCountryByIso(iso)
    return country ? `fi fi-${country.alpha2.toLowerCase()}` : ''
}

const label = (stop: DraftStop) => stop.name.trim() || t('trip.unnamed')

function moveStop(from: number, to: number) {
    const stops = draft.value.stops
    if (to < 0 || to >= stops.length || from === to) return
    const [stop] = stops.splice(from, 1)
    stops.splice(to, 0, stop)
}

function removeStop(index: number) {
    const [stop] = draft.value.stops.splice(index, 1)
    if (placing.value?.key === stop.key) stopPlacing()
}

// Drag and drop for a mouse; touch uses the up and down buttons
const dragIndex = ref<number | null>(null)

function onDrop(index: number) {
    if (dragIndex.value !== null) moveStop(dragIndex.value, index)
    dragIndex.value = null
}

// --- Validation and saving -----------------------------------------------

const problems = computed(() => {
    const d = draft.value
    const list: string[] = []
    if (!d.title.trim()) list.push(t('trip.titleRequired'))
    if (d.startsOn && d.endsOn && d.endsOn < d.startsOn) {
        list.push(t('trip.endBeforeStart'))
    }
    if (d.stops.some((s) => !s.name.trim())) list.push(t('trip.namesRequired'))
    if (d.stops.length > MAX_STOPS) {
        list.push(t('trip.tooManyStops', { max: MAX_STOPS }))
    }
    return list
})

const readOnly = computed(() => !store.canWrite || notFound.value)

const canAddStop = computed(
    () => !readOnly.value && draft.value.stops.length < MAX_STOPS
)
const addStopTitle = computed(() =>
    !store.canWrite
        ? t('common.offlineHint')
        : draft.value.stops.length >= MAX_STOPS
          ? t('trip.tooManyStops', { max: MAX_STOPS })
          : t('trip.addStop')
)
const canSave = computed(
    () =>
        !readOnly.value &&
        !saving.value &&
        !loading.value &&
        !problems.value.length &&
        (dirty.value || !id.value)
)

// Set once the user confirmed leaving, or after saving
let leaving = false

async function save() {
    if (!canSave.value) return
    const d = draft.value
    const input: TripInput = {
        title: d.title.trim(),
        startsOn: d.startsOn,
        endsOn: d.endsOn,
        notes: d.notes.trim(),
        visibility: d.visibility,
        stops: d.stops.map((s) => ({
            id: s.id,
            name: s.name.trim(),
            country: s.country,
            lng: s.lng,
            lat: s.lat,
            arrival: s.arrival,
            notes: s.notes.trim(),
            rating: s.rating,
            category: s.category as StopCategory,
            links: s.links
                .filter((l) => l.url.trim())
                .map((l) => ({ url: l.url.trim(), label: l.label.trim() })),
        })),
    }
    saving.value = true
    const trip = await store.saveTrip(id.value, input)
    saving.value = false
    if (!trip) return
    leaving = true
    router.push({ name: 'trip', params: { id: trip.id } })
}

function cancel() {
    if (placing.value) return stopPlacing()
    router.push(
        id.value ? { name: 'trip', params: { id: id.value } } : { name: 'home' }
    )
}

// What the "discard changes" dialog does once confirmed
const pendingLeave = ref<(() => void) | null>(null)

onBeforeRouteLeave((to) => {
    if (leaving || !dirty.value) return true
    pendingLeave.value = () => {
        leaving = true
        router.push(to.fullPath)
    }
    return false
})

function onKey(e: KeyboardEvent) {
    if (pendingLeave.value || e.key !== 'Escape') return
    // Fields keep Escape for themselves, e.g. a native date picker
    if ((e.target as HTMLElement)?.closest?.('input, textarea')) return
    cancel()
}

onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))

const visibilityOptions = computed<{ value: Visibility; label: string }[]>(
    () => [
        { value: 'private', label: t('common.onlyMe') },
        { value: 'public', label: t('common.public') },
    ]
)

const heading = computed(() =>
    id.value ? t('trip.editTitle') : t('trip.newTitle')
)
</script>

<template>
    <AppSheet v-model:expanded="expanded" wide :label="heading">
        <!-- Placing: on phones this is what the collapsed sheet shows -->
        <div v-if="naming" class="placing">
            <div>
                <h2>{{ t('trip.nameStop') }}</h2>
                <p class="sub">
                    <span :class="flagOf(naming.iso ?? '')" />
                    {{
                        t('trip.inCountry', {
                            country: nameOf(naming.iso ?? ''),
                        })
                    }}
                </p>
            </div>
            <div>
                <label for="new-stop-name">{{
                    t('trip.stopNamePlaceholder')
                }}</label>
                <input
                    id="new-stop-name"
                    ref="namingInput"
                    v-model="placeName"
                    class="field"
                    maxlength="120"
                    enterkeyhint="done"
                    @keydown.enter.prevent="addNamedStop"
                />
            </div>
        </div>
        <div v-else-if="placing" class="placing">
            <div>
                <h2>
                    {{
                        placingStop
                            ? t('trip.movingStop', { name: label(placingStop) })
                            : t('trip.addStop')
                    }}
                </h2>
                <p class="sub">
                    {{
                        coarsePointer
                            ? t('trip.placeByPan')
                            : t('trip.placeByClick')
                    }}
                </p>
            </div>
            <div
                v-if="!placingStop && !coarsePointer && store.pin?.iso"
                class="place-name"
            >
                <input
                    v-model="placeName"
                    class="field"
                    :placeholder="t('trip.stopNamePlaceholder')"
                    :aria-label="t('trip.stopNamePlaceholder')"
                    maxlength="120"
                    @input="onNameInput"
                    @keydown.enter.prevent="confirmPlace"
                />
            </div>
            <p v-if="store.pin && !store.pin.iso" class="hint">
                {{ t('trip.onLand') }}
            </p>
            <p v-else-if="pinCountry" class="sub">
                {{ t('trip.inCountry', { country: pinCountry }) }}
            </p>
            <div class="actions">
                <button
                    class="btn btn-primary"
                    type="button"
                    :disabled="!store.pin?.iso"
                    @click="confirmPlace"
                >
                    {{
                        placingStop
                            ? t('trip.moveHere')
                            : coarsePointer
                              ? t('trip.addStopHere')
                              : t('trip.addStop')
                    }}
                </button>
                <button
                    class="btn btn-secondary"
                    type="button"
                    @click="stopPlacing"
                >
                    {{
                        !placingStop && draft.stops.length
                            ? t('common.done')
                            : t('common.cancel')
                    }}
                </button>
            </div>
        </div>

        <template v-else>
            <div class="head">
                <div>
                    <h2>{{ draft.title.trim() || heading }}</h2>
                    <p class="sub">
                        {{ t('trip.stopCount', { count: draft.stops.length }) }}
                    </p>
                </div>
            </div>

            <p v-if="loading" class="sub">{{ t('trip.loading') }}</p>
            <p v-else-if="notFound" class="hint">
                {{ store.canWrite ? t('trip.notFound') : t('trip.notOffline') }}
            </p>

            <fieldset class="fields" :disabled="readOnly">
                <div>
                    <label for="trip-title">{{ t('trip.title') }}</label>
                    <input
                        id="trip-title"
                        v-model="draft.title"
                        class="field"
                        :placeholder="t('trip.titlePlaceholder')"
                        :maxlength="MAX_TITLE"
                    />
                </div>
                <div class="dates">
                    <div>
                        <label for="trip-start">{{ t('trip.startsOn') }}</label>
                        <input
                            id="trip-start"
                            v-model="draft.startsOn"
                            class="field"
                            type="date"
                            min="1900-01-01"
                            max="2100-12-31"
                        />
                    </div>
                    <div>
                        <label for="trip-end">{{ t('trip.endsOn') }}</label>
                        <input
                            id="trip-end"
                            v-model="draft.endsOn"
                            class="field"
                            type="date"
                            :min="draft.startsOn || '1900-01-01'"
                            max="2100-12-31"
                        />
                    </div>
                </div>
                <SegmentedControl
                    v-model="draft.visibility"
                    class="visibility"
                    :options="visibilityOptions"
                    :label="t('trip.visibility')"
                    :disabled="readOnly"
                />

                <div>
                    <h3>{{ t('trip.stops') }}</h3>
                    <p v-if="!draft.stops.length" class="sub">
                        {{ t('trip.noStops') }}
                    </p>
                    <ol class="stops">
                        <li
                            v-for="(stop, i) in draft.stops"
                            :key="stop.key"
                            :class="{ dragging: dragIndex === i }"
                            :draggable="!coarsePointer && !readOnly"
                            @dragstart="dragIndex = i"
                            @dragend="dragIndex = null"
                            @dragover.prevent
                            @drop.prevent="onDrop(i)"
                        >
                            <div class="row">
                                <span
                                    class="number"
                                    :title="
                                        coarsePointer
                                            ? undefined
                                            : t('trip.dragStop', {
                                                  name: label(stop),
                                              })
                                    "
                                    >{{ i + 1 }}</span
                                >
                                <div class="stop-name">
                                    <input
                                        :ref="
                                            (el) =>
                                                (nameInputs[stop.key] =
                                                    el as HTMLInputElement | null)
                                        "
                                        v-model="stop.name"
                                        class="field"
                                        :aria-label="
                                            t('trip.stopName', { n: i + 1 })
                                        "
                                        :placeholder="
                                            t('trip.stopNamePlaceholder')
                                        "
                                        maxlength="120"
                                    />
                                    <span class="country">
                                        <span :class="flagOf(stop.country)" />
                                        {{ nameOf(stop.country) }}
                                    </span>
                                </div>
                                <div class="tools">
                                    <button
                                        type="button"
                                        class="icon"
                                        :disabled="i === 0"
                                        :aria-label="
                                            t('trip.moveUp', {
                                                name: label(stop),
                                            })
                                        "
                                        :title="
                                            t('trip.moveUp', {
                                                name: label(stop),
                                            })
                                        "
                                        @click="moveStop(i, i - 1)"
                                    >
                                        ↑
                                    </button>
                                    <button
                                        type="button"
                                        class="icon"
                                        :disabled="i === draft.stops.length - 1"
                                        :aria-label="
                                            t('trip.moveDown', {
                                                name: label(stop),
                                            })
                                        "
                                        :title="
                                            t('trip.moveDown', {
                                                name: label(stop),
                                            })
                                        "
                                        @click="moveStop(i, i + 1)"
                                    >
                                        ↓
                                    </button>
                                    <button
                                        type="button"
                                        class="icon"
                                        :aria-label="
                                            t('trip.move', {
                                                name: label(stop),
                                            })
                                        "
                                        :title="
                                            t('trip.move', {
                                                name: label(stop),
                                            })
                                        "
                                        @click="startPlacing(stop)"
                                    >
                                        📍
                                    </button>
                                    <button
                                        type="button"
                                        class="icon"
                                        :aria-label="
                                            t('trip.removeStop', {
                                                name: label(stop),
                                            })
                                        "
                                        :title="
                                            t('trip.removeStop', {
                                                name: label(stop),
                                            })
                                        "
                                        @click="removeStop(i)"
                                    >
                                        ×
                                    </button>
                                </div>
                            </div>
                            <details
                                class="stop-details"
                                :open="!!stop.arrival"
                            >
                                <summary>{{ t('trip.stopDetails') }}</summary>
                                <label :for="`arrival-${stop.key}`">{{
                                    t('trip.arrival')
                                }}</label>
                                <input
                                    :id="`arrival-${stop.key}`"
                                    v-model="stop.arrival"
                                    class="field"
                                    type="date"
                                    min="1900-01-01"
                                    max="2100-12-31"
                                />
                            </details>
                        </li>
                    </ol>
                    <button
                        class="add-stop"
                        type="button"
                        :disabled="!canAddStop"
                        :title="addStopTitle"
                        @click="startPlacing()"
                    >
                        + {{ t('trip.addStop') }}
                    </button>
                </div>

                <div>
                    <label for="trip-notes">{{ t('trip.notes') }}</label>
                    <textarea
                        id="trip-notes"
                        v-model="draft.notes"
                        class="field"
                        rows="3"
                        maxlength="4000"
                        :placeholder="t('trip.notesPlaceholder')"
                    />
                </div>
            </fieldset>

            <div class="footer">
                <p v-if="!store.canWrite" class="hint offline">
                    {{ t('common.offlineHint') }}
                </p>
                <p v-for="problem in problems" :key="problem" class="hint">
                    {{ problem }}
                </p>
                <div class="actions">
                    <button
                        class="btn btn-secondary"
                        type="button"
                        @click="cancel"
                    >
                        {{ t('common.cancel') }}
                    </button>
                    <button
                        class="btn btn-primary"
                        type="button"
                        :disabled="!canSave"
                        :title="
                            store.canWrite ? undefined : t('common.offlineHint')
                        "
                        @click="save"
                    >
                        {{ t('common.save') }}
                    </button>
                </div>
            </div>
        </template>
    </AppSheet>

    <ConfirmModal
        v-if="pendingLeave"
        :title="t('trip.discardTitle')"
        :message="t('trip.discardMessage')"
        :confirm-label="t('trip.discard')"
        @confirm="(pendingLeave!(), (pendingLeave = null))"
        @cancel="pendingLeave = null"
    />
</template>

<style scoped>
h2 {
    font-family: 'DM Serif Display', serif;
    font-size: 1rem;
    font-weight: 400;
    overflow-wrap: anywhere;
}

h3 {
    margin: 0 0 0.3rem;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-soft);
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

textarea.field {
    resize: vertical;
}

.placing,
.fields {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
}

.fields {
    min-width: 0;
    margin: 0;
    padding: 0;
    border: none;
}

/* Also the peek of the collapsed phone sheet */
.head {
    display: flex;
    flex: none;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
}

.head > div {
    min-width: 0;
}

.dates {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.5rem;
}

/* Full width like the fields above, so both halves share it */
.visibility {
    display: flex;
}

.visibility :deep(button) {
    flex: 1;
}

.actions {
    display: flex;
    gap: 0.5rem;
}

.btn {
    flex: 1;
    min-height: var(--touch);
    padding: 0.5rem 0.9rem;
    border: 1px solid transparent;
    border-radius: 0.5rem;
    font: inherit;
    font-size: 0.82rem;
    font-weight: 500;
    cursor: pointer;
}

.btn-primary {
    background: #3a7d5c;
    color: #fff;
}

.btn-secondary {
    border-color: var(--line-strong);
    background: transparent;
    color: var(--text);
}

.btn:disabled {
    opacity: 0.45;
    cursor: not-allowed;
}

.hint {
    margin: 0;
    font-size: 0.72rem;
    color: #c0392b;
}

.hint.offline {
    color: #8a5a12;
}

.stops {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin: 0;
    padding: 0;
    list-style: none;
}

.stops li {
    padding: 0.4rem;
    border: 1px solid var(--line);
    border-radius: 0.5rem;
    background: var(--raised);
}

.stops li.dragging {
    opacity: 0.5;
}

/* Below the list, where the next stop will appear */
.add-stop {
    width: 100%;
    min-height: var(--touch);
    margin-top: 0.5rem;
    border: 1px dashed var(--line-strong);
    border-radius: 0.5rem;
    background: transparent;
    color: #3a7d5c;
    font: inherit;
    font-size: 0.82rem;
    font-weight: 500;
    cursor: pointer;
}

.add-stop:hover:not(:disabled) {
    background: var(--line-faint);
}

.add-stop:disabled {
    color: var(--muted);
    opacity: 0.6;
    cursor: not-allowed;
}

.row {
    display: flex;
    align-items: flex-start;
    gap: 0.4rem;
}

.number {
    display: grid;
    flex: none;
    place-items: center;
    width: 1.6rem;
    height: 1.6rem;
    margin-top: 0.55rem;
    border-radius: 50%;
    background: var(--trip);
    color: var(--on-status);
    font-size: 0.75rem;
    font-weight: 600;
}

li[draggable='true'] .number {
    cursor: grab;
}

.stop-name {
    flex: 1;
    min-width: 0;
}

.country {
    display: block;
    margin-top: 0.2rem;
    font-size: 0.72rem;
    color: var(--muted);
}

.tools {
    display: flex;
    flex: none;
    flex-wrap: wrap;
    justify-content: flex-end;
    max-width: calc(2 * var(--touch));
}

.icon {
    width: var(--touch);
    height: var(--touch);
    padding: 0;
    border: none;
    background: none;
    color: var(--muted);
    font-size: 1rem;
    line-height: 1;
    cursor: pointer;
}

.icon:disabled {
    opacity: 0.3;
}

.stop-details {
    margin-top: 0.2rem;
}

.stop-details summary {
    padding: 0.3rem 0;
    font-size: 0.72rem;
    color: var(--muted);
    cursor: pointer;
}

.stop-details .field + label {
    margin-top: 0.4rem;
}

/* Sticky at the bottom of the sheet, like the country details */
.footer {
    position: sticky;
    bottom: calc(-1 * var(--sheet-pad, 0px));
    z-index: 1;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    margin: 0 calc(-1 * var(--sheet-pad, 0px)) calc(-1 * var(--sheet-pad, 0px));
    padding: 0.75rem var(--sheet-pad, 0px) var(--sheet-pad, 0px);
    background: var(--panel);
    border-top: 1px solid var(--line);
}

/* Mouse: the tools fit in one row */
@media (pointer: fine) {
    .tools {
        flex-wrap: nowrap;
        max-width: none;
    }

    .icon {
        width: 1.9rem;
        height: 1.9rem;
    }

    .field,
    .btn {
        min-height: 0;
    }
}
</style>
