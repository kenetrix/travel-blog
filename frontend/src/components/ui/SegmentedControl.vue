<script setup lang="ts" generic="T extends string">
defineProps<{
    modelValue: T
    // color: background of the picked option instead of the accent
    options: { value: T; label: string; title?: string; color?: string }[]
    label: string
    disabled?: boolean
    // As wide as the container, the options sharing it
    stretch?: boolean
}>()
defineEmits<{ 'update:modelValue': [value: T] }>()
</script>

<template>
    <div
        class="segmented"
        :class="{ stretch }"
        role="radiogroup"
        :aria-label="label"
    >
        <button
            v-for="o in options"
            :key="o.value"
            type="button"
            role="radio"
            :aria-checked="o.value === modelValue"
            :class="{ on: o.value === modelValue }"
            :style="
                o.color && o.value === modelValue
                    ? { background: o.color, color: 'var(--on-status)' }
                    : undefined
            "
            :title="o.title"
            :disabled="disabled"
            @click="$emit('update:modelValue', o.value)"
        >
            {{ o.label }}
        </button>
    </div>
</template>

<style scoped>
.segmented {
    display: inline-flex;
    padding: 0.2rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.6rem;
    background: var(--page);
}

.segmented.stretch {
    display: flex;
}

.stretch button {
    flex: 1;
}

button {
    min-height: 2.2rem;
    padding: 0.3rem 0.8rem;
    border: none;
    border-radius: 0.45rem;
    background: transparent;
    color: var(--muted);
    font: inherit;
    font-size: 0.8rem;
    white-space: nowrap;
    cursor: pointer;
}

button:hover:not(:disabled):not(.on) {
    color: var(--text);
}

button.on {
    background: var(--accent);
    color: #fff;
}

button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}
</style>
