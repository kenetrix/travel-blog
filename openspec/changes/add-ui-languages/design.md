# Design

## Context

- The frontend has about 25 components with English literals, plus English strings in `.ts` helpers (`offline.ts`, `problemMessage` fallbacks, `OIDC_ERRORS` in `AuthPage.vue`). Formatting is pinned to `'en'`, `'en-US'` and `'en-GB'` (`CountryDetailModal.vue`, `distance.ts`, `FlightDetails.vue`). Sorting uses `'de'` (`CountryPanel.vue`).
- The profile already stores `language` (any BCP 47 tag, default `en`). Registration preselects `navigator.language` from a list of eight languages (`account/options.ts`). Single sign-on takes the provider's `locale` (`oidcLanguage`). The value is never used by the UI.
- Country names come from `countries.json` (English). Country facts are English strings in `countryFacts.ts` (`[alpha3, capital, region, area, population, currencies, languages]`), derived from mledoze/countries. Airports are 1172 rows `[iata, name, city, country, lat, lon, alpha2]` in `airports.json`.
- The map's country labels come from the `countries-label` layer of the demotiles base style; flight labels are set by `MapLibreProvider.ts`.
- Server errors are `"<kind>: <text>"` errors turned into `Problem{title, status, detail}` by `problem()` in `handler/handler.go`; the frontend shows `detail`.
- The frontend has no unit test runner; checks are `vue-tsc`, the build and the running app. The backend has Go tests per package.

## Goals / Non-Goals

**Goals:**
- One place that knows the UI language; components only call `t()` and the format helpers.
- Type-checked message keys, so a missing or misspelled key fails `vue-tsc`.
- English stays in the main bundle; other languages are lazy chunks that the service worker precaches.

**Non-Goals:**
- Right-to-left layout or languages beyond the four.
- Translating user content (bio, trip notes, display names) or the PWA manifest's name.
- Localized URLs or a language in the path.
- Server-side rendering of messages by `Accept-Language`.

## Decisions

### Own small i18n module instead of vue-i18n

`src/i18n/` holds `index.ts` (state, `t`, `setLanguage`, detection), `format.ts` (dates, numbers, distances, durations, display names) and `messages/{en,de,es,ru}.ts`. `en.ts` is the source of truth: its type is `Messages`, the other catalogs are checked with `satisfies Messages`, so a missing or unknown key is a type error; at runtime a missing key still falls back to English. `t(key, params)` takes a dotted key typed from `Messages`, interpolates `{name}` placeholders, and picks plural forms with `Intl.PluralRules` from objects like `{ one, few, many, other }`. The current language is a `ref`, so templates re-render on change.

Alternative: vue-i18n. It brings its own message compiler, needs a custom plural rule for Russian anyway, and the project avoids dependencies where the platform suffices (`Intl` covers plurals, numbers, dates and display names). The module is about 100 lines.

### Language resolution

`resolveLanguage()` returns the first supported primary subtag from: the profile language (when `store.profile` is set), `localStorage['language']` (the visitor's choice, wrapped in try/catch), `navigator.languages`, then `en`. A `watch` on `store.profile?.language` calls `setLanguage`, which loads the catalog and data chunks, then sets the ref and `document.documentElement.lang`. The login page picker writes `localStorage` and calls `setLanguage`. Logout does not reset the ref (no flash of another language); the next load resolves again.

### Registration and single sign-on

`AuthPage.vue` preselects the current UI language. The single sign-on link becomes `/api/auth/oidc/login?next=…&lang=<ui language>`; the backend keeps `lang` in the existing login state next to `next` and uses it in `oidcLanguage` when the provider locale is unsupported. `checkLanguage` accepts only `en|de|es|ru`; `oidcLanguage` reduces the provider locale to its primary subtag first.

Alternative: send the language after the first login with a `PATCH`. Rejected: an extra request and a moment where the profile holds the wrong value.

### Country names from `Intl.DisplayNames`

`countryName(alpha2)` uses `new Intl.DisplayNames([lang], { type: 'region' })`, falling back to the English name in `countries.json` when the result equals the code. Lists sort with `Intl.Collator(lang)`. Search compares a lower-cased prefix against both the localized and the English name. The `Country` view model keeps `name` (English) and gains a computed localized name via the helper, so `countries.json` needs no change.

The map's `countries-label` layer gets `text-field` set to a `match` expression from the base style's English `NAME` to the localized names, rebuilt on language change: the demotiles `centroids` layer has only `NAME` and `ABBREV`, no ISO code. `worldmap/countryLabels.ts` matches `NAME` against our English names, the browser's English names and an alias table for Natural Earth abbreviations ("Czech Rep."). Names it does not know (e.g. "Gaza") stay English.

### Country facts with codes

`countryFacts.ts` is regenerated from mledoze/countries keeping codes instead of English names: region as a key (`africa`, `americas`, …, translated in the catalogs), currencies as ISO 4217 codes and languages as ISO 639 codes, both rendered with `Intl.DisplayNames` (`type: 'currency'` / `'language'`); a code the browser cannot name falls back to the English name kept beside it. Capitals keep their English name in the row; translations live in `src/i18n/data/capitals.{de,es,ru}.json` (`alpha3 → name`).

### Airport and city names

`src/i18n/data/airports.{de,es,ru}.json` map `iata → [name, city]`, only for entries with a translation. `airports.ts` exposes `airportName(a)` / `airportCity(a)` that look up the loaded data and fall back to English; the country column uses `countryName(alpha2)`. Search builds its index from both languages.

The capital and airport files are produced by `frontend/scripts/i18n-data.ts`, a one-off script that queries Wikidata (capitals by country ISO code, airports by IATA code `P238`) for `de`, `es`, `ru` labels and writes the JSON files sorted by key. The output is committed; the script is rerun only when the source data changes. The files are imported with dynamic `import()` per language so they become chunks covered by the existing `globPatterns` precache.

Alternative: bundle all languages in the main chunk. Rejected for size (airport names ×3).

### Error codes in problem responses

`Problem` gains `code` (string) and `params` (string map). Service errors become a typed `service.Error{Kind, Code, Params, Msg}` (`Kind` is the existing `ErrInvalid`/`ErrConflict`/… sentinel, so `errors.Is` and the status mapping keep working); `Error()` still returns `"<kind>: <msg>"`. `problem()` copies `Code`/`Params` when the error is one, otherwise sets `code: "unknown"`. Codes are snake_case and documented in the OpenAPI schema description; each existing error site gets a code (about 66 sites, many are validation errors such as `invalid_language`, `username_taken`, `flight_exists`).

`problemMessage(error, fallbackKey)` looks up `errors.<code>` in the catalog with `params`, otherwise uses `detail`, otherwise the translated fallback. The OIDC `?error=` codes join the same `errors.*` namespace.

Alternative: translate on the server by `Accept-Language`. Rejected: a second set of catalogs in Go, and the frontend's offline and validation messages would still need their own.

### Formatting

`format.ts` wraps `Intl.NumberFormat`, `Intl.DateTimeFormat` and the unit formatting from `distance.ts` (`km`/`mi` via `Intl.NumberFormat` with `style: 'unit'`); durations use catalog messages with plurals. All hard-coded locales are removed.

## Risks / Trade-offs

- [Older browsers lack `Intl.DisplayNames` for some codes] → fall back to the English names kept in the data.
- [Wikidata labels are incomplete or odd for small airports] → only entries with a label are written; the rest show English. The script output is reviewed in the diff.
- [Translation quality: about a few hundred strings written by the implementer] → keep messages short and consistent; native review of `de`, `es`, `ru` is listed as a follow-up check.
- [Text length: German and Russian labels are longer] → check the phone layout (375 px) in each language for buttons and the tab bar.
- [Breaking language validation] → existing local profiles with other values are reset when the test database is recreated; no migration before v1.0.0.

## Migration Plan

Change `checkLanguage` and the comment on `language` in `001_init/01_accounts.sql` (the values become `en|de|es|ru`, the default stays `en`) in place and recreate the test database. Rollback is a revert of the change.
