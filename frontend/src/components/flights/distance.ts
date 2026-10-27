// Typical average jet speed over a whole route, plus taxi, climb and approach
const AVERAGE_KMH = 800
const GROUND_MINUTES = 30

export function estimateDurationMinutes(km: number): number {
    return Math.round(((km / AVERAGE_KMH) * 60 + GROUND_MINUTES) / 5) * 5
}
