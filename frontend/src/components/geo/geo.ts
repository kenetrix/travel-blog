// Plain geometry without imports, so `node --test` runs its tests directly

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

// Sum of the distances between consecutive points
export function pathKm(points: [number, number][]): number {
    let km = 0
    for (let i = 1; i < points.length; i++) {
        km += distanceKm(points[i - 1], points[i])
    }
    return km
}

export interface Place {
    name: string
    lng: number
    lat: number
    population: number
}

// The place closest to [lng, lat] within maxKm, or null
export function nearestPlace<P extends Place>(
    places: Iterable<P>,
    point: [number, number],
    maxKm = 15
): P | null {
    let best: P | null = null
    let bestKm = maxKm
    for (const p of places) {
        // Cheap reject first: one degree of latitude is about 111 km
        if (Math.abs(p.lat - point[1]) * 111 > bestKm) continue
        const km = distanceKm([p.lng, p.lat], point)
        if (km <= bestKm) {
            best = p
            bestKm = km
        }
    }
    return best
}
