import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'
import {infoTipPosition} from '../src/infoTipPosition.ts'

test('info popup sits below its trigger and flips above near the bottom', () => {
  const anchor = {left: 200, right: 220, top: 100, bottom: 120}
  const size = {width: 320, height: 100}
  assert.deepEqual(infoTipPosition(anchor, size, {width: 900, height: 600}), {left: 50, top: 128})
  assert.deepEqual(infoTipPosition({...anchor, top: 550, bottom: 570}, size, {width: 900, height: 600}), {left: 50, top: 442})
})

test('popup stays within the viewport at left and right edges', () => {
  for (const width of [320, 900]) {
    for (const left of [0, width - 20]) {
      const size = {width: Math.min(320, width - 24), height: 120}
      const result = infoTipPosition({left, right: left + 20, top: 5, bottom: 25}, size, {width, height: 600})
      assert.ok(result.left >= 12)
      assert.ok(result.left + size.width <= width - 12)
      assert.ok(result.top >= 12 && result.top + size.height <= 588)
    }
  }
})

test('info uses a portal popup without changing document layout and supports dismissal', () => {
  const component = readFileSync(new URL('../src/components/InfoTip.tsx', import.meta.url), 'utf8')
  assert.doesNotMatch(component, /<details|<summary/)
  assert.match(component, /createPortal/)
  assert.match(component, /role="tooltip"/)
  assert.match(component, /aria-expanded=\{open\}/)
  assert.match(component, /event\.key !== 'Escape'/)
  assert.match(component, /addEventListener\('pointerdown', outside/)
  assert.match(component, /removeEventListener\('scroll', place, true\)/)
  assert.match(component, /onPointerEnter=/)
  assert.match(component, /event\.pointerType !== 'touch'/)
  assert.match(component, /onPointerEnter=\{cancelClose\}/)
  assert.match(component, /matches\(':focus-visible'\)/)
  assert.match(component, /setTimeout\(\(\) => setOpen\(false\), 160\)/)
})
