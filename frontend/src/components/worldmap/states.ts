// public/states/<ISO3>.json, written by scripts/states.ts
export type BorderLine = [lng: number, lat: number][]

export interface StateBorders {
    // Between first-level divisions (Bundesländer, US states, ...)
    main: BorderLine[]
    // Between their subdivisions, where the data has them (départements, ...)
    minor: BorderLine[]
}

const NONE: StateBorders = { main: [], minor: [] }
const loaded = new Map<string, StateBorders>()
const pending = new Map<string, Promise<StateBorders>>()

// The borders of a country, fetched once. A failed fetch (offline, or no file
// for the country) yields none and is tried again next time.
function loadBorders(iso: string): Promise<StateBorders> {
    const done = loaded.get(iso)
    if (done) return Promise.resolve(done)
    let request = pending.get(iso)
    if (!request) {
        request = fetch(`/states/${iso}.json`)
            .then((r) => (r.ok ? (r.json() as Promise<StateBorders>) : null))
            .then((borders) => {
                if (borders) loaded.set(iso, borders)
                return borders ?? NONE
            })
            .catch(() => NONE)
            .finally(() => pending.delete(iso))
        pending.set(iso, request)
    }
    return request
}

// By country, so that the map can colour each country's lines to its fill
export async function stateBordersOf(
    isos: string[]
): Promise<Record<string, StateBorders>> {
    const all = await Promise.all(isos.map(loadBorders))
    return Object.fromEntries(isos.map((iso, i) => [iso, all[i]]))
}
