// Writes public/places/<ISO3>.json from the GeoNames dump of places with at
// least 500 inhabitants (CC BY 4.0, https://www.geonames.org). Run when
// updating the place names:
//   download and unzip https://download.geonames.org/export/dump/cities500.zip
//   node scripts/places.ts <path to cities500.txt>
// Each file is a JSON array of [name, lat, lng, population], largest first.
import { mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const input = process.argv[2]
if (!input) {
    console.error('usage: node scripts/places.ts <cities500.txt>')
    process.exit(1)
}

const root = fileURLToPath(new URL('..', import.meta.url))
const outDir = `${root}public/places/`

const countries: Record<string, string>[] = JSON.parse(
    readFileSync(`${root}src/components/countries/countries.json`, 'utf8')
)
const alpha3 = new Map(countries.map((c) => [c['alpha-2'], c['alpha-3']]))
// Kosovo has no ISO code; the map's country layer calls it KOS
alpha3.set('XK', 'KOS')

// Historical, abandoned and destroyed places, and parts of other places
const SKIP = new Set(['PPLH', 'PPLQ', 'PPLW', 'PPLX', 'PPLCH'])

type Row = [string, number, number, number]
const byCountry = new Map<string, Row[]>()
const unknown = new Map<string, number>()

for (const line of readFileSync(input, 'utf8').split('\n')) {
    const f = line.split('\t')
    if (f.length < 15 || SKIP.has(f[7])) continue
    const iso = alpha3.get(f[8])
    if (!iso) {
        unknown.set(f[8], (unknown.get(f[8]) ?? 0) + 1)
        continue
    }
    // About 10 m; more would only make the files bigger
    const round = (n: string) => Math.round(Number(n) * 1e4) / 1e4
    const row: Row = [f[1], round(f[4]), round(f[5]), Number(f[14])]
    const rows = byCountry.get(iso) ?? []
    rows.push(row)
    byCountry.set(iso, rows)
}

rmSync(outDir, { recursive: true, force: true })
mkdirSync(outDir, { recursive: true })
let total = 0
for (const [iso, rows] of byCountry) {
    rows.sort((a, b) => b[3] - a[3] || a[0].localeCompare(b[0]))
    writeFileSync(`${outDir}${iso}.json`, JSON.stringify(rows) + '\n')
    total += rows.length
}
console.log(`${total} places in ${byCountry.size} files under ${outDir}`)
if (unknown.size) {
    console.log('Skipped country codes without an alpha-3 code:', [...unknown])
}
