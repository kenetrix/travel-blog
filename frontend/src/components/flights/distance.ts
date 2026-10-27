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

// Typical average jet speed over a whole route, plus taxi, climb and approach
const AVERAGE_KMH = 800
const GROUND_MINUTES = 30

export function estimateDurationMinutes(km: number): number {
    return Math.round(((km / AVERAGE_KMH) * 60 + GROUND_MINUTES) / 5) * 5
}
