import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'
import { thirdPartyLicenses } from './licenses.ts'

// Same-origin API in dev and preview; the backend listens on localhost only
const proxy = { '/api': 'http://localhost:8080' }

const headers = {
    'X-Content-Type-Options': 'nosniff',
    'X-Frame-Options': 'DENY',
    'Referrer-Policy': 'strict-origin-when-cross-origin',
}

// Fonts are bundled; map data comes from the MapLibre demo tiles.
// Dev keeps the CSP off: Vite's HMR injects inline scripts.
const csp = [
    "default-src 'self'",
    "script-src 'self'",
    "style-src 'self' 'unsafe-inline'",
    "font-src 'self'",
    "img-src 'self' data: blob: https://demotiles.maplibre.org",
    "connect-src 'self' https://demotiles.maplibre.org",
    "worker-src 'self' blob:",
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "frame-ancestors 'none'",
].join('; ')

// https://vite.dev/config/
export default defineConfig({
    server: { proxy, headers },
    preview: {
        proxy,
        headers: { ...headers, 'Content-Security-Policy': csp },
    },
    plugins: [
        vue(),
        thirdPartyLicenses(),
        VitePWA({
            registerType: 'autoUpdate',
            includeAssets: [
                'favicon.svg',
                'favicon.ico',
                'apple-touch-icon-180x180.png',
            ],
            // Browser navigations to /api (e.g. the OIDC login redirect) must reach the server,
            // not get the cached app shell
            workbox: {
                navigateFallbackDenylist: [/^\/api\//],
                // The airport and country data are bundled into the JS
                maximumFileSizeToCacheInBytes: 8 * 1024 * 1024,
                globPatterns: ['**/*.{js,css,html,ico,png,svg,woff2,txt}'],
                // Map style and tiles already seen stay available offline.
                // API data is not cached here; the store keeps its own
                // snapshot (see stores/snapshot.ts).
                runtimeCaching: [
                    {
                        // Regexes: workbox copies these into the service worker
                        urlPattern:
                            /^https:\/\/demotiles\.maplibre\.org\/(?!.*\.pbf$)/,
                        handler: 'StaleWhileRevalidate',
                        options: {
                            cacheName: 'map-style',
                            cacheableResponse: { statuses: [200] },
                        },
                    },
                    {
                        urlPattern:
                            /^https:\/\/demotiles\.maplibre\.org\/.*\.pbf$/,
                        handler: 'CacheFirst',
                        options: {
                            cacheName: 'map-tiles',
                            cacheableResponse: { statuses: [200] },
                            expiration: {
                                maxEntries: 1000,
                                maxAgeSeconds: 30 * 24 * 60 * 60,
                            },
                        },
                    },
                ],
            },
            manifest: {
                name: 'Travel Blog',
                short_name: 'TravelBlog',
                description: 'Travel tracker',
                // Map water colour, as theme-color in index.html
                theme_color: '#373838',
                background_color: '#373838',
                display: 'standalone',
                orientation: 'any',
                start_url: '/',
                icons: [
                    {
                        src: 'pwa-64x64.png',
                        sizes: '64x64',
                        type: 'image/png',
                    },
                    {
                        src: 'pwa-192x192.png',
                        sizes: '192x192',
                        type: 'image/png',
                    },
                    {
                        src: 'pwa-512x512.png',
                        sizes: '512x512',
                        type: 'image/png',
                    },
                    {
                        src: 'maskable-icon-512x512.png',
                        sizes: '512x512',
                        type: 'image/png',
                        purpose: 'maskable',
                    },
                ],
            },
        }),
    ],
})
