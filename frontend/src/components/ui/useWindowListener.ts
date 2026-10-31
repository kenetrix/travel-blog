import { onBeforeUnmount, onMounted } from 'vue'

// Listens on window while the calling component is mounted
export function useWindowListener<K extends keyof WindowEventMap>(
    type: K,
    listener: (event: WindowEventMap[K]) => void
) {
    onMounted(() => window.addEventListener(type, listener))
    onBeforeUnmount(() => window.removeEventListener(type, listener))
}
