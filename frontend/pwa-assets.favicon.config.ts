import { defineConfig } from '@vite-pwa/assets-generator/config'

// favicon.ico for browsers without SVG favicons: just globe and plane, like
// favicon.svg, on a transparent background
export default defineConfig({
    preset: {
        transparent: { sizes: [], favicons: [[48, 'favicon.ico']] },
        maskable: { sizes: [] },
        apple: { sizes: [] },
    },
    images: ['public/favicon.svg'],
})
