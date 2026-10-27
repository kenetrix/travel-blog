# Design

## Context

- `001_init/02_travel.sql` already has a `locations` table (id, user, `country_id`, name, lat/lng, `visited_at`, notes, visibility) and `03_photos.sql` attaches photos to locations with `ON DELETE CASCADE`. `domain.Location`, `LocationRepository`, `LocationService` (add/delete with photo-file cleanup) and `PhotoService` exist but are not wired into the router or the OpenAPI contract; no client uses them.
- The frontend has a placeholder `trips/TripPage.vue` on `/countries/:iso/trips/new`, rendered through App's `<RouterView>` as a full page (`PageWrapper`) on top of the map. Only `CountryDetailModal.vue` links to it, and the route was never used beyond the placeholder.
- App shows either `CountryPanel` or `FlightPanel` over the always-rendered `WorldMap`. The map provider already resolves a clicked country via `ADM0_A3` on the country layer and draws flight routes from a GeoJSON source.
- The server validates every input in the service and returns `ErrInvalid` (`400`) with a code; country codes are checked for format only (`checkCode`). The offline snapshot stores API objects as returned.
- Pre-`v1.0.0`: `001_init` is edited in place, no migration scripts.

## Goals / Non-Goals

**Goals:**
- One trip model for domestic and cross-border trips, with stops as the only source of a trip's countries.
- An editor where stops are placed on the map itself, usable on a 320px phone inside the bottom sheet.
- Atomic saves, so the stored order never mixes an old and a new arrangement.

**Non-Goals:**
- Place search by name, reverse geocoding through an external service, translated place names, routing along roads, travel modes between stops, linking flights to trips.
- Photo upload UI for stops (the data model keeps photos on stops so it can follow).
- Trips on the public page and a global trip overview; both can build on `GET /api/trips` later.

## Decisions

### Stops are the existing `locations`, owned by a trip
New table `trips (id, user_id, title, starts_on, ends_on, notes, visibility, created_at, updated_at)`. `locations` gains `trip_id TEXT NOT NULL REFERENCES trips(id) ON DELETE CASCADE` and `position INTEGER NOT NULL`, with `UNIQUE (trip_id, position)` and an index on `(user_id, country_id)` kept for the per-country lookup. Reusing `locations` keeps the photo relation intact instead of adding a parallel `trip_stops` table and migrating photos later. `trip_id NOT NULL` is safe because no location can exist today without an API. Alternative considered: a join table `trip_locations` so a place can be shared by trips — rejected, a revisited place is a new stop with its own date and notes.

### Countries are derived, not stored
`TripSummary.countries` is computed in SQL (`country_id` of the stops ordered by `position`, deduplicated in first-seen order in Go). No `trips.country_id`: a trip started from Italy that ends in France must not be "the Italy trip". The editor's starting country is only a query parameter (`/trips/new?country=ITA`) used to fit the map.

### API
- `GET /api/trips` → `TripSummary[]` (id, title, dates, visibility, `countries`, `stopCount`). Enough for the per-country lists; the client filters by `countries`.
- `GET /api/trips/{id}` → `Trip` (summary fields, notes, `stops[]`).
- `POST /api/trips` → `201 Trip`, body `TripInput` (title, dates, notes, visibility?, `stops[]`).
- `PUT /api/trips/{id}` → `200 Trip`, same `TripInput`: full replacement of trip fields and stops.
- `DELETE /api/trips/{id}` → `204`.

A stop in `TripInput` has an optional `id`; with an id it updates that stop (must belong to the trip, else `400 unknown_stop`), without one it is created. Array index is the position. `PUT` over `PATCH` + per-stop endpoints because the editor always owns the whole list and reordering would otherwise need many requests that can half-fail. Limits: 200 stops, 120-character names and titles, 4000-character notes, dates `YYYY-MM-DD` in 1900–2100, `lat` in [-90, 90], `lng` in [-180, 180]. Error codes: `title_required`, `name_required`, `invalid_date_range`, `too_many_stops`, `unknown_stop`, `trip_not_found`; documented in the existing error code reference.

### Save in one transaction
`TripRepository.Save(user, trip, stops)` runs in one SQLite transaction: upsert the trip, delete stops not in the new list (collecting their photo storage keys first), shift remaining positions to negative values to avoid `UNIQUE (trip_id, position)` collisions, then update/insert each stop with its final position. Photo files of removed stops are deleted after commit, logging failures like `LocationService.Delete` does today. `TripService` replaces `LocationService` (removed with its port methods that become unused); `PhotoService` keeps using `LocationRepository.Get` for ownership.

### Visibility
Trip visibility defaults to the user's default visibility; stops inherit the trip's visibility (the `locations.visibility` column is set from the trip on save) so a later public page cannot leak single stops of a private trip. The public travel handler is not touched.

### Summary is a full page, the editor a map panel
Routes: `trip` at `/trips/:id` (summary), `trip-edit` at `/trips/:id/edit` and `trip-new` at `/trips/new` (editor). The placeholder route `/countries/:iso/trips/new` was never linked from anywhere else, so it is removed without a redirect.

`TripPage.vue` keeps its full-page `PageWrapper` and becomes the read-only summary: reading a trip needs room for text, not the map. Header actions are "Edit" and "Delete". Length in days is `ends_on - starts_on + 1`, shown only with both dates; the distance is the sum of great-circle distances of consecutive stops with `distanceKm` from `flights/distance.ts` (moved to a shared helper), shown only with two or more stops. Countries are listed by name with the existing `countryName`. A `404` from `GET /api/trips/{id}` (or a missing snapshot entry offline) shows a not-found state.

The new `TripEditor.vue` is an `AppSheet` panel like `CountryPanel`; App hides the country/flight panel while an editor route is active, because stops are placed with the pin on the visible map. Layout: header (title input, dates, visibility, save/cancel), then the stop list; on phones the collapsed sheet shows the title and stop count so the map stays usable. Save and cancel of an existing trip go to its summary; cancel of a new trip goes back to the map.

### Pin placement
One pin component (an inline SVG pin, tip at the bottom centre) with two modes, chosen by `coarsePointer` from `useLayout.ts`:
- **Click mode** (fine pointer): a `maplibregl.Marker` with `anchor: 'bottom'` is set on map click and moved on the next click. A click on a place label (`queryRenderedFeatures` on the place layer) snaps to that place's coordinates.
- **Centre mode** (coarse pointer): an absolutely positioned pin over the map, centred in the visible area (the map insets of `mapInsets.ts` already describe the area the sheet leaves free), so it needs no map marker. The point is `map.unproject` of the pin tip, read on `moveend`. The sheet collapses while placing; map taps are ignored.

Both modes report `(lngLat, iso | null)`. The country is resolved with `queryRenderedFeatures` on the country layer at the tip's screen point; `null` (water) disables confirming. While a trip editor is open, country clicks do not open the country details. "Move" on a stop reuses the same flow starting at the stop.

`MapProvider` gets `showTrip(stops)`, `clearTrip()`, `setPin(lngLat | null)`, `onMapClick(cb)`, `pointAt(screenPoint)` (lngLat + iso), `showPlaces(visible)`. Stops are a symbol layer (numbered circles) plus a `LineString` source; the route line uses a new `--trip` colour token next to `--flight`. Alternative considered: draggable markers instead of the centre pin — rejected because dragging a small marker under a finger hides the target and fights with map panning.

### Place names from a bundled GeoNames extract
The demo tiles have no place layer, and switching the base map to a full vector tile provider (e.g. OpenFreeMap) would replace the country layer the app is built on (`ADM0_A3`, feature-state colouring) and add an external host to the CSP. Instead, a script (`frontend/scripts/places.ts`, run with `node` like `scripts/i18n-data.ts`) reads the GeoNames `cities5000.txt` dump (places with ≥ 5,000 inhabitants, ~50k entries, CC BY 4.0), maps the alpha-2 country codes to the alpha-3 codes of `countries.ts`, and writes `frontend/public/places/<ISO3>.json` as compact arrays `[name, lat, lng, population]` (coordinates rounded to 4 decimals). Files are committed like `airports.json`; the script is rerun by hand when updating.

`places.ts` loads the files of the countries intersecting the viewport once the zoom reaches 5, caches them in memory, and feeds one GeoJSON source. The symbol layer shows places by population tiers via `minzoom`/filters (≥ 500k from zoom 5, ≥ 100k from 7, all from 9), with `symbol-sort-key` = −population and collision detection so labels never overlap. Labels use the GeoNames `name` (local name in Latin script); translated place names are out of scope. Workbox caches `/places/*.json` with `CacheFirst`, so viewed areas work offline. GeoNames is credited in the map attribution while place names are shown.

Name suggestion: the nearest place within 15 km (great-circle distance over the pin's country file plus every country file already loaded for the view, which covers towns across a nearby border) is suggested; a linear scan of a few thousand entries per country is fast enough without a spatial index.

### Client state and offline
The store gets `trips: TripSummary[]` loaded with the other travel data and stored in the snapshot, and `tripDetails: Record<id, Trip>` filled on open and snapshotted as well, so a trip opened once is viewable offline. The editor edits a local draft copy; `dirty` compares it with the loaded trip and drives the leave confirmation (`ConfirmModal`), as in `CountryDetailModal`.

### Ordering on touch
Up/down buttons are the primary reorder control (accessible, work in the sheet); HTML5 drag-and-drop on the list is added for pointer devices only. No drag library.

## Risks / Trade-offs

- [The demo tiles' simplified country polygons put coastal or border pins in the wrong country or the sea] → The stop shows its country name so a wrong one is visible and can be moved; if this proves frequent, the country of a suggested place can be preferred over the polygon (follow-up, not in this change).
- [Place files add ~2–3 MB to the repository and the deployed assets] → Split per country and loaded only when zoomed into that country; gzip transfers a fraction of that.
- [Places below 5,000 inhabitants have no label or suggestion] → The name stays editable; a bigger extract (`cities1000`) can replace the file set later without code changes.
- [Natural Earth `ADM0_A3` codes are not always ISO (e.g. Kosovo)] → Same codes as the existing country entries, so lists stay consistent; the server checks the format only.
- [Full replacement on `PUT` overwrites edits made on another device in between] → Acceptable for a single-user app; `updated_at` is returned and could back an `If-Match` check later.
- [Deleting stops deletes photo files outside the transaction] → Files are deleted only after commit; a failed file delete leaves an orphan file, never a dangling record.
- [Large snapshots with many trip details] → Only trips the user opened are stored in detail; summaries are small.

## Migration Plan

Edit `001_init/02_travel.sql` (new `trips` table before `locations`, new columns and constraint on `locations`) and recreate the local and test databases, since `locations` changes shape and holds no data. Rollback is reverting the commit and recreating the database again.
