import {LanguageCatalog, registerCatalogs} from './language'

// New locale JSON files are discovered automatically on the next build.
// Built-in catalogs are imported with JSON attributes by the pure helper (also
// used in Node tests); exclude them here to avoid duplicate import attributes.
const catalogs = import.meta.glob<LanguageCatalog>(['./locales/*.json', '!./locales/en.json', '!./locales/ru.json'], {eager: true, import: 'default'})
registerCatalogs(Object.values(catalogs))
