// IANA time zones known to the browser (UTC first, as the server's default)
export const TIMEZONES = [
    'UTC',
    ...Intl.supportedValuesOf('timeZone').filter((tz) => tz !== 'UTC'),
]
