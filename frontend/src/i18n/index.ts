import { ref, shallowRef } from 'vue'
import en from './messages/en'
import type { Plural } from './plural'

export const LANGUAGES = [
    { code: 'en', name: 'English' },
    { code: 'de', name: 'Deutsch' },
    { code: 'es', name: 'Español' },
    { code: 'ru', name: 'Русский' },
] as const

export type Language = (typeof LANGUAGES)[number]['code']

// A group of messages may have an `other` key too (trip.categories), so only
// an object with nothing but plural forms counts as a plural
type IsPlural<T> = T extends Plural
    ? [Exclude<keyof T, Intl.LDMLPluralRule>] extends [never]
        ? true
        : false
    : false

// Catalog shape with plain strings, so other languages are not bound to the
// English wording
type Widen<T> = T extends string
    ? string
    : IsPlural<T> extends true
      ? Plural
      : { [K in keyof T]: Widen<T[K]> }
export type Messages = Widen<typeof en>

type Leaves<T, P extends string = ''> = {
    [K in keyof T & string]: T[K] extends string
        ? `${P}${K}`
        : IsPlural<T[K]> extends true
          ? `${P}${K}`
          : Leaves<T[K], `${P}${K}.`>
}[keyof T & string]
export type MessageKey = Leaves<Messages>

// Names for codes the browser cannot translate, keyed by code
export interface LocalizedData {
    capitals: Record<string, string>
    // IATA → [airport name, city]; '' where only English exists
    airports: Record<string, string[]>
}

interface Bundle {
    messages: Messages
    data: LocalizedData
}

const EMPTY_DATA: LocalizedData = { capitals: {}, airports: {} }

// One lazy chunk per language; the service worker precaches them for offline use
const loaders: Record<Exclude<Language, 'en'>, () => Promise<Bundle>> = {
    de: () => import('./bundles/de'),
    es: () => import('./bundles/es'),
    ru: () => import('./bundles/ru'),
}

const STORAGE_KEY = 'language'

const current = ref<Language>('en')
const catalog = shallowRef<Messages>(en)
const data = shallowRef<LocalizedData>(EMPTY_DATA)

export const language = current

export function localizedData(): LocalizedData {
    return data.value
}

// Reduces a BCP 47 tag like "de-AT" to a supported language
export function supported(tag: string | undefined | null): Language | null {
    const primary = tag?.split(/[-_]/)[0].toLowerCase()
    return LANGUAGES.find((l) => l.code === primary)?.code ?? null
}

function deviceChoice(): Language | null {
    try {
        return supported(localStorage.getItem(STORAGE_KEY))
    } catch {
        return null
    }
}

// Profile, then the choice made on this device, then the browser languages
export function resolveLanguage(profileLanguage?: string): Language {
    const candidates = [
        profileLanguage,
        deviceChoice(),
        ...(navigator.languages ?? [navigator.language]),
    ]
    for (const tag of candidates) {
        const l = supported(tag)
        if (l) return l
    }
    return 'en'
}

let requested: Language = 'en'

export async function setLanguage(lang: Language) {
    requested = lang
    const bundle = lang === 'en' ? null : await loaders[lang]()
    // A later call won while this one was loading
    if (requested !== lang) return
    catalog.value = bundle?.messages ?? en
    data.value = bundle?.data ?? EMPTY_DATA
    current.value = lang
    document.documentElement.lang = lang
}

// A visitor's pick on the login page; remembered on this device
export function chooseLanguage(lang: Language) {
    try {
        localStorage.setItem(STORAGE_KEY, lang)
    } catch {
        // Private mode: the choice lasts until the page is reloaded
    }
    return setLanguage(lang)
}

function lookup(messages: unknown, key: string): string | Plural | undefined {
    let node = messages
    for (const part of key.split('.')) {
        if (node === null || typeof node !== 'object') return undefined
        node = (node as Record<string, unknown>)[part]
    }
    return typeof node === 'string' || (node && typeof node === 'object')
        ? (node as string | Plural)
        : undefined
}

const pluralRules = new Map<string, Intl.PluralRules>()
const numberFormats = new Map<string, Intl.NumberFormat>()

function pluralForm(message: Plural, count: number, lang: string): string {
    let rules = pluralRules.get(lang)
    if (!rules) {
        rules = new Intl.PluralRules(lang)
        pluralRules.set(lang, rules)
    }
    return message[rules.select(count)] ?? message.other
}

function formatParam(value: string | number, lang: string): string {
    if (typeof value === 'string') return value
    let format = numberFormats.get(lang)
    if (!format) {
        format = new Intl.NumberFormat(lang)
        numberFormats.set(lang, format)
    }
    return format.format(value)
}

export type Params = Record<string, string | number>

// Message in the UI language, English if it is missing. {name} is replaced
// by params.name; a plural message picks its form by params.count.
export function t(key: MessageKey, params?: Params): string {
    const lang = current.value
    let message = lookup(catalog.value, key) ?? lookup(en, key)
    if (message === undefined) return key
    if (typeof message !== 'string') {
        message = pluralForm(message, Number(params?.count ?? 0), lang)
    }
    if (!params) return message
    return message.replace(/\{(\w+)\}/g, (match, name: string) =>
        name in params ? formatParam(params[name], lang) : match
    )
}

// For codes that come from the server, e.g. error codes
export function hasMessage(key: string): key is MessageKey {
    return (
        lookup(catalog.value, key) !== undefined ||
        lookup(en, key) !== undefined
    )
}
