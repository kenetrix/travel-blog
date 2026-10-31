import type { TripInput } from '../stores/store'

export type StopCategory = NonNullable<TripInput['stops'][number]['category']>

// The API's '' (no category) is never offered as a choice
export const STOP_CATEGORIES: {
    value: Exclude<StopCategory, ''>
    icon: string
}[] = [
    { value: 'sight', icon: '🏛️' },
    { value: 'food', icon: '🍽️' },
    { value: 'stay', icon: '🛏️' },
    { value: 'activity', icon: '🎯' },
    { value: 'nature', icon: '🌿' },
    { value: 'transport', icon: '🚆' },
    { value: 'other', icon: '📍' },
]

export function categoryIcon(category: string): string {
    return STOP_CATEGORIES.find((c) => c.value === category)?.icon ?? ''
}
