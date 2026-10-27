<script setup lang="ts" generic="T extends string">
defineProps<{
    modelValue: T
    options: { value: T; label: string }[]
    label: string
    disabled?: boolean
}>()
defineEmits<{ 'update:modelValue': [value: T] }>()
</script>

<template>
    <div class="segmented" role="radiogroup" :aria-label="label">
        <button
            v-for="o in options"
            :key="o.value"
            type="button"
            role="radio"
            :aria-checked="o.value === modelValue"
            :class="{ on: o.value === modelValue }"
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

button.on {
    background: #3a7d5c;
    color: #fff;
}

button:disabled {
    opacity: 0.5;
    cursor: default;
}
</style>
