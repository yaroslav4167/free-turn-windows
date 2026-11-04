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
