import data from './airports.json'
import { localizedData } from '../../i18n'
import { regionName } from '../../i18n/format'

// Names are English; airportName() and friends give the UI language
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

export function airportName(a: Airport): string {
    return localizedData().airports[a.iata]?.[0] || a.name
}

export function airportCity(a: Airport): string {
    return localizedData().airports[a.iata]?.[1] || a.city
}

export function airportCountry(a: Airport): string {
    return regionName(a.countryCode, a.country)
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
        if (
            normalize(a.city).startsWith(q) ||
            normalize(airportCity(a)).startsWith(q)
        )
            return 2
        return 3
    }

    // English and UI language, so "Munich" and "München" both work
    const text = (a: Airport) =>
        normalize(
            [
                a.iata,
                a.city,
                a.name,
                a.country,
                airportCity(a),
                airportName(a),
                airportCountry(a),
            ].join(' ')
        )

    return airports
        .filter((a) => text(a).includes(q))
        .sort((a, b) => rank(a) - rank(b))
        .slice(0, limit)
        .sort(
            (a, b) => Number(rank(a) > 1) - Number(rank(b) > 1) || byCode(a, b)
        )
}
