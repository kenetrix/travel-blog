import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import { useTravelStore } from './components/stores/store'
import '@fontsource/dm-sans/400.css'
import '@fontsource/dm-sans/500.css'
import '@fontsource/dm-sans/600.css'
import '@fontsource/dm-serif-display/400.css'
import 'maplibre-gl/dist/maplibre-gl.css'
import './style.css'
import 'flag-icons/css/flag-icons.min.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')

useTravelStore().load()
