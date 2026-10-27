<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useTravelStore, type Flight, type FlightUpdate } from '../stores/store'
import { t } from '../../i18n'
import { vEnterNext } from '../ui/enterNext'
import { distanceKm, estimateDurationMinutes } from './distance'
import { formatDay, formatDistance, formatDuration } from '../../i18n/format'
import {
    flightradarUrl,
    formatFlightNumber,
    normalizeFlightNumber,
} from './flightNumber'
import { AIRCRAFT_TYPES } from './aircraft'

const OTHER = 'other'
const MAX_MINUTES = 24 * 60

const props = defineProps<{ flight: Flight }>()
const store = useTravelStore()

const km = computed(() => distanceKm(props.flight.from, props.flight.to))
const estimate = computed(() => estimateDurationMinutes(km.value))

const number = ref('')
const date = ref('')
const hours = ref<number | ''>('')
const minutes = ref<number | ''>('')
const aircraft = ref('')
const otherAircraft = ref('')
// The duration inputs start with the estimate, which is only stored once
// the user changes them
const initialDuration = ref(0)

function reset() {
    const f = props.flight
    number.value = f.flightNumber ? formatFlightNumber(f.flightNumber) : ''
    date.value = f.date ?? ''
    initialDuration.value = f.durationMinutes ?? estimate.value
    hours.value = Math.floor(initialDuration.value / 60)
    minutes.value = initialDuration.value % 60
    const known = !f.aircraft || AIRCRAFT_TYPES.includes(f.aircraft)
    aircraft.value = known ? (f.aircraft ?? '') : OTHER
    otherAircraft.value = known ? '' : (f.aircraft ?? '')
}

// Also after saving, so a newly saved number shows the other fields at once
watch(
    () => [
        props.flight.id,
        props.flight.flightNumber,
        props.flight.date,
        props.flight.durationMinutes,
        props.flight.aircraft,
    ],
    reset,
    { immediate: true }
)

const normalizedNumber = computed(() =>
    number.value.trim() ? normalizeFlightNumber(number.value) : ''
)
const durationMinutes = computed(() => {
    const total = Number(hours.value || 0) * 60 + Number(minutes.value || 0)
    return Number.isInteger(total) ? total : NaN
})
const aircraftValue = computed(() =>
    aircraft.value === OTHER ? otherAircraft.value.trim() : aircraft.value
)

const durationValid = computed(
    () =>
        durationMinutes.value >= 0 &&
        durationMinutes.value <= MAX_MINUTES &&
        Number(minutes.value || 0) < 60
)

const patch = computed<FlightUpdate>(() => {
    const f = props.flight
    const out: FlightUpdate = {}
    if (normalizedNumber.value !== (f.flightNumber ?? ''))
        out.flightNumber = normalizedNumber.value ?? ''
    // The other fields are only shown once a number is stored
    if (!f.flightNumber) return out
    if (date.value !== (f.date ?? '')) out.date = date.value
    if (durationMinutes.value !== initialDuration.value)
        out.durationMinutes = durationMinutes.value
    if (aircraftValue.value !== (f.aircraft ?? ''))
        out.aircraft = aircraftValue.value
    return out
})

const canSave = computed(
    () =>
        store.canWrite &&
        normalizedNumber.value !== null &&
        durationValid.value &&
        aircraftValue.value.length <= 60 &&
        Object.keys(patch.value).length > 0
)

const durationLabel = computed(() =>
    props.flight.durationMinutes
        ? formatDuration(props.flight.durationMinutes)
        : `≈ ${formatDuration(estimate.value)}`
)

const formatDate = (day: string) =>
    formatDay(day, { day: 'numeric', month: 'short', year: 'numeric' })

async function save() {
    if (!canSave.value) return
    // On failure the store restores the flight; show its values again
    if (!(await store.updateFlight(props.flight.id, patch.value))) reset()
}
</script>

<template>
    <div class="details">
        <dl>
            <div>
                <dt>{{ t('flights.distance') }}</dt>
                <dd>{{ formatDistance(km, store.settings.distanceUnit) }}</dd>
            </div>
            <div>
                <dt>{{ t('flights.duration') }}</dt>
                <dd>
                    {{ durationLabel }}
                    <span v-if="!flight.durationMinutes" class="muted">
                        {{ t('flights.estimate') }}
                    </span>
                </dd>
            </div>
            <div v-if="flight.date">
                <dt>{{ t('flights.date') }}</dt>
                <dd>{{ formatDate(flight.date) }}</dd>
            </div>
            <div v-if="flight.aircraft">
                <dt>{{ t('flights.aircraft') }}</dt>
                <dd>{{ flight.aircraft }}</dd>
            </div>
        </dl>

        <a
            v-if="flight.flightNumber"
            class="link"
            :href="flightradarUrl(flight.flightNumber)"
            target="_blank"
            rel="noopener noreferrer"
        >
            {{
                t('flights.onFlightradar', {
                    number: formatFlightNumber(flight.flightNumber),
                })
            }}
        </a>

        <form v-enter-next class="fields" @submit.prevent="save">
            <div>
                <label :for="`number-${flight.id}`">{{
                    t('flights.number')
                }}</label>
                <input
                    :id="`number-${flight.id}`"
                    v-model="number"
                    :placeholder="t('flights.numberPlaceholder')"
                    autocapitalize="characters"
                    autocomplete="off"
                    spellcheck="false"
                    enterkeyhint="done"
                    maxlength="16"
                    :disabled="!store.canWrite"
                />
                <p v-if="normalizedNumber === null" class="hint">
                    {{ t('flights.numberRule') }}
                </p>
                <p v-else-if="!flight.flightNumber" class="muted small">
                    {{ t('flights.numberFirst') }}
                </p>
            </div>

            <template v-if="flight.flightNumber">
                <div>
                    <label :for="`date-${flight.id}`">{{
                        t('flights.date')
                    }}</label>
                    <input
                        :id="`date-${flight.id}`"
                        v-model="date"
                        type="date"
                        min="1900-01-01"
                        max="2100-12-31"
                        :disabled="!store.canWrite"
                    />
                </div>

                <div>
                    <label :for="`hours-${flight.id}`">{{
                        t('flights.duration')
                    }}</label>
                    <div class="duration">
                        <input
                            :id="`hours-${flight.id}`"
                            v-model.number="hours"
                            type="number"
                            inputmode="numeric"
                            min="0"
                            max="24"
                            :aria-label="t('flights.hours')"
                            :disabled="!store.canWrite"
                        />
                        <span>{{ t('flights.hoursUnit') }}</span>
                        <input
                            v-model.number="minutes"
                            type="number"
                            inputmode="numeric"
                            min="0"
                            max="59"
                            step="5"
                            :aria-label="t('flights.minutes')"
                            :disabled="!store.canWrite"
                        />
                        <span>{{ t('flights.minutesUnit') }}</span>
                    </div>
                    <p v-if="!durationValid" class="hint">
                        {{ t('flights.durationRule') }}
                    </p>
                </div>

                <div>
                    <label :for="`aircraft-${flight.id}`">{{
                        t('flights.aircraft')
                    }}</label>
                    <select
                        :id="`aircraft-${flight.id}`"
                        v-model="aircraft"
                        :disabled="!store.canWrite"
                    >
                        <option value="">{{ t('flights.unknown') }}</option>
                        <option v-for="t in AIRCRAFT_TYPES" :key="t" :value="t">
                            {{ t }}
                        </option>
                        <option :value="OTHER">
                            {{ t('flights.otherType') }}
                        </option>
                    </select>
                    <input
                        v-if="aircraft === OTHER"
                        v-model="otherAircraft"
                        class="other"
                        :placeholder="t('flights.aircraftType')"
                        maxlength="60"
                        :aria-label="t('flights.otherAircraft')"
                        :disabled="!store.canWrite"
                    />
                </div>
            </template>

            <button
                class="btn"
                type="submit"
                :disabled="!canSave"
                :title="store.canWrite ? undefined : t('common.offlineHint')"
            >
                {{ t('common.save') }}
            </button>
            <p v-if="!store.canWrite" class="hint offline">
                {{ t('common.offlineHint') }}
            </p>
        </form>
    </div>
</template>

<style scoped>
.details {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    padding: 0.2rem 0.5rem 0.75rem;
}

dl {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(7.5rem, 1fr));
    gap: 0.4rem 0.75rem;
    margin: 0;
}

dt {
    font-size: 0.7rem;
    color: var(--muted);
}

dd {
    margin: 0;
    font-size: 0.85rem;
}

.muted {
    color: var(--muted);
    font-size: 0.72rem;
}

.small {
    margin: 0.3rem 0 0;
}

.link {
    display: flex;
    align-items: center;
    min-height: var(--touch);
    color: #3a7d5c;
    font-size: 0.85rem;
    font-weight: 500;
    text-decoration: none;
}

.link:hover {
    text-decoration: underline;
}

.fields {
    display: flex;
    flex-direction: column;
    gap: 0.7rem;
}

label {
    display: block;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--muted);
    margin-bottom: 0.25rem;
}

input,
select {
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
}

input:focus,
select:focus {
    border-color: #3a7d5c;
    outline: none;
}

input:disabled,
select:disabled {
    opacity: 0.6;
}

.other {
    margin-top: 0.4rem;
}

.duration {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.8rem;
    color: var(--muted);
}

.duration input {
    width: 4.5rem;
}

.hint {
    margin: 0.3rem 0 0;
    font-size: 0.72rem;
    color: #c0392b;
}

.hint.offline {
    margin: -0.4rem 0 0;
    color: #8a5a12;
}

.btn {
    box-sizing: border-box;
    width: 100%;
    min-height: var(--touch);
    padding: 0.5rem;
    border: none;
    border-radius: 0.5rem;
    background: #1a1c1e;
    color: white;
    font: inherit;
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
}

.btn:disabled {
    opacity: 0.5;
}

/* Mouse: denser controls, as in the rest of the panel */
@media (pointer: fine) {
    input,
    select,
    .btn,
    .link {
        min-height: 2rem;
    }

    .btn {
        font-size: 0.82rem;
    }

    .link {
        min-height: 1.6rem;
    }
}
</style>
