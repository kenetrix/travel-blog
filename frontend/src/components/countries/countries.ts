import countries from './countries.json'

export interface Country {
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
