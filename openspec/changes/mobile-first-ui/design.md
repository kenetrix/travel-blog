# Design

## Context

- All layout lives in scoped `<style>` blocks. Panels (`CountryPanel`, `FlightPanel`, `CountryDetailModal`) position themselves absolutely (`top: 5.2rem`, `left/right: 1.2rem`, fixed widths 240px / 340px). The only breakpoint is `max-width: 640px` in `App.vue`, which moves the account actions below the top bar.
- `App.vue` renders the map, the top bar, the account actions and one panel depending on `view`; full-page routes render through `PageWrapper` on top (`z-index: 30`).
- `MapLibreProvider` fits country bounds with a fixed left padding for the panel (`MapLibreProvider.ts:528`).
- The store (`store.ts`) loads `/user`, `/countries` and `/flights` in `load()`. A `401` sets `authState = 'anonymous'`; any other failure only shows a toast and leaves `authState = 'loading'`. `App.vue` redirects to `login` only for `anonymous`.
- The session is a server-side session with an HttpOnly, SameSite=Lax cookie valid for 30 days (`SessionTTL`). It survives offline periods; the client cannot read or refresh it.
- `vite-plugin-pwa` runs in `generateSW` mode with `autoUpdate`; it precaches the build and denies `/api/` for the navigation fallback. There is no runtime caching. The manifest icons `pwa-192x192.png` and `pwa-512x512.png` are missing from `public/`.
- Fonts come from Google Fonts via `@import` in `App.vue`; map style, tiles and glyphs come from `demotiles.maplibre.org`. The production CSP is defined in `vite.config.ts`.
- The frontend has no unit-test setup; checks are `vue-tsc` and manual browser tests.

## Goals / Non-Goals

**Goals:**
- One mobile-first stylesheet structure with shared breakpoints and tokens instead of per-component magic numbers.
- One reusable sheet component that every map panel uses, so phone, tablet and desktop behaviour is defined in one place.
- Offline start with the last known data, driven by the store, so the UI knows exactly when it is offline.

**Non-Goals:**
- Offline editing or a sync queue (offline is read-only, see `specs/offline-access`).
- Full offline map coverage; only tiles already viewed are available.
- Native apps, push notifications, background sync.
- Any backend or API change.

## Decisions

### 1. Breakpoints and tokens in `style.css`
Base styles target phones. `min-width: 600px` adds tablet, `min-width: 1024px` adds desktop. Add CSS custom properties for spacing, radii, the app bar height, `--touch: 44px` and safe-area insets (`--safe-top: env(safe-area-inset-top)` etc.). Components use these tokens and the two media queries only.
Input font size: 16px by default, reduced to the current 0.82rem only inside `@media (pointer: fine)`, so touch devices never zoom on focus.
*Alternative:* a CSS framework (Tailwind). Rejected: rewrites every component and adds a build dependency for what two breakpoints solve.

### 2. `useLayout` composable for logic that CSS cannot express
A small composable based on `matchMedia` exposes `layout: 'phone' | 'tablet' | 'desktop'` and `coarsePointer`. Used where markup or behaviour differs: sheet mode, map padding, hover vs tap. Everything else stays pure CSS.

### 3. `AppSheet` component for all map panels
New `components/ui/AppSheet.vue` with a title slot, a content slot and a `v-model:expanded`.
- Phone: fixed bottom sheet, width 100%, rounded top corners, a drag handle. Two snap states: collapsed (handle + title, about 7rem plus the bottom safe area) and expanded (up to `100dvh` minus app bar). Dragging uses pointer events on the handle only, so content scrolling and map gestures are not hijacked; a tap on the handle toggles. Content scrolls inside with `overscroll-behavior: contain`.
- Tablet: side sheet docked left, `min(360px, 45vw)` wide, full height below the app bar, always expanded.
- Desktop: the current floating panel look (countries left, flights right, 240px / 340px).
`CountryPanel`, `FlightPanel` and `CountryDetailModal` drop their own positioning and render their content into `AppSheet`. Country details replace the list inside the same sheet (as today) and open expanded on phones.
*Alternative:* a separate phone-only component tree. Rejected: duplicates the panel logic.

### 4. Map follows the sheet
The visible map area changes with the sheet. `WorldMap` passes the occupied insets (bottom sheet height on phones, sheet width on tablets, panel width on desktop) to `MapProvider`, and `MapLibreProvider` uses them as `padding` for `fitBounds`/`easeTo` instead of the fixed left padding. Attribution and map controls move above the collapsed sheet on phones.

### 5. App bar
`App.vue` merges `.topbar` and `.useractions` into one flex app bar spanning the width on phones and tablets (padding includes `--safe-top`, `--safe-left`, `--safe-right`). On phones the logo text becomes an icon and the switch and icon buttons shrink to `--touch` height. On desktop it keeps the current centred pill plus right-hand actions. The 640px special case is removed.

### 6. Pages and dialogs
`PageWrapper` gets a sticky header with safe-area padding and a content column `max-width: 720px` centred from tablet up; on phones `main` has 1rem padding. `panel.css` rows (`.row`) stack vertically on phones and go side by side from 600px. `ConfirmModal` is fixed, `max-height: calc(100dvh - 2rem)`, and stacks full-width buttons on phones. `AuthPage` and `PasswordPage` keep `max-width: 360px` but with full width below that.

### 7. Touch instead of hover
Flight rows toggle the highlight on `click` in addition to `mouseenter/mouseleave` (hover stays for fine pointers). Icon buttons (`.visibility`, `.remove`, `.close`, stars) get a 44px hit area via padding or a pseudo-element without growing visually. Map hover cursor logic stays; country selection already uses `click`, which maps to tap.

### 8. Viewport
`index.html`: `viewport-fit=cover`, `<meta name="theme-color">` for light and dark, `apple-touch-icon`, `apple-mobile-web-app-capable`. `.wrapper` uses `height: 100vh; height: 100dvh` (fallback first) and `inset` with safe-area padding.

### 9. Offline data is a store snapshot, not an API cache in the service worker
After each successful `load()` the store writes a snapshot `{ username, account, countries, flights, details }` to `localStorage` (one key, JSON, a few hundred KB at most). Loaded country details are added to the snapshot as they arrive.
On start the store calls the API as today:
- `401` → clear state and snapshot, `anonymous`.
- success → apply, write snapshot, `online`.
- network error (fetch throws, or the request times out after 8s) → if a snapshot exists, apply it, `authState = 'ready'`, `connection = 'offline'`; otherwise stay a visitor with `connection = 'offline'`.
A new `connection: 'online' | 'offline'` state is set from `navigator.onLine`, the `online`/`offline` events and network errors seen by an `openapi-fetch` middleware. Going back online calls `load()` again, which either refreshes the data or ends in `401` and logout.
*Alternative:* Workbox `NetworkFirst` runtime caching for `GET /api/*`. Rejected: the store cannot tell a cached answer from a fresh one (needed for the offline indicator and read-only mode), the cache would hold responses outside the store's control, and it does not work in `npm run dev` where the service worker is off. The browser HTTP cache keeps handling ETag revalidation as today.
*Alternative:* IndexedDB. Rejected for now: the data is small and synchronous access keeps the start path simple; can be swapped behind the same two functions.

### 10. Read-only offline
The store exposes `canWrite = connection === 'online'`. Every writing store action returns the message "This needs a connection" without a request when `!canWrite`; components bind `:disabled="!store.canWrite"` and show a shared offline hint. An `OfflineBanner` in the app bar area shows the state. A write that fails with a network error rolls back as today and flips `connection` to `offline`.

### 11. Purge
`clear()` removes the snapshot. It already runs on logout, account deletion and `401`. A snapshot whose `username` differs from the account returned by the server is replaced, never merged.

### 12. Service worker caching for shell, fonts and map
- Precache stays as is; raise `workbox.maximumFileSizeToCacheInBytes` so the chunks with the bundled airport and country data are precached (default limit is 2 MiB).
- Fonts: self-host with `@fontsource/dm-sans` and `@fontsource/dm-serif-display` imported in `main.ts`; they are then part of the precache. Remove the Google Fonts `@import` and the Google entries from the CSP.
- Map: `runtimeCaching` for `https://demotiles.maplibre.org/` – `StaleWhileRevalidate` for `style.json`, `tiles.json`, glyphs and sprites; `CacheFirst` with an expiration limit (about 1000 entries, 30 days) for vector tiles.

### 13. Icons
Add `@vite-pwa/assets-generator` with a `pwa-assets.config.ts` (preset `minimal-2023`) generating `pwa-64x64.png`, `pwa-192x192.png`, `pwa-512x512.png`, `maskable-icon-512x512.png` and `apple-touch-icon-180x180.png` from `public/favicon.svg`. The generated files are committed; the manifest lists the maskable icon with `purpose: 'maskable'`.

## Risks / Trade-offs

- [iOS Safari deletes script-writable storage of sites not used for 7 days, unless installed] → Document that offline start is guaranteed for the installed app; the session cookie itself is unaffected.
- [Personal data in `localStorage` on a shared device] → Purged on logout, account deletion and `401`; no passwords or tokens are stored (the session cookie stays HttpOnly).
- [Session expires during a long offline period] → The user sees stale data until the next online `load()`, which then logs them out; acceptable for read-only data.
- [Captive portals or a slow server look like offline] → The 8s timeout and the explicit `online` event retry; the banner tells the user why editing is disabled.
- [Bottom-sheet drag conflicts with map gestures] → Drag only on the handle; the map below the collapsed sheet keeps all gestures.
- [Large precache on first visit over mobile data] → Accept; airport data is already in the bundle today. Map tiles are cached only as viewed.
- [Demo tiles host changes caching headers or URLs] → Runtime caching matches by origin, not file names.
