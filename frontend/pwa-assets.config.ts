import {
    defineConfig,
    minimal2023Preset,
} from '@vite-pwa/assets-generator/config'

// Background of public/icon.svg; square icons fill its rounded corners with
// it instead of the generator's white padding
const background = '#2a2c2e'

// App icons (PWA, maskable, Apple) from public/icon.svg. favicon.ico comes
// from the background-free favicon.svg, see pwa-assets.favicon.config.ts.
// Regenerate all of them with `npm run generate:icons`.
export default defineConfig({
    preset: {
        ...minimal2023Preset,
        transparent: { ...minimal2023Preset.transparent, favicons: [] },
        maskable: {
            ...minimal2023Preset.maskable,
            padding: 0,
            resizeOptions: { background },
        },
        apple: {
            ...minimal2023Preset.apple,
            padding: 0,
            resizeOptions: { background },
        },
    },
    images: ['public/icon.svg'],
})
