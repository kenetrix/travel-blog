import { ref } from 'vue'
import { onBeforeRouteLeave, useRouter } from 'vue-router'

// Asks before the route is left while `dirty()` is true: pendingLeave holds
// the navigation until the discard dialog confirms or cancels it.
export function useLeaveGuard(dirty: () => boolean) {
    const router = useRouter()
    const pendingLeave = ref<(() => void) | null>(null)
    // Set once the user confirmed leaving, or after saving or deleting
    let leaving = false

    onBeforeRouteLeave((to) => {
        if (leaving || !dirty()) return true
        pendingLeave.value = () => {
            leaving = true
            router.push(to.fullPath)
        }
        return false
    })

    return {
        pendingLeave,
        // Lets the next navigation pass without asking
        allowLeave: () => (leaving = true),
        confirmLeave: () => {
            pendingLeave.value?.()
            pendingLeave.value = null
        },
        cancelLeave: () => (pendingLeave.value = null),
    }
}
