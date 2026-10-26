import createClient from 'openapi-fetch'
import type { paths, components } from '../../generated/schema.gen'

const TIMEOUT_MS = 8000

// Gateway errors: a proxy is up, but the server behind it is not
const UNREACHABLE = new Set([502, 503, 504])

// The server could not be reached (no network, timeout or gateway error)
export class NetworkError extends Error {
    constructor() {
        super('This needs a connection')
    }
}

let reportReachable: (reachable: boolean) => void = () => {}

// Called with false whenever a request cannot reach the server, and with
// true whenever one does
export function onReachability(fn: (reachable: boolean) => void) {
    reportReachable = fn
}

async function fetchWithTimeout(request: Request): Promise<Response> {
    let response: Response
    try {
        response = await fetch(
            new Request(request, { signal: AbortSignal.timeout(TIMEOUT_MS) })
        )
    } catch {
        reportReachable(false)
        throw new NetworkError()
    }
    if (UNREACHABLE.has(response.status)) {
        reportReachable(false)
        throw new NetworkError()
    }
    reportReachable(true)
    return response
}

// Same origin: Vite proxies /api to the backend (see vite.config.ts)
export const api = createClient<paths>({
    baseUrl: '/api',
    fetch: fetchWithTimeout,
})

export type Schemas = components['schemas']

// Readable message for a failed request: the problem detail if the server
// sent one, otherwise the fallback
export function problemMessage(error: unknown, fallback: string): string {
    if (error instanceof NetworkError) return error.message
    const problem = error as Partial<Schemas['Problem']> | undefined
    return problem?.detail ?? fallback
}
