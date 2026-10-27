// Everything the app needs for this language, loaded as one chunk
import type { LocalizedData } from '..'
import capitals from '../data/capitals.ru.json'
import airports from '../data/airports.ru.json'

export { default as messages } from '../messages/ru'
export const data: LocalizedData = { capitals, airports }
