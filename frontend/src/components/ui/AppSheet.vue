<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useLayout } from './useLayout'
import { registerSheet } from './mapInsets'
import { t } from '../../i18n'

// Phone: bottom sheet that collapses to a peek. Tablet: side sheet.
// Desktop: floating panel on the given side. Outside phones, `expanded`
// false slides the sheet off screen, leaving a tab to bring it back.
const props = withDefaults(
    defineProps<{
        side?: 'left' | 'right'
        wide?: boolean
        // Phone: always as tall as allowed, so the content scrolls instead of
        // the sheet growing with it
        fill?: boolean
        label: string
    }>(),
    { side: 'left', wide: false, fill: false }
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
const PEEK = 72
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

const besideMap = computed(() => layout.value !== 'phone')
const hidden = computed(() => besideMap.value && !expanded.value)

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
        if (!el || hidden.value) return {}
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
        :class="[side, { wide, fill, expanded, dragging: dragY !== null }]"
        :style="style"
        :aria-label="label"
    >
        <div
            ref="handle"
            class="handle"
            role="button"
            tabindex="0"
            :aria-expanded="expanded"
            :aria-label="
                t(expanded ? 'sheet.collapse' : 'sheet.expand', { label })
            "
            @pointerdown="onPointerDown"
            @pointermove="onPointerMove"
            @pointerup="onPointerUp"
            @pointercancel="dragY = null"
            @keydown="onKey"
        >
            <span class="grip" />
        </div>
        <button
            type="button"
            class="tab"
            :aria-expanded="expanded"
            :aria-label="t(expanded ? 'sheet.hide' : 'sheet.show', { label })"
            :title="t(expanded ? 'sheet.hide' : 'sheet.show', { label })"
            @click="expanded = !expanded"
        >
            <svg
                viewBox="0 0 24 24"
                width="18"
                height="18"
                fill="none"
                stroke="currentColor"
                stroke-width="2.2"
                stroke-linecap="round"
                stroke-linejoin="round"
                aria-hidden="true"
            >
                <path d="M15 6l-6 6 6 6" />
            </svg>
        </button>
        <div class="content" data-scroll :inert="hidden">
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
    --sheet-max: min(
        72dvh,
        calc(100dvh - var(--appbar-h) - var(--safe-top) - 1.5rem)
    );
    max-height: var(--sheet-max);
    padding: 0 var(--safe-right) var(--safe-bottom) var(--safe-left);
    background: var(--panel);
    backdrop-filter: blur(8px);
    border: 1px solid var(--line);
    border-bottom: none;
    border-radius: var(--radius) var(--radius) 0 0;
    box-shadow: 0 -4px 24px rgb(0 0 0 / 0.12);
    /* 72px: PEEK in the script */
    transform: translateY(calc(100% - 72px - var(--safe-bottom)));
    transition: transform 0.25s ease;
    touch-action: pan-y;
}

.sheet.fill {
    height: var(--sheet-max);
}

.sheet.expanded {
    transform: none;
}

/* Hide/show tab of side sheets; phones use the handle instead */
.tab {
    display: none;
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
    scrollbar-width: none;
    /* --sheet-pad: lets content such as a sticky footer reach the edges */
    --sheet-pad: 1.1rem;
    padding: 0 var(--sheet-pad) var(--sheet-pad);
}

.content::-webkit-scrollbar {
    display: none;
}

/* The content of a filled sheet manages its own scrolling */
.sheet.fill .content {
    overflow: hidden;
}

/* While collapsed only the title shows; scrolling would move hidden content */
.sheet:not(.expanded) .content {
    overflow: hidden;
}

/* Tablet: side sheet below the app bar */
@media (min-width: 600px),
    (orientation: landscape) and (max-height: 500px) and (min-width: 480px) {
    .sheet {
        top: calc(var(--appbar-h) + var(--safe-top) + 2rem);
        left: calc(var(--edge) + var(--safe-left));
        right: auto;
        bottom: calc(var(--edge) + var(--safe-bottom));
        width: var(--sheet-w);
        height: auto;
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
        padding: var(--sheet-pad);
    }

    .sheet.fill .content {
        overflow: hidden;
    }

    /* Hidden: slid out to the left, only the tab stays on screen */
    .sheet:not(.expanded) {
        transform: translateX(calc(-100% - var(--edge) - var(--safe-left)));
    }

    /* Halfway down the edge, away from the panels' own close buttons */
    .tab {
        position: absolute;
        top: 50%;
        left: 100%;
        display: grid;
        place-items: center;
        width: 1.75rem;
        height: 3.5rem;
        transform: translateY(-50%);
        margin-left: 1px;
        padding: 0;
        border: 1px solid var(--line);
        border-left: none;
        border-radius: 0 0.6rem 0.6rem 0;
        background: var(--panel);
        backdrop-filter: blur(8px);
        color: var(--muted);
        cursor: pointer;
    }

    .tab:hover {
        color: var(--text);
    }

    .tab svg {
        transition: transform 0.25s;
    }

    .sheet:not(.expanded) .tab svg {
        transform: rotate(180deg);
    }
}

@media (orientation: landscape) and (max-height: 500px) and (min-width: 480px) {
    .sheet {
        top: calc(var(--safe-top) + var(--edge));
    }

    .content,
    .sheet:not(.expanded) .content {
        gap: 0.6rem;
        --sheet-pad: 0.8rem;
        padding: var(--sheet-pad);
    }
}

/* Desktop: floating panel that grows with its content */
@media (min-width: 1024px) and (min-height: 501px) {
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

    .sheet.right:not(.expanded) {
        transform: translateX(calc(100% + var(--edge) + var(--safe-right)));
    }

    /* Mirrored: the tab sits on the left edge and points right */
    .sheet.right .tab {
        left: auto;
        right: 100%;
        margin: 0 1px 0 0;
        border: 1px solid var(--line);
        border-right: none;
        border-radius: 0.6rem 0 0 0.6rem;
    }

    .sheet.right .tab svg {
        transform: rotate(180deg);
    }

    .sheet.right:not(.expanded) .tab svg {
        transform: none;
    }
}
</style>
