# Proposal

## Why

The app is English only: every label, date, number, country name and server error is English, and the profile's language setting changes nothing. Users who speak German, Spanish or Russian should get the whole app in their language, picked from their browser or device without any setup.

## What Changes

- The app is available in English, German, Spanish and Russian. Every visible text is translated: UI labels and messages, page titles, the offline hint, confirm dialogs, the map's country labels and flight labels, and the `<html lang>` attribute.
- The UI language is chosen in this order: the logged-in user's profile language, a language a visitor picked on this device, the first supported language in the browser's or device's preferred languages, English. Only the primary subtag counts (`de-AT` → German).
- A visitor without a session can switch the language on the login page; the choice is remembered on this device. Registration stores the current UI language as the new profile language.
- The profile's language list offers only the four languages. Changing it switches the UI at once and applies on every device of that user.
- **BREAKING**: the server accepts only `en`, `de`, `es` and `ru` as profile language (was any BCP 47 tag). Single sign-on maps the provider's locale to one of them, otherwise `en`. No migration before v1.0.0.
- Country names are shown in the UI language everywhere (lists, search, details, map labels); search matches the translated name and the English name. Lists are sorted for the UI language.
- Dates, numbers, distances and durations are formatted for the UI language; Russian plural forms are correct.
- Country facts (capital, region, currencies, languages) are shown in the UI language. Airport and city names are shown in the UI language where a translation exists, otherwise in English; airport search matches both.
- Server errors are translated: every problem response carries a stable error `code` (plus parameters where the text needs them); the frontend shows the translated text for the code and falls back to the English `detail`.
- Aircraft type names, IATA codes and flight numbers stay as they are (proper names and codes).

## Capabilities

### New Capabilities

- `localization`: the supported languages, how the UI language is detected, chosen and stored, which content is translated, locale-aware formatting and sorting, the fallback to English, and translated server errors.

### Modified Capabilities

None (`openspec/specs/` has no capabilities yet).

## Impact

- API contract: `api/components/schemas/common.yaml` (`Problem` gets `code` and `params`), `api/components/schemas/account.yaml` (language becomes an enum of the four codes); regenerated `backend/internal/adapter/http/api/api.gen.go` and `frontend/generated/schema.gen.ts`.
- Backend: error values with codes in `application/service` and `domain`, `problem()` in `handler/handler.go`, `checkLanguage` and `oidcLanguage`, the admin defaults. Tests in the service and router packages.
- Frontend: a new i18n module (language detection, message catalogs for en/de/es/ru, formatting helpers), every component under `src/components/` switched from literal strings to catalog keys, `problemMessage` in `api/client.ts`, `account/options.ts`, the store (UI language from the profile), `countries.ts`, `countryFacts.ts` (codes instead of English names, plus translated capitals), `airports.ts` (translated names), `MapLibreProvider.ts` (country label text), `index.html`.
- Data: new per-language data files for capitals and airport/city names, generated once by a script from Wikidata and committed; loaded lazily per language and precached for offline use.
- No new runtime dependency (the browser's `Intl` APIs do pluralization, formatting and country/currency/language names). No environment variable, no CSP change.
