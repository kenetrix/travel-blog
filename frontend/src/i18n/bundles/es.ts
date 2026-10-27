// Everything the app needs for this language, loaded as one chunk
import type { LocalizedData } from '..'
import capitals from '../data/capitals.es.json'
import airports from '../data/airports.es.json'

export { default as messages } from '../messages/es'
export const data: LocalizedData = { capitals, airports }
