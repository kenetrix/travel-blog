import { computed, onScopeDispose, ref, type Ref } from 'vue'

export type Layout = 'phone' | 'tablet' | 'desktop'

// Same breakpoints as the media queries in style.css
const TABLET = '(min-width: 600px)'
const DESKTOP = '(min-width: 1024px)'
const COARSE = '(pointer: coarse)'

function query(media: string): Ref<boolean> {
    const list = window.matchMedia(media)
    const matches = ref(list.matches)
    const update = (e: MediaQueryListEvent) => (matches.value = e.matches)
    list.addEventListener('change', update)
    onScopeDispose(() => list.removeEventListener('change', update))
    return matches
}

// For what CSS alone cannot decide, e.g. which markup or map padding to use
export function useLayout() {
    const tablet = query(TABLET)
    const desktop = query(DESKTOP)
    const coarsePointer = query(COARSE)
    const layout = computed<Layout>(() =>
        desktop.value ? 'desktop' : tablet.value ? 'tablet' : 'phone'
    )
    return { layout, coarsePointer }
}
