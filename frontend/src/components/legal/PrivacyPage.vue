<script setup lang="ts">
import PageWrapper from '../ui/PageWrapper.vue'
import { t, type MessageKey } from '../../i18n'

const sections = [
    'controller',
    'logs',
    'account',
    'storage',
    'map',
    'sso',
    'rights',
]
const key = (section: string, part: 'Title' | 'Text') =>
    `legal.${section}${part}` as MessageKey
</script>

<template>
    <PageWrapper history :title="t('legal.privacyTitle')">
        <article class="legal-page">
            <p>{{ t('legal.privacyIntro') }}</p>
            <section v-for="section in sections" :key="section">
                <h2>{{ t(key(section, 'Title')) }}</h2>
                <p>{{ t(key(section, 'Text')) }}</p>
            </section>
        </article>
    </PageWrapper>
</template>

<style src="./legal.css"></style>
