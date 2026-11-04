import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'

test('VK Calls editor preserves Enter, blank lines, whitespace and pasted lines while typing', () => {
  const component = readFileSync(new URL('../src/components/ServerForm.tsx', import.meta.url), 'utf8')
  assert.match(component, /value=\{\(s\.vkLinks \?\? \[\]\)\.join\('\\n'\)\}/)
  assert.match(component, /vkLinks: e\.target\.value\.split\('\\n'\)/)
  assert.doesNotMatch(component, /vkLinks:.*(?:filter\(Boolean\)|\.trim\(\))/)
  for (const value of ['https://vk.ru/call/join/example\n', 'first\n\nsecond', '\n', ' first \n second \n']) {
    assert.equal(value.split('\n').join('\n'), value)
  }
})
