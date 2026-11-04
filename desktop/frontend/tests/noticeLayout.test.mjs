import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'

const css = readFileSync(new URL('../src/style.css', import.meta.url), 'utf8')
const rule = (selector) => css.match(new RegExp(selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '\\s*\\{([^}]+)\\}'))?.[1] || ''

test('notification text is vertically centered with actions and safely wraps', () => {
  assert.match(rule('.app-error'), /align-items:\s*center/)
  assert.match(rule('.app-error'), /flex-wrap:\s*wrap/)
  assert.match(rule('.app-error > span'), /min-width:\s*0/)
  assert.match(rule('.notice'), /line-height:\s*1\.5/)
  assert.match(rule('.notice'), /overflow-wrap:\s*anywhere/)
  assert.match(rule('.notice > :first-child'), /margin-top:\s*0/)
  assert.match(rule('.notice > :last-child'), /margin-bottom:\s*0/)
})
