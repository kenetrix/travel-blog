import countries from './countries.json'
import { collator, regionName } from '../../i18n/format'

export interface Country {
    // English; countryName() gives the name in the UI language
    name: string
    nameLong?: string
    alpha3: string
    alpha2: string
    countryCode?: number
}

export function getCountryByIso(isoName: string) {
    return Countries.find((c) => isoName === c.alpha3)
}

export function getCountryByAlpha2(alpha2: string) {
    return Countries.find((c) => alpha2 === c.alpha2)
}

export const Countries: Country[] = countries.map((c) => {
    return {
        name: c.name,
        nameLong: c.alternativeName,
        alpha3: c['alpha-3'],
        alpha2: c['alpha-2'],
        countryCode: Number(c['country-code']),
    }
})

export function countryName(c: Country): string {
    return regionName(c.alpha2, c.name)
}

// Name of the country with this alpha-3 code; the code itself if unknown
export function countryNameOf(iso: string): string {
    const country = getCountryByIso(iso)
    return country ? countryName(country) : iso
}

// CSS classes of the flag-icons flag
export function flagClass(c: Country): string {
    return `fi fi-${c.alpha2.toLowerCase()}`
}

// Flag of the country with this alpha-3 code; no classes if unknown
export function flagClassOf(iso: string): string {
    const country = getCountryByIso(iso)
    return country ? flagClass(country) : ''
}

export function byCountryName(a: Country, b: Country): number {
    return collator().compare(countryName(a), countryName(b))
}

// Lowercase and strip diacritics so "osterreich" finds "Österreich"
const normalize = (s: string) =>
    s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()

// Name in the UI language or in English starts with the query, or the query
// is the country's code
export function matchesCountry(c: Country, query: string): boolean {
    const q = normalize(query.trim())
    if (!q) return false
    return (
        [countryName(c), c.name, c.nameLong ?? ''].some((n) =>
            normalize(n).startsWith(q)
        ) ||
        c.alpha3.toLowerCase() === q ||
        c.alpha2.toLowerCase() === q
    )
}
