import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'

// Same-origin API in dev and preview; the backend listens on localhost only
const proxy = { '/api': 'http://localhost:8080' }

const headers = {
    'X-Content-Type-Options': 'nosniff',
    'X-Frame-Options': 'DENY',
    'Referrer-Policy': 'strict-origin-when-cross-origin',
}

// Fonts and map data come from Google Fonts and the MapLibre demo tiles.
// Dev keeps the CSP off: Vite's HMR injects inline scripts.
const csp = [
    "default-src 'self'",
    "script-src 'self'",
    "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
    'font-src https://fonts.gstatic.com',
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
        VitePWA({
            registerType: 'autoUpdate',
            // Browser navigations to /api (e.g. the OIDC login redirect) must reach the server,
            // not get the cached app shell
            workbox: { navigateFallbackDenylist: [/^\/api\//] },
            manifest: {
                name: 'Travel Blog',
                short_name: 'TravelBlog',
                description: 'Travel tracker',
                theme_color: '#1e293b',
                background_color: '#ffffff',
                display: 'standalone',
                icons: [
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
                ],
            },
        }),
    ],
})
