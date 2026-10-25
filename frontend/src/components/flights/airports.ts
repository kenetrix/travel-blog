import data from './airports.json'

export interface Airport {
    iata: string
    name: string
    city: string
    country: string
    countryCode: string // ISO 3166-1 alpha-2
    coords: [number, number] // [lng, lat]
}

export const airports: Airport[] = (
    data as [string, string, string, string, number, number, string][]
).map(([iata, name, city, country, lat, lng, countryCode]) => ({
    iata,
    name,
    city,
    country,
    countryCode,
    coords: [lng, lat],
}))

const byIata = new Map(airports.map((a) => [a.iata, a]))

export function getAirport(iata: string): Airport | undefined {
    return byIata.get(iata)
}

// Lowercase and strip diacritics so "sao paulo" finds "São Paulo"
const normalize = (s: string) =>
    s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()

const byCode = (a: Airport, b: Airport) => a.iata.localeCompare(b.iata)

const sorted = [...airports].sort(byCode)

// The best matches (by relevance): IATA code matches first, then the rest,
// each group alphabetically by IATA code
export function searchAirports(query: string, limit = 20): Airport[] {
    const q = normalize(query.trim())
    if (!q) return sorted.slice(0, limit)

    const rank = (a: Airport) => {
        if (a.iata.toLowerCase() === q) return 0
        if (a.iata.toLowerCase().startsWith(q)) return 1
        if (normalize(a.city).startsWith(q)) return 2
        return 3
    }

    return airports
        .filter((a) =>
            normalize(`${a.iata} ${a.city} ${a.name} ${a.country}`).includes(q)
        )
        .sort((a, b) => rank(a) - rank(b))
        .slice(0, limit)
        .sort(
            (a, b) => Number(rank(a) > 1) - Number(rank(b) > 1) || byCode(a, b)
        )
}
