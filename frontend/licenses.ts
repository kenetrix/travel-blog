import fs from 'node:fs'
import path from 'node:path'
import type { Plugin } from 'vite'

// Served next to index.html; the account page links to it
export const LICENSE_FILE = 'third-party-licenses.txt'

// Vite's own list (build.license) only covers JavaScript chunks. These reach
// the browser another way: CSS-only packages and the generated service worker.
const EXTRA_PACKAGES = [
    '@fontsource/dm-sans',
    '@fontsource/dm-serif-display',
    'flag-icons',
    'vite-plugin-pwa',
    'workbox-cacheable-response',
    'workbox-core',
    'workbox-expiration',
    'workbox-precaching',
    'workbox-routing',
    'workbox-strategies',
]

// Ship prebuilt bundles that already contain their dependencies, which
// neither Vite nor their own license file lists
const PREBUILT_PACKAGES = ['maplibre-gl']

const VITE_LIST = '.vite/licenses.json'

// Data credited here instead of on the map
const DATA_CREDITS = [
    'GeoNames (CC BY 4.0)',
    '',
    'Place names from GeoNames, https://www.geonames.org',
    'Licensed under Creative Commons Attribution 4.0,',
    'https://creativecommons.org/licenses/by/4.0/',
]

interface Entry {
    name: string
    version: string
    identifier?: string
    text?: string
}

// Node's lookup: node_modules next to `from`, then in each parent directory
function packageDir(name: string, from: string): string {
    for (let dir = from; ; dir = path.dirname(dir)) {
        const candidate = path.join(dir, 'node_modules', name)
        if (fs.existsSync(path.join(candidate, 'package.json')))
            return candidate
        if (path.dirname(dir) === dir) throw new Error(`${name} not found`)
    }
}

function readJson(file: string) {
    return JSON.parse(fs.readFileSync(file, 'utf-8'))
}

// All runtime dependencies below a package; type packages carry no code
function dependenciesOf(name: string): Map<string, string> {
    const found = new Map<string, string>()
    const visit = (dir: string) => {
        const deps = readJson(path.join(dir, 'package.json')).dependencies ?? {}
        for (const dep of Object.keys(deps)) {
            if (dep.startsWith('@types/') || found.has(dep)) continue
            const depDir = packageDir(dep, dir)
            found.set(dep, depDir)
            visit(depDir)
        }
    }
    visit(packageDir(name, import.meta.dirname))
    return found
}

function readPackage(
    name: string,
    dir = packageDir(name, import.meta.dirname)
): Entry {
    const pkg = readJson(path.join(dir, 'package.json'))
    const file = fs
        .readdirSync(dir)
        .find((f) => /^(license|licence|copying)/i.test(f))
    return {
        name,
        version: pkg.version,
        identifier: pkg.license,
        text: file
            ? fs.readFileSync(path.join(dir, file), 'utf-8').trim()
            : undefined,
    }
}

function render(entries: Entry[]): string {
    const line = '-'.repeat(72)
    const parts = entries.map((e) =>
        [
            line,
            `${e.name} ${e.version}${e.identifier ? ` (${e.identifier})` : ''}`,
            line,
            '',
            e.text ?? 'No license text shipped with the package.',
            '',
        ].join('\n')
    )
    return [
        'Travel Blog includes the following third-party software.',
        'Each package is listed with its version, license and license text.',
        '',
        ...parts,
        line,
        'Data',
        line,
        '',
        ...DATA_CREDITS,
        '',
    ].join('\n')
}

// Writes LICENSE_FILE into the build output: Vite's list of bundled packages
// plus EXTRA_PACKAGES, with their license texts
export function thirdPartyLicenses(): Plugin {
    return {
        name: 'third-party-licenses',
        apply: 'build',
        config: () => ({ build: { license: { fileName: VITE_LIST } } }),
        generateBundle: {
            // After vite:license has emitted its list
            order: 'post',
            handler(_, bundle) {
                const list = bundle[VITE_LIST]
                if (list?.type !== 'asset') {
                    this.error(
                        `${VITE_LIST} missing; is build.license enabled?`
                    )
                }
                const source =
                    typeof list.source === 'string'
                        ? list.source
                        : new TextDecoder().decode(list.source)
                const entries = new Map<string, Entry>()
                for (const e of JSON.parse(source) as Entry[]) {
                    entries.set(e.name, e)
                }
                for (const name of EXTRA_PACKAGES) {
                    if (!entries.has(name)) entries.set(name, readPackage(name))
                }
                for (const prebuilt of PREBUILT_PACKAGES) {
                    for (const [name, dir] of dependenciesOf(prebuilt)) {
                        if (!entries.has(name)) {
                            entries.set(name, readPackage(name, dir))
                        }
                    }
                }
                delete bundle[VITE_LIST]
                this.emitFile({
                    type: 'asset',
                    fileName: LICENSE_FILE,
                    source: render(
                        [...entries.values()].sort((a, b) =>
                            a.name.localeCompare(b.name)
                        )
                    ),
                })
            },
        },
    }
}
