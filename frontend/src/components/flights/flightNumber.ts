// Same rules as domain.NormalizeFlightNumber in the backend. A three-letter
// (ICAO) designator is only tried when the third character is a letter.
const ICAO = /^([A-Z]{3})([0-9]{1,4})([A-Z]?)$/
const IATA = /^([A-Z0-9]{2})([0-9]{1,4})([A-Z]?)$/

// Canonical form, e.g. "lh 0400" → "LH400"; null if it is no flight number
export function normalizeFlightNumber(input: string): string | null {
    const s = input.trim().replace(/[\s-]/g, '').toUpperCase()
    let m = ICAO.exec(s)
    if (!m) {
        m = IATA.exec(s)
        if (!m || !/[A-Z]/.test(m[1])) return null
    }
    const number = m[2].replace(/^0+/, '')
    return number ? m[1] + number + m[3] : null
}

// "LH400" → "LH 400"
export function formatFlightNumber(canonical: string): string {
    return canonical.replace(/^([A-Z]{3}|[A-Z0-9]{2})(?=\d)/, '$1 ')
}

export function flightradarUrl(canonical: string): string {
    return `https://www.flightradar24.com/data/flights/${canonical.toLowerCase()}`
}
