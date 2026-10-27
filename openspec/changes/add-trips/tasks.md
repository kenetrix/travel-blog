# Tasks

## 1. Data model

- [x] 1.1 Add the `trips` table to `backend/migrations/data/001_init/02_travel.sql` and give `locations` `trip_id NOT NULL` (cascade), `position` and `UNIQUE (trip_id, position)`; recreate the local database and verify `go test ./...` in `backend/` creates the schema without errors
- [x] 1.2 Add `domain/trip.go` (`Trip`, `TripSummary`, derived `Countries()` in first-seen order) and extend `domain.Location` with trip id and position; verify with a domain test for domestic, cross-border and revisited-country stop lists

## 2. Backend persistence and service

- [x] 2.1 Define `TripRepository` in `port/repository.go` (list summaries, get with stops, save in one transaction returning removed photo keys, delete returning photo keys) and drop the now unused `LocationRepository` methods; verify `go build ./...`
- [x] 2.2 Implement the SQLite trip repository with position shifting inside the transaction; verify repository tests for create, reorder, remove-with-photos, rollback on a failing stop and per-user isolation
- [x] 2.3 Implement `TripService` (validation limits and error codes from design, default visibility, stop visibility from the trip, photo-file cleanup after commit) replacing `LocationService`; verify service tests for every error code and for file deletion of removed stops

## 3. API contract and handlers

- [x] 3.1 Add `api/components/schemas/trip.yaml` (`TripSummary`, `Trip`, `TripStop`, `TripInput`), `api/paths/trips.yaml` and `api/paths/trips_{id}.yaml`, reference them in `api/openapi.yaml`, run `npm run generate:api` and verify the Redocly lint and both generated files are updated
- [x] 3.2 Implement the trip handlers in `travel_handler.go` and wire `TripService` in `cmd/api`; verify router tests for list, get, create, replace, delete, `404 trip_not_found` for another user's trip and `400` for invalid input
- [x] 3.3 Document the new error codes in `docs/error-codes.md` and verify each code returned by `TripService` is listed

## 4. Frontend data and map

- [x] 4.1 Add trips to the store (`trips` summaries loaded with travel data, `loadTrip`, `saveTrip`, `deleteTrip`, writes guarded by `canWrite`) and to the snapshot (`trips`, opened `tripDetails`); verify with `npm run build` in `frontend/` and by reloading offline after opening a trip
- [x] 4.2 Extend `MapProvider`/`MapLibreProvider` with `showTrip`, `clearTrip`, numbered stop markers, the route line and a `--trip` colour token; suppress country details clicks while a trip is shown; verify in the browser that markers, numbers and line follow the stop order
- [x] 4.3 Write `frontend/scripts/places.ts` (GeoNames `cities5000` → `public/places/<ISO3>.json`), generate and commit the files, and add the GeoNames attribution; verify a sample file (e.g. `ESP.json`) contains Granada and Seville with plausible coordinates
- [x] 4.4 Add `places.ts` (load files for countries in view from zoom 5, nearest place within 15 km) with unit tests for the nearest-place lookup, the place-name layer with population tiers and collision, `showPlaces`, and a Workbox `CacheFirst` rule for `/places/`; verify in the browser that labels appear only from zoom 5 with the editor open and stay after reloading offline
- [x] 4.5 Add the pin: click mode (marker, moved on click, snap to clicked place labels) and centre mode for coarse pointers (pin centred in the free map area, point read on `moveend`), both resolving the country under the tip; verify in the browser on desktop and in the mobile viewport that the reported point and country match the pin tip

## 5. Trip summary and editor

- [x] 5.1 Change the router: `trip` at `/trips/:id`, `trip-edit` at `/trips/:id/edit`, `trip-new` at `/trips/new` (query `country`); remove `/countries/:iso/trips/new` without a redirect; let App hide the country and flight panels on the editor routes; verify each URL in the browser
- [x] 5.2 Move `distanceKm` from `flights/distance.ts` to a shared helper and rewrite `TripPage.vue` as the summary page (title, dates, length in days, countries, stop count, total distance, stops with country, date and notes, trip notes, "Edit", "Delete" with `ConfirmModal`, not-found state, offline read-only); verify the scenarios of "Trip summary page" in the browser at 320px and desktop width
- [x] 5.3 Create `TripEditor.vue` as an `AppSheet` panel: title, dates, visibility, notes, stop list with name, country, date, notes, up/down, move, remove, desktop drag-and-drop; "Add stop" with the pin, suggested name and confirmation, collapsing the sheet in centre mode; fit the map to the starting country or the trip's stops; verify the scenarios of "Add a stop on the map", "Place a stop on touch devices", "Stop name suggested from the nearest place" and "Edit, reorder and remove stops" in the browser at 320px and desktop width
- [x] 5.4 Add save (client-side validation hints mirroring the server, then open the summary), cancel, leave confirmation for unsaved changes and the disabled state offline; verify saving, cancelling and leaving a dirty draft in the browser
- [x] 5.5 Add all trip summary and editor texts to `frontend/src/i18n/messages/{en,de,es,ru}.ts` and remove `trip.soon`; verify the type check passes and no key is missing in any language

## 6. Country details

- [x] 6.1 Replace the trips section of `CountryDetailModal.vue`: list trips with a stop in the country (sorted by start date, undated last; title, dates, stop count, other countries) linking to the summary at `/trips/:id`, and the "Start a new trip here" link to `/trips/new?country=<iso>`, disabled offline; update `country.addTrip`/`tripsNote` texts in all four languages; verify in the browser that a trip across Austria and Hungary appears in both countries
- [x] 6.2 Run Prettier on all touched frontend files, `npm run build` in `frontend/` and `go test ./...` in `backend/`; verify everything passes

## 7. Integration

- [x] 7.1 Walk through the full flow with `npm run dev`: start a trip from Italy, add Rome and Florence by click on desktop and Nice with the centre pin in the mobile viewport, reorder, save, check the summary, reload, open it from France's details, edit it from the summary, go offline and view the summary, then delete it; verify every step matches the spec and the public travel page shows no trips
