import english from './locales/en.json' with {type: 'json'}
import russian from './locales/ru.json' with {type: 'json'}

export interface LanguageCatalog {
  code: string
  name: string
  source?: boolean
  messages: Record<string, string>
}
export type Language = string
export type LanguagePreference = string
type Parameters = Record<string, string | number>

const catalogs = new Map<string, LanguageCatalog>()
const listeners = new Set<() => void>()
let revision = 0
let preference = 'system'
let detected = 'ru'

export function registerCatalogs(items: readonly LanguageCatalog[]) {
  for (const item of items) {
    if (!/^[a-z]{2,3}(?:-[a-zA-Z0-9]{2,8})*$/.test(item.code) || !item.name ||
        !item.messages || Object.values(item.messages).some(value => typeof value !== 'string')) {
      throw new Error('Invalid language catalog')
    }
    catalogs.set(item.code.toLowerCase(), item)
  }
  revision++
  for (const listener of listeners) listener()
}
registerCatalogs([english, russian])

export function availableLanguages() {
  return [...catalogs.values()].map(({code, name}) => ({code: code.toLowerCase(), name}))
}

export function systemLanguage(locales: readonly string[]): Language {
  const first = (locales.find(value => value.trim()) || 'en').trim().toLowerCase().split(/[.@]/)[0].replaceAll('_', '-')
  if (catalogs.has(first)) return first
  const base = first.split('-')[0]
  return catalogs.has(base) ? base : 'en'
}

export function validLanguage(value: string): value is LanguagePreference {
  return value === 'system' || catalogs.has(value)
}

export function configureLanguage(value: string, system = detected) {
  preference = validLanguage(value) ? value : 'system'
  detected = systemLanguage([system])
  if (typeof document !== 'undefined') document.documentElement.lang = currentLanguage()
  for (const listener of listeners) listener()
}

export function currentLanguage(): Language { return preference === 'system' ? detected : preference }
export function languagePreference(): LanguagePreference { return preference }
export function languageSnapshot() { return `${preference}:${detected}:${revision}` }
export function subscribeLanguage(listener: () => void) {
  listeners.add(listener)
  return () => { listeners.delete(listener) }
}

// Source-based message IDs keep existing Russian text readable. The optional
// English argument is a migration fallback; JSON catalogs always take priority.
// New messages can use tr('message ID', {name: value}) directly.
export function tr(key: string, fallbackOrParameters?: string | Parameters): string {
  const selected = catalogs.get(currentLanguage())
  const lookup = (catalog: LanguageCatalog | undefined) => catalog && Object.hasOwn(catalog.messages, key) ? catalog.messages[key] : undefined
  const text = lookup(selected) ?? (selected?.source ? key : lookup(catalogs.get('en'))) ??
    (typeof fallbackOrParameters === 'string' ? fallbackOrParameters : key)
  const parameters = typeof fallbackOrParameters === 'object' ? fallbackOrParameters : {}
  return text.replace(/\{([a-zA-Z0-9_]+)\}/g, (placeholder, name: string) =>
    Object.hasOwn(parameters, name) ? String(parameters[name]) : placeholder)
}
