<script setup lang="ts">
import { onMounted, onBeforeUnmount } from 'vue'
import { t } from '../../i18n'

const props = defineProps<{
    title: string
    message: string
    confirmLabel?: string
}>()

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
    <!-- On body: sheets and panels would otherwise clip the fixed overlay -->
    <Teleport to="body">
        <div class="overlay" @click="emit('cancel')" role="presentation">
            <div class="modal" role="dialog" aria-modal="true" @click.stop>
                <h3>{{ title }}</h3>
                <p class="message">{{ message }}</p>
                <div class="actions">
                    <button class="btn cancel" @click="emit('cancel')">
                        {{ t('common.cancel') }}
                    </button>
                    <button class="btn confirm" @click="emit('confirm')">
                        {{ props.confirmLabel ?? t('common.confirm') }}
                    </button>
                </div>
            </div>
        </div>
    </Teleport>
</template>

<style scoped>
.overlay {
    position: fixed;
    inset: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem calc(1rem + var(--safe-right)) calc(1rem + var(--safe-bottom))
        calc(1rem + var(--safe-left));
    background: rgba(0, 0, 0, 0.25);
    font-family: 'DM Sans', sans-serif;
}

.modal {
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
    box-sizing: border-box;
    width: 320px;
    max-width: 100%;
    max-height: calc(100dvh - 2rem);
    overflow-y: auto;
    background: var(--panel);
    backdrop-filter: blur(8px);
    border: 1px solid var(--line);
    border-radius: 1rem;
    padding: 1.2rem;
    box-shadow: 0 8px 32px var(--line-strong);
}

h3 {
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-soft);
}

.message {
    font-size: 0.82rem;
    line-height: 1.45;
    color: var(--muted);
}

.actions {
    display: flex;
    flex-direction: column-reverse;
    gap: 0.5rem;
}

@media (min-width: 600px),
    (orientation: landscape) and (max-height: 500px) and (min-width: 480px) {
    .actions {
        flex-direction: row;
        justify-content: flex-end;
    }
}

.btn {
    min-height: var(--touch);
    padding: 0.4rem 0.9rem;
    border-radius: 0.5rem;
    font-size: 0.8rem;
    font-weight: 500;
    cursor: pointer;
    border: 1px solid transparent;
    font-family: inherit;
    transition:
        border-color 0.15s ease,
        background-color 0.15s ease;
}

@media (pointer: fine) {
    .btn {
        min-height: 0;
    }
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
