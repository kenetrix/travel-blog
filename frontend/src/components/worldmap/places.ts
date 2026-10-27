import { nearestPlace, type Place } from '../geo/geo'

// Place names show from this zoom on; below it whole countries are in view
export const PLACES_MIN_ZOOM = 5

// public/places/<ISO3>.json, written by scripts/places.ts
type Row = [name: string, lat: number, lng: number, population: number]

const loaded = new Map<string, Place[]>()
const pending = new Map<string, Promise<Place[]>>()

// The places of a country, fetched once. A failed fetch (offline, or no file
// for the country) yields none and is tried again next time.
export function loadPlaces(iso: string): Promise<Place[]> {
    const done = loaded.get(iso)
    if (done) return Promise.resolve(done)
    let request = pending.get(iso)
    if (!request) {
        request = fetch(`/places/${iso}.json`)
            .then((r) => (r.ok ? (r.json() as Promise<Row[]>) : null))
            .then((rows) => {
                const places = (rows ?? []).map(
                    ([name, lat, lng, population]) => ({
                        name,
                        lat,
                        lng,
                        population,
                    })
                )
                if (rows) loaded.set(iso, places)
                return places
            })
            .catch(() => [])
            .finally(() => pending.delete(iso))
        pending.set(iso, request)
    }
    return request
}

export async function placesOf(isos: string[]): Promise<Place[]> {
    return (await Promise.all(isos.map(loadPlaces))).flat()
}

// Name of the nearest place within 15 km. Besides the pin's own country, the
// places of every country loaded so far count, so a pin near a border finds
// the town across it if names were shown there.
export async function suggestName(
    point: [number, number],
    iso: string
): Promise<string | null> {
    await loadPlaces(iso)
    return nearestPlace([...loaded.values()].flat(), point)?.name ?? null
}
