<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useTravelStore } from '../stores/store'
import { storeToRefs } from 'pinia'
import { Countries, getCountryByIso, type Country } from './countries'
import CountryDetailModal from './CountryDetailModal.vue'
import AppSheet from '../ui/AppSheet.vue'
import { useLayout } from '../ui/useLayout'

const store = useTravelStore()
const { layout } = useLayout()
const { visited, wishlist, selectedIso, details } = storeToRefs(store)

// Phones: the sheet opens fully for the details and for a search
// Phones start with the map; side sheets start open
const expanded = ref(layout.value !== 'phone')
watch(selectedIso, (iso) => iso && (expanded.value = true))

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
    <AppSheet
        v-model:expanded="expanded"
        :wide="!!selectedIso"
        :label="selectedIso ? 'Country details' : 'Countries'"
    >
        <CountryDetailModal
            v-if="selectedIso"
            :iso="selectedIso"
            @close="store.selectCountry(null)"
        />
        <template v-else>
            <div>
                <h2>Countries</h2>
                <p class="sub">Visited countries in the world</p>
            </div>

            <div>
                <input
                    v-model="searchCountry"
                    type="search"
                    placeholder="Search for country"
                    aria-label="Search for country"
                    @focus="expanded = true"
                />
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
                    >
                        <button
                            type="button"
                            @click="store.selectCountry(country.alpha3)"
                        >
                            <span
                                :class="`fi fi-${country.alpha2.toLowerCase()}`"
                            ></span>
                            {{ country.name }}
                        </button>
                    </li>
                </ol>
            </div>

            <div>
                <label>{{ query ? 'Results' : 'Visited' }}</label>
                <ol>
                    <li
                        v-for="country in listedCountries"
                        :key="country.alpha3"
                    >
                        <button
                            type="button"
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
                                v-else-if="
                                    query && wishlist.has(country.alpha3)
                                "
                                class="wish"
                            >
                                ☆
                            </span>
                            <span
                                v-if="details[country.alpha3]?.rating"
                                class="rating"
                            >
                                {{ '★'.repeat(details[country.alpha3].rating) }}
                            </span>
                        </button>
                    </li>
                    <li v-if="!listedCountries.length" class="empty">
                        No countries found
                    </li>
                </ol>
            </div>
        </template>
    </AppSheet>
</template>

<style scoped>
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
    margin: 0;
    padding: 0;
    list-style-type: none;
}

li button {
    display: flex;
    align-items: center;
    width: 100%;
    min-height: var(--touch);
    padding: 0;
    border: none;
    background: none;
    font: inherit;
    font-size: 1rem;
    font-weight: 500;
    text-align: left;
    color: var(--muted);
    cursor: pointer;
}

li button:hover {
    color: var(--text);
}

/* A mouse needs no 44px rows; keep the denser desktop list */
@media (pointer: fine) {
    li button {
        min-height: 1.6rem;
    }
}

/* Desktop: the panel grows with the list up to half the screen */
@media (min-width: 1024px) and (min-height: 501px) {
    ol {
        max-height: 50vh;
        overflow-y: auto;
    }
}

.tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
}

.tag {
    min-height: 2rem;
    padding: 0.1rem 0.7rem;
    border: 1px solid var(--line-strong);
    border-radius: 1rem;
    background: var(--raised);
    font-size: 0.75rem;
    color: var(--muted-strong);
    cursor: pointer;
}

@media (pointer: fine) {
    .tag {
        min-height: 0;
        padding: 0.1rem 0.5rem;
        font-size: 0.72rem;
    }
}

.tag.on {
    background: var(--accent-soft);
    border-color: #3a7d5c;
    color: #2f6a4e;
}

.wish {
    color: var(--wishlist);
    font-size: 0.8em;
    margin-left: 0.3rem;
}

.rating {
    color: #ca8f22;
    font-size: 0.65em;
    margin-left: 0.3rem;
}

li.empty {
    font-size: 0.82rem;
    color: var(--muted);
}

.check {
    color: #3a7d5c;
    font-size: 0.8em;
    margin-left: 0.3rem;
}

.fi {
    margin-right: 0.6rem;
}

input {
    box-sizing: border-box;
    width: 100%;
    padding: 0.5rem 0.6rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    background: var(--raised);
    color: var(--text);
    outline: none;
}

input:focus {
    border-color: #3a7d5c;
}
</style>
