<script setup lang="ts">
import { useRouter } from 'vue-router'
import OfflineBanner from './OfflineBanner.vue'

withDefaults(
    defineProps<{ title: string; subtitle?: string; closable?: boolean }>(),
    { closable: true }
)

const router = useRouter()

// Back to the map, which keeps whatever was open before
function back() {
    router.push({ name: 'home' })
}
</script>

<template>
    <div class="page">
        <header>
            <button
                v-if="closable"
                class="back"
                aria-label="Back to the map"
                @click="back"
            >
                ←<span class="back-text"> Back</span>
            </button>
            <div class="titles">
                <h1>{{ title }}</h1>
                <p v-if="subtitle" class="sub">{{ subtitle }}</p>
            </div>
            <div class="actions"><slot name="actions" /></div>
        </header>
        <OfflineBanner inline />
        <main>
            <slot />
        </main>
    </div>
</template>

<style scoped>
.page {
    position: absolute;
    inset: 0;
    z-index: 30;
    display: flex;
    flex-direction: column;
    background: var(--page);
    font-family: 'DM Sans', sans-serif;
    color: var(--text);
}

/* The header stays put while main scrolls */
header {
    display: flex;
    flex: none;
    align-items: center;
    gap: 0.6rem;
    padding: calc(0.6rem + var(--safe-top))
        calc(var(--edge) + var(--safe-right)) 0.6rem
        calc(var(--edge) + var(--safe-left));
    border-bottom: 1px solid var(--line);
}

@media (min-width: 600px),
    (orientation: landscape) and (max-height: 500px) and (min-width: 480px) {
    header {
        gap: 1.2rem;
        padding: calc(1.2rem + var(--safe-top)) calc(1.5rem + var(--safe-right))
            1.2rem calc(1.5rem + var(--safe-left));
    }
}

.titles {
    min-width: 0;
}

.actions {
    flex: none;
    margin-left: auto;
}

h1 {
    margin: 0;
    font-family: 'DM Serif Display', serif;
    font-size: 1.2rem;
    font-weight: 400;
    overflow-wrap: anywhere;
}

.sub {
    margin: 0.1rem 0 0;
    font-size: 0.8rem;
    color: var(--muted);
}

.back {
    flex: none;
    min-width: var(--touch);
    min-height: var(--touch);
    padding: 0 0.7rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    background: transparent;
    font: inherit;
    font-size: 0.85rem;
    color: var(--muted);
    cursor: pointer;
}

.back-text {
    display: none;
}

@media (min-width: 600px),
    (orientation: landscape) and (max-height: 500px) and (min-width: 480px) {
    h1 {
        font-size: 1.3rem;
    }

    .back-text {
        display: inline;
    }
}

@media (pointer: fine) {
    .back {
        min-height: 0;
        padding: 0.4rem 0.9rem;
        font-size: 0.8rem;
    }
}

@media (orientation: landscape) and (max-height: 500px) and (min-width: 480px) {
    header {
        padding-top: calc(0.4rem + var(--safe-top));
        padding-bottom: 0.4rem;
    }

    h1 {
        font-size: 1.1rem;
    }

    .sub {
        display: none;
    }
}

main {
    flex: 1;
    overflow-y: auto;
    overscroll-behavior: contain;
    padding: 1rem calc(1rem + var(--safe-right))
        calc(1.5rem + var(--safe-bottom)) calc(1rem + var(--safe-left));
}

@media (min-width: 600px),
    (orientation: landscape) and (max-height: 500px) and (min-width: 480px) {
    main {
        padding: 1.5rem calc(1.5rem + var(--safe-right))
            calc(1.5rem + var(--safe-bottom)) calc(1.5rem + var(--safe-left));
    }
}
</style>
