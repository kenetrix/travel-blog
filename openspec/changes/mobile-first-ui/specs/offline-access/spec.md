# Spec Delta

## Purpose
The application stays installable as a PWA and remains usable without a network: a logged-in user keeps their session and sees their last loaded travel data while offline.

## ADDED Requirements

### Requirement: Installable PWA
The application SHALL remain installable as a standalone PWA on Android, iOS/iPadOS and desktop browsers, with a valid web app manifest, icons in 192px and 512px including a maskable variant, an Apple touch icon and a theme colour.

#### Scenario: Install on Android
- **WHEN** the app is opened in Chrome on Android over HTTPS
- **THEN** the browser offers to install it, and the installed app starts in standalone mode with the app icon and theme colour

#### Scenario: Add to home screen on iOS
- **WHEN** the user adds the app to the home screen in Safari
- **THEN** the home screen shows the app icon and the app opens without browser UI

### Requirement: App shell available offline
The application shell (HTML, scripts, styles, fonts, icons and static data) SHALL be served from the device after the first visit, so the app starts without a network.

#### Scenario: Start without network
- **WHEN** the installed app is started in airplane mode after it was used online before
- **THEN** the app loads and shows the map with its interface instead of a browser error page

#### Scenario: New version
- **WHEN** a new version is deployed and the app is opened online
- **THEN** the new version is installed and used without the user clearing any data

### Requirement: Session survives offline
When the server cannot be reached, the client SHALL keep a previously logged-in user logged in and show the account and travel data of their last successful load. Only a `401` response from the server SHALL end the session on the client.

#### Scenario: Offline start of a logged-in user
- **WHEN** a user who was logged in opens the app without network
- **THEN** they see their profile in the app bar and their visited countries, wish list and flights on the map and in the panels, and they are not sent to the log-in page

#### Scenario: Server unreachable while online
- **WHEN** the device has a network but the server does not answer or answers with a network error
- **THEN** the app behaves as offline and keeps the user logged in

#### Scenario: Back online with a valid session
- **WHEN** the network returns
- **THEN** the app reloads the account and travel data from the server and leaves the offline state without a new log-in

#### Scenario: Back online with an expired session
- **WHEN** the network returns and the server answers `401`
- **THEN** the user is logged out, the cached data is removed and the log-in page is shown

#### Scenario: Offline without previous login
- **WHEN** a visitor who was never logged in on this device opens the app offline
- **THEN** the bare map is shown with the offline indicator and log-in is not available

### Requirement: Offline details of viewed countries
Country details that were opened while online SHALL be shown offline from the last loaded copy. Details that were never loaded SHALL show a message that they are not available offline.

#### Scenario: Previously opened country
- **WHEN** the user opens a country offline whose details they opened online before
- **THEN** its notes, food, sightseeing, rating and tags are shown read-only

#### Scenario: Never opened country
- **WHEN** the user opens a visited country offline whose details were never loaded
- **THEN** rating and tags are shown and the long texts show that they are not available offline

### Requirement: Read-only while offline
While offline the application SHALL show a visible offline indicator and SHALL disable every action that needs the server (saving or deleting countries, adding, changing or removing flights, account, password and administration changes, log-in, registration and log-out), with a hint why. Changes SHALL NOT be queued for later.

#### Scenario: Try to add a flight offline
- **WHEN** the user is offline and opens the flight panel
- **THEN** the "Add route" button and the visibility and remove buttons are disabled and a hint says that changes need a connection

#### Scenario: Losing the connection while editing
- **WHEN** the connection drops while the country details are open with unsaved changes
- **THEN** the input is kept, saving is disabled until the connection returns, and nothing is lost

#### Scenario: Save fails because of the network
- **WHEN** a save request fails with a network error although the device reported being online
- **THEN** the change is rolled back as for any failed save, the user is told it needs a connection, and the app switches to the offline state

### Requirement: Cached personal data is removed on logout
Account and travel data stored on the device for offline use SHALL be removed when the user logs out, deletes their account, or the server answers `401`, and SHALL never be shown to a different user.

#### Scenario: Logout on a shared device
- **WHEN** a user logs out and the device then goes offline and the app is opened
- **THEN** no data of that user is shown and the app shows the visitor state

#### Scenario: Different user logs in
- **WHEN** user A logged out and user B logs in on the same device
- **THEN** only user B's data is shown, online and later offline

### Requirement: Offline map
The map base style and the map tiles that were already displayed SHALL be available offline, so the world map and the user's countries and flights render without a network at the zoom levels viewed before.

#### Scenario: World view offline
- **WHEN** the app is opened offline after the world map was shown online before
- **THEN** the world map is drawn with the visited and wish-list countries coloured and the flights drawn
