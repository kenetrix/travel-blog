<script setup lang="ts">
import { t } from '../../i18n'

const version = __APP_VERSION__
</script>

<template>
    <!-- Written by the build (licenses.ts); a plain link, so it opens
         outside the app and also works offline from the precache -->
    <p class="legal">
        <a href="/third-party-licenses.txt" target="_blank" rel="noopener">
            {{ t('profile.licenses') }}
        </a>
        <span class="version">{{ version }}</span>
    </p>
</template>

<style scoped>
.legal {
    max-width: 720px;
    margin: 1.5rem auto 0;
    font-size: 0.78rem;
    text-align: center;
    color: var(--muted);
}

.legal a {
    display: inline-block;
    padding: 0.6rem 0.4rem;
    color: var(--muted);
}

.version::before {
    content: '·';
    margin-right: 0.4rem;
}
</style>
