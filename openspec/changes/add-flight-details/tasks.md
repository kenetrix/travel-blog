# Tasks

## 1. API contract

- [x] 1.1 In `api/components/schemas/flight.yaml` add optional `flightNumber`, `date` (`format: date`), `durationMinutes` and `aircraft` to `Flight`, and make every `FlightUpdate` field optional (`visibility`, `flightNumber`, `date`, `durationMinutes`, `aircraft`; `""` / `0` clears); update the summary and `400` description in `api/paths/flights_{id}.yaml`; verify `npm run generate:api` succeeds and both generated files change

## 2. Backend

- [x] 2.1 Add `flight_number`, `flown_on`, `duration_minutes` and `aircraft` (nullable) to `flights` in `backend/migrations/data/001_init/02_travel.sql` and add the same columns to the local `backend/data/travel-blog.db` with `ALTER TABLE`; verify the backend starts and lists flights
- [x] 2.2 Add the fields to `domain.Flight`, add `domain.FlightUpdate` and `domain.NormalizeFlightNumber` with a table test for valid, normalised and invalid numbers (`go test ./internal/domain/...`)
- [x] 2.3 Read and write the new columns in `FlightRepo` and replace `SetVisibility` with `Update(user, id, domain.FlightUpdate)`; extend the repository test for round trip, partial update and clearing
- [x] 2.4 Replace `TravelService.SetFlightVisibility` with `UpdateFlight` (flight number, date, duration and aircraft validation, empty update rejected); extend `travel_service_test.go` for valid, invalid, clearing and unchanged fields
- [x] 2.5 Map the fields in `UpdateFlight` and `toAPIFlights` and omit them from the public travel response; extend `router_test.go` (`TestTravelFlow`, `TestProblems`, `TestPublicTravel`) and verify `cd backend && go test ./...` passes

## 3. Frontend logic

- [x] 3.1 Add `flights/flightNumber.ts` (normalise, format, Flightradar24 link), `flights/aircraft.ts` (common types) and `estimateDurationMinutes` / `formatDuration` in `distance.ts`; verify with `npm --prefix frontend run type-check`
- [x] 3.2 Extend the store: `Flight` view model with the new fields, optimistic `updateFlight(id, patch)`, `setFlightVisibility` on top of it; verify type-check passes

## 4. Frontend UI

- [x] 4.1 Add the flight number step to the add form in `FlightPanel.vue` (focus, Save, Skip, Enter, Escape, invalid hint, opens the new flight's details after saving); verify in the browser that adding with and without a number works
- [x] 4.2 Add `FlightDetails.vue` and expandable list rows (one open at a time, highlight kept): distance and estimate, number input without a number; link, date, duration, aircraft select with "Other" and Save with a number; verify in the browser that a number added later shows the link and inputs at once and that saved values survive a reload
- [x] 4.3 Disable the number step, detail inputs and Save offline with the hint; verify with the browser offline that details from the snapshot are shown
- [x] 4.4 Run Prettier on the changed frontend files and verify `npm --prefix frontend run type-check` and `npm --prefix frontend run build` pass
- [x] 4.5 Show the list first with a head (count, total, "Add flight" button) and open the From/To form from that button (Cancel, Escape, From focused, sheet expanded; form directly without flights); verify on a 320px and 375px phone (collapsed peek shows the button), a tablet and a 1280px desktop
- [x] 4.6 Share the selected flight through the store, make routes on the map clickable (hit tolerance, hand cursor, empty click clears), fit the selected route into the free map area (also when the phone sheet opens or collapses) and name it in the collapsed phone sheet; verify on a phone, a tablet and a 1280px desktop

## 5. Integration

- [x] 5.1 Verify the whole flow in the browser at phone (375×812) and desktop size: add a route, enter a number, fill the details, edit and clear, add a number later, follow the link target, offline view, public data without the new fields
- [x] 5.2 Check `README.md` and `api/README.md` for flight descriptions that need the new fields; no setup change is expected (no keys, no environment variables)
