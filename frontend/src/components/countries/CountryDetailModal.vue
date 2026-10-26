<script setup lang="ts">
import {
    ref,
    computed,
    watch,
    onMounted,
    onBeforeUnmount,
    type Directive,
} from 'vue'
import {
    useTravelStore,
    emptyDetails,
    type CountryStatus,
    type Visibility,
} from '../stores/store'
import { getCountryByIso } from './countries'
import { getCountryFacts } from './countryFacts'
import { RouterLink, useRoute } from 'vue-router'
import ConfirmModal from '../ui/ConfirmModal.vue'
import { OFFLINE_HINT } from '../ui/offline'

const props = defineProps<{ iso: string }>()
const emit = defineEmits<{ close: [] }>()

const store = useTravelStore()
const route = useRoute()

const country = computed(() => getCountryByIso(props.iso))
const facts = computed(() => getCountryFacts(props.iso))

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

const SUGGESTED_TAGS = ['beach', 'city trip', 'hiking', 'culture', 'nightlife']

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
    store.notify(`${country.value?.name ?? props.iso} saved`)
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

onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))

// --- Facts -------------------------------------------------------------

const formatNumber = (n: number) => n.toLocaleString('en')

const factRows = computed(() => {
    const f = facts.value
    if (!f) return []
    return [
        ['Capital', f.capital],
        ['Region', f.region],
        ['Population', f.population ? formatNumber(f.population) : ''],
        ['Area', f.area ? `${formatNumber(f.area)} km²` : ''],
        ['Currency', f.currencies],
        ['Languages', f.languages],
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
    SUGGESTED_TAGS.filter((t) => !form.value.tags.includes(t))
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

const statusOptions: { value: CountryStatus; label: string }[] = [
    { value: 'visited', label: 'Visited' },
    { value: 'wishlist', label: 'Want to go' },
    { value: 'none', label: 'Not set' },
]

const visibilityOptions: { value: Visibility; label: string }[] = [
    { value: 'private', label: '🔒 Only me' },
    { value: 'public', label: '🌐 Public' },
]
</script>

<template>
    <div class="details-panel">
        <div class="header">
            <div>
                <div class="title-row">
                    <h2>
                        <span
                            v-if="country"
                            :class="`fi fi-${country.alpha2.toLowerCase()}`"
                        ></span>
                        {{ country?.name ?? iso }}
                    </h2>
                    <div class="stars" role="group" aria-label="Rating">
                        <button
                            v-for="n in 5"
                            :key="n"
                            type="button"
                            class="star"
                            :class="{ on: n <= form.rating }"
                            :aria-label="`${n} star${n > 1 ? 's' : ''}`"
                            @click="setRating(n)"
                        >
                            ★
                        </button>
                    </div>
                </div>
                <p class="sub">Country details</p>
            </div>
            <button
                class="close"
                aria-label="Close"
                @click="leave(() => emit('close'))"
            >
                ×
            </button>
        </div>

        <div class="segmented" role="group" aria-label="Status">
            <button
                v-for="o in statusOptions"
                :key="o.value"
                type="button"
                :class="{ on: status === o.value, [o.value]: true }"
                @click="status = o.value"
            >
                {{ o.label }}
            </button>
        </div>

        <div
            v-if="status !== 'none'"
            class="segmented visibility"
            role="group"
            aria-label="Visibility"
        >
            <button
                v-for="o in visibilityOptions"
                :key="o.value"
                type="button"
                :class="{ on: vis === o.value }"
                :title="
                    o.value === 'public'
                        ? 'Shown on your public profile, if it is public'
                        : 'Only visible to you'
                "
                @click="vis = o.value"
            >
                {{ o.label }}
            </button>
        </div>

        <details open>
            <summary>Overview</summary>
            <dl v-if="factRows.length" class="facts">
                <template v-for="[label, value] in factRows" :key="label">
                    <dt>{{ label }}</dt>
                    <dd>{{ value }}</dd>
                </template>
            </dl>
        </details>

        <details open>
            <summary>Trips</summary>
            <p class="empty">Trips are planned on their own page.</p>
            <RouterLink
                class="link-btn"
                :to="{ name: 'trip-new', params: { iso } }"
            >
                + Add trip
            </RouterLink>
        </details>

        <details open>
            <summary>Notes</summary>
            <p v-if="unavailable" class="empty">
                Not available offline. Open this country once while online to
                keep its notes on this device.
            </p>
            <label for="food">Food place recommendations</label>
            <textarea
                id="food"
                v-autogrow
                v-model="form.food"
                rows="3"
                placeholder="Add your first food tip — restaurants, street food, cafés…"
            />
            <label for="sights">Sightseeing</label>
            <textarea
                id="sights"
                v-autogrow
                v-model="form.sightseeing"
                rows="3"
                placeholder="Add your first sight — places worth seeing…"
            />
            <label for="notes">Trip notes &amp; highlights</label>
            <textarea
                id="notes"
                v-autogrow
                v-model="form.notes"
                rows="3"
                placeholder="The story of your trip, tips for next time…"
            />
        </details>

        <details>
            <summary>
                Tags
                <span v-if="form.tags.length" class="count">
                    {{ form.tags.length }}
                </span>
            </summary>
            <div class="chips">
                <span v-for="t in form.tags" :key="t" class="chip on">
                    {{ t }}
                    <button
                        type="button"
                        class="icon"
                        :aria-label="`Remove ${t}`"
                        @click="removeTag(t)"
                    >
                        ×
                    </button>
                </span>
            </div>
            <input
                v-model="tagInput"
                placeholder="Add tag and press Enter"
                @keydown.enter.prevent.stop="addTag(tagInput)"
            />
            <div class="chips">
                <button
                    v-for="t in tagSuggestions"
                    :key="t"
                    type="button"
                    class="chip"
                    @click="addTag(t)"
                >
                    + {{ t }}
                </button>
            </div>
        </details>

        <div class="actions">
            <span v-if="!store.canWrite" class="hint offline">
                {{ OFFLINE_HINT }}
            </span>
            <span v-else class="hint keys">Esc closes · Ctrl+Enter saves</span>
            <button class="btn cancel" @click="leave(() => emit('close'))">
                Cancel
            </button>
            <button
                class="btn save"
                :disabled="!dirty || loadingDetails || !store.canWrite"
                :title="
                    !store.canWrite
                        ? OFFLINE_HINT
                        : dirty
                          ? undefined
                          : 'No changes to save'
                "
                @click="save"
            >
                Save
            </button>
        </div>
    </div>

    <ConfirmModal
        v-if="pendingLeave"
        title="Discard changes"
        message="You have unsaved changes. Discard them?"
        confirm-label="Discard"
        @confirm="(pendingLeave!(), (pendingLeave = null))"
        @cancel="pendingLeave = null"
    />

    <ConfirmModal
        v-if="showRemoveConfirm"
        title="Remove country"
        :message="`Remove ${country?.name ?? iso} from your visited countries?`"
        confirm-label="Remove"
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
    font-family: 'DM Serif Display', serif;
    font-size: 1rem;
    font-weight: 400;
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

/* Status */
.segmented {
    display: flex;
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    overflow: hidden;
}

.segmented button {
    flex: 1;
    min-height: 40px;
    padding: 0.35rem 0;
    border: none;
    background: var(--raised);
    font-size: 0.78rem;
    color: var(--muted);
    cursor: pointer;
}

.segmented button + button {
    border-left: 1px solid var(--line-strong);
}

.segmented button.on.visited {
    background: var(--visited);
    color: var(--on-status);
}

.segmented button.on.wishlist {
    background: var(--wishlist);
    color: var(--on-status);
}

.segmented button.on.none {
    background: var(--chip);
    color: var(--text-soft);
}

.segmented.visibility button {
    min-height: 36px;
    padding: 0.25rem 0;
    font-size: 0.72rem;
}

.segmented.visibility button.on {
    background: var(--accent-soft);
    color: var(--text);
    font-weight: 500;
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
    color: #ca8f22;
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

.chip {
    display: inline-flex;
    align-items: center;
    gap: 0.2rem;
    min-height: 2rem;
    padding: 0.15rem 0.65rem;
    border: 1px solid var(--line-strong);
    border-radius: 1rem;
    background: var(--raised);
    font-size: 0.75rem;
    color: var(--muted-strong);
    cursor: pointer;
}

.chip.on {
    background: var(--accent-soft);
    border-color: #3a7d5c;
    color: #2f6a4e;
    cursor: default;
}

.chip .fi {
    margin-right: 0.2rem;
}

.icon {
    padding: 0.2rem 0.3rem;
    margin-right: -0.3rem;
    border: none;
    background: transparent;
    color: var(--muted);
    font-size: 1rem;
    line-height: 1;
    cursor: pointer;
}

.link-btn {
    display: inline-block;
    padding: 0.5rem 0;
    font-size: 0.78rem;
    color: #3a7d5c;
    text-decoration: none;
    cursor: pointer;
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
    padding: 0.4rem 0.6rem;
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
    border-color: #3a7d5c;
}

input[type='checkbox'] {
    accent-color: #3a7d5c;
}

.actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    align-items: center;
    gap: 0.5rem;
}

/* Keyboard shortcuts mean nothing on a touch screen */
@media (pointer: coarse) {
    .hint.keys {
        display: none;
    }
}

.hint.offline {
    color: #8a5a12;
}

.hint {
    margin-right: auto;
}

.btn {
    min-height: 40px;
    padding: 0.4rem 1.1rem;
    border-radius: 0.5rem;
    font-size: 0.8rem;
    font-weight: 500;
    cursor: pointer;
    border: 1px solid transparent;
}

.btn.cancel {
    background: transparent;
    border-color: var(--line-strong);
    color: var(--muted);
}

.btn.save {
    background: #3a7d5c;
    border-color: #3a7d5c;
    color: #fff;
}

.btn.save:hover:not(:disabled) {
    background: #2f6a4e;
}

.btn.save:disabled {
    opacity: 0.45;
    cursor: not-allowed;
}
</style>
