<script setup lang="ts">
import { onMounted, ref } from 'vue'
import PageWrapper from '../ui/PageWrapper.vue'
import { t } from '../../i18n'

interface Notice {
    name: string
    address: string[]
    email: string
}

const notice = ref<Notice | null>(null)
const failed = ref(false)

// Outside the OpenAPI contract: plain JSON built from the server's LEGAL_* variables
onMounted(async () => {
    try {
        const response = await fetch('/api/public/legal')
        if (!response.ok) throw new Error(String(response.status))
        notice.value = await response.json()
    } catch {
        failed.value = true
    }
})
</script>

<template>
    <PageWrapper :title="t('legal.impressumTitle')">
        <article class="legal-page">
            <p>{{ t('legal.impressumIntro') }}</p>
            <p v-if="failed" class="note">{{ t('legal.unavailable') }}</p>
            <p v-else-if="notice && !notice.name" class="note">
                {{ t('legal.missing') }}
            </p>
            <template v-else-if="notice">
                <h2>{{ t('legal.provider') }}</h2>
                <p>
                    {{ notice.name }}
                    <template v-for="line in notice.address" :key="line">
                        <br />{{ line }}
                    </template>
                </p>
                <template v-if="notice.email">
                    <h2>{{ t('legal.contact') }}</h2>
                    <p>
                        <a :href="`mailto:${notice.email}`">{{
                            notice.email
                        }}</a>
                    </p>
                </template>
                <h2>{{ t('legal.responsible') }}</h2>
                <p>{{ notice.name }}</p>
            </template>
        </article>
    </PageWrapper>
</template>

<style src="./legal.css"></style>
