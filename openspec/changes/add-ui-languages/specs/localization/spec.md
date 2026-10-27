# Spec Delta

## Purpose

Shows the whole app in English, German, Spanish or Russian, picks the language from the user's profile or their browser or device, and formats names, dates and numbers for that language.

## ADDED Requirements

### Requirement: Supported languages

The app SHALL support exactly the UI languages English (`en`), German (`de`), Spanish (`es`) and Russian (`ru`). Any other language SHALL fall back to English.

#### Scenario: Unsupported browser language
- **WHEN** a visitor's browser prefers only French
- **THEN** the app is shown in English

### Requirement: Language detection for visitors

Without a session and without a language picked on this device, the app SHALL use the first entry of the browser's or device's preferred languages whose primary subtag is supported; region and script subtags SHALL be ignored.

#### Scenario: Regional variant
- **WHEN** the browser's preferred languages are `de-AT`, `en-US`
- **THEN** the app is shown in German

#### Scenario: Later preference is supported
- **WHEN** the browser's preferred languages are `fr-FR`, `es`, `en`
- **THEN** the app is shown in Spanish

### Requirement: Visitor language choice

The login page SHALL offer a language picker with the four languages, each named in its own language. A visitor's choice SHALL apply at once and be remembered on this device across reloads; it SHALL take precedence over the browser languages.

#### Scenario: Switch on the login page
- **WHEN** a visitor with an English browser picks "Русский" on the login page and reloads
- **THEN** the login page and the map are shown in Russian

### Requirement: Profile language drives the UI

For a logged-in user the UI language SHALL be the profile language, on every device, taking precedence over a device choice and the browser languages. Changing the profile language SHALL switch the UI without a reload once it is saved.

#### Scenario: Login on another device
- **WHEN** a user whose profile language is German logs in on a device whose browser prefers English
- **THEN** the app is shown in German after login

#### Scenario: Change in the profile
- **WHEN** the user changes the profile language from German to Spanish and saves
- **THEN** the app is shown in Spanish immediately

#### Scenario: Logout
- **WHEN** the user logs out
- **THEN** the app keeps the language it was shown in until the next reload, then follows the device choice or browser languages

### Requirement: New accounts take the current language

Registration SHALL preselect the current UI language as the profile language. A single sign-on account SHALL get the provider's locale if its primary subtag is supported, otherwise the current UI language of the login, otherwise English.

#### Scenario: Register in Spanish
- **WHEN** a visitor whose app is shown in Spanish registers without changing the language field
- **THEN** the new profile language is `es`

#### Scenario: Provider locale unsupported
- **WHEN** a new single sign-on user's provider reports locale `fr-FR` and the login was started from a German UI
- **THEN** the new profile language is `de`

### Requirement: Profile language values

The profile SHALL offer only the four languages, and the server SHALL reject any other profile language with `400`.

#### Scenario: Unsupported language via API
- **WHEN** a client sends `PATCH` with profile language `fr`
- **THEN** the server responds `400` with error code `invalid_language` and the profile is unchanged

### Requirement: All UI text is translated

Every text the app shows SHALL be in the UI language: labels, buttons, placeholders, headings, hints, empty states, confirm dialogs, validation and error messages, the offline hint, accessible names and the document language attribute. A missing translation SHALL show the English text.

#### Scenario: German UI
- **WHEN** the UI language is German
- **THEN** no English UI text is visible on the map, the country and flight panels, the login, profile, password and admin pages, and the document's `lang` is `de`

#### Scenario: Missing key
- **WHEN** a German translation is missing for one message
- **THEN** that message is shown in English and nothing else changes

### Requirement: Plural forms

Messages with a count SHALL use the plural rules of the UI language.

#### Scenario: Russian flight count
- **WHEN** the UI language is Russian and the user has 1, 3 and 5 flights
- **THEN** the panel shows "1 перелёт", "3 перелёта" and "5 перелётов"

### Requirement: Localized country names

Country names SHALL be shown in the UI language in lists, details and the map's country labels. Country lists SHALL be sorted by the rules of the UI language. Country search SHALL match the start of the translated name or of the English name, case-insensitively.

#### Scenario: Search in Russian
- **WHEN** the UI language is Russian and the user types "герм" or "germ" into the country search
- **THEN** the list shows "Германия"

#### Scenario: Map labels
- **WHEN** the UI language is Spanish
- **THEN** the map labels Germany as "Alemania"

### Requirement: Localized country facts

The country details SHALL show region, currencies and languages in the UI language, and the capital in the UI language where a translation exists, otherwise in English.

#### Scenario: German facts
- **WHEN** the UI language is German and the user opens the details of Spain
- **THEN** the capital reads "Madrid", the region "Europa", the currency "Euro" and the language "Spanisch"

### Requirement: Localized airport names

Airport and city names SHALL be shown in the UI language where a translation exists, otherwise in English. Airport search SHALL match the translated name, the English name and the IATA code. IATA codes, flight numbers and aircraft type names SHALL not be translated.

#### Scenario: Russian airport search
- **WHEN** the UI language is Russian and the user types "Мюнх" or "Munich" or "MUC" into the From input
- **THEN** the list offers Munich airport with its city shown as "Мюнхен"

### Requirement: Locale-aware formatting

Dates, numbers, distances and durations SHALL be formatted for the UI language.

#### Scenario: Number and date in German
- **WHEN** the UI language is German and a flight from 2024-03-05 has a distance of 9284 km
- **THEN** the details show "9.284 km" and the date in German, e.g. "5. März 2024"

### Requirement: Translated server errors

Every error response from the server SHALL carry a stable error code and the parameters its message needs, besides the English detail. The app SHALL show the translated message for a known code and the English detail for an unknown one.

#### Scenario: Username taken
- **WHEN** the UI language is Spanish and registration fails because the username exists
- **THEN** the error code is `username_taken` and the app shows the Spanish message

#### Scenario: Unknown code
- **WHEN** the server returns an error code the app does not know
- **THEN** the app shows the problem's English detail

### Requirement: Languages work offline

Switching or detecting the language SHALL work without a connection once the app has been loaded online, including the translated country, airport and capital names.

#### Scenario: Offline switch
- **WHEN** the installed app is offline and the visitor picks German on the login page
- **THEN** the page is shown in German with German country names
