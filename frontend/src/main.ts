import { createApp, watch } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import { useTravelStore } from './components/stores/store'
import { readSnapshot } from './components/stores/snapshot'
import { resolveLanguage, setLanguage } from './i18n'
import '@fontsource/dm-sans/400.css'
import '@fontsource/dm-sans/500.css'
import '@fontsource/dm-sans/600.css'
import '@fontsource/dm-serif-display/400.css'
import './cyrillic.css'
import 'maplibre-gl/dist/maplibre-gl.css'
import './style.css'
import 'flag-icons/css/flag-icons.min.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)

const store = useTravelStore()
// A logged-in user's profile language wins over the device and the browser
watch(
    () => store.profile?.language,
    (profileLanguage) => {
        if (profileLanguage) setLanguage(resolveLanguage(profileLanguage))
    }
)

// Mount once the language is loaded, so nothing shows in the wrong one. The
// last session's profile saves a switch once the account has loaded.
setLanguage(resolveLanguage(readSnapshot()?.profile.language)).finally(() =>
    app.mount('#app')
)
store.load()
