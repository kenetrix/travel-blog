# Design

## Context

A flight is a row in `flights` (`001_init/02_travel.sql`) with `from_iata`, `to_iata`, `visibility`, `created_at` and `UNIQUE (user_id, from_iata, to_iata)`. The client resolves airports from the bundled `airports.json` and computes distances in `flights/distance.ts`. `PATCH /api/flights/{id}` only changes the visibility (`FlightUpdate.visibility` is required). The backend has no request validation middleware: the service validates every input and returns `ErrInvalid` (`400`). The public travel response reuses the `Flight` schema via `toAPIFlights`. The offline snapshot stores `Schemas['Flight'][]` as returned by the server. Migrations are tracked by count in SQLite's `user_version`, without checksums.

Decided with the user: no external service and no keys; the only lookup is a Flightradar24 link; date, duration and aircraft type are entered by hand.

## Goals / Non-Goals

**Goals:**
- Store an optional flight number, date, duration and aircraft type per flight, validated and normalised on the server and mirrored on the client for instant feedback.
- A two-step add flow and an expandable details view that work on a 320px phone inside the bottom sheet and offline (read only).

**Non-Goals:**
- Fetching real flight data, aircraft photos or uploads.
- Several flights on the same route, sorting the list by date, time of day or time zones.

## Decisions

### Columns in `001_init`, nullable
Add `flight_number TEXT` (canonical, e.g. `LH400`), `flown_on TEXT` (`YYYY-MM-DD`), `duration_minutes INTEGER` and `aircraft TEXT` to `flights`, all nullable; the unique route constraint stays. Per `CLAUDE.md`, `001_init` is edited in place. Because migrations are counted, not checksummed, the local `backend/data/travel-blog.db` is brought in line with four `ALTER TABLE flights ADD COLUMN` statements instead of being deleted; test databases are created fresh by the tests. In the domain the fields are plain values with zero meaning unknown; the repository maps zero ↔ `NULL`.

### `POST` unchanged, everything else via `PATCH`
The add flow creates the route first and saves the number in a second step, so `POST /api/flights` needs no new field. `FlightUpdate` gets optional `visibility`, `flightNumber`, `date`, `durationMinutes` and `aircraft`; absent means unchanged, `""` (strings) or `0` (duration) clears. Alternatives: `nullable: true` with merge-patch semantics would need oapi-codegen's `nullable` output option and a new dependency to tell absent from `null`; a separate sub-resource would duplicate routing. An update without any field is `400`. Request `date` is a pattern string (an empty string would not parse as `format: date` in the generated Go types); the response field is `format: date`. The repository replaces `SetVisibility` with `Update(user, id, domain.FlightUpdate)` (pointer fields) and builds the `SET` list from the present fields.

### Validation
- Flight number: `domain.NormalizeFlightNumber(s) (string, error)`: uppercase, drop spaces and hyphens, match designator + digits + optional letter, where a two-character designator must contain a letter and a three-character designator is three letters; leading zeros of the number are stripped (`0` alone is invalid). The three-letter form is tried only when the third character is a letter, so `LH400` is `LH` + `400` and `DLH400` is `DLH` + `400`.
- Date: `time.Parse("2006-01-02", …)`, year 1900–2100; a calendar date because a timestamp would invent a time and shift with the viewer's zone.
- Duration: 1–1440 minutes (the longest scheduled flights are under 20 h).
- Aircraft: trimmed free text, at most 60 characters; the dropdown is a client convenience, so the server does not restrict it to the list.

The frontend mirrors the flight number rules in `flights/flightNumber.ts` (normalise, format with a space after the designator, link) to show a hint before sending; the server normalises again and is the authority.

### Public view omits the new fields
`toAPIFlights` takes a flag so the public response carries only id, route, visibility and creation time. Exact date and flight number tell strangers when someone was where; the route alone is what public flights showed until now.

### Duration estimate
`estimateDurationMinutes(km) = km / 800 * 60 + 30`, rounded to 5 minutes, shown as `≈ 11 h 40 min (estimate)`. 800 km/h is a typical average jet speed over a whole route; 30 minutes covers taxi, climb and approach. Lives next to `distanceKm` in `flights/distance.ts`. The duration inputs are prefilled with the estimate, but a duration is only sent when the user changes them, so saving just a date does not turn the estimate into a stored duration.

### Flightradar24 link
`https://www.flightradar24.com/data/flights/<lowercase>`, `target="_blank" rel="noopener noreferrer"`. The page lists the number's recent and upcoming flights, not the stored date. No CSP change: CSP restricts fetches and embeds, not top-level navigation.

### Aircraft types bundled
`flights/aircraft.ts` exports about 30 common passenger types (Airbus A220 to A380, Boeing 737 to 787 variants, Embraer E-Jets, ATR 72, Dash 8, CRJ). The select shows them plus "Other", which reveals a text input; a stored value that is not in the list is shown as "Other" with the text filled in. A `<select>` instead of a `<datalist>` because the user asked for a dropdown and `<select>` opens the native picker on phones.

### UI flow in `FlightPanel.vue`
- With flights, the panel shows a head (title, count, total distance, "Add flight" pill) and the list; `adding` swaps the list for the From/To form with Cancel, and opening it expands the sheet and focuses From. Without flights, the form is the panel. The head is the first content, so the collapsed phone sheet's peek shows it with the button. Only the list scrolls (`flex: 0 1 auto; min-height: …; overflow-y: auto` inside the sheet's flex column), so the head stays in view; `AppSheet` is unchanged. A container query turns the pill into a round icon button when the head is narrower than 15rem (the 240px desktop panel).
- Add form state `step: 'route' | 'number'` plus the id of the flight just added. After `addFlight` succeeds, `step = 'number'`: the From/To inputs are replaced by "Flight number for BER → HND (optional)", the input is focused, with Save and Skip. Enter saves (empty Enter skips), Escape skips. Saving calls `updateFlight`, then returns to `step = 'route'` and expands the new flight's details.
- List rows: tapping a row toggles its details below (accordion, one open at a time, `openId`). The open flight stays highlighted on the map; on desktop, `mouseleave` restores the open flight's highlight instead of clearing it. The opened row is scrolled into view within the list.
- Details content: distance, duration (stored or estimate), and the flight number input with Save when there is no number. With a number: number (editable), Flightradar24 link, date, duration (hours and minutes inputs), aircraft select, and one Save button that is enabled when something changed and the values are valid. The form is a child component `FlightDetails.vue` to keep the panel readable.
- All inputs keep the 16px touch font from `panel.css`; the flight number input uses `autocapitalize="characters"`, `autocomplete="off"`, `enterkeyhint="done"`, the duration inputs `inputmode="numeric"`.

### Route selection and focus
The selected flight moves from `FlightPanel` into the store (`selectedFlightId`, `selectFlight`), so the map and the list share it; `selectFlight` also sets the highlight. The provider keeps the drawn arc per route id: `onFlightClick` hit-tests a 24px square around a click against the line and label layers (a 1.5px dashed line is too thin to hit by touch) and reports the route id or `null`; `focusFlight` fits the arc's bounds with the same inset logic as countries (`fitInsets`, shared with `fitCountry`), plus a 32px margin so the airports do not touch the edges, max zoom 7. A `flightFocus` counter in the store triggers the fit; it is bumped on a new selection and by the panel when the phone sheet opens or collapses, since the free area changes. On a collapsed phone sheet a map tap keeps the sheet collapsed so the route stays visible, and the sheet's head names the selected route; a tap on it expands the sheet. Alternatives: expanding the sheet on a map tap would cover the route that was just tapped; an invisible wide hit layer would duplicate the line source.

### Store
`Flight` view model gains `flightNumber?`, `date?`, `durationMinutes?`, `aircraft?`. New `updateFlight(id, patch)` applies the patch optimistically and restores the previous record on error, like `setFlightVisibility`, which becomes a thin wrapper around it. Because the view model is computed from `flightRecords`, a saved number shows the link and inputs at once. The snapshot type is `Schemas['Flight'][]`, so the new fields are stored without changing its key; old snapshots lack them.

## Risks / Trade-offs

- [The estimate is off for routes with strong winds or detours] → labelled as an estimate and replaceable by the real duration.
- [Flightradar24 shows current flights for the number, not the trip's date; older history is behind its paid plans] → the details show the stored date; the link is a lookup, not history.
- [Flight number rules reject rare codes] → the number is optional, so a rejected code never blocks adding the route.
- [The number step is lost if the user navigates away] → the flight is already saved; the number can be added later in the details.
