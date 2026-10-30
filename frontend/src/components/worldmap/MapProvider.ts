export interface FlightRoute {
    id: string
    from: [number, number] // [lng, lat]
    to: [number, number]
    label?: string
}

export interface MapMarker {
    id: string
    position: [number, number]
    label?: string
}

export interface TripStopPoint {
    lng: number
    lat: number
}

export interface MapPlace {
    name: string
    lng: number
    lat: number
    population: number
}

// A point on the map with the country under it (null on water)
export interface MapPointInfo {
    lng: number
    lat: number
    iso: string | null
}

// Pixels at each edge that panels cover; the map fits countries inside the rest
export interface MapInsets {
    top: number
    right: number
    bottom: number
    left: number
}

export interface MapProvider {
    mount(container: HTMLElement): void
    destroy(): void
    onReady(callback: () => void): void
    highlightCountries(isoCodes: string[]): void
    highlightWishlist(isoCodes: string[]): void
    showFlightRoutes(routes: FlightRoute[], visible: boolean): void
    highlightFlight(id: string | null): void
    addMarker(marker: MapMarker): void
    onCountryClick(callback: (countryCode: string) => void): void
    // While routes are shown: the clicked route's id, or null for a click
    // away from all routes
    onFlightClick(callback: (id: string | null) => void): void
    focusCountry(isoCode: string | null, insets?: MapInsets): void
    focusFlight(id: string | null, insets?: MapInsets): void
    flyTo(lng: number, lat: number, zoom: number): void
    // Labels by the base map's English country name; null keeps English
    setCountryNames(names: Record<string, string> | null): void
    // Numbered stops joined by a line in visiting order; null hides them
    // The routes of all trips, drawn under the open one
    showTripRoutes(routes: TripStopPoint[][]): void
    showTrip(stops: TripStopPoint[] | null): void
    fitPoints(points: TripStopPoint[], insets?: MapInsets): void
    // Fits the country without marking it as selected
    fitCountry(isoCode: string, insets?: MapInsets): void
    // Place names; null hides them
    showPlaces(places: MapPlace[] | null): void
    getZoom(): number
    // Countries with a part in the visible map
    countriesInView(): string[]
    onMoveEnd(callback: () => void): void
    // Every click on the map, with the place label clicked, if any
    onMapClick(
        callback: (point: MapPointInfo, place: MapPlace | null) => void
    ): void
    // The map point and country at a screen point of the map
    pointAt(x: number, y: number): MapPointInfo | null
    countryAt(lng: number, lat: number): string | null
    // The pin set by click; null removes it
    setPin(point: TripStopPoint | null): void
    // Moves the map so that the point lies under screen point (x, y)
    panTo(point: TripStopPoint, x: number, y: number, minZoom: number): void
}

function mix(top: string, bottom: string, alpha: number): string {
    const channel = (hex: string, i: number) =>
        parseInt(hex.slice(1 + i * 2, 3 + i * 2), 16)
    const rgb = [0, 1, 2].map((i) =>
        Math.round(alpha * channel(top, i) + (1 - alpha) * channel(bottom, i))
    )
    return `rgb(${rgb.join(',')})`
}

export const Colors = {
    // Visited, wishlist, flight and trip colours match --visited, --wishlist,
    // --flight and --trip in style.css, which the legend and panels use
    countryBase: '#151616',
    countryHighlight: '#d9a03f',
    countryFocus: '#fff1c4',
    countryWishlist: '#9d7fd6',
    // countryHighlight slightly toned down by the water colour, so the
    // opaque overlay does not glare on the dark map
    countryHighlightOnBase: mix('#d9a03f', '#373838', 0.85),
    // Cyan stands apart from the gold visited and violet wishlist fills as
    // well as from the dark base
    flight: '#5fd4e4',
    flightLabel: '#c4f1f7',
    // --trip in style.css; hot pink stands apart from the other three
    trip: '#ff3d6e',
    tripNumber: '#1a1c1e',
    place: '#ededed',
    // Also --map-water in style.css and the theme colour of the app
    water: '#373838',
    geoline: '#4c4d4d',
    label: '#dbdbdb',
}
