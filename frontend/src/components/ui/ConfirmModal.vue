<script setup lang="ts">
import { onMounted, onBeforeUnmount } from 'vue'

const props = withDefaults(
    defineProps<{
        title: string
        message: string
        confirmLabel?: string
    }>(),
    { confirmLabel: 'Confirm' }
)

const emit = defineEmits<{
    confirm: []
    cancel: []
}>()

function onKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') emit('cancel')
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
    <div class="overlay" @click="emit('cancel')" role="presentation">
        <div class="modal" role="dialog" aria-modal="true" @click.stop>
            <h3>{{ title }}</h3>
            <p class="message">{{ message }}</p>
            <div class="actions">
                <button class="btn cancel" @click="emit('cancel')">
                    Cancel
                </button>
                <button class="btn confirm" @click="emit('confirm')">
                    {{ props.confirmLabel }}
                </button>
            </div>
        </div>
    </div>
</template>

<style scoped>
.overlay {
    position: fixed;
    inset: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.25);
}

.modal {
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
    width: 320px;
    max-width: calc(100vw - 2.4rem);
    background: var(--panel);
    backdrop-filter: blur(8px);
    border: 1px solid var(--line);
    border-radius: 1rem;
    padding: 1.2rem;
    box-shadow: 0 8px 32px var(--line-strong);
}

h3 {
    font-family: 'DM Serif Display', serif;
    font-size: 1rem;
    font-weight: 400;
    color: var(--text-soft);
}

.message {
    font-size: 0.82rem;
    line-height: 1.45;
    color: var(--muted);
}

.actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
}

.btn {
    padding: 0.4rem 0.9rem;
    border-radius: 0.5rem;
    font-size: 0.8rem;
    font-weight: 500;
    cursor: pointer;
    border: 1px solid transparent;
    transition:
        border-color 0.15s ease,
        background-color 0.15s ease;
}

.btn.cancel {
    background: transparent;
    border-color: var(--line-strong);
    color: var(--muted);
}

.btn.cancel:hover {
    border-color: rgba(0, 0, 0, 0.25);
}

.btn.cancel:focus {
    outline: none;
    border-color: #3a7d5c;
}

.btn.confirm {
    background: #3a7d5c;
    border-color: #3a7d5c;
    color: #fff;
}

.btn.confirm:hover {
    background: #2f6a4e;
}

.btn.confirm:focus {
    outline: none;
    box-shadow: 0 0 0 2px rgba(58, 125, 92, 0.35);
}
</style>
