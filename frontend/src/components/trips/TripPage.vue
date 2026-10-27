<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import PageWrapper from '../ui/PageWrapper.vue'
import { countryName, getCountryByIso } from '../countries/countries'
import { t } from '../../i18n'

const route = useRoute()
const iso = computed(() => String(route.params.iso))
const country = computed(() => getCountryByIso(iso.value))
const name = computed(() =>
    country.value ? countryName(country.value) : iso.value
)
</script>

<template>
    <PageWrapper :title="t('trip.title')" :subtitle="name">
        <!-- Placeholder: the trip planner is a future feature -->
        <p class="soon">{{ t('trip.soon', { country: name }) }}</p>
    </PageWrapper>
</template>

<style scoped>
.soon {
    font-size: 0.9rem;
    color: var(--muted);
}
</style>
