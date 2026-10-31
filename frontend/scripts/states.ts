// Writes public/states/<ISO3>.json from the Natural Earth 1:10m admin-1
// boundary lines (public domain, https://www.naturalearthdata.com). "States"
// stands for each country's first-level divisions: Bundesländer, US states,
// provinces, oblasts, ... Run when updating the borders:
//   download https://raw.githubusercontent.com/nvkelso/natural-earth-vector/master/geojson/ne_10m_admin_1_states_provinces_lines.geojson
//   node scripts/states.ts <path to ne_10m_admin_1_states_provinces_lines.geojson>
// Each file is { main: Line[], minor: Line[] }, a Line being [lng, lat][].
import { mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const input = process.argv[2]
if (!input) {
    console.error('usage: node scripts/states.ts <admin_1 lines .geojson>')
    process.exit(1)
}

// Douglas-Peucker tolerance in degrees, about 500 m: below a pixel up to zoom 8
const TOLERANCE = 0.005
const DIGITS = 3

type Line = [number, number][]
type Feature = {
    properties: { FEATURECLA: string; ADM0_A3: string }
    geometry:
        | { type: 'LineString'; coordinates: Line }
        | { type: 'MultiLineString'; coordinates: Line[] }
        | null
}

const root = fileURLToPath(new URL('..', import.meta.url))
const outDir = `${root}public/states/`
const { features }: { features: Feature[] } = JSON.parse(
    readFileSync(input, 'utf8')
)

// Natural Earth files some countries' first-level divisions (French régions,
// Italian regioni, ...) as region boundaries and their subdivisions
// (départements, provinces) as admin-1 boundaries. Where a country has region
// boundaries, they are the main borders and the rest are minor ones.
// Statistical boundaries are no administrative borders and are left out.
const ADMIN = 'Admin-1 boundary'
const REGION = 'Admin-1 region boundary'

function distance(
    p: [number, number],
    a: [number, number],
    b: [number, number]
) {
    const dx = b[0] - a[0]
    const dy = b[1] - a[1]
    const length = dx * dx + dy * dy
    const t = length
        ? Math.max(
              0,
              Math.min(1, ((p[0] - a[0]) * dx + (p[1] - a[1]) * dy) / length)
          )
        : 0
    return Math.hypot(p[0] - a[0] - t * dx, p[1] - a[1] - t * dy)
}

function simplify(line: Line): Line {
    if (line.length < 3) return line
    let worst = 0
    let index = 0
    for (let i = 1; i < line.length - 1; i++) {
        const d = distance(line[i], line[0], line[line.length - 1])
        if (d > worst) {
            worst = d
            index = i
        }
    }
    if (worst <= TOLERANCE) return [line[0], line[line.length - 1]]
    return [
        ...simplify(line.slice(0, index + 1)).slice(0, -1),
        ...simplify(line.slice(index)),
    ]
}

const scale = 10 ** DIGITS
function compact(line: Line): Line {
    const out: Line = []
    for (const [x, y] of simplify(line)) {
        const point: [number, number] = [
            Math.round(x * scale) / scale,
            Math.round(y * scale) / scale,
        ]
        const last = out[out.length - 1]
        if (!last || last[0] !== point[0] || last[1] !== point[1]) {
            out.push(point)
        }
    }
    return out
}

const byCountry = new Map<string, { admin: Line[]; region: Line[] }>()
for (const { properties: p, geometry: g } of features) {
    if (!g || (p.FEATURECLA !== ADMIN && p.FEATURECLA !== REGION)) continue
    let country = byCountry.get(p.ADM0_A3)
    if (!country)
        byCountry.set(p.ADM0_A3, (country = { admin: [], region: [] }))
    const lines = g.type === 'LineString' ? [g.coordinates] : g.coordinates
    const target = p.FEATURECLA === REGION ? country.region : country.admin
    target.push(...lines.map(compact).filter((l) => l.length > 1))
}

rmSync(outDir, { recursive: true, force: true })
mkdirSync(outDir, { recursive: true })
let bytes = 0
for (const [iso, { admin, region }] of byCountry) {
    const json = JSON.stringify(
        region.length
            ? { main: region, minor: admin }
            : { main: admin, minor: [] }
    )
    bytes += json.length
    writeFileSync(`${outDir}${iso}.json`, json)
}
console.log(`${byCountry.size} countries, ${Math.round(bytes / 1024)} KiB`)
