import { ref } from 'vue'

const SHOWN_MS = 3000

// A status message for pages above the map, whose own toast they would hide
export function useTimedMessage() {
    const message = ref<{ text: string; error: boolean } | null>(null)
    let timer: ReturnType<typeof setTimeout> | undefined

    function say(text: string, error = false) {
        message.value = { text, error }
        clearTimeout(timer)
        timer = setTimeout(() => (message.value = null), SHOWN_MS)
    }

    return { message, say }
}
