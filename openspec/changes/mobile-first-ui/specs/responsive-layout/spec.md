# Spec Delta

## Purpose
The application is usable on any smartphone and tablet, in portrait and landscape, with the phone layout as the baseline and larger layouts added for tablets and desktops.

## ADDED Requirements

### Requirement: Supported screen sizes
The UI SHALL be fully usable from 320px to at least 1366px viewport width in portrait and landscape. No page SHALL scroll horizontally, and no content or control SHALL be cut off or covered by another fixed element.

#### Scenario: Smallest phone
- **WHEN** the app is opened at 320×568px
- **THEN** the app bar, the map and the open panel are visible, every control can be reached, and the page has no horizontal scrollbar

#### Scenario: Orientation change
- **WHEN** a phone or tablet is rotated while a panel or page is open
- **THEN** the layout adapts to the new size and the open panel, its scroll position and unsaved input stay as they were

### Requirement: Layout classes
The UI SHALL use three layout classes: phone (below 600px wide), tablet (600px to 1023px wide, or a landscape viewport at most 500px high and at least 480px wide) and desktop (1024px wide and more than 500px high). Phone is the base layout; tablet and desktop extend it.

#### Scenario: Phone width
- **WHEN** the viewport is narrower than 600px and not a short landscape viewport
- **THEN** panels open as a bottom sheet and pages use the full width

#### Scenario: Tablet width
- **WHEN** the viewport is between 600px and 1023px wide
- **THEN** panels open as a side sheet next to the visible map and page content is shown in a centered column

#### Scenario: Phone held sideways
- **WHEN** the viewport is in landscape, at most 500px high and at least 480px wide (e.g. 844×390 or 932×430)
- **THEN** the side sheet uses the full height on the left, a compact app bar sits beside it, and no content is cut off

#### Scenario: Portrait phone with the keyboard open
- **WHEN** a portrait phone narrower than 480px opens its on-screen keyboard
- **THEN** the phone layout stays in place

#### Scenario: Desktop width
- **WHEN** the viewport is 1024px or wider and more than 500px high
- **THEN** panels float over the map beside each other as before and page content is shown in a centered column

### Requirement: App bar
The app SHALL show one app bar at the top with the app name, the Countries/Flights switch (when logged in) and the account actions (log in, account, administration, log out). On phones the items SHALL fit into one row without overlapping; the app name MAY be shortened to an icon.

#### Scenario: Logged in on a phone
- **WHEN** an administrator is logged in at 360px width
- **THEN** the view switch, the account, administration and log-out buttons are all visible in the app bar and none of them overlaps another

#### Scenario: Visitor without session
- **WHEN** a visitor without a session opens the app on a phone
- **THEN** the app bar shows the app name and the log-in button only

### Requirement: Panels as sheets on phones
On phones the country panel, the flight panel and the country details SHALL open as a bottom sheet over the map. The sheet SHALL have a collapsed state that leaves most of the map visible and an expanded state that shows the full content, and the user SHALL switch between them by tapping or dragging its handle.

#### Scenario: Collapse to see the map
- **WHEN** the country panel is expanded on a phone and the user drags the handle down
- **THEN** the sheet collapses to a peek height showing its title and the map above it can be panned and tapped

#### Scenario: Selecting a country on the map
- **WHEN** the user taps a country on the map on a phone
- **THEN** the country details open in the sheet in the expanded state, and closing them returns to the country list

#### Scenario: Long content
- **WHEN** a sheet holds more content than fits on the screen
- **THEN** the sheet content scrolls inside the sheet and the map behind it does not move

### Requirement: Hide the side sheet
Outside the phone layout the country panel, the flight panel and the country details SHALL be hideable to show more of the map. A tab at the sheet's edge SHALL stay visible to show it again, and the app bar, stats and messages SHALL use the space it frees.

#### Scenario: Hide the country list on a phone held sideways
- **WHEN** the user taps the tab of the open country panel at 844x390
- **THEN** the panel slides off screen, only its tab stays at the left edge, and the app bar and stats move to the left

#### Scenario: Show it again
- **WHEN** the user taps the tab of the hidden panel
- **THEN** the panel slides back with its list and scroll position unchanged

#### Scenario: Selecting a country while hidden
- **WHEN** the panel is hidden and the user taps a country on the map
- **THEN** the panel shows again with the country details

#### Scenario: Default state
- **WHEN** the app opens outside the phone layout
- **THEN** the panel is shown

### Requirement: Full-screen pages
The log-in, password, account, administration and new-trip pages SHALL use the whole screen on phones, keep their header with the back action visible while the content scrolls, and show content in a centered column of limited width on tablets and desktops.

#### Scenario: Account page on a phone
- **WHEN** the account page is opened on a phone and the user scrolls down
- **THEN** the header with the back action stays visible and every form field spans the available width

#### Scenario: Tabs on a narrow screen
- **WHEN** a page with tabs is shown on a phone and the tabs do not fit
- **THEN** the tab bar scrolls horizontally on its own while the page itself does not

### Requirement: Dialogs fit the screen
Confirmation dialogs SHALL fit inside the visible viewport on every supported size, and on phones their buttons SHALL span the dialog width.

#### Scenario: Confirm on a phone
- **WHEN** the user removes a flight on a phone
- **THEN** the confirmation dialog is fully visible with both buttons reachable without scrolling

### Requirement: Touch operation
Every action SHALL be possible by touch alone. Interactive controls SHALL have a touch target of at least 44×44 CSS pixels, and behaviour that is triggered by mouse hover SHALL also be triggered by a tap.

#### Scenario: Highlight a flight without a mouse
- **WHEN** the user taps a flight in the flight list on a touch device
- **THEN** the flight is highlighted on the map, and tapping it again or tapping another flight changes the highlight

#### Scenario: Small icon buttons
- **WHEN** the visibility or remove button of a flight is measured on a phone
- **THEN** its tappable area is at least 44×44 CSS pixels

### Requirement: Readable inputs without zoom
Text inputs, selects and text areas SHALL use a font size of at least 16px on touch devices, so mobile browsers do not zoom the page when a field gets focus.

#### Scenario: Focus a search field on iOS
- **WHEN** the user taps the country search field in Safari on an iPhone
- **THEN** the on-screen keyboard opens and the page does not zoom in

### Requirement: Safe areas and dynamic viewport
The UI SHALL keep its controls out of display cutouts, rounded corners and the home indicator, and SHALL size full-height elements to the visible viewport, including when browser toolbars or the on-screen keyboard appear or disappear.

#### Scenario: Phone with a notch in landscape
- **WHEN** the installed app runs in landscape on a phone with a notch
- **THEN** the app bar and sheets keep clear of the notch and the home indicator

#### Scenario: Browser toolbar collapses
- **WHEN** the mobile browser hides or shows its address bar
- **THEN** the map fills the visible area and no control moves behind the toolbar
