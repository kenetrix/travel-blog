import {
    defineConfig,
    minimal2023Preset,
} from '@vite-pwa/assets-generator/config'

// Background of public/favicon.svg; square icons fill its rounded corners
// with it instead of the generator's white padding
const background = '#2a2c2e'

// Regenerate the icons in public/ with `npm run generate:icons`
export default defineConfig({
    preset: {
        ...minimal2023Preset,
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
    images: ['public/favicon.svg'],
})
