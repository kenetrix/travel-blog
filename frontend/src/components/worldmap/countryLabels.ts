import { Countries, getCountryByAlpha2 } from '../countries/countries'
import { COUNTRY_BOUNDS } from './countryBounds'
import { regionName } from '../../i18n/format'

// The base map labels carry only an English Natural Earth name, no ISO code.
// These are the names that match neither ours nor the browser's English one.
const NATURAL_EARTH: Record<string, string> = {
    Aland: 'AX',
    'Antigua and Barb.': 'AG',
    Bonaire: 'BQ',
    'Bosnia and Herz.': 'BA',
    'Br. Indian Ocean Ter.': 'IO',
    'British Virgin Is.': 'VG',
    'Cayman Is.': 'KY',
    'Central African Rep.': 'CF',
    'Congo (Brazzaville)': 'CG',
    'Congo (Kinshasa)': 'CD',
    'Cook Is.': 'CK',
    // Mis-encoded in the tiles
    'CuraÃ§ao': 'CW',
    'Czech Rep.': 'CZ',
    'Dem. Rep. Congo': 'CD',
    'Dominican Rep.': 'DO',
    'East Timor': 'TL',
    'Eq. Guinea': 'GQ',
    'Falkland Is.': 'FK',
    'Faroe Is.': 'FO',
    'Fr. Polynesia': 'PF',
    'Fr. S. and Antarctic Lands': 'TF',
    'Guinea Bissau': 'GW',
    'Heard I. and McDonald Is.': 'HM',
    'Hong Kong': 'HK',
    'Ivory Coast': 'CI',
    Kosovo: 'XK',
    Macau: 'MO',
    Macedonia: 'MK',
    'Marshall Is.': 'MH',
    'N. Korea': 'KP',
    'N. Mariana Is.': 'MP',
    'Pitcairn Is.': 'PN',
    'S. Geo. and the Islands': 'GS',
    'S. Korea': 'KR',
    'S. Sudan': 'SS',
    'Saint Helena': 'SH',
    'Solomon Is.': 'SB',
    'St. Barthelemy': 'BL',
    'St. Kitts and Nevis': 'KN',
    'St. Martin': 'MF',
    'St. Pierre and Miquelon': 'PM',
    'St. Vin. and Gren.': 'VC',
    Svalbard: 'SJ',
    Swaziland: 'SZ',
    Turkey: 'TR',
    'Turks and Caicos Is.': 'TC',
    'U.S. Virgin Is.': 'VI',
    Vatican: 'VA',
    'W. Sahara': 'EH',
    'Wallis and Futuna Is.': 'WF',
}

// Base map name → ISO alpha-2 code
function baseNameCodes(): Record<string, string> {
    const english = new Intl.DisplayNames(['en'], { type: 'region' })
    const codes: Record<string, string> = {}
    for (const c of Countries) {
        for (const name of [c.name, c.nameLong, english.of(c.alpha2)]) {
            if (name) codes[name] = c.alpha2
        }
    }
    return { ...codes, ...NATURAL_EARTH }
}

// Base map name → name in the UI language; unknown names stay English
export function countryLabels(): Record<string, string> {
    return Object.fromEntries(
        Object.entries(baseNameCodes()).map(([name, alpha2]) => [
            name,
            regionName(alpha2, name),
        ])
    )
}

// Base map name → approximate area of the main landmass in square degrees.
// The tiles carry no rank, so without it labels are placed in alphabetical
// order and small countries (Austria) crowd out big ones (Germany).
export function countryAreas(): Record<string, number> {
    const areas: Record<string, number> = {}
    for (const [name, alpha2] of Object.entries(baseNameCodes())) {
        const alpha3 = getCountryByAlpha2(alpha2)?.alpha3
        const box = alpha3 && COUNTRY_BOUNDS[alpha3]
        if (!box) continue
        const [w, s, e, n] = box
        areas[name] =
            (e - w) * (n - s) * Math.cos((((s + n) / 2) * Math.PI) / 180)
    }
    return areas
}

const VOWELS = /[aeiouyäöüáéíóúàèìòùâêîôûаеёиоуыэюя]/i

// Short label for low zoom, in the style of Natural Earth's English ones
// ("Ger.", "U.A.E."): initials of several words, or the first syllable
export function abbreviate(name: string): string {
    if (name.length <= 7 || !/^\p{L}/u.test(name)) return name
    // Lowercase words are joiners like "und", "de" or "и"
    const words = name.split(/[\s-]+/).filter((w) => /^\p{Lu}/u.test(w))
    if (words.length > 1) {
        return words.map((w) => w[0] + '.').join('')
    }
    let end = 3
    while (end < 4 && end < name.length && !VOWELS.test(name[end])) end++
    const short = name.slice(0, end) + '.'
    return short.length < name.length ? short : name
}
