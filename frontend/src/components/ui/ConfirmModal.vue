<script setup lang="ts">
import { t } from '../../i18n'
import { useWindowListener } from './useWindowListener'

const props = defineProps<{
    title: string
    message: string
    confirmLabel?: string
    confirmDisabled?: boolean
    // Red confirm button for actions that delete or discard something
    danger?: boolean
}>()

const emit = defineEmits<{
    confirm: []
    cancel: []
}>()

useWindowListener('keydown', (event) => {
    if (event.key === 'Escape') emit('cancel')
})
</script>

<template>
    <!-- On body: sheets and panels would otherwise clip the fixed overlay -->
    <Teleport to="body">
        <div class="overlay" @click="emit('cancel')" role="presentation">
            <div class="modal" role="dialog" aria-modal="true" @click.stop>
                <h3>{{ title }}</h3>
                <p class="message">{{ message }}</p>
                <slot />
                <div class="actions">
                    <button class="btn secondary" @click="emit('cancel')">
                        {{ t('common.cancel') }}
                    </button>
                    <button
                        class="btn"
                        :class="{ danger: props.danger }"
                        :disabled="props.confirmDisabled"
                        @click="emit('confirm')"
                    >
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
</style>
