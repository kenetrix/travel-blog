<script setup lang="ts">
import { ref, computed } from 'vue'
import { useTravelStore } from '../stores/store'
import { storeToRefs } from 'pinia'
import { Countries, getCountryByIso, type Country } from './countries'
import CountryDetailModal from './CountryDetailModal.vue'

const store = useTravelStore()
const { visited, wishlist, selectedIso, details, toast } = storeToRefs(store)

const searchCountry = ref('')

const query = computed(() => searchCountry.value.trim().toLowerCase())

const activeTag = ref('')

const allTags = computed(() => [
    ...new Set(Object.values(details.value).flatMap((d) => d.tags ?? [])),
])

const byName = (a: Country, b: Country) => a.name.localeCompare(b.name, 'de')

function fromIsos(isos: Iterable<string>): Country[] {
    return [...isos].flatMap((iso) => {
        const country = getCountryByIso(iso)
        return country ? [country] : []
    })
}

const matchesTag = (c: Country) =>
    !activeTag.value || details.value[c.alpha3]?.tags?.includes(activeTag.value)

const wishlistCountries = computed(() =>
    query.value ? [] : fromIsos(wishlist.value).filter(matchesTag).sort(byName)
)

// Without a search term: visited countries. With one: any matching country.
const listedCountries = computed(() => {
    const list = query.value
        ? Countries.filter(
              (c) =>
                  c.name.toLowerCase().startsWith(query.value) ||
                  c.nameLong?.toLowerCase().startsWith(query.value) ||
                  c.alpha3.toLowerCase() === query.value ||
                  c.alpha2.toLowerCase() === query.value
          )
        : fromIsos(visited.value).filter(matchesTag)
    return list.sort(byName)
})
</script>

<template>
    <div v-if="toast" class="toast" role="status">{{ toast }}</div>
    <CountryDetailModal
        v-if="selectedIso"
        :iso="selectedIso"
        @close="store.selectCountry(null)"
    />
    <div v-else class="panel">
        <div>
            <h2>Countries</h2>
            <p class="sub">Visited countries in the world</p>
        </div>

        <div>
            <input v-model="searchCountry" placeholder="Search for country" />
        </div>

        <div v-if="allTags.length && !query" class="tags">
            <button
                v-for="t in allTags"
                :key="t"
                type="button"
                class="tag"
                :class="{ on: activeTag === t }"
                @click="activeTag = activeTag === t ? '' : t"
            >
                {{ t }}
            </button>
        </div>

        <div v-if="wishlistCountries.length">
            <label>Want to go</label>
            <ol>
                <li
                    v-for="country in wishlistCountries"
                    :key="country.alpha3"
                    @click="store.selectCountry(country.alpha3)"
                >
                    <span
                        :class="`fi fi-${country.alpha2.toLowerCase()}`"
                    ></span>
                    {{ country.name }}
                </li>
            </ol>
        </div>

        <div>
            <label>{{ query ? 'Results' : 'Visited' }}</label>
            <ol>
                <li
                    v-for="country in listedCountries"
                    :key="country.alpha3"
                    @click="store.selectCountry(country.alpha3)"
                >
                    <span
                        :class="`fi fi-${country.alpha2.toLowerCase()}`"
                    ></span>
                    {{ country.name }}
                    <span
                        v-if="query && visited.has(country.alpha3)"
                        class="check"
                    >
                        ✓
                    </span>
                    <span
                        v-else-if="query && wishlist.has(country.alpha3)"
                        class="wish"
                    >
                        ☆
                    </span>
                    <span v-if="details[country.alpha3]?.rating" class="rating">
                        {{ '★'.repeat(details[country.alpha3].rating) }}
                    </span>
                </li>
                <li v-if="!listedCountries.length" class="empty">
                    No countries found
                </li>
            </ol>
        </div>
    </div>
</template>

<style scoped>
.panel {
    position: absolute;
    top: 5.2rem;
    left: 1.2rem;
    z-index: 10;
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
    width: 240px;
    background: var(--panel);
    backdrop-filter: blur(8px);
    border: 1px solid var(--line);
    border-radius: 1rem;
    padding: 1.1rem;
}

h2 {
    font-family: 'DM Serif Display', serif;
    font-size: 1rem;
    font-weight: 400;
}

.sub {
    font-size: 0.75rem;
    color: var(--muted);
    margin-top: 0.1rem;
}

label {
    display: block;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--muted);
    margin-bottom: 0.25rem;
}

ol {
    max-height: 50vh;
    overflow-y: auto;
    display: block;
    font-size: 1rem;
    font-weight: 500;
    color: var(--muted);
    margin-bottom: 0.25rem;
    list-style-type: none;
}

li {
    cursor: pointer;
}

li:hover {
    color: var(--text);
}

.tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
}

.tag {
    padding: 0.1rem 0.5rem;
    border: 1px solid var(--line-strong);
    border-radius: 1rem;
    background: var(--raised);
    font-size: 0.72rem;
    color: var(--muted-strong);
    cursor: pointer;
}

.tag.on {
    background: var(--accent-soft);
    border-color: #3a7d5c;
    color: #2f6a4e;
}

.wish {
    color: var(--wishlist);
    font-size: 0.8em;
}

.rating {
    color: #ca8f22;
    font-size: 0.65em;
    margin-left: 0.3rem;
}

.toast {
    position: absolute;
    bottom: 1.5rem;
    left: 1.2rem;
    z-index: 20;
    padding: 0.5rem 0.9rem;
    border-radius: 0.5rem;
    background: #1a1c1e;
    color: #fff;
    font-size: 0.8rem;
}

li.empty {
    cursor: default;
    font-size: 0.82rem;
}

.check {
    color: #3a7d5c;
    font-size: 0.8em;
}

.fi {
    margin-right: 0.6rem;
}

.flag {
    margin-right: 0.4rem;
    font-size: 1.05em;
}

input {
    padding: 0.4rem 0.6rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    font-size: 0.82rem;
    background: var(--raised);
    outline: none;
}

input:focus {
    border-color: #3a7d5c;
}
</style>
