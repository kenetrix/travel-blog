// Writes src/i18n/data/{capitals,airports}.{de,es,ru}.json from Wikidata labels.
// Run once when countryFacts.ts or airports.json change:
//   node scripts/i18n-data.ts
// Only names that differ from the English ones are written; the app falls
// back to English for everything else.
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const LANGUAGES = ['de', 'es', 'ru'] as const
type Lang = (typeof LANGUAGES)[number]

const root = fileURLToPath(new URL('..', import.meta.url))
const dataDir = `${root}src/i18n/data/`

type Binding = Record<string, { value: string } | undefined>

async function sparql(query: string): Promise<Binding[]> {
    const response = await fetch('https://query.wikidata.org/sparql', {
        method: 'POST',
        headers: {
            Accept: 'application/sparql-results+json',
            'Content-Type': 'application/x-www-form-urlencoded',
            'User-Agent': 'travel-blog i18n-data script',
        },
        body: new URLSearchParams({ query }),
    })
    if (!response.ok) {
        throw new Error(`Wikidata: ${response.status} ${await response.text()}`)
    }
    return (await response.json()).results.bindings
}

const labels = (v: string) =>
    LANGUAGES.map(
        (l) =>
            `OPTIONAL { ${v} rdfs:label ?${v.slice(1)}_${l} FILTER(LANG(?${v.slice(1)}_${l}) = "${l}") }`
    ).join('\n')

function write(name: string, lang: Lang, entries: [string, unknown][]) {
    entries.sort(([a], [b]) => a.localeCompare(b))
    const file = `${dataDir}${name}.${lang}.json`
    writeFileSync(
        file,
        JSON.stringify(Object.fromEntries(entries), null, 1) + '\n'
    )
    console.log(`${file}: ${entries.length} entries`)
}

// Several candidates per key: prefer the one whose English label matches ours
function pick(rows: Binding[], englishVar: string, english: string) {
    return rows.find((r) => r[englishVar]?.value === english) ?? rows[0]
}

function group(rows: Binding[], key: string) {
    const out = new Map<string, Binding[]>()
    for (const r of rows) {
        const k = r[key]?.value
        if (k) out.set(k, [...(out.get(k) ?? []), r])
    }
    return out
}

async function capitals() {
    const source = readFileSync(
        `${root}src/components/countries/countryFacts.ts`,
        'utf8'
    )
    const ours = new Map(
        [...source.matchAll(/\[\s*'([A-Z]{3})',\s*'((?:[^'\\]|\\.)*)'/g)].map(
            (m) => [m[1], m[2].replace(/\\'/g, "'")]
        )
    )
    const rows = await sparql(`
        SELECT ?iso ?cap_en ${LANGUAGES.map((l) => `?cap_${l}`).join(' ')} WHERE {
            ?country wdt:P298 ?iso; wdt:P36 ?cap.
            OPTIONAL { ?cap rdfs:label ?cap_en FILTER(LANG(?cap_en) = "en") }
            ${labels('?cap')}
        }`)
    const byIso = group(rows, 'iso')
    for (const lang of LANGUAGES) {
        const entries: [string, string][] = []
        for (const [iso, capital] of ours) {
            const candidates = byIso.get(iso)
            if (!capital || !candidates) continue
            const name = pick(candidates, 'cap_en', capital)[`cap_${lang}`]
                ?.value
            if (name && name !== capital) entries.push([iso, name])
        }
        write('capitals', lang, entries)
    }
}

async function airports() {
    const ours = JSON.parse(
        readFileSync(`${root}src/components/flights/airports.json`, 'utf8')
    ) as [string, string, string, string, number, number, string][]
    const rows: Binding[] = []
    // Large VALUES lists time out; a few hundred codes per query are fine
    for (let i = 0; i < ours.length; i += 300) {
        const codes = ours.slice(i, i + 300).map(([iata]) => `"${iata}"`)
        rows.push(
            ...(await sparql(`
                SELECT ?iata ?city_en ${LANGUAGES.map((l) => `?airport_${l} ?city_${l}`).join(' ')} WHERE {
                    VALUES ?iata { ${codes.join(' ')} }
                    ?airport wdt:P238 ?iata.
                    FILTER NOT EXISTS { ?airport wdt:P576 [] }
                    ${labels('?airport')}
                    OPTIONAL {
                        ?airport wdt:P931 ?city.
                        OPTIONAL { ?city rdfs:label ?city_en FILTER(LANG(?city_en) = "en") }
                        ${labels('?city')}
                    }
                }`))
        )
    }
    const byIata = group(rows, 'iata')
    for (const lang of LANGUAGES) {
        const entries: [string, [string, string]][] = []
        for (const [iata, name, city] of ours) {
            const candidates = byIata.get(iata)
            if (!candidates) continue
            const best = pick(candidates, 'city_en', city)
            const localCity = best[`city_${lang}`]?.value ?? ''
            // Some labels are only the city ("Токио" for Haneda); English says more
            let localName = best[`airport_${lang}`]?.value ?? ''
            if (localName === localCity) localName = ''
            // '' keeps the English text for that part
            const entry: [string, string] = [
                localName !== name ? localName : '',
                localCity !== city ? localCity : '',
            ]
            if (entry[0] || entry[1]) entries.push([iata, entry])
        }
        write('airports', lang, entries)
    }
}

await capitals()
await airports()
