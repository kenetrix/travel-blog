import { test } from 'node:test'
import assert from 'node:assert/strict'
import { distanceKm, nearestPlace, pathKm, type Place } from './geo.ts'

const granada: Place = {
    name: 'Granada',
    lng: -3.6067,
    lat: 37.1882,
    population: 234325,
}
const seville: Place = {
    name: 'Seville',
    lng: -5.9845,
    lat: 37.3891,
    population: 684234,
}
const armilla: Place = {
    name: 'Armilla',
    lng: -3.6223,
    lat: 37.1407,
    population: 24000,
}
const places = [seville, armilla, granada]

test('distance between Berlin and Tokyo', () => {
    const km = distanceKm([13.405, 52.52], [139.6917, 35.6895])
    assert.ok(Math.abs(km - 8918) < 20, `${km}`)
})

test('path sums consecutive legs', () => {
    const legs =
        distanceKm([granada.lng, granada.lat], [seville.lng, seville.lat]) * 2
    const km = pathKm([
        [granada.lng, granada.lat],
        [seville.lng, seville.lat],
        [granada.lng, granada.lat],
    ])
    assert.ok(Math.abs(km - legs) < 1e-9)
    assert.equal(pathKm([[granada.lng, granada.lat]]), 0)
})

test('nearest place wins over a bigger one further away', () => {
    // A few hundred metres from the centre of Granada
    assert.equal(nearestPlace(places, [-3.6, 37.19])?.name, 'Granada')
    // Closer to Armilla than to Granada
    assert.equal(nearestPlace(places, [-3.625, 37.145])?.name, 'Armilla')
})

test('nothing within the radius', () => {
    // In the Sahara
    assert.equal(nearestPlace(places, [2, 25]), null)
    // Seville is about 210 km from Granada
    assert.equal(nearestPlace([seville], [granada.lng, granada.lat], 15), null)
    assert.equal(
        nearestPlace([seville], [granada.lng, granada.lat], 250)?.name,
        'Seville'
    )
})
