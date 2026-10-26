import type { Schemas } from '../../api/client'

// The last known account and travel data, so a logged-in user keeps seeing
// them offline. Only ever one user's data; removed on logout and on 401.
export interface Snapshot {
    username: string
    profile: Schemas['Profile']
    role: Schemas['Role']
    mustChangePassword: boolean
    settings: Schemas['Settings']
    visited: string[]
    wishlist: string[]
    details: Record<string, Schemas['CountryDetails']>
    // Countries whose long texts are in `details`
    detailsLoaded: string[]
    visibility: Record<string, Schemas['Visibility']>
    flights: Schemas['Flight'][]
}

const KEY = 'travel-blog:offline:v1'

// Storage can be missing or full (private mode, quota); offline start is a
// convenience, so failures are ignored
export function readSnapshot(): Snapshot | null {
    try {
        const raw = localStorage.getItem(KEY)
        return raw ? (JSON.parse(raw) as Snapshot) : null
    } catch {
        return null
    }
}

export function writeSnapshot(snapshot: Snapshot) {
    try {
        localStorage.setItem(KEY, JSON.stringify(snapshot))
    } catch {
        // see readSnapshot
    }
}

export function removeSnapshot() {
    try {
        localStorage.removeItem(KEY)
    } catch {
        // see readSnapshot
    }
}
