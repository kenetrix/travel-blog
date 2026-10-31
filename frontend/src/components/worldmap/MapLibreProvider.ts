import { COUNTRY_BOUNDS } from './countryBounds'
import { abbreviate, countryAreas } from './countryLabels'
import {
    Map as MLMap,
    GeoJSONSource,
    LngLatBounds,
    Marker,
    MercatorCoordinate,
    NavigationControl,
    setWorkerUrl,
    type ExpressionSpecification,
} from 'maplibre-gl'
import type {
    MapProvider,
    FlightRoute,
    MapMarker,
    MapInsets,
    MapPlace,
    MapPointInfo,
    TripStopPoint,
} from './MapProvider'
import workerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url'
import { Colors } from './MapProvider'
import { pinElement } from './pin'
import type { FeatureCollection } from 'geojson'

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
// Half the side of the square around a click that may hit a route
const ROUTE_HIT_PX = 12
// The same for a place name
const PLACE_HIT_PX = 6

// Base map (Natural Earth) names of countries too small for their label to
// fit at low zoom: tier 1 is named from zoom 5, tier 2 from zoom 6.
const SMALL_COUNTRIES_TIER1 = [
    'Albania',
    'Belgium',
    'Bosnia and Herz.',
    'Costa Rica',
    'Croatia',
    'Denmark',
    'El Salvador',
    'Estonia',
    'Haiti',
    'Israel',
    'Jamaica',
    'Kosovo',
    'Kuwait',
    'Latvia',
    'Lebanon',
    'Lithuania',
    'Macedonia',
    'Moldova',
    'Montenegro',
    'Netherlands',
    'Qatar',
    'Slovenia',
    'Switzerland',
    'Taiwan',
    'Vanuatu',
]
const SMALL_COUNTRIES_TIER2 = [
    'Andorra',
    'Antigua and Barb.',
    'Bahrain',
    'Barbados',
    'Brunei',
    'Cyprus',
    'Dominica',
    'Fiji',
    'Grenada',
    'Liechtenstein',
    'Luxembourg',
    'Malta',
    'Mauritius',
    'Monaco',
    'N. Cyprus',
    'Palestine',
    'San Marino',
    'Singapore',
    'St. Kitts and Nevis',
    'St. Vin. and Gren.',
    'Vatican',
]

// Place names by population: big cities from zoom 5, villages from zoom 11.
// Listed smallest first, so the biggest places end up on top and win when
// labels collide.
const PLACE_TIERS = [
    { id: 'places-tiny', min: 0, max: 1_500, minZoom: 11, size: 10 },
    { id: 'places-village', min: 1_500, max: 5_000, minZoom: 10, size: 10.5 },
    { id: 'places-small', min: 5_000, max: 20_000, minZoom: 9, size: 11 },
    { id: 'places-town', min: 20_000, max: 100_000, minZoom: 8, size: 11.5 },
    { id: 'places-medium', min: 100_000, max: 500_000, minZoom: 7, size: 12 },
    { id: 'places-large', min: 500_000, max: 1e10, minZoom: 5, size: 13 },
]
const PLACE_LAYERS = PLACE_TIERS.map((t) => t.id)

const EMPTY: FeatureCollection = {
    type: 'FeatureCollection',
    features: [],
}

export class MapLibreProvider implements MapProvider {
    private map?: MLMap
    private clickCallback?: (countryCode: string) => void
    private flightClickCallback?: (id: string | null) => void
    private mapClickCallback?: (
        point: MapPointInfo,
        place: MapPlace | null
    ) => void
    private moveEndCallback?: () => void
    private pin?: Marker
    private placesVisible = false
    // Drawn arc per route id, for hit tests and fitting the map to a route
    private arcs = new Map<string, [number, number][]>()
    private routesVisible = false
    private countryLayerId = ''
    private highlighted = new Set<string>()
    private wished = new Set<string>()
    private focused = ''
    private boundsCache = new Map<string, LngLatBounds>()
    private readyCallback?: () => void
    private loaded = false
    private countryNames: Record<string, string> | null = null
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
            attributionControl: false,
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
            // Bigger countries are placed first and win label collisions
            const bySize = [
                'match',
                ['get', 'NAME'],
                ...Object.entries(countryAreas()).flatMap(([name, area]) => [
                    name,
                    -area,
                ]),
                0,
            ]
            this.map!.setLayoutProperty(
                'countries-label',
                'symbol-sort-key',
                bySize as ExpressionSpecification
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

            // Place names, filled while a trip is edited
            this.map!.addSource('places', {
                type: 'geojson',
                data: EMPTY,
            })
            for (const tier of PLACE_TIERS) {
                this.map!.addLayer({
                    id: tier.id,
                    type: 'symbol',
                    source: 'places',
                    minzoom: tier.minZoom,
                    filter: [
                        'all',
                        ['>=', ['get', 'p'], tier.min],
                        ['<', ['get', 'p'], tier.max],
                    ],
                    layout: {
                        visibility: 'none',
                        'text-field': ['get', 'name'],
                        'text-font': ['Open Sans Semibold'],
                        'text-size': tier.size,
                        'text-padding': 4,
                        'symbol-sort-key': ['-', 0, ['get', 'p']],
                    },
                    paint: {
                        'text-color': Colors.place,
                        // Dark outline keeps names readable on the gold and violet fills
                        'text-halo-color': Colors.countryBase,
                        'text-halo-width': 2,
                        'text-halo-blur': 0.5,
                    },
                })
            }

            // The open trip: a line in visiting order and numbered stops
            this.map!.addSource('trip', { type: 'geojson', data: EMPTY })
            const isLine: ExpressionSpecification = [
                '==',
                ['geometry-type'],
                'LineString',
            ]
            const isPoint: ExpressionSpecification = [
                '==',
                ['geometry-type'],
                'Point',
            ]
            // All trips as thin lines, below the open trip
            this.map!.addSource('trip-routes', { type: 'geojson', data: EMPTY })
            this.map!.addLayer({
                id: 'trip-routes-line',
                type: 'line',
                source: 'trip-routes',
                filter: isLine,
                layout: { 'line-join': 'round', 'line-cap': 'round' },
                paint: {
                    'line-color': Colors.trip,
                    'line-width': 2.5,
                },
            })
            this.map!.addLayer({
                id: 'trip-routes-dots',
                type: 'circle',
                source: 'trip-routes',
                filter: isPoint,
                paint: {
                    'circle-radius': 3.5,
                    'circle-color': Colors.trip,
                },
            })
            this.map!.addLayer({
                id: 'trip-line',
                type: 'line',
                source: 'trip',
                filter: isLine,
                layout: { 'line-join': 'round', 'line-cap': 'round' },
                paint: {
                    'line-color': Colors.trip,
                    'line-width': 3,
                },
            })
            this.map!.addLayer({
                id: 'trip-stops',
                type: 'circle',
                source: 'trip',
                filter: isPoint,
                paint: {
                    'circle-radius': 10,
                    'circle-color': Colors.trip,
                    'circle-stroke-color': Colors.water,
                    'circle-stroke-width': 1.5,
                },
            })
            this.map!.addLayer({
                id: 'trip-numbers',
                type: 'symbol',
                source: 'trip',
                filter: isPoint,
                layout: {
                    'text-field': ['to-string', ['get', 'n']],
                    'text-font': ['Open Sans Semibold'],
                    'text-size': 11,
                    'text-allow-overlap': true,
                    'text-ignore-placement': true,
                },
                paint: { 'text-color': Colors.tripNumber },
            })

            this.map!.on('click', (e) => {
                if (!this.mapClickCallback) return
                const point = this.pointAt(e.point.x, e.point.y)
                if (point) this.mapClickCallback(point, this.placeAt(e.point))
            })
            this.map!.on('moveend', () => this.moveEndCallback?.())

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

            // Routes are thin dashed lines; a tap anywhere near one counts
            this.map!.on('click', (e) => {
                if (!this.routesVisible) return
                this.flightClickCallback?.(this.routeAt(e.point))
            })
            this.map!.on('mousemove', (e) => {
                if (!this.routesVisible) return
                this.map!.getCanvas().classList.toggle(
                    'route-hover',
                    this.routeAt(e.point) !== null
                )
            })

            this.loaded = true
            this.applyCountryNames()
            this.readyCallback?.()
            this.reveal()
        })
    }

    setCountryNames(names: Record<string, string> | null): void {
        this.countryNames = names
        if (this.loaded) this.applyCountryNames()
    }

    // The base style shows abbreviations ("Ger.") below zoom 4; translated
    // ones are derived from the translated names
    private applyCountryNames(): void {
        const names = this.countryNames
        const full = names
            ? [
                  'match',
                  ['get', 'NAME'],
                  ...Object.entries(names).flat(),
                  ['get', 'NAME'],
              ]
            : ['get', 'NAME']
        const short = names
            ? [
                  'match',
                  ['get', 'NAME'],
                  ...Object.entries(names).flatMap(([base, local]) => [
                      base,
                      abbreviate(local),
                  ]),
                  ['get', 'ABBREV'],
              ]
            : ['get', 'ABBREV']
        // Labels of small countries are blanked until zoomed in, where they
        // no longer collide. Zoom can only be the input of a top-level step.
        const hide = (label: unknown, hidden: string[]) => [
            'case',
            ['in', ['get', 'NAME'], ['literal', hidden]],
            '',
            label,
        ]
        const name = [
            'step',
            ['zoom'],
            hide(short, [...SMALL_COUNTRIES_TIER1, ...SMALL_COUNTRIES_TIER2]),
            4,
            hide(full, [...SMALL_COUNTRIES_TIER1, ...SMALL_COUNTRIES_TIER2]),
            5,
            hide(full, SMALL_COUNTRIES_TIER2),
            6,
            full,
        ]
        this.map!.setLayoutProperty(
            'countries-label',
            'text-field',
            name as ExpressionSpecification
        )
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

    // Id of the route within ROUTE_HIT_PX of a screen point, if any
    private routeAt(point: { x: number; y: number }): string | null {
        if (!this.map) return null
        const { x, y } = point
        const r = ROUTE_HIT_PX
        const hits = this.map.queryRenderedFeatures(
            [
                [x - r, y - r],
                [x + r, y + r],
            ],
            { layers: ['flights-line', 'flights-label'] }
        )
        return (hits[0]?.properties?.id as string | undefined) ?? null
    }

    onFlightClick(callback: (id: string | null) => void): void {
        this.flightClickCallback = callback
    }

    focusFlight(id: string | null, insets?: MapInsets): void {
        if (insets) this.insets = insets
        const points = id ? this.arcs.get(id) : undefined
        if (!points) return
        const lngs = points.map((p) => p[0])
        const lats = points.map((p) => p[1])
        this.fitInsets(
            new LngLatBounds(
                [Math.min(...lngs), Math.min(...lats)],
                [Math.max(...lngs), Math.max(...lats)]
            ),
            // The ends are airports: keep them clear of the edges
            { maxZoom: 7, squeezeLarge: false, margin: 32 }
        )
    }

    showFlightRoutes(routes: FlightRoute[], visible: boolean): void {
        if (!this.map?.getSource('flights')) return

        const arcs = routes.map((r) => ({ r, points: curve(r.from, r.to) }))
        this.arcs = new Map(arcs.map(({ r, points }) => [r.id, points]))
        this.routesVisible = visible
        if (!visible) this.map.getCanvas().classList.remove('route-hover')

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
        const bounds = this.countryBounds(isoCode)
        if (bounds) this.fitCountryBounds(bounds)
    }

    fitCountry(isoCode: string, insets?: MapInsets): void {
        if (insets) this.insets = insets
        const bounds = this.countryBounds(isoCode)
        if (bounds) this.fitCountryBounds(bounds)
    }

    private countryBounds(isoCode: string): LngLatBounds | undefined {
        if (!this.map || !this.countryLayerId) return
        const layer = this.map.getLayer(this.countryLayerId)
        if (!layer) return

        let bounds = this.boundsCache.get(isoCode)
        if (bounds) return bounds

        const known = COUNTRY_BOUNDS[isoCode]
        if (known) {
            bounds = new LngLatBounds(
                [known[0], known[1]],
                [known[2], known[3]]
            )
            this.boundsCache.set(isoCode, bounds)
            return bounds
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
        return bounds
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

    private fitCountryBounds(bounds: LngLatBounds): void {
        this.fitInsets(bounds, { maxZoom: 8, squeezeLarge: true })
    }

    private fitInsets(
        bounds: LngLatBounds,
        {
            maxZoom,
            squeezeLarge,
            margin = 0,
        }: { maxZoom: number; squeezeLarge: boolean; margin?: number }
    ): void {
        if (!this.map) return
        // Leave room for the panel or sheet covering the map. On desktops very
        // large countries (e.g. Russia) may be overlapped by the floating
        // panel, so they get less padding there and zoom in at least a bit.
        // Tablet and phone sheets take too much of the screen for that: the
        // country is fitted into the free area, even if that zooms out.
        const ne = bounds.getNorthEast()
        const sw = bounds.getSouthWest()
        const isLarge =
            squeezeLarge && (ne.lng - sw.lng > 50 || ne.lat - sw.lat > 30)
        const { clientWidth: width, clientHeight: height } =
            this.map.getContainer()
        const padding = {
            top: this.insets.top + margin,
            right: this.insets.right + margin,
            bottom: this.insets.bottom + margin,
            left: this.insets.left + margin,
        }
        const desktop = width >= 1024
        if (isLarge && desktop) padding.left = Math.min(padding.left, 120)
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
        const camera = this.map.cameraForBounds(bounds, { padding, maxZoom })
        if (!camera) return
        let zoom = camera.zoom ?? 3
        if (isLarge && desktop) zoom = Math.max(zoom, 3)
        // The map never zooms out past its minimum; on a phone the strip
        // above the sheet often wants less
        zoom = Math.max(zoom, this.map.getMinZoom())
        // cameraForBounds' centre only fits its own zoom. Recompute it for
        // the final one: the country's middle sits in the middle of the free
        // area, i.e. off the screen centre by half the padding difference.
        const a = MercatorCoordinate.fromLngLat(bounds.getSouthWest())
        const b = MercatorCoordinate.fromLngLat(bounds.getNorthEast())
        const world = 512 * 2 ** zoom
        const center = new MercatorCoordinate(
            (a.x + b.x) / 2 - (padding.left - padding.right) / 2 / world,
            (a.y + b.y) / 2 - (padding.top - padding.bottom) / 2 / world
        ).toLngLat()
        this.map.easeTo({ center, zoom })
    }

    flyTo(lng: number, lat: number, zoom: number): void {
        this.map?.flyTo({ center: [lng, lat], zoom })
    }

    addMarker(_marker: MapMarker): void {}

    showTripRoutes(routes: TripStopPoint[][]): void {
        const source = this.map?.getSource('trip-routes') as
            GeoJSONSource | undefined
        source?.setData({
            type: 'FeatureCollection',
            features: routes.flatMap((stops) => [
                ...(stops.length > 1
                    ? [
                          {
                              type: 'Feature' as const,
                              geometry: {
                                  type: 'LineString' as const,
                                  coordinates: stops.map((s) => [s.lng, s.lat]),
                              },
                              properties: {},
                          },
                      ]
                    : []),
                ...stops.map((s) => ({
                    type: 'Feature' as const,
                    geometry: {
                        type: 'Point' as const,
                        coordinates: [s.lng, s.lat],
                    },
                    properties: {},
                })),
            ]),
        })
    }

    showTrip(stops: TripStopPoint[] | null): void {
        const source = this.map?.getSource('trip') as GeoJSONSource | undefined
        if (!source) return
        const points = (stops ?? []).map((s) => [s.lng, s.lat])
        source.setData({
            type: 'FeatureCollection',
            features: [
                ...(points.length > 1
                    ? [
                          {
                              type: 'Feature' as const,
                              geometry: {
                                  type: 'LineString' as const,
                                  coordinates: points,
                              },
                              properties: {},
                          },
                      ]
                    : []),
                ...points.map((coordinates, i) => ({
                    type: 'Feature' as const,
                    geometry: { type: 'Point' as const, coordinates },
                    properties: { n: i + 1 },
                })),
            ],
        })
    }

    fitPoints(points: TripStopPoint[], insets?: MapInsets): void {
        if (insets) this.insets = insets
        if (!points.length) return
        const lngs = points.map((p) => p.lng)
        const lats = points.map((p) => p.lat)
        this.fitInsets(
            new LngLatBounds(
                [Math.min(...lngs), Math.min(...lats)],
                [Math.max(...lngs), Math.max(...lats)]
            ),
            // Close enough to tell the places apart, and the markers clear of
            // the edges
            { maxZoom: 9, squeezeLarge: false, margin: 32 }
        )
    }

    showPlaces(places: MapPlace[] | null): void {
        const source = this.map?.getSource('places') as
            GeoJSONSource | undefined
        if (!source) return
        this.placesVisible = !!places
        source.setData({
            type: 'FeatureCollection',
            features: (places ?? []).map((p) => ({
                type: 'Feature',
                geometry: { type: 'Point', coordinates: [p.lng, p.lat] },
                properties: { name: p.name, p: p.population },
            })),
        })
        for (const layer of PLACE_LAYERS) {
            this.map!.setLayoutProperty(
                layer,
                'visibility',
                places ? 'visible' : 'none'
            )
        }
    }

    getZoom(): number {
        return this.map?.getZoom() ?? 0
    }

    countriesInView(): string[] {
        if (!this.map || !this.countryLayerId) return []
        const features = this.map.queryRenderedFeatures({
            layers: [this.countryLayerId],
        })
        return [
            ...new Set(features.flatMap((f) => f.properties?.ADM0_A3 ?? [])),
        ] as string[]
    }

    onMoveEnd(callback: () => void): void {
        this.moveEndCallback = callback
    }

    onMapClick(
        callback: (point: MapPointInfo, place: MapPlace | null) => void
    ): void {
        this.mapClickCallback = callback
    }

    pointAt(x: number, y: number): MapPointInfo | null {
        if (!this.map) return null
        const { lng, lat } = this.map.unproject([x, y])
        return { lng, lat, iso: this.countryAtPoint(x, y) }
    }

    countryAt(lng: number, lat: number): string | null {
        if (!this.map) return null
        const { x, y } = this.map.project([lng, lat])
        return this.countryAtPoint(x, y)
    }

    private countryAtPoint(x: number, y: number): string | null {
        if (!this.map || !this.countryLayerId) return null
        const hits = this.map.queryRenderedFeatures([x, y], {
            layers: [this.countryLayerId],
        })
        return (hits[0]?.properties?.ADM0_A3 as string | undefined) ?? null
    }

    // The place whose label is near a screen point, if names are shown
    private placeAt(point: { x: number; y: number }): MapPlace | null {
        if (!this.map || !this.placesVisible) return null
        const { x, y } = point
        const r = PLACE_HIT_PX
        const hit = this.map.queryRenderedFeatures(
            [
                [x - r, y - r],
                [x + r, y + r],
            ],
            { layers: PLACE_LAYERS }
        )[0]
        if (!hit || hit.geometry.type !== 'Point') return null
        const [lng, lat] = hit.geometry.coordinates
        return {
            name: String(hit.properties?.name ?? ''),
            lng,
            lat,
            population: Number(hit.properties?.p ?? 0),
        }
    }

    setPin(point: TripStopPoint | null): void {
        if (!this.map) return
        if (!point) {
            this.pin?.remove()
            this.pin = undefined
            return
        }
        if (!this.pin) {
            this.pin = new Marker({ element: pinElement(), anchor: 'bottom' })
                .setLngLat([point.lng, point.lat])
                .addTo(this.map)
        } else {
            this.pin.setLngLat([point.lng, point.lat])
        }
    }

    panTo(point: TripStopPoint, x: number, y: number, minZoom: number): void {
        if (!this.map) return
        const { clientWidth: width, clientHeight: height } =
            this.map.getContainer()
        this.map.easeTo({
            center: [point.lng, point.lat],
            zoom: Math.max(this.map.getZoom(), minZoom),
            offset: [x - width / 2, y - height / 2],
        })
    }
}
