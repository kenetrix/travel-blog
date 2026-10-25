import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, problemMessage, type Schemas } from '../../api/client'
import { getAirport, type Airport } from '../flights/airports'
import { getCountryByAlpha2 } from '../countries/countries'

// A flight resolved against the airport data, ready for the map and lists
export interface Flight {
    id: string
    from: [number, number]
    to: [number, number]
    label: string
    visibility: Visibility
}

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

    const selectedIso = ref<string | null>(null)
    // Unsaved checkbox state of the open country modal, shown on the map only
    const preview = ref<{ iso: string; status: CountryStatus } | null>(null)
    // Flight hovered in the flight panel, emphasised on the map
    const highlightedFlightId = ref<string | null>(null)

    const toast = ref('')
    let toastTimer: ReturnType<typeof setTimeout> | undefined

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
        selectedIso.value = null
    }

    // Three requests in parallel (account, countries, flights); the browser
    // revalidates each with its ETag, so an unchanged copy costs only a 304
    async function load() {
        try {
            const [account, countries, flightList] = await Promise.all([
                api.GET('/user'),
                api.GET('/countries'),
                api.GET('/flights'),
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
                applyCountries(countries.data)
                flightRecords.value = flightList.data
            }
            authState.value = 'ready'
        } catch (e) {
            notify(problemMessage(e, 'Could not load your travel data'))
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

    // --- Account -----------------------------------------------------------
    // Each returns an error message, or null on success

    // Whether the username has an account (log in) or not (register), and
    // whether registering is allowed. A string is an error message.
    async function checkUsername(
        username: string
    ): Promise<Schemas['UsernameCheck'] | string> {
        try {
            const { data, error } = await api.GET('/auth/users/{username}', {
                params: { path: { username } },
            })
            if (error || !data) throw error
            return data
        } catch (e) {
            return problemMessage(e, 'Could not check the username')
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
        try {
            const { error } = await api.POST('/auth/login', {
                body: { username, password },
            })
            if (error) throw error
            await load()
            return null
        } catch (e) {
            return problemMessage(e, 'Could not log in')
        }
    }

    async function register(input: RegisterInput) {
        try {
            const { error } = await api.POST('/auth/register', { body: input })
            if (error) throw error
            await load()
            return null
        } catch (e) {
            return problemMessage(e, 'Could not create the account')
        }
    }

    // Loading again moves the state to anonymous
    async function logout() {
        try {
            await api.POST('/auth/logout')
        } finally {
            clear()
            await load()
        }
    }

    async function updateProfile(changes: ProfileUpdate) {
        try {
            const { data, error } = await api.PATCH('/user/profile', {
                body: changes,
            })
            if (error || !data) throw error
            profile.value = data
            return null
        } catch (e) {
            return problemMessage(e, 'Could not save the profile')
        }
    }

    // Applied right away, rolled back if the server rejects it
    async function saveSettings(next: Settings) {
        const before = settings.value
        settings.value = next
        try {
            const { error } = await api.PUT('/user/settings', { body: next })
            if (error) throw error
            return null
        } catch (e) {
            settings.value = before
            return problemMessage(e, 'Could not save the settings')
        }
    }

    async function changePassword(
        currentPassword: string,
        newPassword: string
    ) {
        try {
            const { error } = await api.PUT('/user/password', {
                body: { currentPassword, newPassword },
            })
            if (error) throw error
            return null
        } catch (e) {
            return problemMessage(e, 'Could not change the password')
        }
    }

    async function deleteAccount() {
        try {
            const { error } = await api.DELETE('/user')
            if (error) throw error
            clear()
            await load()
            return null
        } catch (e) {
            return problemMessage(e, 'Could not delete the account')
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
        try {
            const { data, error } = await api.GET('/countries/{iso}', {
                params: { path: { iso } },
            })
            if (error || !data) throw error
            details.value[iso] = data.details
            detailsLoaded.add(iso)
            return true
        } catch (e) {
            notify(problemMessage(e, 'Could not load the country details'))
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
            notify(problemMessage(e, 'Could not save the country'))
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

    async function addFlight(from: Airport, to: Airport): Promise<boolean> {
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
            return true
        } catch (e) {
            notify(problemMessage(e, 'Could not add the flight'))
            return false
        }
    }

    // Applied right away, restored if the server rejects it
    async function setFlightVisibility(id: string, vis: Visibility) {
        const flight = flightRecords.value.find((f) => f.id === id)
        if (!flight) return
        const before = flight.visibility
        flight.visibility = vis
        try {
            const { error } = await api.PATCH('/flights/{id}', {
                params: { path: { id } },
                body: { visibility: vis },
            })
            if (error) throw error
        } catch (e) {
            flight.visibility = before
            notify(problemMessage(e, 'Could not change the flight'))
        }
    }

    // Applied right away, restored if the server rejects it
    async function removeFlight(id: string) {
        const index = flightRecords.value.findIndex((f) => f.id === id)
        if (index < 0) return
        const [removed] = flightRecords.value.splice(index, 1)
        if (highlightedFlightId.value === id) highlightedFlightId.value = null
        try {
            const { error } = await api.DELETE('/flights/{id}', {
                params: { path: { id } },
            })
            if (error) throw error
        } catch (e) {
            flightRecords.value.splice(index, 0, removed)
            notify(problemMessage(e, 'Could not remove the flight'))
        }
    }

    function highlightFlight(id: string | null) {
        highlightedFlightId.value = id
    }

    function notify(message: string) {
        toast.value = message
        clearTimeout(toastTimer)
        toastTimer = setTimeout(() => (toast.value = ''), 2200)
    }

    return {
        authState,
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
        removeFlight,
        highlightedFlightId,
        highlightFlight,
    }
})
