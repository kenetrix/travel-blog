<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import PageWrapper from '../ui/PageWrapper.vue'
import ConfirmModal from '../ui/ConfirmModal.vue'
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

function setRating(n: number) {
    form.value.rating = form.value.rating === n ? 0 : n
}

// Saves the whole trip with the given stops; the editor's PUT replaces them all
async function saveStops(stops: Trip['stops']): Promise<boolean> {
    const current = trip.value
    if (!current) return false
    saving.value = true
    const saved = await store.saveTrip(current.id, {
        title: current.title,
        startsOn: current.startsOn ?? '',
        endsOn: current.endsOn ?? '',
        notes: current.notes,
        visibility: current.visibility,
        stops: stops.map((s) => ({ ...s, arrival: s.arrival ?? '' })),
    })
    saving.value = false
    if (!saved) return false
    trip.value = saved
    return true
}

async function saveDetails() {
    const current = trip.value
    if (!current || !editingId.value || !linksValid.value) return
    const ok = await saveStops(
        current.stops.map((s) =>
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
        )
    )
    if (ok) editingId.value = null
}

// Stops are only changed in edit mode
const editMode = ref(false)
const removing = ref<Trip['stops'][number] | null>(null)

function toggleEditMode() {
    editMode.value = !editMode.value
    editingId.value = null
}

async function removeStop() {
    const stop = removing.value
    removing.value = null
    if (!stop || !trip.value) return
    await saveStops(trip.value.stops.filter((s) => s.id !== stop.id))
}

async function remove() {
    confirmDelete.value = false
    if (!trip.value || !(await store.deleteTrip(trip.value.id))) return
    store.notify(t('trip.deleted'))
    router.push({ name: 'home' })
}
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
                    :disabled="!store.canWrite"
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
                <p v-if="trip.countries.length" class="countries">
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
                <p v-if="trip.notes" class="text">{{ trip.notes }}</p>
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
                                <span
                                    v-if="stop.rating"
                                    class="rating"
                                    :aria-label="
                                        t('trip.stars', { count: stop.rating })
                                    "
                                >
                                    <span aria-hidden="true">{{
                                        '★'.repeat(stop.rating)
                                    }}</span
                                    ><span class="rest" aria-hidden="true">{{
                                        '★'.repeat(5 - stop.rating)
                                    }}</span>
                                </span>
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
                                        class="icon"
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
                                        class="chip"
                                        @click="editingId = null"
                                    >
                                        {{ t('common.cancel') }}
                                    </button>
                                    <button
                                        type="submit"
                                        class="chip on"
                                        :disabled="saving || !linksValid"
                                    >
                                        {{ t('common.save') }}
                                    </button>
                                </div>
                            </form>
                            <div v-else-if="editMode" class="stop-actions">
                                <button
                                    type="button"
                                    class="edit-details"
                                    @click="startEdit(stop)"
                                >
                                    ✎ {{ t('trip.editDetails') }}
                                </button>
                                <button
                                    type="button"
                                    class="edit-details"
                                    :disabled="saving"
                                    @click="removing = stop"
                                >
                                    × {{ t('common.remove') }}
                                </button>
                            </div>
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
                    :to="{ name: 'trip-edit', params: { id: trip.id } }"
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
        @confirm="removeStop"
        @cancel="removing = null"
    />
    <ConfirmModal
        v-if="confirmDelete && trip"
        :title="t('trip.deleteTitle')"
        :message="t('trip.deleteMessage', { title: trip.title })"
        :confirm-label="t('common.delete')"
        @confirm="remove"
        @cancel="confirmDelete = false"
    />
</template>

<style src="../ui/panel.css"></style>

<style scoped>
.header-actions {
    display: flex;
    gap: 0.4rem;
}

.header-actions .btn {
    display: inline-flex;
    align-items: center;
    box-sizing: border-box;
    min-height: var(--touch);
    padding: 0.4rem 0.9rem;
    border: none;
    border-radius: 0.5rem;
    background: #3a7d5c;
    color: #fff;
    font: inherit;
    font-size: 0.8rem;
    font-weight: 500;
    text-decoration: none;
    cursor: pointer;
}

.header-actions .btn.secondary {
    border: 1px solid var(--line-strong);
    background: transparent;
    color: var(--text);
}

.header-actions .btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
}

@media (pointer: fine) {
    .header-actions .btn {
        min-height: 0;
    }
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

.chip {
    min-height: 30px;
    padding: 0.1rem 0.6rem;
    border: 1px solid var(--line-strong);
    border-radius: 999px;
    background: transparent;
    color: var(--text);
    font: inherit;
    font-size: 0.74rem;
    cursor: pointer;
}

.chip.on {
    border-color: var(--trip);
    background: color-mix(in srgb, var(--trip) 18%, transparent);
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
    align-items: baseline;
    justify-content: space-between;
    gap: 0 0.8rem;
}

.rating {
    color: #ca8f22;
    font-size: 0.8rem;
    letter-spacing: 0.05em;
    white-space: nowrap;
}

.rating .rest {
    color: var(--faint);
}

.stop-actions {
    display: flex;
    gap: 0.9rem;
}

.add-stop {
    display: block;
    margin-top: 0.2rem;
    padding: 0.6rem;
    border: 1px dashed var(--line-strong);
    border-radius: 0.6rem;
    color: var(--muted);
    font-size: 0.82rem;
    text-align: center;
    text-decoration: none;
}

.edit-details {
    margin-top: 0.3rem;
    padding: 0.2rem 0;
    border: none;
    background: transparent;
    color: var(--muted);
    font: inherit;
    font-size: 0.74rem;
    cursor: pointer;
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
    color: #ca8f22;
}

.link-row {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 0.3rem;
}

.link-row .icon {
    grid-column: 2;
    grid-row: 1 / span 2;
}

.add-link {
    align-self: flex-start;
    border: none;
    background: transparent;
    padding: 0.2rem 0;
    color: var(--muted);
    font: inherit;
    font-size: 0.78rem;
    cursor: pointer;
}

.error {
    color: var(--danger, #e5645c);
    font-size: 0.75rem;
}

.form-actions {
    display: flex;
    gap: 0.4rem;
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
