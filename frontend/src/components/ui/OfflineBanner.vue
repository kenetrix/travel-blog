<script setup lang="ts">
import { useTravelStore } from '../stores/store'

// inline: in the flow of a page instead of floating over the map
defineProps<{ inline?: boolean }>()

const store = useTravelStore()
</script>

<template>
    <p
        v-if="store.connection === 'offline'"
        class="offline"
        :class="{ inline }"
        role="status"
    >
        <template v-if="inline">
            Offline · showing your last saved data, changes are paused
        </template>
        <template v-else>Offline · read-only</template>
    </p>
</template>

<style scoped>
.offline {
    position: absolute;
    /* Phones: below the stats */
    top: calc(var(--safe-top) + var(--appbar-h) + 3.2rem);
    left: 50%;
    z-index: 19;
    transform: translateX(-50%);
    width: max-content;
    max-width: calc(100vw - 2 * var(--edge));
    box-sizing: border-box;
    margin: 0;
    padding: 0.3rem 0.8rem;
    border-radius: 1rem;
    background: #8a5a12;
    color: #fff;
    font-size: 0.75rem;
    text-align: center;
}

.offline.inline {
    position: static;
    transform: none;
    width: auto;
    max-width: none;
    border-radius: 0;
}

/* Tablet: beside the side sheet */
@media (min-width: 600px),
    (orientation: landscape) and (max-height: 500px) and (min-width: 480px) {
    .offline:not(.inline) {
        top: calc(var(--safe-top) + var(--appbar-h) + 2rem);
        left: calc(var(--safe-left) + var(--sheet-space) + 1rem);
        max-width: calc(100vw - var(--sheet-space) - 4rem);
        transform: none;
    }
}

@media (orientation: landscape) and (max-height: 500px) and (min-width: 480px) {
    .offline:not(.inline) {
        top: calc(var(--safe-top) + var(--appbar-h) + 1rem);
        left: auto;
        right: calc(var(--safe-right) + var(--edge));
    }
}

@media (min-width: 1024px) and (min-height: 501px) {
    .offline:not(.inline) {
        top: 4.6rem;
        left: 50%;
        max-width: calc(100vw - 2 * var(--edge));
        transform: translateX(-50%);
    }
}
</style>
