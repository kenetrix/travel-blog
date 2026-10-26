# Proposal

## Why

The UI was built for a desktop window: the country and flight panels float as fixed 240–340px boxes over the map, the top bar and account actions are absolutely positioned and collide on narrow screens (patched by a single `max-width: 640px` rule), flight highlighting needs a mouse hover, and `100vh` breaks on mobile browsers with dynamic toolbars. The app is installable as a PWA but is mostly used on the go, where the network is unreliable: offline, the first `GET /api/user` fails, the store stays in `loading`, the user sees an empty map and effectively loses their session view even though the 30-day session cookie is still valid.

## What Changes

- Rework all styles mobile first: base styles target a phone in portrait; `min-width` media queries add the tablet and desktop layouts. Supported range: 320px wide phones up to large tablets in both orientations, desktop keeps working.
- Phones: the country panel, flight panel and country details open as a bottom sheet over the map (collapsed / expanded), the top bar becomes a single compact app bar with the view switch and the account actions, full-page routes (account, admin, login, password, trip) use the whole screen with a sticky header.
- Tablets: panels become a side sheet next to the map; pages use a centered, wider content column.
- Touch: all interactive controls reach a 44×44px touch target, hover-only behaviour (flight highlight, remove/visibility buttons) gets a tap equivalent, form inputs use at least 16px font size so iOS does not zoom.
- Viewport: `viewport-fit=cover`, safe-area insets for notches and home indicators, dynamic viewport units (`dvh`) instead of `100vh`.
- PWA kept and completed: the manifest references `pwa-192x192.png` and `pwa-512x512.png`, which do not exist in `frontend/public/`; add them plus a maskable icon and an Apple touch icon, and set `theme-color` in `index.html`.
- Offline session: the last loaded account and travel data are kept in the service worker cache. When the device is offline (or the server is unreachable) the app starts with that data and the user stays logged in; only an actual `401` from the server ends the session. Logging out or a `401` purges the cached data.
- Offline mode is read-only: an offline banner is shown and actions that need the server (saving countries, adding/removing flights, account and admin changes, login) are disabled with a hint. They are not queued.
- The map base style and tiles that were already viewed, and the web fonts, are available offline (fonts are self-hosted instead of loaded from Google Fonts).

## Capabilities

### New Capabilities

- `responsive-layout`: mobile-first layout of the map, panels, sheets and pages across phone, tablet and desktop sizes, including touch targets, safe areas and orientation changes.
- `offline-access`: installable PWA, offline start with the cached session and travel data, read-only behaviour while offline, and cache purge on logout or session expiry.

### Modified Capabilities

None (`openspec/specs/` has no capabilities yet).

## Impact

- Frontend only: `App.vue`, `style.css`, `index.html`, `vite.config.ts` (Workbox runtime caching, CSP), the store (`store.ts`: offline state, cache purge), all components under `components/` (panels, `CountryDetailModal`, `ConfirmModal`, `PageWrapper`, `TabBar`, `panel.css`, account and admin pages, `WorldMap`).
- New static assets in `frontend/public/` (PWA icons).
- New dependency: self-hosted fonts (`@fontsource/dm-sans`, `@fontsource/dm-serif-display`); the Google Fonts entries leave the CSP.
- Backend and API: no change. The session stays a server-side session with an HttpOnly cookie (30 days); the client never stores a token.
- Privacy: account and travel data are stored in the browser's Cache Storage until logout or a `401`.
