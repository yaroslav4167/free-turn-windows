import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'

const css = readFileSync(new URL('../src/style.css', import.meta.url), 'utf8')

test('connection details cannot shrink underneath traffic cards', () => {
  assert.match(css, /\.connection-center\s*\{[^}]*flex:\s*1 0 auto/)
  assert.match(css, /\.connection-center > \*\s*\{\s*flex-shrink:\s*0/)
  assert.match(css, /\.connection-bottom\s*\{[^}]*flex-shrink:\s*0/)
  assert.match(css, /\.connection-center h2\s*\{[^}]*margin:\s*8px 0 6px/)
  assert.match(css, /@media \(max-height: 700px\)/)
})

test('short Windows client areas compact spacing without hiding overflow', () => {
  const compact = css.slice(css.indexOf('@media (max-height: 600px)'))
  assert.match(compact, /\.connection-center\s*\{[^}]*min-height:\s*0/)
  assert.match(compact, /\.connection-center \.power-button\s*\{[^}]*height:\s*178px/)
  assert.match(compact, /\.connection-bottom \.traffic-card\s*\{[^}]*padding-top:\s*12px/)
  assert.match(compact, /\.connection-bottom \.profile-picker\s*\{[^}]*margin-top:\s*12px/)
  assert.match(css, /\.connection-page\s*\{[^}]*overflow-y:\s*auto/)
})
