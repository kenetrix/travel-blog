<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import PageWrapper from '../ui/PageWrapper.vue'
import { getCountryByIso } from '../countries/countries'

const route = useRoute()
const iso = computed(() => String(route.params.iso))
const country = computed(() => getCountryByIso(iso.value))
</script>

<template>
    <PageWrapper title="New trip" :subtitle="country?.name ?? iso">
        <!-- Placeholder: the trip planner is a future feature -->
        <p class="soon">
            Trip planning for {{ country?.name ?? iso }} is coming soon.
        </p>
    </PageWrapper>
</template>

<style scoped>
.soon {
    font-size: 0.9rem;
    color: var(--muted);
}
</style>
