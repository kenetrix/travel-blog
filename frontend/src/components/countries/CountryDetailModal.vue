<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount, type Directive } from 'vue'
import {
    useTravelStore,
    emptyDetails,
    type CountryStatus,
    type Visibility,
} from '../stores/store'
import {
    countryName,
    countryNameOf,
    flagClass,
    getCountryByIso,
} from './countries'
import { getCountryFacts } from './countryFacts'
import {
    currencyName,
    formatList,
    formatNumber,
    languageName,
} from '../../i18n/format'
import { RouterLink, useRoute } from 'vue-router'
import ConfirmModal from '../ui/ConfirmModal.vue'
import SegmentedControl from '../ui/SegmentedControl.vue'
import { useWindowListener } from '../ui/useWindowListener'
import { visibilityOptions } from '../ui/visibility'
import { localizedData, t } from '../../i18n'
import { tripDates } from '../trips/dates'

const props = defineProps<{ iso: string }>()
const emit = defineEmits<{ close: [] }>()

const store = useTravelStore()
const route = useRoute()

const country = computed(() => getCountryByIso(props.iso))
const facts = computed(() => getCountryFacts(props.iso))
const name = computed(() =>
    country.value ? countryName(country.value) : props.iso
)

const status = ref<CountryStatus>('visited')
const form = ref(emptyDetails())
const vis = ref<Visibility>('private')
const snapshot = ref('')
// What the "discard changes" dialog should do once confirmed
const pendingLeave = ref<(() => void) | null>(null)
const showRemoveConfirm = ref(false)
// The long texts are fetched when the modal opens
const loadingDetails = ref(false)
// Offline and never loaded before, so only rating and tags are known
const unavailable = ref(false)

const SUGGESTED_TAGS = [
    'beach',
    'cityTrip',
    'hiking',
    'culture',
    'nightlife',
] as const

function load(iso: string) {
    status.value = store.visited.has(iso)
        ? 'visited'
        : store.wishlist.has(iso)
          ? 'wishlist'
          : 'none'
    form.value = JSON.parse(
        JSON.stringify({ ...emptyDetails(), ...store.details[iso] })
    )
    vis.value = store.visibilityOf(iso)
    snapshot.value = JSON.stringify([status.value, vis.value, form.value])
}

async function loadAndFill(iso: string) {
    load(iso)
    unavailable.value = false
    loadingDetails.value = true
    const ok = await store.loadCountryDetails(iso)
    // Skip if the user moved on or has already started editing
    if (props.iso !== iso) return
    loadingDetails.value = false
    unavailable.value = !ok && store.connection === 'offline'
    if (ok && !dirty.value) load(iso)
}

watch(() => props.iso, loadAndFill, { immediate: true })

const dirty = computed(
    () =>
        JSON.stringify([status.value, vis.value, form.value]) !== snapshot.value
)

watch(
    [() => props.iso, status],
    ([iso, s]) => store.setPreview({ iso, status: s }),
    { immediate: true }
)

onBeforeUnmount(() => store.setPreview(null))

// Run `action` right away, or after confirming that unsaved edits may go
function leave(action: () => void) {
    if (dirty.value) pendingLeave.value = action
    else action()
}

async function persist() {
    showRemoveConfirm.value = false
    const saved = await store.saveCountry(
        props.iso,
        status.value,
        {
            ...form.value,
            food: form.value.food.trim(),
            sightseeing: form.value.sightseeing.trim(),
            notes: form.value.notes.trim(),
        },
        vis.value
    )
    // On failure the store reports the error and the form stays open
    if (!saved) return
    store.notify(t('country.saved', { name: name.value }))
    emit('close')
}

function save() {
    // Nothing to save; also covers Ctrl+Enter
    if (!dirty.value || loadingDetails.value || !store.canWrite) return
    if (status.value === 'none' && store.visited.has(props.iso)) {
        showRemoveConfirm.value = true
    } else {
        persist()
    }
}

function onKey(e: KeyboardEvent) {
    // The trip page sits on top of this panel
    if (route.name !== 'home') return
    if (pendingLeave.value || showRemoveConfirm.value) return
    if (e.key === 'Escape') leave(() => emit('close'))
    else if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) save()
}

useWindowListener('keydown', onKey)

// --- Facts -------------------------------------------------------------

// Codes (EUR, de) are named by the browser; the rest are English names
const isCurrencyCode = (s: string) => /^[A-Z]{3}$/.test(s)
const isLanguageCode = (s: string) => /^[a-z]{2,3}$/.test(s)

const factRows = computed(() => {
    const f = facts.value
    if (!f) return []
    const area = formatNumber(f.area)
    return [
        [
            t('country.capital'),
            localizedData().capitals[props.iso] ?? f.capital,
        ],
        [t('country.region'), t(`regions.${f.region}`)],
        [
            t('country.population'),
            f.population ? formatNumber(f.population) : '',
        ],
        [t('country.area'), f.area ? t('format.area', { area }) : ''],
        [
            t('country.currency'),
            formatList(
                f.currencies.map((c) =>
                    isCurrencyCode(c) ? currencyName(c, c) : c
                )
            ),
        ],
        [
            t('country.languages'),
            formatList(
                f.languages.map((l) =>
                    isLanguageCode(l) ? languageName(l, l) : l
                )
            ),
        ],
    ].filter(([, value]) => value)
})

// --- Rating ------------------------------------------------------------

function setRating(n: number) {
    form.value.rating = form.value.rating === n ? 0 : n
}

// --- Tags --------------------------------------------------------------

const tagInput = ref('')

function addTag(raw: string) {
    const tag = raw.trim().toLowerCase()
    if (tag && !form.value.tags.includes(tag)) form.value.tags.push(tag)
    tagInput.value = ''
}

function removeTag(tag: string) {
    form.value.tags = form.value.tags.filter((t) => t !== tag)
}

const tagSuggestions = computed(() =>
    SUGGESTED_TAGS.map((key) => t(`country.suggestedTags.${key}`)).filter(
        (tag) => !form.value.tags.includes(tag)
    )
)

// --- Misc --------------------------------------------------------------

// Grow textareas with their content
const vAutogrow: Directive<HTMLTextAreaElement> = {
    mounted: fit,
    updated: fit,
}

function fit(el: HTMLTextAreaElement) {
    el.style.height = 'auto'
    el.style.height = `${el.scrollHeight + 2}px`
}

// Picked statuses take their map colour
const statusOptions = computed(() => [
    {
        value: 'visited' as const,
        label: t('country.visited'),
        color: 'var(--visited)',
    },
    {
        value: 'wishlist' as const,
        label: t('country.wishlist'),
        color: 'var(--wishlist)',
    },
    {
        value: 'none' as const,
        label: t('country.notSet'),
        color: 'var(--chip)',
    },
])

// Trips with a stop in this country: latest start first, undated ones last
const trips = computed(() =>
    store.trips
        .filter((trip) => trip.countries.includes(props.iso))
        .sort((a, b) => {
            if (!a.startsOn || !b.startsOn)
                return a.startsOn ? -1 : b.startsOn ? 1 : 0
            return b.startsOn.localeCompare(a.startsOn)
        })
)

// The other countries a trip visits, by name
function otherCountries(countries: string[]): string {
    return formatList(
        countries.filter((iso) => iso !== props.iso).map(countryNameOf)
    )
}

const visibilityChoices = computed(() => visibilityOptions(true))
</script>

<template>
    <div class="details-panel">
        <div class="header">
            <div>
                <div class="title-row">
                    <h2>
                        <span v-if="country" :class="flagClass(country)"></span>
                        {{ name }}
                    </h2>
                    <div
                        class="stars"
                        role="group"
                        :aria-label="t('country.rating')"
                    >
                        <button
                            v-for="n in 5"
                            :key="n"
                            type="button"
                            class="star"
                            :class="{ on: n <= form.rating }"
                            :aria-label="t('country.stars', { count: n })"
                            @click="setRating(n)"
                        >
                            ★
                        </button>
                    </div>
                </div>
                <p class="sub">{{ t('country.details') }}</p>
            </div>
            <button
                class="close"
                :aria-label="t('common.close')"
                @click="leave(() => emit('close'))"
            >
                ×
            </button>
        </div>

        <SegmentedControl
            v-model="status"
            :options="statusOptions"
            :label="t('country.status')"
            stretch
        />

        <SegmentedControl
            v-if="status !== 'none'"
            v-model="vis"
            :options="visibilityChoices"
            :label="t('country.visibility')"
            stretch
        />

        <details open>
            <summary>{{ t('country.overview') }}</summary>
            <dl v-if="factRows.length" class="facts">
                <template v-for="[label, value] in factRows" :key="label">
                    <dt>{{ label }}</dt>
                    <dd>{{ value }}</dd>
                </template>
            </dl>
        </details>

        <details open>
            <summary>{{ t('country.trips') }}</summary>
            <ul v-if="trips.length" class="trips">
                <li v-for="trip in trips" :key="trip.id">
                    <RouterLink :to="{ name: 'trip', params: { id: trip.id } }">
                        <span class="trip-title">{{ trip.title }}</span>
                        <span class="mini">
                            <template v-if="tripDates(trip)"
                                >{{ tripDates(trip) }} ·
                            </template>
                            {{ t('trip.stopCount', { count: trip.stopCount }) }}
                            <template v-if="trip.countries.length > 1">
                                ·
                                {{
                                    t('country.alsoIn', {
                                        countries: otherCountries(
                                            trip.countries
                                        ),
                                    })
                                }}
                            </template>
                        </span>
                    </RouterLink>
                </li>
            </ul>
            <p v-else class="empty">{{ t('country.noTrips') }}</p>
            <RouterLink
                v-if="store.canWrite"
                class="link-btn"
                :to="{ name: 'trip-new', query: { country: iso } }"
            >
                {{ t('country.startTrip') }}
            </RouterLink>
            <span
                v-else
                class="link-btn disabled"
                :title="t('common.offlineHint')"
                aria-disabled="true"
            >
                {{ t('country.startTrip') }}
            </span>
        </details>

        <details open>
            <summary>{{ t('country.notes') }}</summary>
            <p v-if="unavailable" class="empty">
                {{ t('country.notesOffline') }}
            </p>
            <label for="food">{{ t('country.food') }}</label>
            <textarea
                id="food"
                v-autogrow
                v-model="form.food"
                rows="3"
                :placeholder="t('country.foodPlaceholder')"
            />
            <label for="sights">{{ t('country.sights') }}</label>
            <textarea
                id="sights"
                v-autogrow
                v-model="form.sightseeing"
                rows="3"
                :placeholder="t('country.sightsPlaceholder')"
            />
            <label for="notes">{{ t('country.tripNotes') }}</label>
            <textarea
                id="notes"
                v-autogrow
                v-model="form.notes"
                rows="3"
                :placeholder="t('country.tripNotesPlaceholder')"
            />
        </details>

        <details>
            <summary>
                {{ t('country.tags') }}
                <span v-if="form.tags.length" class="count">
                    {{ form.tags.length }}
                </span>
            </summary>
            <div class="chips">
                <span v-for="tag in form.tags" :key="tag" class="chip on">
                    {{ tag }}
                    <button
                        type="button"
                        class="remove-tag"
                        :aria-label="t('country.removeTag', { tag })"
                        @click="removeTag(tag)"
                    >
                        ×
                    </button>
                </span>
            </div>
            <input
                v-model="tagInput"
                :placeholder="t('country.addTag')"
                @keydown.enter.prevent.stop="addTag(tagInput)"
            />
            <div class="chips">
                <button
                    v-for="tag in tagSuggestions"
                    :key="tag"
                    type="button"
                    class="chip"
                    @click="addTag(tag)"
                >
                    + {{ tag }}
                </button>
            </div>
        </details>

        <div class="actions">
            <p v-if="!store.canWrite" class="hint offline">
                {{ t('common.offlineHint') }}
            </p>
            <button class="btn secondary" @click="leave(() => emit('close'))">
                {{ t('common.cancel') }}
            </button>
            <button
                class="btn"
                :disabled="!dirty || loadingDetails || !store.canWrite"
                :title="
                    !store.canWrite
                        ? t('common.offlineHint')
                        : dirty
                          ? undefined
                          : t('country.noChanges')
                "
                @click="save"
            >
                {{ t('common.save') }}
            </button>
        </div>
    </div>

    <ConfirmModal
        v-if="pendingLeave"
        :title="t('country.discardTitle')"
        :message="t('country.discardMessage')"
        :confirm-label="t('country.discard')"
        danger
        @confirm="(pendingLeave!(), (pendingLeave = null))"
        @cancel="pendingLeave = null"
    />

    <ConfirmModal
        v-if="showRemoveConfirm"
        :title="t('country.removeTitle')"
        :message="t('country.removeMessage', { name })"
        :confirm-label="t('common.remove')"
        danger
        @confirm="persist"
        @cancel="showRemoveConfirm = false"
    />
</template>

<style scoped>
.details-panel {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
}

.details-panel > * {
    flex-shrink: 0;
}

.header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
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

.fi {
    margin-right: 0.6rem;
}

.close {
    flex: none;
    width: var(--touch);
    height: var(--touch);
    margin: -0.6rem -0.7rem 0 0;
    border: none;
    background: transparent;
    font-size: 1.4rem;
    line-height: 1;
    color: var(--muted);
    cursor: pointer;
}

/* Rating */
.title-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0 0.6rem;
}

.star {
    min-width: 32px;
    min-height: 40px;
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

/* Sections */
details {
    border-top: 1px solid var(--line);
    padding-top: 0.6rem;
}

summary {
    padding: 0.4rem 0;
    cursor: pointer;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-soft);
    margin-bottom: 0.4rem;
}

details > :not(summary) + :not(summary) {
    margin-top: 0.4rem;
}

.count {
    margin-left: 0.3rem;
    padding: 0 0.4rem;
    border-radius: 1rem;
    background: var(--chip);
    font-size: 0.7rem;
    font-weight: 500;
    color: var(--muted-strong);
}

.empty,
.mini,
.hint {
    font-size: 0.75rem;
    color: var(--muted);
}

.facts {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 0.2rem 0.8rem;
    font-size: 0.8rem;
}

.facts dt {
    color: var(--muted);
}

.facts dd {
    margin: 0;
    color: var(--text-soft);
}

/* Chips */
.chips {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
}

/* Picked tags are removed with their ×, not by a click on the chip */
.chip.on {
    cursor: default;
}

.remove-tag {
    padding: 0.2rem 0.3rem;
    margin-right: -0.3rem;
    border: none;
    background: transparent;
    color: var(--muted);
    font-size: 1rem;
    line-height: 1;
    cursor: pointer;
}

.remove-tag:hover {
    color: var(--danger);
}

.link-btn {
    display: inline-block;
    padding: 0.5rem 0;
    font-size: 0.78rem;
    color: var(--accent);
    text-decoration: none;
    cursor: pointer;
}

.link-btn.disabled {
    color: var(--muted);
    cursor: not-allowed;
}

.trips {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    margin: 0;
    padding: 0;
    list-style: none;
}

.trips a {
    display: flex;
    flex-direction: column;
    min-height: var(--touch);
    justify-content: center;
    padding: 0.25rem 0.4rem;
    border-radius: 0.4rem;
    color: var(--text);
    text-decoration: none;
}

.trips a:hover {
    background: var(--line-faint);
}

.trip-title {
    font-size: 0.85rem;
    overflow-wrap: anywhere;
}

/* Form */
label {
    display: block;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--muted);
    margin: 0.4rem 0 0.25rem;
}

textarea,
input:not([type='checkbox']) {
    width: 100%;
    box-sizing: border-box;
    padding: 0.5rem 0.6rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    font-family: inherit;
    font-size: var(--field-font);
    background: var(--raised);
    color: var(--text);
    outline: none;
}

textarea {
    resize: none;
    overflow: hidden;
}

textarea:focus,
input:focus {
    border-color: var(--accent);
}

input[type='checkbox'] {
    accent-color: var(--accent);
}

/* Stays at the bottom of the panel while the details scroll; reaches the
   panel's edges through the sheet's padding */
.actions {
    position: sticky;
    bottom: calc(-1 * var(--sheet-pad, 0px));
    z-index: 1;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.5rem;
    margin: 0 calc(-1 * var(--sheet-pad, 0px)) calc(-1 * var(--sheet-pad, 0px));
    padding: 0.75rem var(--sheet-pad, 0px) var(--sheet-pad, 0px);
    border-top: 1px solid var(--line);
    background: var(--panel);
    backdrop-filter: blur(8px);
}

.hint.offline {
    grid-column: 1 / -1;
    margin: 0;
    color: var(--warn);
}
</style>
