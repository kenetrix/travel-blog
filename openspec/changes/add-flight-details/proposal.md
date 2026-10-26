# Proposal

## Why

A flight is stored only as a route (two IATA codes), so the flight log cannot say which flight it was, when it happened or how long it took. Users want to add a flight number and a few details they fill in themselves, and look the flight up on a public tracker.

## What Changes

- Adding a flight keeps the From/To airport pickers: Enter or "Add route" adds the route right away. The same form then shows an optional flight number input (e.g. `lh0400` → `LH 400`) with "Save" and "Skip". The number is normalised and validated on the client and the server.
- Saving a flight number opens the flight's details in the list with a link to the flight on Flightradar24 and three fields the user fills in: date (native date picker), duration (hours and minutes, prefilled with an estimate) and aircraft type (dropdown of common types, or "Other" as free text).
- Skipping is always possible: the flight number can be added later in the flight's details. After it is saved, the link and the fields appear immediately, without a page reload.
- Without a flight number, the details show the route, the great-circle distance and an estimated duration derived from the distance, plus the flight number input.
- Details can be edited later (number, date, duration, aircraft type, also clearing them); disabled offline like every other write. Details of loaded flights stay viewable offline from the stored snapshot.
- No external service is called and no API key is needed: distance and the duration estimate are computed in the browser, the Flightradar24 link is built from the flight number. There is no aircraft photo.
- A route still exists at most once per user (unchanged `409`). `POST /api/flights` is unchanged.
- `PATCH /api/flights/{id}` accepts flight number, date, duration and aircraft type besides the visibility; every field becomes optional there. `GET /api/flights` returns the fields that are set.
- The public travel page does not expose any of the new fields; it keeps showing the route only.

## Capabilities

### New Capabilities

- `flight-details`: the optional flight number step after adding a flight, the manually filled details (date, duration, aircraft type), their normalisation, validation, editing and privacy, and the details view with distance, duration estimate and the Flightradar24 link.

### Modified Capabilities

None (`openspec/specs/` has no capabilities yet).

## Impact

- API contract: `api/components/schemas/flight.yaml` (`Flight`, `FlightUpdate`), `api/paths/flights_{id}.yaml`; regenerated `backend/internal/adapter/http/api/api.gen.go` and `frontend/generated/schema.gen.ts`.
- Backend: `domain/flight.go` (fields, flight number normalisation), `TravelService` (update), `FlightRepo` (columns, partial update), handlers (mapping, public view without the new fields), `001_init/02_travel.sql` (four nullable columns). Tests in the domain, service, repository and router packages.
- Frontend: `FlightPanel.vue` (number step, expandable details with form), the store (`Flight` view model, `updateFlight`), new helpers for flight numbers, the duration estimate and the Flightradar24 link, a bundled list of aircraft types. The offline snapshot picks up the new fields without a format change.
- No new dependency, no environment variable, no CSP change (the link is a navigation, not a fetch).
