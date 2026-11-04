# Добавление языка / Adding a language

UI translations live in `desktop/frontend/src/locales/*.json`. The frontend and
native Linux UI use these same files; they are bundled into the release.

## Add a language

1. Copy `en.json` to a new file, for example `de.json`.
2. Set `code` to its BCP 47 language code (`de`) and `name` to its native name
   (`Deutsch`). Translate the **values** in `messages`; keep the message IDs.
3. Rebuild. Vite discovers every JSON catalog automatically. Go embeds the same
   directory for preference validation, system-language detection and native UI.
   No component, settings-menu or language-list changes are needed.

`ru.json` declares the source language with `source: true`: message IDs are the
original Russian text, so missing Russian values resolve to their IDs rather
than duplicating every source string. Explicit overrides can be added to its
`messages` object. Other languages should not set `source: true`.

## Translation helper

```tsx
import {tr} from '../language'

<h1>{tr('Настройки')}</h1>
<p>{tr('Последние {count} записей', {count: 500})}</p>
```

JSON catalogs always take priority. Some existing calls retain an optional
English string as a migration fallback; new text belongs in the catalogs.

- Missing translations fall back to `en.json`, then to the message ID.
- Named `{parameters}` may be reordered. Preserve their names in every language.
- Parameters are inserted once as plain text, never parsed as HTML or translated.
- Native language names, profile names, keys, configs, logs and diagnostic
  messages are not rewritten by the frontend.
- Language preference is saved in `desktop.json`; `system` uses Linux locale
  variables (`LC_ALL`, `LC_MESSAGES`, `LANG`, in that order). Unsupported system
  locales fall back to English. A manual selection overrides the system.
- Switching is reactive and does not remount connection screens or reconnect VPN.

Run `npm test` and `npm run build` in `desktop/frontend`, and `go test ./desktop/...`
from the repository root. Tests check catalog coverage, fallback, parameter
interpolation, locale detection and saved preferences.
