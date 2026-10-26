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
    focusCountry(isoCode: string | null, insets?: MapInsets): void
    flyTo(lng: number, lat: number, zoom: number): void
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
    // Visited, wishlist and flight colours match --visited, --wishlist and
    // --flight in style.css, which the legend and panels use
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
    // Also --map-water in style.css and the theme colour of the app
    water: '#373838',
    geoline: '#4c4d4d',
    label: '#dbdbdb',
}
