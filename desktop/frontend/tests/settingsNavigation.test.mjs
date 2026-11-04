import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'

const read = path => readFileSync(new URL('../src/' + path, import.meta.url), 'utf8')

test('settings mount on first visit and remain mounted when switching pages', () => {
  const app = read('App.tsx')
  assert.match(app, /if \(next === 'settings'\) setSettingsOpened\(true\)/)
  assert.match(app, /<Navigation view=\{view\} onChange=\{changeView\}/)
  assert.match(app, /\{settingsOpened && <DesktopPreferences[^>]*active=\{view === 'settings'\}[^>]*servers=\{servers\}/)
  assert.doesNotMatch(app, /view === 'settings' && <DesktopPreferences/)
  assert.match(read('components/DesktopPreferences.tsx'), /className="settings-page" hidden=\{!active\}/)
  assert.match(read('style.css'), /\.settings-page\[hidden\]\s*\{\s*display:\s*none/)
})

test('settings use the live profile list rather than fetching a stale copy on mount', () => {
  const preferences = read('components/DesktopPreferences.tsx')
  assert.doesNotMatch(preferences, /ListServers|setServers/)
  assert.match(preferences, /servers: backend\.Server\[\]/)
  assert.match(preferences, /await action\(\)\s*await refresh\(\)/)
})

test('update polling pauses while settings are hidden without discarding state', () => {
  const updates = read('components/UpdatePreferences.tsx')
  assert.match(updates, /if \(!enabled\) return/)
  assert.match(updates, /\[interval, enabled\]/)
  assert.match(updates, /window\.clearTimeout\(timer\)/)
  assert.match(updates, /useUpdateState\(500, active\)/)
  assert.match(read('components/DesktopPreferences.tsx'), /<UpdatePreferences active=\{active\}/)
})
