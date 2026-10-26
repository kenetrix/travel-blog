<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useLayout } from './useLayout'
import { registerSheet } from './mapInsets'

// Phone: bottom sheet that collapses to a peek. Tablet: side sheet.
// Desktop: floating panel on the given side.
const props = withDefaults(
    defineProps<{ side?: 'left' | 'right'; wide?: boolean; label: string }>(),
    { side: 'left', wide: false }
)
const expanded = defineModel<boolean>('expanded', { default: false })

const { layout } = useLayout()
const sheet = ref<HTMLElement | null>(null)
const handle = ref<HTMLElement | null>(null)

// Pixels the sheet is dragged by; null while not dragging
const dragY = ref<number | null>(null)
let startY = 0
let startOffset = 0
let moved = false

// Visible height while collapsed; the bottom padding is the safe area
const PEEK = 104
function peek(): number {
    const el = sheet.value
    if (!el) return 0
    return PEEK + (parseFloat(getComputedStyle(el).paddingBottom) || 0)
}

// How far the sheet is pushed down in the given state
function offsetFor(open: boolean): number {
    const el = sheet.value
    if (!el || open) return 0
    return Math.max(0, el.offsetHeight - peek())
}

function onPointerDown(e: PointerEvent) {
    if (layout.value !== 'phone') return
    startY = e.clientY
    startOffset = offsetFor(expanded.value)
    moved = false
    dragY.value = startOffset
    handle.value?.setPointerCapture(e.pointerId)
}

function onPointerMove(e: PointerEvent) {
    if (dragY.value === null) return
    const dy = e.clientY - startY
    if (Math.abs(dy) > 4) moved = true
    dragY.value = Math.min(offsetFor(false), Math.max(0, startOffset + dy))
}

function onPointerUp(e: PointerEvent) {
    if (dragY.value === null) return
    const dy = e.clientY - startY
    // A tap toggles; a drag snaps by direction, or to the nearer state
    if (!moved) expanded.value = !expanded.value
    else if (Math.abs(dy) > 30) expanded.value = dy < 0
    else expanded.value = dragY.value < offsetFor(false) / 2
    dragY.value = null
}

function onKey(e: KeyboardEvent) {
    if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        expanded.value = !expanded.value
    }
}

const style = computed(() =>
    dragY.value === null
        ? undefined
        : { transform: `translateY(${dragY.value}px)`, transition: 'none' }
)

let unregister = () => {}
onMounted(() => {
    unregister = registerSheet(() => {
        const el = sheet.value
        if (!el) return {}
        const gap = 16
        if (layout.value === 'phone') {
            return { bottom: el.offsetHeight - offsetFor(expanded.value) + gap }
        }
        const rect = el.getBoundingClientRect()
        return props.side === 'right' && layout.value === 'desktop'
            ? { right: window.innerWidth - rect.left + gap }
            : { left: rect.right + gap }
    })
})
onBeforeUnmount(() => unregister())
</script>

<template>
    <section
        ref="sheet"
        class="sheet"
        :class="[side, { wide, expanded, dragging: dragY !== null }]"
        :style="style"
        :aria-label="label"
    >
        <div
            ref="handle"
            class="handle"
            role="button"
            tabindex="0"
            :aria-expanded="expanded"
            :aria-label="expanded ? `Collapse ${label}` : `Expand ${label}`"
            @pointerdown="onPointerDown"
            @pointermove="onPointerMove"
            @pointerup="onPointerUp"
            @pointercancel="dragY = null"
            @keydown="onKey"
        >
            <span class="grip" />
        </div>
        <div class="content" data-scroll>
            <slot />
        </div>
    </section>
</template>

<style scoped>
/* Phone: bottom sheet */
.sheet {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 15;
    display: flex;
    flex-direction: column;
    max-height: min(
        72dvh,
        calc(100dvh - var(--appbar-h) - var(--safe-top) - 1.5rem)
    );
    padding: 0 var(--safe-right) var(--safe-bottom) var(--safe-left);
    background: var(--panel);
    backdrop-filter: blur(8px);
    border: 1px solid var(--line);
    border-bottom: none;
    border-radius: var(--radius) var(--radius) 0 0;
    box-shadow: 0 -4px 24px rgb(0 0 0 / 0.12);
    /* 104px: PEEK in the script */
    transform: translateY(calc(100% - 104px - var(--safe-bottom)));
    transition: transform 0.25s ease;
    touch-action: pan-y;
}

.sheet.expanded {
    transform: none;
}

.handle {
    display: grid;
    flex: none;
    place-items: center;
    height: 28px;
    cursor: grab;
    touch-action: none;
}

.sheet.dragging .handle {
    cursor: grabbing;
}

.grip {
    width: 2.5rem;
    height: 5px;
    border-radius: 3px;
    background: var(--faint);
}

.content {
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
    min-height: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
    padding: 0 1.1rem 1.1rem;
}

/* While collapsed only the title shows; scrolling would move hidden content */
.sheet:not(.expanded) .content {
    overflow: hidden;
}

/* Tablet: side sheet below the app bar */
@media (min-width: 600px) {
    .sheet {
        top: calc(var(--appbar-h) + var(--safe-top) + 2rem);
        left: calc(var(--edge) + var(--safe-left));
        right: auto;
        bottom: calc(var(--edge) + var(--safe-bottom));
        width: min(360px, 45vw);
        max-height: none;
        padding: 0;
        border: 1px solid var(--line);
        border-radius: var(--radius);
        box-shadow: none;
        transform: none;
    }

    .handle {
        display: none;
    }

    .content,
    .sheet:not(.expanded) .content {
        overflow-y: auto;
        padding: 1.1rem;
    }
}

/* Desktop: floating panel that grows with its content */
@media (min-width: 1024px) {
    .sheet {
        top: 5.2rem;
        bottom: auto;
        width: 240px;
        max-height: calc(100dvh - 7rem);
    }

    .sheet.wide {
        width: 340px;
    }

    .sheet.right {
        left: auto;
        right: calc(var(--edge) + var(--safe-right));
    }
}
</style>
