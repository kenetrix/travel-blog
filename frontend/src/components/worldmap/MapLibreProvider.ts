import { COUNTRY_BOUNDS } from './countryBounds'
import {
    Map as MLMap,
    GeoJSONSource,
    LngLatBounds,
    NavigationControl,
    setWorkerUrl,
} from 'maplibre-gl'
import type {
    MapProvider,
    FlightRoute,
    MapMarker,
    MapInsets,
} from './MapProvider'
import workerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url'
import { Colors } from './MapProvider'

setWorkerUrl(workerUrl)

// Quadratic Bezier bulging slightly to one side of the straight route
function curve(
    from: [number, number],
    to: [number, number],
    bend = 0.15,
    steps = 32
): [number, number][] {
    const dx = to[0] - from[0]
    const dy = to[1] - from[1]
    const control = [
        (from[0] + to[0]) / 2 - dy * bend,
        (from[1] + to[1]) / 2 + dx * bend,
    ]
    return Array.from({ length: steps + 1 }, (_, i) => {
        const t = i / steps
        const a = (1 - t) ** 2
        const b = 2 * (1 - t) * t
        const c = t ** 2
        return [
            a * from[0] + b * control[0] + c * to[0],
            a * from[1] + b * control[1] + c * to[1],
        ]
    })
}

const HIGHLIGHT_SOURCE = 'countries-state'

export class MapLibreProvider implements MapProvider {
    private map?: MLMap
    private clickCallback?: (countryCode: string) => void
    private countryLayerId = ''
    private highlighted = new Set<string>()
    private wished = new Set<string>()
    private focused = ''
    private boundsCache = new Map<string, LngLatBounds>()
    private readyCallback?: () => void
    private insets: MapInsets = { top: 80, right: 60, bottom: 80, left: 360 }

    mount(container: HTMLElement): void {
        // Keep the map hidden until our custom styling is applied, so the
        // base style's colours never flash on load.
        container.style.background = Colors.water
        this.map = new MLMap({
            container,
            style: 'https://demotiles.maplibre.org/style.json',
            center: [15, 32],
            zoom: 2.5,
            minZoom: 2,
        })

        const canvasContainer = this.map.getCanvasContainer()
        canvasContainer.style.opacity = '0'
        canvasContainer.style.transition = 'opacity 0.3s'

        this.map.addControl(
            new NavigationControl({ showCompass: false }),
            'bottom-right'
        )
        this.map.dragRotate.disable()
        this.map.keyboard.disable()
        this.map.touchZoomRotate.disableRotation()

        this.map.on('load', () => {
            // The base style (demotiles) already has a country fill layer;
            // restyle it instead of adding an error-prone source and layer of our own.
            const baseCountryLayer = this.map!.getStyle()?.layers?.find(
                (l) => l.type === 'fill' && l.id.startsWith('countries')
            )

            if (baseCountryLayer) {
                this.countryLayerId = baseCountryLayer.id

                this.map!.setPaintProperty(
                    this.countryLayerId,
                    'fill-color',
                    Colors.countryBase
                )
                this.map!.setPaintProperty(
                    'crimea-fill',
                    'fill-color',
                    Colors.countryBase
                )

                this.map!.setPaintProperty(
                    this.countryLayerId,
                    'fill-opacity',
                    0.75
                )
                this.map!.setPaintProperty('crimea-fill', 'fill-opacity', 0)
            } else {
                console.warn('No country fill layer found in the base style.')
            }

            this.map!.setPaintProperty(
                'countries-boundary',
                'line-color',
                Colors.water
            )
            this.map!.setPaintProperty(
                'countries-label',
                'text-color',
                Colors.label
            )
            this.map!.setPaintProperty(
                'countries-label',
                'text-halo-color',
                undefined
            )

            this.map!.setPaintProperty(
                'background',
                'background-color',
                Colors.water
            )
            this.map!.setPaintProperty('coastline', 'line-color', Colors.water)

            this.map!.setPaintProperty('geolines', 'line-color', Colors.geoline)
            this.map!.removeLayer('geolines-label')

            // Overlay used for the visited-country colour. It reads the same
            // tiles but promotes ADM0_A3 to the feature id so that feature-state
            // can be used.
            this.map!.addSource(HIGHLIGHT_SOURCE, {
                type: 'vector',
                url: 'https://demotiles.maplibre.org/tiles/tiles.json',
                promoteId: { countries: 'ADM0_A3' },
            })
            this.map!.addLayer(
                {
                    id: 'countries-visited',
                    type: 'fill',
                    source: HIGHLIGHT_SOURCE,
                    'source-layer': 'countries',
                    paint: {
                        'fill-color': Colors.countryHighlightOnBase,
                        'fill-opacity': [
                            'case',
                            ['boolean', ['feature-state', 'visited'], false],
                            1,
                            0,
                        ],
                    },
                },
                'countries-boundary'
            )

            this.map!.addLayer(
                {
                    id: 'countries-wishlist',
                    type: 'fill',
                    source: HIGHLIGHT_SOURCE,
                    'source-layer': 'countries',
                    paint: {
                        'fill-color': Colors.countryWishlist,
                        'fill-opacity': [
                            'case',
                            ['boolean', ['feature-state', 'wishlist'], false],
                            0.4,
                            0,
                        ],
                    },
                },
                'countries-boundary'
            )

            // Wishlist countries get a light fill plus an outline: planned,
            // not yet visited
            this.map!.addLayer(
                {
                    id: 'countries-wishlist-outline',
                    type: 'line',
                    source: HIGHLIGHT_SOURCE,
                    'source-layer': 'countries',
                    paint: {
                        'line-color': Colors.countryWishlist,
                        'line-width': 1.5,
                        'line-opacity': [
                            'case',
                            ['boolean', ['feature-state', 'wishlist'], false],
                            1,
                            0,
                        ],
                    },
                },
                'countries-label'
            )

            this.map!.addLayer({
                id: 'countries-focus',
                type: 'line',
                source: HIGHLIGHT_SOURCE,
                'source-layer': 'countries',
                paint: {
                    'line-color': Colors.countryFocus,
                    'line-width': 2.5,
                    'line-opacity': [
                        'case',
                        ['boolean', ['feature-state', 'focused'], false],
                        1,
                        0,
                    ],
                },
            })

            this.map!.addSource('flights', {
                type: 'geojson',
                data: { type: 'FeatureCollection', features: [] },
            })

            this.map!.addLayer({
                id: 'flights-line',
                type: 'line',
                source: 'flights',
                paint: {
                    'line-color': Colors.flight,
                    'line-width': 1.5,
                    'line-dasharray': [3, 2],
                    'line-opacity': 0.85,
                },
                layout: { visibility: 'none' },
            })

            // One point per route at the middle of its arc, so each route
            // gets exactly one horizontal label
            this.map!.addSource('flight-labels', {
                type: 'geojson',
                data: { type: 'FeatureCollection', features: [] },
            })

            this.map!.addLayer({
                id: 'flights-label',
                type: 'symbol',
                source: 'flight-labels',
                layout: {
                    visibility: 'none',
                    'text-field': ['get', 'label'],
                    'text-font': ['Open Sans Semibold'],
                    'text-size': 13,
                    'text-allow-overlap': true,
                    'text-ignore-placement': true,
                },
                paint: {
                    'text-color': Colors.flightLabel,
                },
            })

            if (this.countryLayerId) {
                this.map!.on('click', this.countryLayerId, (e) => {
                    const iso = e.features?.[0]?.properties?.ADM0_A3
                    if (iso && this.clickCallback) this.clickCallback(iso)
                })

                this.map!.on('mouseenter', this.countryLayerId, () => {
                    this.map!.getCanvas().style.cursor = 'pointer'
                })
                this.map!.on('mouseleave', this.countryLayerId, () => {
                    this.map!.getCanvas().style.cursor = ''
                })
            }

            this.readyCallback?.()
            this.reveal()
        })
    }

    onReady(callback: () => void): void {
        this.readyCallback = callback
    }

    // Show the map once the styled layers have actually been rendered
    private reveal(): void {
        this.map?.once('idle', () => {
            this.map!.getCanvasContainer().style.opacity = '1'
        })
    }

    destroy(): void {
        this.map?.remove()
    }

    highlightCountries(isoCodes: string[]): void {
        if (!this.map?.getSource(HIGHLIGHT_SOURCE)) return
        const next = new Set(isoCodes)

        // Feature-state changes don't reload tiles, unlike changing a
        // data-driven paint property, so large countries recolour at once.
        this.highlighted.forEach((iso) => {
            if (!next.has(iso)) this.setVisited(iso, false)
        })
        next.forEach((iso) => {
            if (!this.highlighted.has(iso)) this.setVisited(iso, true)
        })
        this.highlighted = next
    }

    highlightWishlist(isoCodes: string[]): void {
        if (!this.map?.getSource(HIGHLIGHT_SOURCE)) return
        const next = new Set(isoCodes)
        const set = (id: string, wishlist: boolean) =>
            this.map!.setFeatureState(
                { source: HIGHLIGHT_SOURCE, sourceLayer: 'countries', id },
                { wishlist }
            )
        this.wished.forEach((iso) => {
            if (!next.has(iso)) set(iso, false)
        })
        next.forEach((iso) => {
            if (!this.wished.has(iso)) set(iso, true)
        })
        this.wished = next
    }

    private setFocused(iso: string | null): void {
        if (!this.map?.getSource(HIGHLIGHT_SOURCE)) return
        const state = (id: string, focused: boolean) =>
            this.map!.setFeatureState(
                { source: HIGHLIGHT_SOURCE, sourceLayer: 'countries', id },
                { focused }
            )
        if (this.focused) state(this.focused, false)
        this.focused = iso ?? ''
        if (iso) state(iso, true)
    }

    private setVisited(iso: string, visited: boolean): void {
        this.map?.setFeatureState(
            {
                source: HIGHLIGHT_SOURCE,
                sourceLayer: 'countries',
                id: iso,
            },
            { visited }
        )
    }

    // Emphasise one route (and its label) and dim the others
    highlightFlight(id: string | null): void {
        if (!this.map?.getLayer('flights-line')) return
        const hit = ['==', ['get', 'id'], id ?? '']
        this.map.setPaintProperty('flights-line', 'line-width', [
            'case',
            hit,
            3.5,
            1.5,
        ])
        this.map.setPaintProperty('flights-line', 'line-opacity', [
            'case',
            hit,
            1,
            id ? 0.3 : 0.85,
        ])
        this.map.setLayoutProperty('flights-label', 'text-size', [
            'case',
            hit,
            16,
            13,
        ])
        this.map.setPaintProperty('flights-label', 'text-opacity', [
            'case',
            hit,
            1,
            id ? 0.4 : 1,
        ])
    }

    showFlightRoutes(routes: FlightRoute[], visible: boolean): void {
        if (!this.map?.getSource('flights')) return

        const arcs = routes.map((r) => ({ r, points: curve(r.from, r.to) }))

        ;(this.map.getSource('flights') as GeoJSONSource).setData({
            type: 'FeatureCollection',
            features: arcs.map(({ r, points }) => ({
                type: 'Feature',
                geometry: { type: 'LineString', coordinates: points },
                properties: { id: r.id },
            })),
        })
        ;(this.map.getSource('flight-labels') as GeoJSONSource).setData({
            type: 'FeatureCollection',
            features: arcs.map(({ r, points }) => ({
                type: 'Feature',
                geometry: {
                    type: 'Point',
                    coordinates: points[Math.floor(points.length / 2)],
                },
                properties: { id: r.id, label: r.label ?? '' },
            })),
        })
        for (const layer of ['flights-line', 'flights-label']) {
            this.map.setLayoutProperty(
                layer,
                'visibility',
                visible ? 'visible' : 'none'
            )
        }

        // While routes are shown, the countries step back so the routes are
        // the only strong colour: visited and wishlist countries fade
        const byState = (state: string, opacity: number) => [
            'case',
            ['boolean', ['feature-state', state], false],
            opacity,
            0,
        ]
        this.map.setPaintProperty(
            'countries-visited',
            'fill-opacity',
            byState('visited', visible ? 0.3 : 1)
        )
        this.map.setPaintProperty(
            'countries-wishlist',
            'fill-opacity',
            byState('wishlist', visible ? 0.15 : 0.4)
        )
        this.map.setPaintProperty(
            'countries-wishlist-outline',
            'line-opacity',
            byState('wishlist', visible ? 0.3 : 1)
        )
    }

    onCountryClick(callback: (countryCode: string) => void): void {
        this.clickCallback = callback
    }

    focusCountry(isoCode: string | null, insets?: MapInsets): void {
        if (insets) this.insets = insets
        if (!this.map || !this.countryLayerId) return
        this.setFocused(isoCode)
        if (!isoCode) return
        const layer = this.map.getLayer(this.countryLayerId)
        if (!layer) return

        let bounds = this.boundsCache.get(isoCode)
        if (bounds) return this.fitCountry(bounds)

        const known = COUNTRY_BOUNDS[isoCode]
        if (known) {
            bounds = new LngLatBounds(
                [known[0], known[1]],
                [known[2], known[3]]
            )
            this.boundsCache.set(isoCode, bounds)
            return this.fitCountry(bounds)
        }

        const features = this.map.querySourceFeatures(layer.source, {
            sourceLayer: layer.sourceLayer,
            filter: ['==', ['get', 'ADM0_A3'], isoCode],
        })

        // One point group per polygon, so that small overseas territories
        // and islands (Canary Islands, French Guiana, ...) can be dropped.
        type Group = { lngs: number[]; lats: number[] }
        const groups: Group[] = []
        const collect = (c: unknown, g: Group): void => {
            if (Array.isArray(c) && typeof c[0] === 'number') {
                g.lngs.push(c[0])
                g.lats.push(c[1])
            } else if (Array.isArray(c)) {
                c.forEach((x) => collect(x, g))
            }
        }
        features.forEach((f) => {
            const g = f.geometry
            if (g.type === 'MultiPolygon') {
                g.coordinates.forEach((poly: unknown) => {
                    const grp: Group = { lngs: [], lats: [] }
                    collect(poly, grp)
                    groups.push(grp)
                })
            } else if (g.type === 'Polygon') {
                const grp: Group = { lngs: [], lats: [] }
                collect(g.coordinates, grp)
                groups.push(grp)
            }
        })
        const extent = (g: Group): number =>
            (Math.max(...g.lngs) - Math.min(...g.lngs)) *
            (Math.max(...g.lats) - Math.min(...g.lats))
        const largest = Math.max(0, ...groups.map(extent))
        const main = groups.filter((g) => extent(g) >= largest * 0.05)
        const lngs = main.flatMap((g) => g.lngs)
        const lats = main.flatMap((g) => g.lats)
        if (!lngs.length) return

        bounds = this.computeBounds(lngs, lats)
        this.boundsCache.set(isoCode, bounds)
        this.fitCountry(bounds)
    }

    // Bounds of a country, handling shapes that cross the antimeridian
    // (e.g. Russia), which would otherwise span the whole world.
    private computeBounds(lngs: number[], lats: number[]): LngLatBounds {
        let xs = lngs
        if (Math.min(...xs) < -150 && Math.max(...xs) > 150) {
            xs = xs.map((x) => (x < 0 ? x + 360 : x))
            if (Math.max(...xs) - Math.min(...xs) > 100) {
                // Ignore far-flung outliers (e.g. the Aleutian Islands)
                const sorted = [...xs].sort((a, b) => a - b)
                const cut = Math.floor(sorted.length * 0.02)
                xs = sorted.slice(cut, sorted.length - cut)
            }
        }
        return new LngLatBounds(
            [Math.min(...xs), Math.min(...lats)],
            [Math.max(...xs), Math.max(...lats)]
        )
    }

    private fitCountry(bounds: LngLatBounds): void {
        if (!this.map) return
        // Leave room for the panel or sheet covering the map. Very large
        // countries (e.g. Russia) may be overlapped by it, so they get less
        // padding there and always zoom in at least a bit.
        const ne = bounds.getNorthEast()
        const sw = bounds.getSouthWest()
        const isLarge = ne.lng - sw.lng > 50 || ne.lat - sw.lat > 30
        const { clientWidth: width, clientHeight: height } =
            this.map.getContainer()
        const padding = { ...this.insets }
        if (isLarge) {
            padding.left = Math.min(padding.left, 120)
            padding.bottom = Math.min(padding.bottom, height * 0.3)
        }
        // Keep at least 120px of map to fit the country into
        const shrinkX = Math.min(
            1,
            (width - 120) / (padding.left + padding.right)
        )
        const shrinkY = Math.min(
            1,
            (height - 120) / (padding.top + padding.bottom)
        )
        padding.left *= Math.max(0, shrinkX)
        padding.right *= Math.max(0, shrinkX)
        padding.top *= Math.max(0, shrinkY)
        padding.bottom *= Math.max(0, shrinkY)
        const camera = this.map.cameraForBounds(bounds, {
            padding,
            maxZoom: 8,
        })
        if (!camera || !camera.center) return
        const zoom = camera.zoom ?? 3
        this.map.easeTo({
            center: camera.center,
            zoom: isLarge ? Math.max(zoom, 3) : zoom,
        })
    }

    flyTo(lng: number, lat: number, zoom: number): void {
        this.map?.flyTo({ center: [lng, lat], zoom })
    }

    addMarker(_marker: MapMarker): void {}
}
