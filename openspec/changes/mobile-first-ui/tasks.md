# Tasks

## 1. Foundation: viewport, tokens, fonts, icons

- [x] 1.1 `index.html`: add `viewport-fit=cover`, `theme-color` (light and dark), `apple-touch-icon` and `apple-mobile-web-app-capable`; verify in the built HTML (`npm run build`, `dist/index.html`)
- [x] 1.2 `style.css`: add breakpoint convention (base = phone, `min-width: 600px`, `min-width: 1024px`), spacing/radius tokens, `--touch: 44px`, `--appbar-h`, safe-area tokens, and the 16px input rule with the `(pointer: fine)` reduction; verify `npm run type-check` and that an input at 375px emulation computes `font-size: 16px`
- [x] 1.3 Replace the Google Fonts `@import` with `@fontsource/dm-sans` and `@fontsource/dm-serif-display` imported in `main.ts`, remove `fonts.googleapis.com`/`fonts.gstatic.com` from the CSP in `vite.config.ts`; verify in `npm run preview` that no request goes to Google and the fonts render
- [x] 1.4 Add `@vite-pwa/assets-generator` and `pwa-assets.config.ts` (preset `minimal-2023`, source `public/favicon.svg`), generate and commit the icons, list `pwa-64x64`, `pwa-192x192`, `pwa-512x512` and the maskable icon in the manifest; verify Chrome DevTools > Application > Manifest shows no icon errors and reports the app as installable
- [x] 1.5 Add `useLayout` composable (`phone | tablet | desktop`, `coarsePointer`) based on `matchMedia` with listeners cleaned up; verify by logging the value while resizing across 599/600 and 1023/1024px

## 2. App bar and map wrapper

- [x] 2.1 `App.vue`: merge top bar and account actions into one app bar (phone/tablet full width with safe-area padding, desktop keeps the centred pill and right-hand actions), logo icon on phones, controls at `--touch` height, remove the 640px special case; verify at 320, 360, 768 and 1280px that nothing overlaps and the admin user sees all four actions
- [x] 2.2 `.wrapper` sizing with `100vh` fallback and `100dvh`; verify in mobile emulation that the map fills the screen and no control sits behind the browser toolbar
- [x] 2.3 `WorldMap.vue`: move stats and MapLibre attribution/controls above the collapsed sheet on phones and inside safe areas; verify at 375×667 and 667×375

## 3. Sheet component and panels

- [x] 3.1 Create `components/ui/AppSheet.vue` (phone bottom sheet with handle drag and tap toggle, collapsed/expanded snap states, inner scrolling with `overscroll-behavior: contain`; tablet side sheet; desktop floating panel with left/right placement); verify drag, tap and scroll by hand in touch emulation at 375px and that content scroll does not move the map
- [x] 3.2 Port `CountryPanel.vue` into `AppSheet` (remove own positioning, list fills sheet height instead of `max-height: 50vh`, larger touch rows, toast above the sheet); verify search, tags and list selection at 375, 768 and 1280px
- [x] 3.3 Port `CountryDetailModal.vue` into the same sheet (opens expanded on phones, close returns to the list, `.close` and star buttons with 44px hit area, segmented controls full width); verify editing and saving a country at 375px with the keyboard open
- [x] 3.4 Port `FlightPanel.vue` into `AppSheet` (tap on a row toggles `highlightFlight`, hover kept for fine pointers, 44px hit area for visibility and remove, list grows with the sheet); verify on touch emulation that tapping highlights the flight on the map
- [x] 3.5 `AirportSelect.vue`: dropdown stays inside the viewport and above the keyboard on phones, 44px option rows; verify picking an airport at 320px
- [x] 3.6 `MapProvider`/`MapLibreProvider`: accept the occupied insets from `WorldMap` and use them as `padding` for `fitBounds`/`easeTo` instead of the fixed left padding; verify that selecting a country centres it in the visible map area on phone (above the sheet), tablet (beside the side sheet) and desktop
- [x] 3.7 Verify orientation change with an open sheet and unsaved country edits keeps state and input (rotate in DevTools between 390×844 and 844×390)

## 4. Pages and dialogs

- [x] 4.1 `PageWrapper.vue`: sticky header with safe-area padding and 44px back button, `main` padding 1rem on phones, centred 720px column from tablet; verify the header stays visible while scrolling the account page at 375px
- [x] 4.2 `panel.css` and `TabBar.vue`: rows stack on phones and sit side by side from 600px, buttons at touch height, tab bar scrolls horizontally on its own; verify at 320px that the page has no horizontal scrollbar (`document.documentElement.scrollWidth === innerWidth`)
- [x] 4.3 Account and admin pages (`ProfilePage`, `ProfileHeader`, `PasswordPage`, `AuthPage`, `AdminPage`, `OidcSettings`, `TripPage`): remove fixed widths that overflow, full-width forms on phones; verify each route at 320 and 768px with the horizontal-scroll check from 4.2
- [x] 4.4 `ConfirmModal.vue`: `max-height: calc(100dvh - 2rem)`, inner scroll, full-width stacked buttons on phones; verify removing a flight at 320×568
- [x] 4.5 Run Prettier on all changed frontend files and `npm run type-check`; both succeed

## 5. Offline session and read-only mode

- [x] 5.1 `api/client.ts`: `openapi-fetch` middleware that reports network errors and an 8s request timeout to a connection callback; verify by stopping the backend while `npm run dev` runs and seeing the store switch to offline
- [x] 5.2 Store: `connection` state from `navigator.onLine`, `online`/`offline` events and the middleware; `canWrite` computed; `online` event triggers `load()`; verify toggling DevTools "Offline" flips the state both ways
- [x] 5.3 Store: snapshot write after a successful `load()` and after `loadCountryDetails`; on network error apply the snapshot and set `authState = 'ready'`; `clear()` removes the snapshot; a snapshot for another username is replaced; verify reload with DevTools "Offline" shows profile, countries, flights and opened details, and that after logout the snapshot key is gone
- [x] 5.4 Store: every writing action (country, flight, profile, settings, password, account deletion, login, register, logout) returns "This needs a connection" without a request when `!canWrite`; a network error during a write rolls back and sets `offline`; verify by attempting each write offline and checking the Network tab shows no request
- [x] 5.5 UI: `OfflineBanner` in the app bar area, disabled controls with the shared hint in panels, sheets, pages and the log-in page, "not available offline" text for details that were never loaded; verify each screen offline at 375px
- [x] 5.6 Verify the 401 path: with a snapshot present, delete the session row (or wait for expiry in a test DB), go online and confirm the app logs out, removes the snapshot and shows the log-in page
- [x] 5.7 Document offline behaviour, read-only mode and the iOS storage limitation in `README.md`; verify the README renders and links resolve

## 6. Service worker caching

- [x] 6.1 `vite.config.ts`: raise `workbox.maximumFileSizeToCacheInBytes` so all bundles are precached, keep `navigateFallbackDenylist` for `/api/`; verify `npm run build` prints no "will not be precached" warning
- [ ] 6.2 `runtimeCaching` for `https://demotiles.maplibre.org/` (StaleWhileRevalidate for style, tilejson, glyphs and sprites; CacheFirst with expiration for tiles); verify in `npm run preview`: load the map online, switch DevTools to "Offline", reload, the world map renders with coloured countries and flights
- [ ] 6.3 Verify an app update: build twice with a visible change, open the old version, reload online, the new version is active without clearing data

## 7. Integration checks

- [ ] 7.1 With `npm run build && npm run preview` behind HTTPS (or Chrome on localhost), install the PWA, use it online, go offline, restart it: the user is still logged in and sees their data read-only
- [ ] 7.2 Run the full flow (log in, add a flight, edit a country, account page, admin page, log out) at 320×568, 375×812, 390×844 landscape, 768×1024, 1024×1366 and 1280×800; no overlap, no horizontal scroll, all actions reachable by touch
- [ ] 7.3 Test on a real Android phone (Chrome) and an iPhone/iPad (Safari, added to home screen) if available: install, safe areas, keyboard, offline start
