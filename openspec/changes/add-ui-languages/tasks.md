# Tasks

## 1. API contract

- [x] 1.1 Add `code` (string) and `params` (string map) to `Problem` in `api/components/schemas/common.yaml`, document the code list in its description, and restrict the profile/registration `language` in `api/components/schemas/account.yaml` to the enum `en|de|es|ru`; verify `npm run generate:api` in `frontend/` and the Go generator both run cleanly and `go build ./...` passes

## 2. Backend

- [x] 2.1 Add `service.Error{Kind, Code, Params, Msg}` whose `Error()` keeps the `"<kind>: <msg>"` text and whose `Unwrap()` returns the kind sentinel; verify a unit test asserts `errors.Is(err, ErrInvalid)` and the text
- [x] 2.2 Give every error site in `application/service` and `domain` a snake_case code (and params where the message has values); verify `grep` finds no remaining `fmt.Errorf("%w: …", Err…)` returned to handlers and `go test ./...` passes
- [x] 2.3 Copy `Code`/`Params` into the problem in `handler.problem()`, `unknown` otherwise; verify a router test checks `code: "username_taken"` on a duplicate registration and `code: "invalid_language"` on `PATCH /api/user/profile` with `fr`
- [x] 2.4 Restrict `checkLanguage` to `en|de|es|ru`, reduce the provider locale to its primary subtag in `oidcLanguage`, and update the `language` comment in `001_init/01_accounts.sql`; verify service tests for `de-AT` → `de` and `fr` → rejected
- [x] 2.5 Accept `lang` on `GET /api/auth/oidc/login`, keep it in the login state next to `next`, and use it when the provider locale is unsupported; verify an OIDC service test for provider `fr-FR` with `lang=de` → profile `de`, and document the parameter in `auth_oidc_config.yaml`

## 3. Frontend i18n core

- [x] 3.1 Create `src/i18n/index.ts` with the language ref, `SUPPORTED`, `resolveLanguage()` (profile → `localStorage` → `navigator.languages` → `en`), `setLanguage()` (lazy catalog load, `document.documentElement.lang`) and a typed `t(key, params)` with `{name}` interpolation and `Intl.PluralRules` plurals; verify `vue-tsc` rejects an unknown key
- [x] 3.2 Create `src/i18n/format.ts` (numbers, dates, distances via `Intl.NumberFormat` unit style, durations, `countryName`, `currencyName`, `languageName` via `Intl.DisplayNames` with English fallback, `collator`); move `distance.ts` formatting onto it; verify in the browser that `de` shows `9.284 km`
- [x] 3.3 Wire it up in `main.ts` and the store: resolve before mount, watch `profile.language`, keep the language on logout; verify in the browser that a `de-AT` browser shows German and that logging in as a Spanish-profile user switches to Spanish
- [x] 3.4 Make `problemMessage` take a fallback key and translate `errors.<code>` with params, falling back to `detail`; move `OIDC_ERRORS` into `errors.*`; verify a duplicate registration shows the translated message

## 4. Translate the UI

- [x] 4.1 Create `messages/en.ts` and move all literals of `App.vue`, `ui/*` (`AppSheet`, `ConfirmModal`, `OfflineBanner`, `PageWrapper`, `TabBar`, `offline.ts`) to keys; verify the map page shows no change in English
- [x] 4.2 Move the literals of `account/*` (`AuthPage`, `PasswordPage`, `ProfileHeader`, `ProfilePage`, `options.ts`) and `admin/*` (`AdminPage`, `OidcSettings`) to keys; cut `LANGUAGES` to the four, each named in its own language; add the language picker to the login page that writes `localStorage`; verify the picker switches the page and survives a reload
- [x] 4.3 Move the literals of `countries/*`, `flights/*` and `trips/*` to keys, including plural messages (flight and country counts, durations); verify `vue-tsc` and the English UI unchanged
- [x] 4.4 Write `messages/de.ts`, `es.ts`, `ru.ts` with every key of `en.ts`, including all `errors.*` codes from 2.2; verify with a type-level or script check that no key is missing, and that Russian shows "1 перелёт / 3 перелёта / 5 перелётов"

## 5. Localized data

- [x] 5.1 Use `countryName` and the collator in `countries.ts`, `CountryPanel.vue` and `CountryDetailModal.vue`; match search against localized and English names; verify "герм" and "germ" both find Germany in Russian
- [x] 5.2 Set the `countries-label` `text-field` in `MapLibreProvider.ts` to a `match` on the base style's ISO property with localized names, rebuilt on language change, and localize flight labels; verify Germany reads "Alemania" on the map in Spanish
- [x] 5.3 Write `frontend/scripts/i18n-data.ts` that queries Wikidata for `de`/`es`/`ru` labels of capitals (by country ISO) and airports/cities (by IATA `P238`) and writes `src/i18n/data/{capitals,airports}.{de,es,ru}.json`; run it and verify the files load and the diff looks sane for a sample of 20 entries
- [x] 5.4 Regenerate `countryFacts.ts` with region keys, ISO 4217 currency codes and ISO 639 language codes (English names kept as fallback); render them with the format helpers and the capitals file; verify Spain in German shows "Europa", "Euro", "Spanisch", "Madrid"
- [x] 5.5 Add `airportName`/`airportCity` lookups to `airports.ts`, use them in `AirportSelect.vue`, `FlightPanel.vue` and `FlightDetails.vue`, and index both languages for search; verify "Мюнх", "Munich" and "MUC" find Munich in Russian
- [x] 5.6 Load the data files with dynamic `import()` per language; verify after `npm run build` that they are separate chunks matched by the service worker's precache list, and that switching to German offline shows German country names

## 6. Docs and integration

- [x] 6.1 Document the languages, the resolution order and the data script in `README.md` (Features and Common tasks) and the error codes in `docs/`; verify the documented script command runs
- [x] 6.2 Check each language at 375 px and desktop width on the map, both panels, login, profile, password and admin pages; verify no clipped buttons or English leftovers, and `npm run build` plus `go test ./...` pass
