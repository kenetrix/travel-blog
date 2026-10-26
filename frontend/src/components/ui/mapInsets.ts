import type { MapInsets } from '../worldmap/MapProvider'

// Room the open sheet takes from the map, so the map can centre countries
// in the part that is still visible. Only one sheet is open at a time.
type Source = () => Partial<MapInsets>

let source: Source | null = null

export function registerSheet(fn: Source): () => void {
    source = fn
    return () => {
        if (source === fn) source = null
    }
}

export function sheetInsets(): Partial<MapInsets> {
    return source?.() ?? {}
}
