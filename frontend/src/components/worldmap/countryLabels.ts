import { Countries, countryName } from '../countries/countries'
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

// Base map name → name in the UI language; unknown names stay English
export function countryLabels(): Record<string, string> {
    const english = new Intl.DisplayNames(['en'], { type: 'region' })
    const labels: Record<string, string> = {}
    for (const c of Countries) {
        const local = countryName(c)
        for (const name of [c.name, c.nameLong, english.of(c.alpha2)]) {
            if (name) labels[name] = local
        }
    }
    for (const [name, alpha2] of Object.entries(NATURAL_EARTH)) {
        labels[name] = regionName(alpha2, name)
    }
    return labels
}
