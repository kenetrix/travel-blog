import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import {
    api,
    NetworkError,
    offlineMessage,
    onReachability,
    problemMessage,
    type Schemas,
} from '../../api/client'
import { getAirport, type Airport } from '../flights/airports'
import { getCountryByAlpha2 } from '../countries/countries'
import { readSnapshot, removeSnapshot, writeSnapshot } from './snapshot'

// While offline but with a network, how often to look for the server again
const RETRY_MS = 30_000

// A flight resolved against the airport data, ready for the map and lists
export interface Flight {
    id: string
    from: [number, number]
    to: [number, number]
    label: string
    visibility: Visibility
    // Entered by the user; unset if unknown
    flightNumber?: string
    date?: string
    durationMinutes?: number
    aircraft?: string
}

export type FlightUpdate = Schemas['FlightUpdate']

export type TripSummary = Schemas['TripSummary']
export type Trip = Schemas['Trip']
export type TripInput = Schemas['TripInput']

// A point on the map with the country under it (null on water)
export interface MapPoint {
    lng: number
    lat: number
    iso: string | null
}

// What the map shows of the trip open in the summary or editor
export interface TripMap {
    stops: { lng: number; lat: number; name: string }[]
    // Editing: place names are shown and clicks do not open countries
    editing: boolean
}

// How the pin is placed: by click, or fixed in the middle of the map while
// the map moves underneath (touch)
export type PinMode = 'off' | 'click' | 'center'

export type CountryStatus = Schemas['CountryStatus']
export type CountryDetails = Schemas['CountryDetails']
export type Visibility = Schemas['Visibility']
export type Profile = Schemas['Profile']
export type ProfileUpdate = Schemas['ProfileUpdate']
export type Settings = Schemas['Settings']
export type RegisterInput = Schemas['RegisterInput']

export function emptyDetails(): CountryDetails {
    return {
        food: '',
        sightseeing: '',
        notes: '',
        rating: 0,
        tags: [],
    }
}

// Shown until the account is loaded; matches the server's defaults
const defaultSettings: Settings = {
    distanceUnit: 'km',
    theme: 'system',
    defaultView: 'countries',
    defaultVisibility: 'private',
    profileVisibility: 'private',
}

export const useTravelStore = defineStore('travel', () => {
    // loading: first GET /travel pending; anonymous: no session (401)
    const authState = ref<'loading' | 'anonymous' | 'ready'>('loading')
    const profile = ref<Profile | null>(null)
    const role = ref<Schemas['Role']>('user')
    // The password was generated or set by an administrator and has to change
    const mustChangePassword = ref(false)
    const isAdmin = computed(() => role.value === 'admin')
    const settings = ref<Settings>({ ...defaultSettings })

    const visited = ref<Set<string>>(new Set())
    const wishlist = ref<Set<string>>(new Set())
    const details = ref<Record<string, CountryDetails>>({})
    // Countries whose long texts are in `details`; GET /travel only has
    // rating and tags, the rest comes with loadCountryDetails
    const detailsLoaded = new Set<string>()
    const visibility = ref<Record<string, Visibility>>({})
    // Flights as stored on the server (IATA codes only)
    const flightRecords = ref<Schemas['Flight'][]>([])
    const trips = ref<TripSummary[]>([])
    // Trips with their stops, filled when a trip is opened
    const tripDetails = ref<Record<string, Trip>>({})

    const tripMap = ref<TripMap | null>(null)
    // Bumped to make the map fit the trip's stops, or the country when the
    // trip has none yet
    const tripFocus = ref<{ count: number; iso: string | null }>({
        count: 0,
        iso: null,
    })
    const pinMode = ref<PinMode>('off')
    // Where the pin points; set by the map
    const pin = ref<MapPoint | null>(null)
    // Name of a place label clicked on the map, which set the pin exactly
    const pinPlace = ref<string | null>(null)

    const selectedIso = ref<string | null>(null)
    // Unsaved checkbox state of the open country modal, shown on the map only
    const preview = ref<{ iso: string; status: CountryStatus } | null>(null)
    // Flight hovered in the flight panel, emphasised on the map
    const highlightedFlightId = ref<string | null>(null)
    // Flight picked in the list or on the map: its details are open and the
    // map shows it
    const selectedFlightId = ref<string | null>(null)
    // Bumped to make the map fit the selected flight again, e.g. after the
    // sheet covering it changed size
    const flightFocus = ref(0)

    const toast = ref('')
    let toastTimer: ReturnType<typeof setTimeout> | undefined

    // offline: no network, or the server cannot be reached. The app then
    // shows the last snapshot and allows no changes.
    const connection = ref<'online' | 'offline'>(
        navigator.onLine ? 'online' : 'offline'
    )
    const canWrite = computed(() => connection.value === 'online')

    const flights = computed<Flight[]>(() =>
        flightRecords.value.flatMap((f) => {
            const from = getAirport(f.from)
            const to = getAirport(f.to)
            if (!from || !to) return []
            return [
                {
                    id: f.id,
                    from: from.coords,
                    to: to.coords,
                    label: `${f.from} → ${f.to}`,
                    visibility: f.visibility,
                    flightNumber: f.flightNumber,
                    date: f.date,
                    durationMinutes: f.durationMinutes,
                    aircraft: f.aircraft,
                },
            ]
        })
    )

    const visitedCount = computed(() => visited.value.size)
    const flightCount = computed(() => flights.value.length)

    // --- Loading -----------------------------------------------------------

    function clear() {
        profile.value = null
        role.value = 'user'
        mustChangePassword.value = false
        settings.value = { ...defaultSettings }
        visited.value = new Set()
        wishlist.value = new Set()
        details.value = {}
        detailsLoaded.clear()
        visibility.value = {}
        flightRecords.value = []
        trips.value = []
        tripDetails.value = {}
        selectedIso.value = null
        selectedFlightId.value = null
        removeSnapshot()
    }

    // --- Offline snapshot --------------------------------------------------

    function saveSnapshot() {
        if (!profile.value || authState.value !== 'ready') return
        writeSnapshot({
            username: profile.value.username,
            profile: profile.value,
            role: role.value,
            mustChangePassword: mustChangePassword.value,
            settings: settings.value,
            visited: [...visited.value],
            wishlist: [...wishlist.value],
            details: details.value,
            detailsLoaded: [...detailsLoaded],
            visibility: visibility.value,
            flights: flightRecords.value,
            trips: trips.value,
            tripDetails: tripDetails.value,
        })
    }

    // True if there was a snapshot to show
    function restoreSnapshot(): boolean {
        const s = readSnapshot()
        if (!s) return false
        profile.value = s.profile
        role.value = s.role
        mustChangePassword.value = s.mustChangePassword
        settings.value = s.settings
        visited.value = new Set(s.visited)
        wishlist.value = new Set(s.wishlist)
        details.value = s.details
        detailsLoaded.clear()
        for (const iso of s.detailsLoaded) detailsLoaded.add(iso)
        visibility.value = s.visibility
        flightRecords.value = s.flights
        trips.value = s.trips ?? []
        tripDetails.value = s.tripDetails ?? {}
        authState.value = 'ready'
        return true
    }

    // Batched: a save touches several of these at once
    let snapshotTimer: ReturnType<typeof setTimeout> | undefined
    watch(
        [
            profile,
            settings,
            visited,
            wishlist,
            details,
            visibility,
            flightRecords,
            trips,
            tripDetails,
        ],
        () => {
            clearTimeout(snapshotTimer)
            snapshotTimer = setTimeout(saveSnapshot, 300)
        },
        { deep: true }
    )

    // --- Connection --------------------------------------------------------

    let loading = false

    onReachability((reachable) => {
        const wasOffline = connection.value === 'offline'
        connection.value = reachable ? 'online' : 'offline'
        // Back online: refresh, which also ends an expired session
        if (reachable && wasOffline && !loading) load()
    })

    window.addEventListener('online', () => !loading && load())
    window.addEventListener('offline', () => (connection.value = 'offline'))
    setInterval(() => {
        if (connection.value === 'offline' && navigator.onLine && !loading) {
            load()
        }
    }, RETRY_MS)

    // --- Loading -----------------------------------------------------------

    // Four requests in parallel (account, countries, flights, trips); the
    // browser revalidates each with its ETag, so an unchanged copy costs only
    // a 304
    async function load() {
        loading = true
        try {
            const [account, countries, flightList, tripList] =
                await Promise.all([
                    api.GET('/user'),
                    api.GET('/countries'),
                    api.GET('/flights'),
                    api.GET('/trips'),
                ])
            if (account.response.status === 401) {
                clear()
                authState.value = 'anonymous'
                return
            }
            if (account.error || !account.data) throw account.error
            profile.value = account.data.profile
            role.value = account.data.role
            mustChangePassword.value = account.data.mustChangePassword
            settings.value = account.data.settings
            // Until the password is changed the server only serves the account
            if (!mustChangePassword.value) {
                if (countries.error || !countries.data) throw countries.error
                if (flightList.error || !flightList.data) throw flightList.error
                if (tripList.error || !tripList.data) throw tripList.error
                applyCountries(countries.data)
                flightRecords.value = flightList.data
                applyTrips(tripList.data)
            }
            authState.value = 'ready'
            saveSnapshot()
        } catch (e) {
            if (e instanceof NetworkError) {
                // Keep what is already shown; else the last snapshot, if any
                if (authState.value !== 'ready' && !restoreSnapshot()) {
                    authState.value = 'anonymous'
                }
                return
            }
            notify(problemMessage(e, 'store.loadFailed'))
        } finally {
            loading = false
        }
    }

    function applyCountries(entries: Schemas['CountrySummary'][]) {
        visited.value = new Set(
            entries.filter((e) => e.status === 'visited').map((e) => e.iso)
        )
        wishlist.value = new Set(
            entries.filter((e) => e.status === 'wishlist').map((e) => e.iso)
        )
        details.value = Object.fromEntries(
            entries.map((e) => [
                e.iso,
                { ...emptyDetails(), rating: e.rating, tags: e.tags },
            ])
        )
        detailsLoaded.clear()
        visibility.value = Object.fromEntries(
            entries.map((e) => [e.iso, e.visibility])
        )
    }

    // Trips opened before, also in an earlier session, stay available
    // offline while unchanged; deleted or changed ones drop out
    function applyTrips(list: TripSummary[]) {
        trips.value = list
        const stored = readSnapshot()
        const earlier =
            stored?.username === profile.value?.username
                ? (stored?.tripDetails ?? {})
                : {}
        const current = new Map(list.map((t) => [t.id, t.updatedAt]))
        tripDetails.value = Object.fromEntries(
            Object.entries({ ...earlier, ...tripDetails.value }).filter(
                ([id, trip]) => current.get(id) === trip.updatedAt
            )
        )
    }

    // --- Account -----------------------------------------------------------
    // Each returns an error message, or null on success

    // Whether the username has an account (log in) or not (register), and
    // whether registering is allowed. A string is an error message.
    async function checkUsername(
        username: string
    ): Promise<Schemas['UsernameCheck'] | string> {
        if (!canWrite.value) return offlineMessage()
        try {
            const { data, error } = await api.GET('/auth/users/{username}', {
                params: { path: { username } },
            })
            if (error || !data) throw error
            return data
        } catch (e) {
            return problemMessage(e, 'store.checkFailed')
        }
    }

    // Whether the login page offers single sign-on; null if unknown, in
    // which case the password form is all there is
    async function oidcConfig(): Promise<Schemas['OidcConfig'] | null> {
        try {
            const { data } = await api.GET('/auth/oidc/config')
            return data ?? null
        } catch {
            return null
        }
    }

    async function login(username: string, password: string) {
        if (!canWrite.value) return offlineMessage()
        try {
            const { error } = await api.POST('/auth/login', {
                body: { username, password },
            })
            if (error) throw error
            await load()
            return null
        } catch (e) {
            return problemMessage(e, 'store.loginFailed')
        }
    }

    async function register(input: RegisterInput) {
        if (!canWrite.value) return offlineMessage()
        try {
            const { error } = await api.POST('/auth/register', { body: input })
            if (error) throw error
            await load()
            return null
        } catch (e) {
            return problemMessage(e, 'store.registerFailed')
        }
    }

    // Loading again moves the state to anonymous. Offline the server could
    // not end the session, so logging out waits for the connection.
    async function logout(): Promise<string | null> {
        if (!canWrite.value) return offlineMessage()
        try {
            await api.POST('/auth/logout')
        } catch (e) {
            if (e instanceof NetworkError) return offlineMessage()
        }
        clear()
        await load()
        return null
    }

    async function updateProfile(changes: ProfileUpdate) {
        if (!canWrite.value) return offlineMessage()
        try {
            const { data, error } = await api.PATCH('/user/profile', {
                body: changes,
            })
            if (error || !data) throw error
            profile.value = data
            return null
        } catch (e) {
            return problemMessage(e, 'store.profileFailed')
        }
    }

    // Applied right away, rolled back if the server rejects it
    async function saveSettings(next: Settings) {
        if (!canWrite.value) return offlineMessage()
        const before = settings.value
        settings.value = next
        try {
            const { error } = await api.PUT('/user/settings', { body: next })
            if (error) throw error
            return null
        } catch (e) {
            settings.value = before
            return problemMessage(e, 'store.settingsFailed')
        }
    }

    async function changePassword(
        currentPassword: string,
        newPassword: string
    ) {
        if (!canWrite.value) return offlineMessage()
        try {
            const { error } = await api.PUT('/user/password', {
                body: { currentPassword, newPassword },
            })
            if (error) throw error
            return null
        } catch (e) {
            return problemMessage(e, 'store.passwordFailed')
        }
    }

    async function deleteAccount() {
        if (!canWrite.value) return offlineMessage()
        try {
            const { error } = await api.DELETE('/user')
            if (error) throw error
            clear()
            await load()
            return null
        } catch (e) {
            return problemMessage(e, 'store.deleteFailed')
        }
    }

    // --- Countries ---------------------------------------------------------

    function statusOf(iso: string): CountryStatus {
        if (visited.value.has(iso)) return 'visited'
        if (wishlist.value.has(iso)) return 'wishlist'
        return 'none'
    }

    // Visibility an entry has or would get when saved without one
    function visibilityOf(iso: string): Visibility {
        return visibility.value[iso] ?? settings.value.defaultVisibility
    }

    // Fetches the long texts of a country once; false if that failed
    async function loadCountryDetails(iso: string): Promise<boolean> {
        if (detailsLoaded.has(iso) || statusOf(iso) === 'none') return true
        // Offline only the snapshot has details; the panel says so
        if (!canWrite.value) return false
        try {
            const { data, error } = await api.GET('/countries/{iso}', {
                params: { path: { iso } },
            })
            if (error || !data) throw error
            details.value[iso] = data.details
            detailsLoaded.add(iso)
            return true
        } catch (e) {
            if (!(e instanceof NetworkError)) {
                notify(problemMessage(e, 'store.countryLoadFailed'))
            }
            return false
        }
    }

    function applyCountry(
        iso: string,
        status: CountryStatus,
        data: CountryDetails | undefined,
        vis: Visibility | undefined
    ) {
        visited.value.delete(iso)
        wishlist.value.delete(iso)
        if (status === 'visited') visited.value.add(iso)
        else if (status === 'wishlist') wishlist.value.add(iso)
        if (status === 'none' || !data) {
            delete details.value[iso]
            detailsLoaded.delete(iso)
            delete visibility.value[iso]
        } else {
            details.value[iso] = data
            if (vis) visibility.value[iso] = vis
        }
    }

    // Applied right away, rolled back if the server rejects it
    async function saveCountry(
        iso: string,
        status: CountryStatus,
        data: CountryDetails,
        vis?: Visibility
    ): Promise<boolean> {
        if (!canWrite.value) {
            notify(offlineMessage())
            return false
        }
        const before = {
            status: statusOf(iso),
            details: details.value[iso],
            loaded: detailsLoaded.has(iso),
            visibility: visibility.value[iso],
        }
        applyCountry(iso, status, data, vis)
        if (status !== 'none') detailsLoaded.add(iso)
        try {
            const { data: entry, error } = await api.PUT('/countries/{iso}', {
                params: { path: { iso } },
                body: { status, details: data, visibility: vis },
            })
            if (error) throw error
            // The server normalizes the details (e.g. tags)
            if (entry) {
                details.value[iso] = entry.details
                visibility.value[iso] = entry.visibility
            }
            return true
        } catch (e) {
            applyCountry(iso, before.status, before.details, before.visibility)
            if (before.loaded) detailsLoaded.add(iso)
            else detailsLoaded.delete(iso)
            notify(problemMessage(e, 'store.countryFailed'))
            return false
        }
    }

    function setPreview(value: { iso: string; status: CountryStatus } | null) {
        preview.value = value
    }

    function selectCountry(iso: string | null) {
        selectedIso.value = iso
    }

    // --- Flights -----------------------------------------------------------

    // Returns the new flight's id, or null if it could not be added
    async function addFlight(
        from: Airport,
        to: Airport
    ): Promise<string | null> {
        if (!canWrite.value) {
            notify(offlineMessage())
            return null
        }
        // Only the client knows the airports' countries; the server marks
        // them visited
        const countries = [
            ...new Set(
                [from, to].flatMap(
                    (a) => getCountryByAlpha2(a.countryCode)?.alpha3 ?? []
                )
            ),
        ]
        try {
            const { data, error } = await api.POST('/flights', {
                body: { from: from.iata, to: to.iata, countries },
            })
            if (error || !data) throw error
            flightRecords.value.push({
                id: data.id,
                from: from.iata,
                to: to.iata,
                visibility: data.visibility,
                createdAt: data.createdAt,
            })
            for (const iso of data.countriesUpdated) {
                visited.value.add(iso)
                wishlist.value.delete(iso)
                if (!details.value[iso]) {
                    details.value[iso] = emptyDetails()
                    detailsLoaded.add(iso)
                }
                visibility.value[iso] ??= settings.value.defaultVisibility
            }
            return data.id
        } catch (e) {
            notify(problemMessage(e, 'store.flightAddFailed'))
            return null
        }
    }

    // Applied right away, restored if the server rejects it. Empty values
    // ('' or 0) clear a field, as on the server.
    async function updateFlight(
        id: string,
        patch: FlightUpdate
    ): Promise<boolean> {
        const index = flightRecords.value.findIndex((f) => f.id === id)
        if (index < 0) return false
        if (!canWrite.value) {
            notify(offlineMessage())
            return false
        }
        const before = flightRecords.value[index]
        const after = { ...before }
        for (const [key, value] of Object.entries(patch)) {
            if (value === '' || value === 0)
                delete after[key as keyof FlightUpdate]
            else Object.assign(after, { [key]: value })
        }
        flightRecords.value[index] = after
        try {
            const { error } = await api.PATCH('/flights/{id}', {
                params: { path: { id } },
                body: patch,
            })
            if (error) throw error
            return true
        } catch (e) {
            const current = flightRecords.value.findIndex((f) => f.id === id)
            if (current >= 0) flightRecords.value[current] = before
            notify(problemMessage(e, 'store.flightChangeFailed'))
            return false
        }
    }

    function setFlightVisibility(id: string, visibility: Visibility) {
        return updateFlight(id, { visibility })
    }

    // Applied right away, restored if the server rejects it
    async function removeFlight(id: string) {
        const index = flightRecords.value.findIndex((f) => f.id === id)
        if (index < 0) return
        if (!canWrite.value) return notify(offlineMessage())
        const [removed] = flightRecords.value.splice(index, 1)
        if (highlightedFlightId.value === id) highlightedFlightId.value = null
        if (selectedFlightId.value === id) selectedFlightId.value = null
        try {
            const { error } = await api.DELETE('/flights/{id}', {
                params: { path: { id } },
            })
            if (error) throw error
        } catch (e) {
            flightRecords.value.splice(index, 0, removed)
            notify(problemMessage(e, 'store.flightRemoveFailed'))
        }
    }

    function highlightFlight(id: string | null) {
        highlightedFlightId.value = id
    }

    function selectFlight(id: string | null) {
        if (id && id !== selectedFlightId.value) flightFocus.value++
        selectedFlightId.value = id
        highlightedFlightId.value = id
    }

    function focusSelectedFlight() {
        if (selectedFlightId.value) flightFocus.value++
    }

    // --- Trips -------------------------------------------------------------

    function summaryOf(trip: Trip): TripSummary {
        const { notes: _notes, stops: _stops, ...summary } = trip
        return summary
    }

    function forgetTrip(id: string) {
        trips.value = trips.value.filter((t) => t.id !== id)
        delete tripDetails.value[id]
    }

    // The trip with its stops, from the server or, offline, as stored
    // before; null if there is no such trip
    async function loadTrip(id: string): Promise<Trip | null> {
        if (!canWrite.value) return tripDetails.value[id] ?? null
        try {
            const { data, error, response } = await api.GET('/trips/{id}', {
                params: { path: { id } },
            })
            if (response.status === 404) {
                forgetTrip(id)
                return null
            }
            if (error || !data) throw error
            tripDetails.value[id] = data
            return data
        } catch (e) {
            if (!(e instanceof NetworkError)) {
                notify(problemMessage(e, 'store.tripLoadFailed'))
            }
            return tripDetails.value[id] ?? null
        }
    }

    // Creates the trip (id null) or replaces it with all of its stops;
    // returns the saved trip, or null after reporting the error
    async function saveTrip(
        id: string | null,
        input: TripInput
    ): Promise<Trip | null> {
        if (!canWrite.value) {
            notify(offlineMessage())
            return null
        }
        try {
            const { data, error } = id
                ? await api.PUT('/trips/{id}', {
                      params: { path: { id } },
                      body: input,
                  })
                : await api.POST('/trips', { body: input })
            if (error || !data) throw error
            tripDetails.value[data.id] = data
            const summary = summaryOf(data)
            const index = trips.value.findIndex((t) => t.id === data.id)
            if (index >= 0) trips.value[index] = summary
            else trips.value.push(summary)
            // A planned country is at least on the wishlist
            for (const iso of data.countries) {
                if (statusOf(iso) === 'none') {
                    await saveCountry(iso, 'wishlist', emptyDetails())
                }
            }
            return data
        } catch (e) {
            notify(problemMessage(e, 'store.tripSaveFailed'))
            return null
        }
    }

    async function deleteTrip(id: string): Promise<boolean> {
        if (!canWrite.value) {
            notify(offlineMessage())
            return false
        }
        try {
            const { error } = await api.DELETE('/trips/{id}', {
                params: { path: { id } },
            })
            if (error) throw error
            forgetTrip(id)
            return true
        } catch (e) {
            notify(problemMessage(e, 'store.tripDeleteFailed'))
            return false
        }
    }

    function showTrip(value: TripMap | null) {
        tripMap.value = value
        if (!value) setPinMode('off')
    }

    function focusTrip(iso: string | null = null) {
        tripFocus.value = { count: tripFocus.value.count + 1, iso }
    }

    function setPinMode(mode: PinMode) {
        pinMode.value = mode
        if (mode === 'off') setPin(null)
    }

    function setPin(point: MapPoint | null, place: string | null = null) {
        pin.value = point
        pinPlace.value = place
    }

    function notify(message: string) {
        toast.value = message
        clearTimeout(toastTimer)
        toastTimer = setTimeout(() => (toast.value = ''), 2200)
    }

    return {
        authState,
        connection,
        canWrite,
        hasDetails: (iso: string) => detailsLoaded.has(iso),
        profile,
        role,
        isAdmin,
        mustChangePassword,
        settings,
        visited,
        wishlist,
        toast,
        notify,
        selectedIso,
        preview,
        details,
        visibility,
        visibilityOf,
        flights,
        visitedCount,
        flightCount,
        load,
        checkUsername,
        oidcConfig,
        login,
        register,
        logout,
        updateProfile,
        saveSettings,
        changePassword,
        deleteAccount,
        selectCountry,
        setPreview,
        saveCountry,
        loadCountryDetails,
        addFlight,
        setFlightVisibility,
        updateFlight,
        removeFlight,
        highlightedFlightId,
        highlightFlight,
        selectedFlightId,
        selectFlight,
        flightFocus,
        focusSelectedFlight,
        trips,
        tripDetails,
        loadTrip,
        saveTrip,
        deleteTrip,
        tripMap,
        showTrip,
        tripFocus,
        focusTrip,
        pinMode,
        setPinMode,
        pin,
        pinPlace,
        setPin,
    }
})
