const EARTH_RADIUS_KM = 6371

// Great-circle distance between two [lng, lat] points
export function distanceKm(a: [number, number], b: [number, number]): number {
    const rad = (deg: number) => (deg * Math.PI) / 180
    const dLat = rad(b[1] - a[1])
    const dLng = rad(b[0] - a[0])
    const h =
        Math.sin(dLat / 2) ** 2 +
        Math.cos(rad(a[1])) * Math.cos(rad(b[1])) * Math.sin(dLng / 2) ** 2
    return 2 * EARTH_RADIUS_KM * Math.asin(Math.sqrt(h))
}

const KM_PER_MILE = 1.609344

// Distance in the unit the user picked in the settings
export function formatDistance(km: number, unit: 'km' | 'mi' = 'km'): string {
    const value = unit === 'mi' ? km / KM_PER_MILE : km
    return `${Math.round(value).toLocaleString('en-US')} ${unit}`
}
