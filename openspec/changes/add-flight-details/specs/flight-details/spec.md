# Spec Delta

## Purpose

Lets a user record which flight a route was, fill in its date, duration and aircraft type, and look it up on a public flight tracker, without the app calling any external service.

## ADDED Requirements

### Requirement: Flight list first, add form on request

When the user has flights, the flight panel SHALL show a head with the number of flights, the total distance and an "Add flight" button, followed by the flight list; the From/To form opens only from that button. Without flights, the form is shown directly. The head stays visible while the list scrolls; on phones it is what the collapsed sheet shows.

#### Scenario: Open the form
- **WHEN** the user has flights and taps "Add flight" on the collapsed phone sheet
- **THEN** the sheet expands, the list is replaced by the From/To form and the From input is focused

#### Scenario: Cancel
- **WHEN** the user presses "Cancel" or Escape in the form
- **THEN** the form closes without adding a flight and the list is shown again

#### Scenario: First flight
- **WHEN** the user has no flights
- **THEN** the panel shows the From/To form without a "Cancel" button

#### Scenario: Offline
- **WHEN** the app is offline
- **THEN** the "Add flight" button is disabled with the offline hint

### Requirement: Flight number step after adding a route

After a route is added with Enter or the "Add route" button, the flight form SHALL show an optional flight number input for that flight in the same view, focused, with "Save" and "Skip". Skip, Escape, or Enter on an empty input SHALL close the step without changing the flight and return to the flight list.

#### Scenario: Number entered
- **WHEN** the user adds BER → HND, then types `nh 204` into the flight number input and presses Enter
- **THEN** the flight number `NH204` is saved for that flight, the step closes and the flight's details open in the list

#### Scenario: Step skipped
- **WHEN** the user adds BER → HND and presses "Skip"
- **THEN** the flight stays without a number and the panel shows the flight list again

### Requirement: Flight number normalisation

The flight number SHALL be stored in a canonical form: the airline designator (two letters or digits, not both digits, or three letters) followed by one to four digits without leading zeros and an optional letter suffix, uppercase and without separators. Spaces and hyphens in the input are ignored. It is displayed with a space after the designator.

#### Scenario: Normalised input
- **WHEN** the user enters `lh0400`, `LH 400` or `lh-400`
- **THEN** the flight number is stored as `LH400` and displayed as `LH 400`

#### Scenario: Three-letter designator
- **WHEN** the user enters `dlh 400`
- **THEN** the flight number is stored as `DLH400`

#### Scenario: Invalid flight number
- **WHEN** the user enters `400`, `L 400`, `LH` or `LH 40000`
- **THEN** the input shows a hint and cannot be saved, and the server responds `400` if it receives the value

### Requirement: Details of a flight with a number

The details of a flight with a flight number SHALL show the number, a link to the flight on Flightradar24, the distance, and inputs for date (native date picker), duration (hours and minutes) and aircraft type (dropdown of common types with "Other" as free text). The user fills them in and saves them together.

#### Scenario: Fill in the details
- **WHEN** the user opens the details of BER → HND `NH204`, picks 2026-03-14, enters 11 h 50 min, selects "Boeing 787-9" and saves
- **THEN** the date, duration and aircraft type are stored and shown when the details are opened again, also after a reload

#### Scenario: Other aircraft type
- **WHEN** the user selects "Other" and types `Ilyushin Il-96`
- **THEN** that text is stored as the aircraft type and shown in the details

#### Scenario: Invalid values
- **WHEN** a client sends a date that is not a real `YYYY-MM-DD` date, a duration outside 1–1440 minutes or an aircraft type longer than 60 characters
- **THEN** the server responds `400` and stores nothing

### Requirement: Duration estimate

Until a duration is entered, the details SHALL show an estimated duration derived from the distance, rounded to five minutes and labelled as an estimate; the duration input is prefilled with it. An entered duration replaces the estimate.

#### Scenario: Estimate shown
- **WHEN** the user opens the details of a flight without a stored duration
- **THEN** the duration shows as an estimate, e.g. `≈ 11 h 40 min (estimate)`

#### Scenario: Entered duration
- **WHEN** the flight has a stored duration of 710 minutes
- **THEN** the details show `11 h 50 min` without the estimate label

### Requirement: Details of a flight without a number

The details of a flight without a flight number SHALL show the route, the distance and the estimated duration, and offer the flight number input. Saving a number SHALL show the link and the detail inputs at once, without a page reload. Stored date, duration and aircraft type stay stored while the number is cleared and are shown again when a number is added.

#### Scenario: Number added later
- **WHEN** the user opens the details of BER → HND without a number, enters `NH 204` and saves
- **THEN** the details show the Flightradar24 link and the date, duration and aircraft inputs right away

#### Scenario: No number
- **WHEN** the user opens the details of a flight without a number
- **THEN** no tracker link and no date, duration or aircraft inputs are shown

### Requirement: Opening and closing details

Each flight in the flight list SHALL be expandable to show its details below it; only one flight is expanded at a time and the expanded flight is highlighted on the map.

#### Scenario: Open details
- **WHEN** the user taps or clicks a flight in the list
- **THEN** its details appear below it and the flight is highlighted on the map

#### Scenario: Close details
- **WHEN** the user taps the expanded flight again or expands another flight
- **THEN** the first flight's details close

### Requirement: Selecting a route on the map

In the flights view, a click or tap on a route line or its label (within about 12px) SHALL select that flight, as if it was picked in the list. A click away from all routes clears the selection. The pointer turns into a hand over a route.

#### Scenario: Route tapped on a phone
- **WHEN** the phone sheet is collapsed and the user taps the MUC → LIS route
- **THEN** the flight is selected and highlighted, the sheet stays collapsed and its head shows `MUC → LIS`, the flight number and distance with "tap for details"

#### Scenario: Open from the head
- **WHEN** the user then taps that head
- **THEN** the sheet expands with the flight's details open in the list

#### Scenario: Route clicked on a larger screen
- **WHEN** the user clicks a route on a tablet or desktop
- **THEN** the flight's details open in the panel and are scrolled into view

#### Scenario: Click on empty map
- **WHEN** a flight is selected and the user clicks the map away from all routes
- **THEN** the selection and its details close

### Requirement: Map focuses the selected route

When a flight is selected, from the list, the map, or after its number was saved, the map SHALL move and zoom so the whole route fits into the part of the map not covered by the panel or sheet, with some room around its ends, but not beyond the map's minimum zoom. On phones, the route is fitted again whenever the sheet opens or collapses.

#### Scenario: Selected in the list
- **WHEN** the user selects BER → HND in the list on a desktop
- **THEN** the map shows the whole route beside the panel

#### Scenario: Sheet collapsed on a phone
- **WHEN** a flight is selected and the user collapses the phone sheet
- **THEN** the map fits the route into the now larger free area

### Requirement: Editing details

The user SHALL be able to change or clear the flight number, date, duration and aircraft type of an existing flight. An update changes only the fields it contains and clears a field given as empty; an update without any field is rejected with `400`. Changes are shown at once and restored if the server rejects them.

#### Scenario: Clear the date
- **WHEN** the user clears the date of a flight and saves
- **THEN** the flight has no date afterwards and the other fields are unchanged

#### Scenario: Rejected update
- **WHEN** the server rejects a change (invalid value, flight not found, network error)
- **THEN** the previous values are shown again and an error message is displayed

### Requirement: Flightradar24 link

The Flightradar24 link SHALL be built from the flight number as `https://www.flightradar24.com/data/flights/<number in lowercase>` and open in a new browser tab without passing the referrer. The app SHALL NOT send any request to Flightradar24 or another flight-data service itself.

#### Scenario: Link for a flight number
- **WHEN** the user opens the details of a flight with the number `LH400`
- **THEN** the details show a link to `https://www.flightradar24.com/data/flights/lh400` that opens in a new tab

### Requirement: Offline behaviour of flight details

While offline the details of loaded flights SHALL remain viewable from the stored data, and the flight number step, the detail inputs and saving SHALL be disabled with the offline hint.

#### Scenario: Details offline
- **WHEN** the app starts offline with a stored snapshot containing a flight with a number, date, duration and aircraft type
- **THEN** the flight's details show all of them, and the inputs and the save button are disabled with the offline hint

### Requirement: Flight details stay private

The public travel page SHALL NOT expose the flight number, date, duration or aircraft type of a flight, even when the flight is public.

#### Scenario: Public profile
- **WHEN** a visitor opens the public travel data of a user whose public flight has details
- **THEN** the response contains the flight's route but none of `flightNumber`, `date`, `durationMinutes` and `aircraft`

### Requirement: Mobile-first input and details

The new inputs and controls SHALL follow the responsive layout: inputs use at least a 16px font size on touch devices, buttons, selects and links reach a 44×44px touch target, and the date uses the device's native date picker.

#### Scenario: Phone
- **WHEN** the flight panel is opened as a bottom sheet on a 375px wide phone
- **THEN** the flight number step, the expanded details and their inputs fit the width without horizontal scrolling and meet the touch target size
