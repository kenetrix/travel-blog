// Writes src/components/worldmap/lakes.json from the Natural Earth 1:50m lakes
// (public domain, https://www.naturalearthdata.com). The base map leaves some
// large lakes out (e.g. Lake Onega) or loses them at low zoom (Lake Ladoga),
// so the map draws them itself. Run when updating the lakes:
//   download https://raw.githubusercontent.com/nvkelso/natural-earth-vector/master/geojson/ne_50m_lakes.geojson
//   node scripts/lakes.ts <path to ne_50m_lakes.geojson>
// The output is a GeoJSON FeatureCollection without properties.
import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const input = process.argv[2]
if (!input) {
    console.error('usage: node scripts/lakes.ts <ne_50m_lakes.geojson>')
    process.exit(1)
}

// Smaller lakes are hardly visible up to zoom 8 and would only grow the bundle
const MIN_AREA_KM2 = 1000
// About 100 m, finer than a pixel up to zoom 8
const DIGITS = 3

type Ring = [number, number][]
type Geometry =
    | { type: 'Polygon'; coordinates: Ring[] }
    | { type: 'MultiPolygon'; coordinates: Ring[][] }

const root = fileURLToPath(new URL('..', import.meta.url))
const lakes: { features: { geometry: Geometry }[] } = JSON.parse(
    readFileSync(input, 'utf8')
)

// Shoelace area on an equirectangular projection around the ring's latitude
function areaKm2(ring: Ring): number {
    const lat = ring.reduce((sum, [, y]) => sum + y, 0) / ring.length
    const kx = 111.32 * Math.cos((lat * Math.PI) / 180)
    let twice = 0
    for (let i = 0; i < ring.length; i++) {
        const [x1, y1] = ring[i]
        const [x2, y2] = ring[(i + 1) % ring.length]
        twice += x1 * kx * y2 * 110.57 - x2 * kx * y1 * 110.57
    }
    return Math.abs(twice) / 2
}

const scale = 10 ** DIGITS
function round(ring: Ring): Ring {
    const out: Ring = []
    for (const [x, y] of ring) {
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

const polygons = lakes.features
    .flatMap(({ geometry: g }) =>
        g.type === 'Polygon' ? [g.coordinates] : g.coordinates
    )
    .filter((polygon) => areaKm2(polygon[0]) >= MIN_AREA_KM2)
    .map((polygon) => polygon.map(round).filter((ring) => ring.length >= 4))

writeFileSync(
    `${root}src/components/worldmap/lakes.json`,
    JSON.stringify({
        type: 'FeatureCollection',
        features: polygons.map((coordinates) => ({
            type: 'Feature',
            properties: {},
            geometry: { type: 'Polygon', coordinates },
        })),
    })
)
console.log(`${polygons.length} lakes`)
