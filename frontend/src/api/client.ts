import createClient from 'openapi-fetch'
import type { paths, components } from '../../generated/schema.gen'

// Same origin: Vite proxies /api to the backend (see vite.config.ts)
export const api = createClient<paths>({ baseUrl: '/api' })

export type Schemas = components['schemas']

// Readable message for a failed request: the problem detail if the server
// sent one, otherwise the fallback (also for network errors)
export function problemMessage(error: unknown, fallback: string): string {
    const problem = error as Partial<Schemas['Problem']> | undefined
    return problem?.detail ?? fallback
}
