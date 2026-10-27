# Proposal

## Why

The country details already link to a trip page (`/countries/:iso/trips/new`), but that page is only a "coming soon" placeholder. A trip is an ordered series of places the user visited, and it often crosses borders, so it cannot belong to one country. Users want to record such trips, starting from the country they are looking at, whether the trip stays in that country or continues into others.

## What Changes

- New trip entity: a title, optional start and end dates, notes, a visibility, and an ordered list of stops. A stop is a named place on the map (coordinates) with its country, an optional arrival date and notes.
- A trip's countries are derived from its stops, in the order they are first visited; a trip with stops in one country is a domestic trip, one with stops in several countries spans them. There is no separate "country trip" vs. "multi-country trip" type.
- The country details' trips section lists the user's trips that have at least one stop in that country, and replaces "+ Add trip" with "Start a new trip here". It opens the trip editor with the map focused on that country; the first stop the user places there is in that country, later stops may be anywhere.
- Each trip gets a summary page (route `/trips/:id`, full page like the current placeholder): title, dates and length in days, the countries in visiting order, number of stops, total distance between the stops, the stops in order with their dates and notes, and the trip notes. It offers "Edit" and "Delete"; the trips in the country details link to it.
- The trip editor is a panel over the map (routes `/trips/new`, `/trips/:id/edit`): places are chosen with a pin icon. On desktop a click sets the pin; on phones and tablets the pin stays in the centre of the map and the user moves the map underneath it. "Add stop" confirms. Stops are then reordered (up/down, drag on desktop), renamed, moved with the pin, given a date or notes, or removed.
- While the editor is open, the map renders place names (cities and towns) once zoomed in far enough, larger places first. The nearest place to the pin is suggested as the stop name, and clicking a place name on desktop sets the pin on it. The place data is a bundled extract of GeoNames (CC BY 4.0) split per country, served from the app's own origin and cached for offline use; the current demo tiles have no place names. The map shows the stops as numbered markers connected by a line in visiting order.
- Saving creates or updates the trip with all of its stops in one request; the stop order is the order the user arranged. After saving, the summary page opens.
- Trips can be deleted (with confirmation); deleting a trip deletes its stops and their photos.
- Offline: trips and their stops are part of the stored snapshot and stay viewable; every write is disabled with the offline hint.
- The unused placeholder route `/countries/:iso/trips/new` is removed without a redirect.
- No geocoding or other external service: the name is suggested from the bundled places and can be edited, the country comes from the country polygon under the pin tip.
- Not in this change: showing trips on the public travel page, attaching photos to stops in the UI, a global trip overview, marking countries as visited automatically.

## Capabilities

### New Capabilities

- `trips`: trips as an ordered list of stops across one or more countries; starting a trip from the country details; the trip summary page; the trip editor over the map (adding, ordering, editing and removing stops); the trip's countries and route on the map; listing trips per country; saving, deleting, validation, privacy and offline behaviour.

### Modified Capabilities

None (`openspec/specs/` has no capabilities yet).

## Impact

- Database: new `trips` table in `backend/migrations/data/001_init/02_travel.sql`; `locations` becomes the trip stops (`trip_id NOT NULL` with cascade, `position`). Pre-`v1.0.0`, so `001_init` is changed and the test database recreated.
- API contract: new `api/paths/trips.yaml`, `api/paths/trips_{id}.yaml`, `api/components/schemas/trip.yaml`; regenerated `backend/internal/adapter/http/api/api.gen.go` and `frontend/generated/schema.gen.ts`.
- Backend: `domain/trip.go`, `domain/location.go` (trip id, position); a `TripService` that replaces the unused `LocationService` and reuses its photo-file cleanup; `TripRepository` in `port/repository.go` and a SQLite implementation (stops are replaced in one transaction); handlers in `travel_handler.go`; new error codes. Tests in domain, service, repository and router packages.
- Frontend: `TripPage.vue` rewritten as the trip summary page, a new `TripEditor.vue` map panel, router changes, `CountryDetailModal.vue` trips section, store (`trips`, load/save/delete) and snapshot, `MapProvider`/`MapLibreProvider` (pin, place-name layer, trip route layer), a place lookup module, i18n messages in all four languages.
- Data: a generator script that turns the GeoNames `cities5000` dump into per-country files under `frontend/public/places/`, committed like `airports.json`; GeoNames attribution in the map attribution.
- No new npm or Go dependency, no environment variable, no CSP change (place files come from the app's own origin).
