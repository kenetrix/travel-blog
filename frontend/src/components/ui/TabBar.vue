<script setup lang="ts">
defineProps<{ tabs: { id: string; label: string }[] }>()
const model = defineModel<string>({ required: true })
</script>

<template>
    <div class="tabs" role="tablist">
        <button
            v-for="t in tabs"
            :key="t.id"
            type="button"
            role="tab"
            :aria-selected="model === t.id"
            :class="{ active: model === t.id }"
            @click="model = t.id"
        >
            {{ t.label }}
        </button>
    </div>
</template>

<style scoped>
.tabs {
    display: flex;
    gap: 0.25rem;
    max-width: 720px;
    margin: 0 auto 1.2rem;
    padding: 0.25rem;
    overflow-x: auto;
    border-radius: 0.7rem;
    background: var(--chip);
}

button {
    flex: 1;
    padding: 0.45rem 0.9rem;
    border: none;
    border-radius: 0.5rem;
    background: transparent;
    color: var(--muted);
    font: inherit;
    font-size: 0.82rem;
    font-weight: 500;
    white-space: nowrap;
    cursor: pointer;
}

button.active {
    background: var(--raised);
    color: var(--text);
    box-shadow: 0 1px 3px rgb(0 0 0 / 0.12);
}
</style>
