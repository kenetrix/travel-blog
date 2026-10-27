import { Colors } from './MapProvider'

// The pin that picks a stop's place; its tip is the bottom centre
export const PIN_WIDTH = 30
export const PIN_HEIGHT = 40

export const PIN_SVG = `<svg viewBox="0 0 30 40" width="${PIN_WIDTH}" height="${PIN_HEIGHT}" aria-hidden="true">
<path d="M15 39C15 39 2 23.5 2 14a13 13 0 0 1 26 0c0 9.5-13 25-13 25z" fill="${Colors.trip}" stroke="${Colors.water}" stroke-width="2"/>
<circle cx="15" cy="14" r="5" fill="${Colors.water}"/>
</svg>`

export function pinElement(): HTMLElement {
    const el = document.createElement('div')
    el.className = 'trip-pin'
    el.innerHTML = PIN_SVG
    return el
}
