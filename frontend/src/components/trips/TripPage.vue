<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import PageWrapper from '../ui/PageWrapper.vue'
import ConfirmModal from '../ui/ConfirmModal.vue'
import SegmentedControl from '../ui/SegmentedControl.vue'
import { useTravelStore, type Trip } from '../stores/store'
import { countryName, getCountryByIso } from '../countries/countries'
import { pathKm } from '../geo/geo'
import { formatDay, formatDistance } from '../../i18n/format'
import { t } from '../../i18n'
import { tripDates, tripDays } from './dates'
import { STOP_CATEGORIES, categoryIcon, type StopCategory } from './categories'

const store = useTravelStore()
const route = useRoute()
const router = useRouter()

const id = computed(() => String(route.params.id))
const trip = ref<Trip | null>(null)
const loading = ref(true)
const confirmDelete = ref(false)

watch(
    id,
    async (tripId) => {
        // A trip opened before shows at once; the server copy replaces it
        trip.value = store.tripDetails[tripId] ?? null
        loading.value = !trip.value
        const loaded = await store.loadTrip(tripId)
        if (tripId !== id.value) return
        trip.value = loaded
        loading.value = false
    },
    { immediate: true }
)

function nameOf(iso: string): string {
    const country = getCountryByIso(iso)
    return country ? countryName(country) : iso
}

function flagOf(iso: string): string {
    const country = getCountryByIso(iso)
    return country ? `fi fi-${country.alpha2.toLowerCase()}` : ''
}

const days = computed(() => (trip.value ? tripDays(trip.value) : null))

const km = computed(() =>
    trip.value && trip.value.stops.length > 1
        ? pathKm(trip.value.stops.map((s) => [s.lng, s.lat]))
        : null
)

// Kinds of stops used in this trip, as filter chips
const usedCategories = computed(() =>
    STOP_CATEGORIES.filter((c) =>
        trip.value?.stops.some((s) => s.category === c.value)
    )
)
const filter = ref('')
const visibleStops = computed(() =>
    (trip.value?.stops ?? [])
        .map((stop, i) => ({ stop, n: i + 1 }))
        .filter(({ stop }) => !filter.value || stop.category === filter.value)
)

function linkLabel(link: { url: string; label?: string }): string {
    if (link.label) return link.label
    try {
        return new URL(link.url).hostname.replace(/^www\./, '')
    } catch {
        return link.url
    }
}

// --- Stop details: category, rating and links are edited here, not in the
// map editor
const MAX_LINKS = 10
const LINK_PATTERN = /^https?:\/\/\S+$/i

interface DetailsForm {
    notes: string
    category: StopCategory | ''
    rating: number
    links: { url: string; label: string }[]
}

const editingId = ref<string | null>(null)
const form = ref<DetailsForm>({
    notes: '',
    category: '',
    rating: 0,
    links: [],
})
const saving = ref(false)

const linksValid = computed(() =>
    form.value.links.every(
        (l) => !l.url.trim() || LINK_PATTERN.test(l.url.trim())
    )
)

function startEdit(stop: Trip['stops'][number]) {
    editingId.value = stop.id
    form.value = {
        notes: stop.notes,
        category: stop.category,
        rating: stop.rating,
        links: stop.links.map((l) => ({ url: l.url, label: l.label ?? '' })),
    }
}

const editingStop = computed(() =>
    trip.value?.stops.find((s) => s.id === editingId.value)
)

const detailsDirty = computed(() => {
    const stop = editingStop.value
    if (!stop) return false
    return (
        JSON.stringify(form.value) !==
        JSON.stringify({
            notes: stop.notes,
            category: stop.category,
            rating: stop.rating,
            links: stop.links.map((l) => ({
                url: l.url,
                label: l.label ?? '',
            })),
        })
    )
})

function setRating(n: number) {
    form.value.rating = form.value.rating === n ? 0 : n
}

// Saves the whole trip, as the editor's PUT replaces everything, with the
// given stops or trip fields changed
async function saveTrip(
    changes: { stops?: Trip['stops'] } & Partial<TripForm>
): Promise<boolean> {
    const current = trip.value
    if (!current) return false
    saving.value = true
    const saved = await store.saveTrip(current.id, {
        title: current.title,
        startsOn: current.startsOn ?? '',
        endsOn: current.endsOn ?? '',
        notes: current.notes,
        visibility: current.visibility,
        ...changes,
        stops: (changes.stops ?? current.stops).map((s) => ({
            ...s,
            arrival: s.arrival ?? '',
        })),
    })
    saving.value = false
    if (!saved) return false
    trip.value = saved
    return true
}

async function saveDetails() {
    const current = trip.value
    if (!current || !editingId.value || !linksValid.value) return
    const ok = await saveTrip({
        stops: current.stops.map((s) =>
            s.id !== editingId.value
                ? s
                : {
                      ...s,
                      notes: form.value.notes.trim(),
                      rating: form.value.rating,
                      category: form.value.category,
                      links: form.value.links
                          .filter((l) => l.url.trim())
                          .map((l) => ({
                              url: l.url.trim(),
                              label: l.label.trim(),
                          })),
                  }
        ),
    })
    if (ok) editingId.value = null
}

// Edit mode manages the stops and the trip's own fields; stop details can
// be edited any time. Coming back from adding stops in the editor, it is
// still on; the query is dropped like the editor's.
const editMode = ref(Boolean(route.query.edit))
if (route.query.edit) {
    router.replace({ query: { ...route.query, edit: undefined } })
}
const removing = ref<Trip['stops'][number] | null>(null)

interface TripForm {
    title: string
    startsOn: string
    endsOn: string
    notes: string
    visibility: Trip['visibility']
}

const MAX_TITLE = 120
const tripForm = ref<TripForm>({
    title: '',
    startsOn: '',
    endsOn: '',
    notes: '',
    visibility: 'private',
})

function formOf(trip: Trip): TripForm {
    return {
        title: trip.title,
        startsOn: trip.startsOn ?? '',
        endsOn: trip.endsOn ?? '',
        notes: trip.notes,
        visibility: trip.visibility,
    }
}

watch(
    [editMode, () => trip.value?.id],
    () => {
        if (editMode.value && trip.value) tripForm.value = formOf(trip.value)
    },
    { immediate: true }
)

const tripDirty = computed(
    () =>
        !!trip.value &&
        JSON.stringify(tripForm.value) !== JSON.stringify(formOf(trip.value))
)

const tripProblems = computed(() => {
    const f = tripForm.value
    const list: string[] = []
    if (!f.title.trim()) list.push(t('trip.titleRequired'))
    if (f.startsOn && f.endsOn && f.endsOn < f.startsOn) {
        list.push(t('trip.endBeforeStart'))
    }
    return list
})

const visibilityOptions = computed(() => [
    { value: 'private' as const, label: t('common.onlyMe') },
    { value: 'public' as const, label: t('common.public') },
])

// "Done" saves the trip fields; with problems or a failed save it stays open
async function toggleEditMode() {
    editingId.value = null
    if (!editMode.value) {
        editMode.value = true
        return
    }
    if (tripDirty.value) {
        if (tripProblems.value.length) return
        const f = tripForm.value
        const ok = await saveTrip({
            ...f,
            title: f.title.trim(),
            notes: f.notes.trim(),
        })
        if (!ok) return
    }
    editMode.value = false
}

async function removeStop() {
    const stop = removing.value
    removing.value = null
    if (!stop || !trip.value) return
    await saveTrip({
        stops: trip.value.stops.filter((s) => s.id !== stop.id),
    })
}

async function remove() {
    confirmDelete.value = false
    if (!trip.value || !(await store.deleteTrip(trip.value.id))) return
    store.notify(t('trip.deleted'))
    leaving = true
    router.push({ name: 'home' })
}

// Like the editors: leaving with unsaved trip fields or stop details asks first
const pendingLeave = ref<(() => void) | null>(null)
let leaving = false

onBeforeRouteLeave((to) => {
    if (leaving || !(tripDirty.value || detailsDirty.value)) return true
    pendingLeave.value = () => {
        leaving = true
        router.push(to.fullPath)
    }
    return false
})
</script>

<template>
    <PageWrapper
        :title="trip?.title ?? ''"
        :subtitle="trip ? tripDates(trip) : undefined"
    >
        <template v-if="trip" #actions>
            <div class="header-actions">
                <button
                    class="btn"
                    :disabled="
                        !store.canWrite ||
                        saving ||
                        (editMode && tripProblems.length > 0)
                    "
                    :title="
                        store.canWrite ? undefined : t('common.offlineHint')
                    "
                    @click="toggleEditMode"
                >
                    {{ editMode ? t('common.done') : t('trip.edit') }}
                </button>
                <button
                    class="btn secondary"
                    :disabled="!store.canWrite"
                    :title="
                        store.canWrite ? undefined : t('common.offlineHint')
                    "
                    @click="confirmDelete = true"
                >
                    {{ t('common.delete') }}
                </button>
            </div>
        </template>

        <p v-if="loading" class="state">{{ t('trip.loading') }}</p>
        <div v-else-if="!trip" class="panel state">
            <p>
                {{ store.canWrite ? t('trip.notFound') : t('trip.notOffline') }}
            </p>
            <RouterLink class="btn secondary" :to="{ name: 'home' }">
                {{ t('common.backToMap') }}
            </RouterLink>
        </div>

        <template v-else>
            <section class="panel overview">
                <div class="tiles">
                    <div v-if="days" class="tile">
                        <strong>{{ days }}</strong>
                        <span>{{ t('trip.length') }}</span>
                    </div>
                    <div class="tile">
                        <strong>{{ trip.stops.length }}</strong>
                        <span>{{ t('trip.stops') }}</span>
                    </div>
                    <div v-if="trip.countries.length" class="tile">
                        <strong>{{ trip.countries.length }}</strong>
                        <span>{{ t('trip.countries') }}</span>
                    </div>
                    <div v-if="km !== null" class="tile">
                        <strong>{{
                            formatDistance(km, store.settings.distanceUnit)
                        }}</strong>
                        <span>{{ t('trip.distance') }}</span>
                    </div>
                </div>
                <div v-if="editMode" class="trip-form">
                    <div class="field-group">
                        <label for="trip-title">{{ t('trip.title') }}</label>
                        <input
                            id="trip-title"
                            v-model="tripForm.title"
                            :placeholder="t('trip.titlePlaceholder')"
                            :maxlength="MAX_TITLE"
                        />
                    </div>
                    <div class="dates">
                        <div class="field-group">
                            <label for="trip-start">{{
                                t('trip.startsOn')
                            }}</label>
                            <input
                                id="trip-start"
                                v-model="tripForm.startsOn"
                                type="date"
                                min="1900-01-01"
                                max="2100-12-31"
                            />
                        </div>
                        <div class="field-group">
                            <label for="trip-end">{{ t('trip.endsOn') }}</label>
                            <input
                                id="trip-end"
                                v-model="tripForm.endsOn"
                                type="date"
                                :min="tripForm.startsOn || '1900-01-01'"
                                max="2100-12-31"
                            />
                        </div>
                    </div>
                    <SegmentedControl
                        v-model="tripForm.visibility"
                        :options="visibilityOptions"
                        :label="t('trip.visibility')"
                    />
                    <div class="field-group">
                        <label for="trip-notes">{{ t('trip.notes') }}</label>
                        <textarea
                            id="trip-notes"
                            v-model="tripForm.notes"
                            rows="3"
                            maxlength="4000"
                            :placeholder="t('trip.notesPlaceholder')"
                        />
                    </div>
                    <p v-for="p in tripProblems" :key="p" class="error">
                        {{ p }}
                    </p>
                </div>
                <p v-else-if="trip.countries.length" class="countries">
                    <span v-for="iso in trip.countries" :key="iso" class="pill">
                        <span :class="flagOf(iso)" />
                        {{ nameOf(iso) }}
                    </span>
                    <span class="pill muted">{{
                        trip.visibility === 'public'
                            ? t('common.public')
                            : t('common.onlyMe')
                    }}</span>
                </p>
                <p v-if="trip.notes && !editMode" class="text">
                    {{ trip.notes }}
                </p>
            </section>

            <section v-if="trip.stops.length || editMode" class="panel">
                <div class="stops-head">
                    <h2>{{ t('trip.route') }}</h2>
                    <div
                        v-if="usedCategories.length"
                        class="filters"
                        role="group"
                        :aria-label="t('trip.category')"
                    >
                        <button
                            type="button"
                            class="chip"
                            :class="{ on: !filter }"
                            :aria-pressed="!filter"
                            @click="filter = ''"
                        >
                            {{ t('trip.filterAll') }}
                        </button>
                        <button
                            v-for="c in usedCategories"
                            :key="c.value"
                            type="button"
                            class="chip"
                            :class="{ on: filter === c.value }"
                            :aria-pressed="filter === c.value"
                            @click="filter = filter === c.value ? '' : c.value"
                        >
                            {{ c.icon }} {{ t(`trip.categories.${c.value}`) }}
                        </button>
                    </div>
                </div>
                <ol class="stops">
                    <li v-for="{ stop, n } in visibleStops" :key="stop.id">
                        <span class="number">{{ n }}</span>
                        <div class="stop">
                            <div class="stop-top">
                                <p class="name">{{ stop.name }}</p>
                                <div class="stop-side">
                                    <span
                                        v-if="stop.rating"
                                        class="rating"
                                        :aria-label="
                                            t('trip.stars', {
                                                count: stop.rating,
                                            })
                                        "
                                    >
                                        <span aria-hidden="true">{{
                                            '★'.repeat(stop.rating)
                                        }}</span
                                        ><span
                                            class="rest"
                                            aria-hidden="true"
                                            >{{
                                                '★'.repeat(5 - stop.rating)
                                            }}</span
                                        >
                                    </span>
                                    <button
                                        v-if="
                                            store.canWrite &&
                                            editingId !== stop.id
                                        "
                                        type="button"
                                        class="icon-btn"
                                        :aria-label="t('trip.editDetails')"
                                        :title="t('trip.editDetails')"
                                        @click="startEdit(stop)"
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
                                    <button
                                        v-if="
                                            store.canWrite &&
                                            editMode &&
                                            editingId !== stop.id
                                        "
                                        type="button"
                                        class="icon-btn danger"
                                        :disabled="saving"
                                        :aria-label="t('trip.removeStopTitle')"
                                        :title="t('trip.removeStopTitle')"
                                        @click="removing = stop"
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
                                            <path d="M3 6h18" />
                                            <path
                                                d="M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2"
                                            />
                                            <path
                                                d="M19 6l-1 14a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1L5 6"
                                            />
                                        </svg>
                                    </button>
                                </div>
                            </div>
                            <p class="meta">
                                <span :class="flagOf(stop.country)" />
                                {{ nameOf(stop.country) }}
                                <template v-if="stop.arrival">
                                    · {{ formatDay(stop.arrival) }}
                                </template>
                                <template v-if="stop.category">
                                    ·
                                    {{ categoryIcon(stop.category) }}
                                    {{ t(`trip.categories.${stop.category}`) }}
                                </template>
                            </p>
                            <p v-if="stop.notes" class="text">
                                {{ stop.notes }}
                            </p>
                            <form
                                v-if="editingId === stop.id"
                                class="details-form"
                                @submit.prevent="saveDetails"
                                @keydown.esc="editingId = null"
                            >
                                <div class="filters" role="group">
                                    <button
                                        v-for="c in STOP_CATEGORIES"
                                        :key="c.value"
                                        type="button"
                                        class="chip"
                                        :class="{
                                            on: form.category === c.value,
                                        }"
                                        :aria-pressed="
                                            form.category === c.value
                                        "
                                        @click="
                                            form.category =
                                                form.category === c.value
                                                    ? ''
                                                    : c.value
                                        "
                                    >
                                        {{ c.icon }}
                                        {{ t(`trip.categories.${c.value}`) }}
                                    </button>
                                </div>
                                <textarea
                                    v-model="form.notes"
                                    class="field"
                                    rows="3"
                                    maxlength="4000"
                                    :placeholder="t('trip.stopNotes')"
                                    :aria-label="t('trip.stopNotes')"
                                />
                                <div
                                    class="star-input"
                                    role="group"
                                    :aria-label="t('trip.rating')"
                                >
                                    <button
                                        v-for="n in 5"
                                        :key="n"
                                        type="button"
                                        class="star"
                                        :class="{ on: n <= form.rating }"
                                        :aria-label="
                                            t('trip.stars', { count: n })
                                        "
                                        @click="setRating(n)"
                                    >
                                        ★
                                    </button>
                                </div>
                                <div
                                    v-for="(link, li) in form.links"
                                    :key="li"
                                    class="link-row"
                                >
                                    <input
                                        v-model="link.url"
                                        class="field"
                                        type="url"
                                        inputmode="url"
                                        maxlength="500"
                                        :placeholder="t('trip.linkUrl')"
                                        :aria-label="t('trip.links')"
                                    />
                                    <input
                                        v-model="link.label"
                                        class="field"
                                        maxlength="80"
                                        :placeholder="t('trip.linkLabel')"
                                        :aria-label="t('trip.linkLabel')"
                                    />
                                    <button
                                        type="button"
                                        class="icon-btn danger"
                                        :aria-label="t('trip.removeLink')"
                                        :title="t('trip.removeLink')"
                                        @click="form.links.splice(li, 1)"
                                    >
                                        ×
                                    </button>
                                </div>
                                <button
                                    v-if="form.links.length < MAX_LINKS"
                                    type="button"
                                    class="add-link"
                                    @click="
                                        form.links.push({ url: '', label: '' })
                                    "
                                >
                                    + {{ t('trip.addLink') }}
                                </button>
                                <p v-if="!linksValid" class="error">
                                    {{ t('trip.invalidLink') }}
                                </p>
                                <div class="form-actions">
                                    <button
                                        type="button"
                                        class="btn secondary"
                                        @click="editingId = null"
                                    >
                                        {{ t('common.cancel') }}
                                    </button>
                                    <button
                                        type="submit"
                                        class="btn"
                                        :disabled="
                                            saving ||
                                            !linksValid ||
                                            !detailsDirty
                                        "
                                    >
                                        {{ t('common.save') }}
                                    </button>
                                </div>
                            </form>
                            <p v-if="stop.links.length" class="links">
                                <a
                                    v-for="(link, li) in stop.links"
                                    :key="li"
                                    :href="link.url"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    :title="t('trip.opensNew')"
                                >
                                    🔗 {{ linkLabel(link) }}
                                </a>
                            </p>
                        </div>
                    </li>
                </ol>
                <RouterLink
                    v-if="editMode"
                    class="add-stop"
                    :to="{
                        name: 'trip-edit',
                        params: { id: trip.id },
                        query: { add: '1', from: 'trip' },
                    }"
                >
                    + {{ t('trip.addStop') }}
                </RouterLink>
            </section>
        </template>
    </PageWrapper>

    <ConfirmModal
        v-if="removing"
        :title="t('trip.removeStopTitle')"
        :message="t('trip.removeStopMessage', { name: removing.name })"
        :confirm-label="t('common.remove')"
        danger
        @confirm="removeStop"
        @cancel="removing = null"
    />
    <ConfirmModal
        v-if="confirmDelete && trip"
        :title="t('trip.deleteTitle')"
        :message="t('trip.deleteMessage', { title: trip.title })"
        :confirm-label="t('common.delete')"
        danger
        @confirm="remove"
        @cancel="confirmDelete = false"
    />
    <ConfirmModal
        v-if="pendingLeave"
        :title="t('trip.discardTitle')"
        :message="t('trip.discardMessage')"
        :confirm-label="t('trip.discard')"
        danger
        @confirm="(pendingLeave!(), (pendingLeave = null))"
        @cancel="pendingLeave = null"
    />
</template>

<style src="../ui/panel.css"></style>

<style scoped>
.header-actions {
    display: flex;
    gap: 0.4rem;
}

.state {
    max-width: 720px;
    margin: 0 auto;
    font-size: 0.9rem;
    color: var(--muted);
}

.state p {
    margin: 0 0 0.8rem;
}

.tiles {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(7rem, 1fr));
    gap: 0.6rem;
}

.tile {
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
    padding: 0.6rem 0.8rem;
    border-radius: 0.6rem;
    background: color-mix(in srgb, var(--trip) 10%, transparent);
}

.tile strong {
    font-size: 1.15rem;
    font-weight: 600;
}

.tile span {
    font-size: 0.72rem;
    color: var(--muted);
}

.countries {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem;
    margin: 0.8rem 0 0;
}

.pill {
    padding: 0.2rem 0.65rem;
    border: 1px solid var(--line-strong);
    border-radius: 999px;
    font-size: 0.78rem;
}

.pill.muted {
    color: var(--muted);
}

.trip-form {
    display: flex;
    flex-direction: column;
    gap: 0.7rem;
    margin-top: 0.9rem;
}

.field-group {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    min-width: 0;
}

.dates {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.6rem;
}

.trip-form .segmented {
    align-self: flex-start;
}

.trip-form .error {
    margin: 0;
}

.overview .text {
    margin-top: 0.8rem;
}

.stops-head {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 0.4rem 1rem;
    margin-bottom: 0.4rem;
}

.stops-head h2 {
    margin: 0;
}

.filters {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
}

.stops {
    display: flex;
    flex-direction: column;
    margin: 0;
    padding: 0;
    list-style: none;
}

.stops li {
    position: relative;
    display: flex;
    gap: 0.7rem;
    padding-bottom: 1.1rem;
}

/* The timeline runs through the numbers down to the next stop */
.stops li:not(:last-child)::before {
    content: '';
    position: absolute;
    top: 1.6rem;
    bottom: 0;
    left: 0.75rem;
    width: 2px;
    background: color-mix(in srgb, var(--trip) 35%, transparent);
}

.stop-top {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 0 0.8rem;
}

.stop-side {
    display: flex;
    align-items: center;
    gap: 0.3rem;
}

/* Touch targets reach into the row's spacing instead of stretching it */
@media (pointer: coarse) {
    .stop-side .icon-btn {
        margin: -6px -6px -6px 0;
    }
}

.rating {
    color: var(--star);
    font-size: 0.8rem;
    letter-spacing: 0.05em;
    white-space: nowrap;
}

.rating .rest {
    color: var(--faint);
}

/* Below the list, where the next stop will appear; as in the editor */
.add-stop {
    display: grid;
    place-items: center;
    min-height: var(--touch);
    margin-top: 0.2rem;
    border: 1px dashed var(--line-strong);
    border-radius: 0.5rem;
    color: var(--accent);
    font-size: 0.82rem;
    font-weight: 500;
    text-decoration: none;
}

.add-stop:hover {
    background: var(--line-faint);
}

@media (pointer: fine) {
    .add-stop {
        min-height: 2rem;
    }
}

.details-form {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-top: 0.5rem;
}

.star-input {
    display: flex;
}

.star {
    min-width: 32px;
    min-height: 36px;
    border: none;
    background: transparent;
    padding: 0 0.05rem;
    font-size: 1.25rem;
    line-height: 1;
    color: var(--faint);
    cursor: pointer;
}

.star.on {
    color: var(--star);
}

.link-row {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 0.3rem;
}

.link-row .icon-btn {
    grid-column: 2;
    grid-row: 1 / span 2;
}

.add-link {
    align-self: flex-start;
    border: none;
    background: transparent;
    padding: 0.2rem 0;
    color: var(--accent);
    font: inherit;
    font-size: 0.78rem;
    font-weight: 500;
    cursor: pointer;
}

.error {
    color: var(--danger);
    font-size: 0.75rem;
}

.form-actions {
    display: flex;
    gap: 0.5rem;
}

.form-actions .btn {
    flex: 1;
}

.links {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem;
    margin-top: 0.45rem !important;
}

.links a {
    padding: 0.15rem 0.6rem;
    border: 1px solid var(--line-strong);
    border-radius: 999px;
    color: var(--text);
    font-size: 0.75rem;
    text-decoration: none;
    overflow-wrap: anywhere;
}

.links a:hover {
    border-color: var(--trip);
}

.number {
    display: grid;
    flex: none;
    place-items: center;
    z-index: 1;
    width: 1.6rem;
    height: 1.6rem;
    border-radius: 50%;
    background: var(--trip);
    color: var(--on-status);
    font-size: 0.75rem;
    font-weight: 600;
}

.stop {
    flex: 1;
    min-width: 0;
}

.stop p {
    margin: 0;
}

.name {
    font-weight: 500;
    font-size: 0.9rem;
    overflow-wrap: anywhere;
}

.meta {
    font-size: 0.75rem;
    color: var(--muted);
}

.text {
    margin: 0.3rem 0 0;
    font-size: 0.85rem;
    line-height: 1.45;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
}
</style>
