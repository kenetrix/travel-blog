// Everything the app needs for this language, loaded as one chunk
import type { LocalizedData } from '..'
import capitals from '../data/capitals.de.json'
import airports from '../data/airports.de.json'

export { default as messages } from '../messages/de'
export const data: LocalizedData = { capitals, airports }
