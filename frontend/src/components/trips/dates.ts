import { formatDay } from '../../i18n/format'
import { t } from '../../i18n'

type Dated = { startsOn?: string; endsOn?: string }

// "1 May 2024 – 14 May 2024", "From 1 May 2024", or '' without dates
export function tripDates(trip: Dated): string {
    const { startsOn, endsOn } = trip
    if (startsOn && endsOn) {
        return t('trip.dateRange', {
            from: formatDay(startsOn),
            to: formatDay(endsOn),
        })
    }
    if (startsOn) return t('trip.dateFrom', { date: formatDay(startsOn) })
    if (endsOn) return t('trip.dateUntil', { date: formatDay(endsOn) })
    return ''
}

// Days including the first and the last; null unless both are known
export function tripDays(trip: Dated): number | null {
    if (!trip.startsOn || !trip.endsOn) return null
    const ms =
        Date.parse(`${trip.endsOn}T00:00:00Z`) -
        Date.parse(`${trip.startsOn}T00:00:00Z`)
    return Math.round(ms / 86_400_000) + 1
}
