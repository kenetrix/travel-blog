# Spec Delta

## Purpose

Lets a user record a trip as an ordered series of places they visited, inside one country or across several, starting from the details of a country and editing the stops directly on the map.

## ADDED Requirements

### Requirement: Trip with ordered stops

A trip SHALL have a title (1 to 120 characters after trimming), optional start and end dates, notes, a visibility and an ordered list of stops. A stop SHALL have a name (1 to 120 characters after trimming), coordinates, the ISO 3166 alpha-3 code of its country, an optional arrival date and notes. A trip MAY have no stops yet.

#### Scenario: Trip saved
- **WHEN** the user saves a trip titled "Japan 2025" with the stops Tokyo, Kyoto and Osaka in that order
- **THEN** the trip is stored and loading it again returns the three stops in the same order

#### Scenario: Missing title
- **WHEN** the user tries to save a trip with an empty title
- **THEN** the editor shows a hint and does not save, and the server responds `400` with the code `title_required` if it receives such a trip

#### Scenario: End before start
- **WHEN** a trip's end date is before its start date
- **THEN** the editor shows a hint and does not save, and the server responds `400` with the code `invalid_date_range`

### Requirement: Trip countries derived from stops

A trip's countries SHALL be the distinct countries of its stops in the order they are first reached. A trip has no country field of its own; the same trip SHALL appear for every country it has a stop in.

#### Scenario: Domestic trip
- **WHEN** a trip has the stops Munich, Nuremberg and Berlin
- **THEN** its countries are `DEU` only

#### Scenario: Trip across borders
- **WHEN** a trip has the stops Vienna, Bratislava, Budapest and back to Vienna
- **THEN** its countries are `AUT`, `SVK`, `HUN` in that order

### Requirement: Start a trip from the country details

The trips section of the country details SHALL offer a "Start a new trip here" link. It SHALL open the trip editor for a new trip with the map focused on that country. Nothing is stored until the user saves.

#### Scenario: Start a trip
- **WHEN** the user opens the details of Italy and follows "Start a new trip here"
- **THEN** the trip editor opens for a new, empty trip and the map is fitted to Italy

#### Scenario: Trip continues abroad
- **WHEN** a trip started from Italy gets a stop in Rome and then one in Nice
- **THEN** both stops are kept and the trip's countries are `ITA` and `FRA`

#### Scenario: Leave without saving
- **WHEN** the user has added stops to a new trip and leaves the editor
- **THEN** the app asks for confirmation before discarding the unsaved trip

### Requirement: Trips listed per country

The trips section of the country details SHALL list the user's trips that have at least one stop in that country, most recent start date first and undated trips last, each with its title, dates, number of stops and the other countries it visits. Selecting a trip SHALL open its summary page.

#### Scenario: Multi-country trip in each country
- **WHEN** the user has a trip with stops in Austria and Hungary
- **THEN** the trip is listed in the details of Austria and of Hungary, and in Austria's list it names Hungary as another country

#### Scenario: No trips
- **WHEN** the user has no trip with a stop in the country
- **THEN** the section shows a short note and the "Start a new trip here" link

### Requirement: Trip summary page

Each saved trip SHALL have a summary page showing its title, dates and length in days, its countries in visiting order, the number of stops, the total straight-line distance between consecutive stops, the stops in order with country, arrival date and notes, and the trip notes. The page SHALL offer "Edit", which opens the trip editor, and "Delete".

#### Scenario: Open the summary
- **WHEN** the user selects the trip "Balkans 2024" in the details of Croatia
- **THEN** the summary page of that trip opens and lists its stops in visiting order with their countries

#### Scenario: Length and distance
- **WHEN** a trip runs from 2024-05-01 to 2024-05-14 and has the stops Zagreb, Split and Dubrovnik
- **THEN** the summary shows 14 days and the sum of the distances Zagreb–Split and Split–Dubrovnik

#### Scenario: Missing details
- **WHEN** a trip has no dates, no notes or fewer than two stops
- **THEN** the summary leaves out the length, the notes or the distance instead of showing empty values

#### Scenario: Edit from the summary
- **WHEN** the user presses "Edit" on the summary page
- **THEN** the trip editor opens over the map with the trip's stops shown

#### Scenario: Unknown trip
- **WHEN** the user opens the summary URL of a trip that does not exist or was deleted
- **THEN** the app shows that the trip was not found and offers the way back to the map

### Requirement: Add a stop on the map

A place SHALL be chosen with a pin icon on the map, followed by a confirmation ("Add stop") that appends the stop at the end of the list with the pin's coordinates and the country under the pin tip. On devices with a fine pointer, clicking the map SHALL set the pin at the clicked point; clicking again moves it. The pin is shown only while the trip editor is open.

#### Scenario: Pin set by click
- **WHEN** the user clicks a point in southern Spain on a desktop and confirms "Add stop"
- **THEN** a stop is appended with the country `ESP` and the clicked coordinates

#### Scenario: Pin moved before confirming
- **WHEN** the user clicks one point, then another, and confirms
- **THEN** only one stop is added, at the second point

#### Scenario: Pin on the sea
- **WHEN** the pin tip is on water or outside any country
- **THEN** "Add stop" is disabled and the editor shows that stops must be placed on land

### Requirement: Place a stop on touch devices

On devices with a coarse pointer, "Add stop" SHALL show the pin fixed at the centre of the visible map area (above the sheet) and collapse the sheet. The user moves the map underneath the pin; the pin tip marks the place. "Add stop here" SHALL confirm and "Cancel" SHALL leave without adding a stop. Taps on the map SHALL not set the pin in this mode.

#### Scenario: Pan to place
- **WHEN** a user on a phone presses "Add stop", pans the map until the pin points at Porto and presses "Add stop here"
- **THEN** a stop is appended with the coordinates under the pin tip and the country `PRT`, and the sheet shows the new stop

#### Scenario: Cancel placing
- **WHEN** the user presses "Cancel" while the centre pin is shown
- **THEN** no stop is added and the pin disappears

### Requirement: Place names on the map

While the trip editor is open, the map SHALL render place names (cities and towns) from a zoom level on which single places can be told apart, larger places first. Place data SHALL come from the app's own origin without a key and SHALL stay available offline for areas already viewed.

#### Scenario: Zoomed out
- **WHEN** the editor is open and the map shows a whole continent
- **THEN** no place names are rendered besides the country names

#### Scenario: Zoomed in
- **WHEN** the user zooms into a region at or beyond the place-name zoom level
- **THEN** the names of the larger cities appear, and more towns appear as the user zooms further without labels overlapping

#### Scenario: Editor closed
- **WHEN** the trip editor is not open
- **THEN** the map renders no place names

### Requirement: Stop name suggested from the nearest place

When the pin is set or the map stops moving under the centre pin, the editor SHALL show the name of the nearest known place within 15 km of the pin tip as the suggested stop name, editable before and after confirming. On a fine pointer, clicking a rendered place name SHALL set the pin exactly on that place. Without a place nearby the name stays empty.

#### Scenario: Suggested name
- **WHEN** the pin is set a few hundred metres from the centre of Granada
- **THEN** the editor suggests "Granada" and the confirmed stop is named "Granada"

#### Scenario: Click a place name
- **WHEN** the user clicks the rendered label "Sevilla" on a desktop
- **THEN** the pin is set on Sevilla's coordinates and "Sevilla" is suggested

#### Scenario: Remote place
- **WHEN** the pin is set in a desert with no known place within 15 km
- **THEN** the stop is added without a name and its name input is focused

#### Scenario: Name required
- **WHEN** a stop's name is left empty
- **THEN** the trip cannot be saved until the stop is named or removed

### Requirement: Edit, reorder and remove stops

In the trip editor the user SHALL be able to rename a stop, set or clear its arrival date and notes, move it up or down in the order, drag it to another position in the list on devices with a fine pointer, and remove it. "Move" on a stop SHALL place it again with the pin (click on a fine pointer, centre pin on a coarse pointer), starting at its current position; its country SHALL follow the new position and its name is kept.

#### Scenario: Reorder
- **WHEN** the stops are Tokyo, Osaka, Kyoto and the user moves Kyoto up
- **THEN** the order is Tokyo, Kyoto, Osaka, and the map's route line and marker numbers follow

#### Scenario: Remove a stop
- **WHEN** the user removes a stop
- **THEN** it disappears from the list and the map; it is deleted from the server when the trip is saved

#### Scenario: Marker moved into another country
- **WHEN** the user moves the stop "Strasbourg" with the pin across the Rhine to Kehl and confirms
- **THEN** the stop's coordinates and its country `DEU` are updated and it is still named "Strasbourg"

### Requirement: Trip route on the map

While a trip is open in the editor, the map SHALL show each stop as a marker numbered by its position and a line connecting the stops in visiting order, and SHALL fit the view to the stops when an existing trip is opened. When the editor closes, the trip's markers and line SHALL be removed.

#### Scenario: Open an existing trip
- **WHEN** the user opens a trip with stops in Portugal and Spain
- **THEN** the map fits both countries' stops and shows numbered markers joined by a line

#### Scenario: Close the editor
- **WHEN** the user leaves the editor
- **THEN** the markers and the route line of the trip are gone

### Requirement: Save a trip with its stops

Saving SHALL store the trip together with its complete, ordered list of stops in one request. Stops that already exist SHALL keep their identity; stops missing from the list SHALL be deleted together with their photos. A failed save SHALL leave the stored trip unchanged.

#### Scenario: Save changes
- **WHEN** the user renames one stop, removes another, adds a new one and saves
- **THEN** the stored trip has exactly the stops shown in the editor, in that order, the renamed stop keeps its id, and the trip's summary page opens

#### Scenario: Invalid stop
- **WHEN** the server receives a stop with a malformed country code (not three uppercase letters) or coordinates out of range
- **THEN** it responds `400` and stores nothing of that save

#### Scenario: Too many stops
- **WHEN** a trip would have more than 200 stops
- **THEN** the editor does not allow adding another stop and the server responds `400` with the code `too_many_stops`

### Requirement: Delete a trip

The user SHALL be able to delete a saved trip from its summary page or the editor after confirming. Deleting a trip SHALL delete its stops and the photos of those stops, including the stored files.

#### Scenario: Delete
- **WHEN** the user deletes a trip and confirms
- **THEN** the trip, its stops and their photos are gone, the app returns to the map, and the trip no longer appears in any country's list

#### Scenario: Unknown trip
- **WHEN** the server is asked to delete or update a trip that does not exist or belongs to another user
- **THEN** it responds `404` with the code `trip_not_found`

### Requirement: Trips are private to their owner

Trips SHALL only be readable and writable by their owner. A trip's visibility SHALL default to the user's default visibility and can be changed in the editor; this change SHALL NOT expose trips on the public travel page.

#### Scenario: Another user's trip
- **WHEN** a logged-in user requests a trip id of another user
- **THEN** the server responds `404`

#### Scenario: Public page unchanged
- **WHEN** a visitor opens the public travel page of a user with public trips
- **THEN** the page shows no trips

### Requirement: Trips offline

Trips and their stops SHALL be part of the stored offline snapshot. Offline, the country details SHALL still list trips and the summary page of a trip opened before SHALL still be shown; starting, editing and deleting a trip SHALL be disabled with the offline hint.

#### Scenario: View offline
- **WHEN** the app is offline and the user opens a trip from the country details
- **THEN** the summary page with the trip's stops is shown and "Edit" and "Delete" are disabled

#### Scenario: Start offline
- **WHEN** the app is offline
- **THEN** the "Start a new trip here" link is disabled with the offline hint
