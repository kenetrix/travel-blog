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
                <RouterLink
                    v-if="store.canWrite"
                    class="btn"
                    :to="{ name: 'trip-edit', params: { id: trip.id } }"
                >
                    {{ t('trip.edit') }}
                </RouterLink>
                <button
                    v-else
                    class="btn"
                    disabled
                    :title="t('common.offlineHint')"
                >
                    {{ t('trip.edit') }}
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
            <section class="panel">
                <dl class="facts">
                    <template v-if="trip.startsOn || trip.endsOn">
                        <dt>{{ t('trip.dates') }}</dt>
                        <dd>{{ tripDates(trip) }}</dd>
                    </template>
                    <template v-if="days">
                        <dt>{{ t('trip.length') }}</dt>
                        <dd>{{ t('trip.days', { count: days }) }}</dd>
                    </template>
                    <template v-if="trip.countries.length">
                        <dt>{{ t('trip.countries') }}</dt>
                        <dd class="countries">
                            <span v-for="iso in trip.countries" :key="iso">
                                <span :class="flagOf(iso)" />
                                {{ nameOf(iso) }}
                            </span>
                        </dd>
                    </template>
                    <dt>{{ t('trip.stops') }}</dt>
                    <dd>
                        {{ t('trip.stopCount', { count: trip.stops.length }) }}
                    </dd>
                    <template v-if="km !== null">
                        <dt>{{ t('trip.distance') }}</dt>
                        <dd>
                            {{
                                formatDistance(km, store.settings.distanceUnit)
                            }}
                            <span class="muted">{{
                                t('trip.distanceNote')
                            }}</span>
                        </dd>
                    </template>
                    <dt>{{ t('trip.visibility') }}</dt>
                    <dd>
                        {{
                            trip.visibility === 'public'
                                ? t('common.public')
                                : t('common.onlyMe')
                        }}
                    </dd>
                </dl>
            </section>

            <section v-if="trip.stops.length" class="panel">
                <h2>{{ t('trip.stops') }}</h2>
                <ol class="stops">
                    <li v-for="(stop, i) in trip.stops" :key="stop.id">
                        <span class="number">{{ i + 1 }}</span>
                        <div class="stop">
                            <p class="name">{{ stop.name }}</p>
                            <p class="meta">
                                <span :class="flagOf(stop.country)" />
                                {{ nameOf(stop.country) }}
                                <template v-if="stop.arrival">
                                    · {{ formatDay(stop.arrival) }}
                                </template>
                            </p>
                            <p v-if="stop.notes" class="text">
                                {{ stop.notes }}
                            </p>
                        </div>
                    </li>
                </ol>
            </section>

            <section v-if="trip.notes" class="panel">
                <h2>{{ t('trip.notes') }}</h2>
                <p class="text">{{ trip.notes }}</p>
            </section>
        </template>
    </PageWrapper>

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

.facts {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 0.45rem 1rem;
    margin: 0;
    font-size: 0.85rem;
}

.facts dt {
    color: var(--muted);
}

.facts dd {
    margin: 0;
}

.countries {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem 0.8rem;
}

.muted {
    color: var(--muted);
    font-size: 0.75rem;
}

.stops {
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
    margin: 0;
    padding: 0;
    list-style: none;
}

.stops li {
    display: flex;
    gap: 0.7rem;
}

.number {
    display: grid;
    flex: none;
    place-items: center;
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
