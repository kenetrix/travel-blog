<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
import { searchAirports, type Airport } from './airports'
import { focusNext } from '../ui/enterNext'

defineProps<{ placeholder?: string }>()
const emit = defineEmits<{ focus: [] }>()
const model = defineModel<Airport | null>({ default: null })

const query = ref('')
const editing = ref(false)
const active = ref(0)
const list = ref<HTMLElement | null>(null)

// The list only opens once something has been typed
const open = computed(() => editing.value && query.value.trim() !== '')
const results = computed(() => searchAirports(query.value))

function label(a: Airport) {
    return `${a.iata} · ${a.city}`
}

function onFocus(e: FocusEvent) {
    query.value = ''
    editing.value = true
    emit('focus')
    // Phones: keep the field and its results above the on-screen keyboard
    if (!matchMedia('(pointer: coarse)').matches) return
    // Only the sheet scrolls; scrollIntoView would also shift the map page
    const input = e.target as HTMLElement
    const box = input.closest<HTMLElement>('[data-scroll]')
    if (!box) return
    setTimeout(() => {
        const top =
            input.getBoundingClientRect().top - box.getBoundingClientRect().top
        box.scrollBy({ top: top - 8, behavior: 'smooth' })
    }, 300)
}

function onInput(e: Event) {
    editing.value = true
    query.value = (e.target as HTMLInputElement).value
    active.value = 0
}

function move(step: number) {
    const count = results.value.length
    if (!count) return
    active.value = (active.value + step + count) % count
    nextTick(() =>
        list.value
            ?.querySelector('.active')
            ?.scrollIntoView({ block: 'nearest' })
    )
}

function onKeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        if (!open.value) return
        e.preventDefault()
        move(e.key === 'ArrowDown' ? 1 : -1)
    } else if (e.key === 'Enter') {
        const choice = open.value && results.value[active.value]
        if (!choice) return
        e.preventDefault()
        pick(choice)
        // Picking with Enter carries on to the next field
        focusNext(e.target as HTMLElement)
    } else if (e.key === 'Escape') {
        editing.value = false
        query.value = ''
    }
}

function clear() {
    model.value = null
    query.value = ''
}

function pick(a: Airport) {
    model.value = a
    editing.value = false
    query.value = ''
}

function onBlur() {
    editing.value = false
    query.value = ''
}
</script>

<template>
    <div class="select">
        <input
            :value="editing ? query : model ? label(model) : ''"
            :placeholder="placeholder"
            autocomplete="off"
            @focus="onFocus"
            @blur="onBlur"
            @input="onInput"
            @keydown="onKeydown"
        />
        <button
            v-if="model && !editing"
            type="button"
            class="clear"
            tabindex="-1"
            aria-label="Clear airport"
            @mousedown.prevent="clear"
        >
            ×
        </button>
        <ul v-if="open" ref="list" class="list">
            <li
                v-for="(a, i) in results"
                :key="a.iata"
                :class="{ active: i === active }"
                @mousedown.prevent="pick(a)"
                @mousemove="active = i"
            >
                <strong>{{ a.iata }}</strong>
                <span>{{ a.city }}, {{ a.country }}</span>
                <small>{{ a.name }}</small>
            </li>
            <li v-if="!results.length" class="empty">No airports found</li>
        </ul>
        <p v-if="model && !open" class="selected">{{ model.name }}</p>
    </div>
</template>

<style scoped>
.select {
    position: relative;
}

.clear {
    position: absolute;
    top: 0;
    right: 0;
    width: 2.6rem;
    height: 100%;
    max-height: var(--touch);
    border: none;
    background: none;
    color: var(--muted);
    font-size: 1rem;
    line-height: 1.5rem;
    cursor: pointer;
}

.clear:hover {
    color: #c0392b;
}

.selected {
    margin: 0.2rem 0 0;
    font-size: 0.7rem;
    color: var(--muted);
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
}

input {
    box-sizing: border-box;
    width: 100%;
    padding: 0.5rem 2.6rem 0.5rem 0.6rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    font-size: var(--field-font);
    background: var(--raised);
    color: var(--text);
    outline: none;
}

input:focus {
    border-color: #3a7d5c;
}

.list {
    position: absolute;
    z-index: 20;
    top: calc(100% + 0.25rem);
    left: 0;
    right: 0;
    max-height: min(13rem, 40dvh);
    overflow-y: auto;
    margin: 0;
    padding: 0.25rem;
    list-style: none;
    background: var(--raised);
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    box-shadow: 0 6px 18px var(--line-strong);
}

li {
    display: grid;
    grid-template-columns: auto 1fr;
    column-gap: 0.5rem;
    min-height: var(--touch);
    box-sizing: border-box;
    align-content: center;
    padding: 0.35rem 0.5rem;
    border-radius: 0.35rem;
    font-size: 0.8rem;
    cursor: pointer;
}

@media (pointer: fine) {
    li {
        min-height: 0;
    }
}

li.active {
    background: rgba(58, 125, 92, 0.1);
}

li small {
    grid-column: 2;
    color: var(--muted);
    font-size: 0.7rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.empty {
    color: var(--muted);
    cursor: default;
}

.empty:hover {
    background: none;
}
</style>
