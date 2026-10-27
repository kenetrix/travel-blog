import { language, t } from '.'

// Formatters read `language`, so templates and computeds using them update
// when it changes. Intl objects are cached per language and options.
const cache = new Map<string, object>()

function cached<T extends object>(
    kind: string,
    options: object,
    make: () => T
): T {
    const key = `${kind}|${language.value}|${JSON.stringify(options)}`
    let value = cache.get(key) as T | undefined
    if (!value) {
        value = make()
        cache.set(key, value)
    }
    return value
}

export function formatNumber(
    n: number,
    options: Intl.NumberFormatOptions = {}
) {
    return cached(
        'number',
        options,
        () => new Intl.NumberFormat(language.value, options)
    ).format(n)
}

const KM_PER_MILE = 1.609344

// Distance in the unit the user picked in the settings
export function formatDistance(km: number, unit: 'km' | 'mi' = 'km'): string {
    const value = unit === 'mi' ? km / KM_PER_MILE : km
    return formatNumber(Math.round(value), {
        style: 'unit',
        unit: unit === 'mi' ? 'mile' : 'kilometer',
        maximumFractionDigits: 0,
    })
}

// 710 → "11 h 50 min"
export function formatDuration(minutes: number): string {
    const h = Math.floor(minutes / 60)
    const m = minutes % 60
    if (!h) return t('format.minutes', { m })
    return m ? t('format.hoursMinutes', { h, m }) : t('format.hours', { h })
}

// A calendar day ("2024-03-05"), independent of the time zone
export function formatDay(
    day: string,
    options: Intl.DateTimeFormatOptions = { dateStyle: 'long' }
) {
    return cached(
        'date',
        options,
        () =>
            new Intl.DateTimeFormat(language.value, {
                ...options,
                timeZone: 'UTC',
            })
    ).format(new Date(`${day}T00:00:00Z`))
}

export function formatDate(
    date: Date | string,
    options: Intl.DateTimeFormatOptions = { dateStyle: 'medium' }
) {
    return cached(
        'date',
        options,
        () => new Intl.DateTimeFormat(language.value, options)
    ).format(new Date(date))
}

function displayName(
    type: Intl.DisplayNamesType,
    code: string
): string | undefined {
    try {
        const name = cached(
            `names-${type}`,
            {},
            () =>
                new Intl.DisplayNames([language.value], {
                    type,
                    fallback: 'none',
                })
        ).of(code)
        return name && name !== code ? name : undefined
    } catch {
        // Invalid code or no DisplayNames support
        return undefined
    }
}

// ISO 3166-1 alpha-2 → name in the UI language
export function regionName(alpha2: string, fallback: string): string {
    return displayName('region', alpha2) ?? fallback
}

// ISO 4217 code → name in the UI language
export function currencyName(code: string, fallback: string): string {
    return displayName('currency', code) ?? fallback
}

// ISO 639 code → name in the UI language
export function languageName(code: string, fallback: string): string {
    return displayName('language', code) ?? fallback
}

export function collator(): Intl.Collator {
    return cached('collator', {}, () => new Intl.Collator(language.value))
}

// Joins names the way the UI language lists them ("A, B and C")
export function formatList(items: string[]): string {
    return cached(
        'list',
        {},
        () =>
            new Intl.ListFormat(language.value, {
                type: 'unit',
                style: 'short',
            })
    ).format(items)
}
