# Travel Blog — Frontend

Interactive travel tracker: a world map showing visited countries (markers) and flights (routes), with side panels to manage each. Ships as an installable PWA.

## Architecture

- **Layers**
    - `App.vue` — layout, top bar, and Countries/Flights view toggle
    - Feature components — `worldmap/` (MapLibre GL map), `countries/` (list & search), `flights/` (routes & city geocoding via Nominatim/OpenStreetMap)
    - `stores/store.ts` — single Pinia store, the source of truth for visited countries and flights

- **Data flow** — components read and mutate the shared store reactively; the map re-renders from store state. Flight geocoding is the only external API call.

- **Stack**
    - Vue 3 (`<script setup>` SFCs) + TypeScript
    - Vite (dev server, builds, PWA via `vite-plugin-pwa`)
    - Pinia (state management)
    - MapLibre GL (map rendering)
    - Country name lookups come from a local `countries.json` dataset (no backend dependency)

## Getting started

```bash
npm install
npm run dev        # start the Vite dev server
npm run build      # type-check and build for production
npm run preview    # preview the production build
```
