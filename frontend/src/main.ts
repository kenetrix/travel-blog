import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import { useTravelStore } from './components/stores/store'
import 'maplibre-gl/dist/maplibre-gl.css'
import './style.css'
import 'flag-icons/css/flag-icons.min.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')

useTravelStore().load()
