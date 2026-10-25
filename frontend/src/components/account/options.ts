// Languages offered in the profile; any BCP 47 tag is valid on the server
export const LANGUAGES = [
    { code: 'en', name: 'English' },
    { code: 'de', name: 'Deutsch' },
    { code: 'fr', name: 'Français' },
    { code: 'es', name: 'Español' },
    { code: 'it', name: 'Italiano' },
    { code: 'nl', name: 'Nederlands' },
    { code: 'pt', name: 'Português' },
    { code: 'ja', name: '日本語' },
]

// IANA time zones known to the browser (UTC first, as the server's default)
export const TIMEZONES = [
    'UTC',
    ...Intl.supportedValuesOf('timeZone').filter((tz) => tz !== 'UTC'),
]
