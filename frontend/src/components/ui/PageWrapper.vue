<script setup lang="ts">
import { useRouter } from 'vue-router'

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
            <button v-if="closable" class="back" @click="back">← Back</button>
            <div>
                <h1>{{ title }}</h1>
                <p v-if="subtitle" class="sub">{{ subtitle }}</p>
            </div>
            <div class="actions"><slot name="actions" /></div>
        </header>
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

header {
    display: flex;
    align-items: center;
    gap: 1.2rem;
    padding: 1.2rem 1.5rem;
    border-bottom: 1px solid var(--line);
}

.actions {
    margin-left: auto;
}

h1 {
    font-family: 'DM Serif Display', serif;
    font-size: 1.3rem;
    font-weight: 400;
}

.sub {
    font-size: 0.8rem;
    color: var(--muted);
    margin-top: 0.1rem;
}

.back {
    padding: 0.4rem 0.9rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    background: transparent;
    font-size: 0.8rem;
    color: var(--muted);
    cursor: pointer;
}

main {
    flex: 1;
    overflow-y: auto;
    padding: 1.5rem;
}
</style>
