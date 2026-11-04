import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'
import {powerShapeAt} from '../src/powerShape.ts'
import {advancePowerMotion} from '../src/powerMotion.ts'
import {expressiveLoaderFrame} from '../src/expressiveLoader.ts'
import {connectionState} from '../src/connectionState.ts'

const start = () => ({morph: 0, morphVelocity: 0, angle: 0, speed: 0})

test('matching star and circle contours morph through real intermediate geometry', () => {
  const paths = [0, 0.5, 1].map(powerShapeAt)
  assert.equal(new Set(paths).size, 3)
  for (const path of paths) {
    assert.equal((path.match(/Q /g) || []).length, 20)
    assert.doesNotMatch(path, /NaN|Infinity/)
  }
})

test('power motion is frame-rate independent and settles without snapping orientation', () => {
  let fast = start(), slow = start()
  for (let i = 0; i < 60; i++) fast = advancePowerMotion(fast, true, true, 1 / 60)
  for (let i = 0; i < 30; i++) slow = advancePowerMotion(slow, true, true, 1 / 30)
  assert.ok(Math.abs(fast.morph - slow.morph) < 0.001)
  assert.ok(Math.abs(fast.angle - slow.angle) < 0.001)
  const before = fast.angle
  for (let i = 0; i < 240; i++) fast = advancePowerMotion(fast, true, false, 1 / 60)
  assert.equal(fast.morph, 1)
  assert.equal(fast.speed, 0)
  assert.ok(fast.angle >= before)
})

test('loader begins with horizontal oval and continuously changes contour', () => {
  const first = expressiveLoaderFrame(0)
  assert.equal(first.angle, 0)
  assert.notEqual(first.path, expressiveLoaderFrame(0.4).path)
  const numbers = first.path.match(/-?\d+(?:\.\d+)?/g).map(Number)
  const xs = numbers.filter((_, i) => i % 2 === 0)
  const ys = numbers.filter((_, i) => i % 2 === 1)
  assert.ok(Math.max(...xs) - Math.min(...xs) > Math.max(...ys) - Math.min(...ys))
  assert.doesNotMatch(expressiveLoaderFrame(NaN).path, /NaN|Infinity/)
})

test('VPN is not reported enabled until route setup completes; active captcha retains disconnect', () => {
  assert.equal(connectionState('connected', true, true).connected, false)
  assert.equal(connectionState('connected', false, true).heading, 'VPN включён')
  assert.equal(connectionState('connected', false, true).action, 'Отключить')
  assert.equal(connectionState('captcha', false, true).action, 'Отменить')
  assert.equal(connectionState('connected', false, false).heading, 'Реле подключено')
})

test('only the contour rotates and motion hooks respect reduced-motion and cleanup', () => {
  const component = readFileSync(new URL('../src/components/ConnectionControl.tsx', import.meta.url), 'utf8')
  assert.match(component, /className="power-outline" style=\{contour\}/)
  assert.doesNotMatch(component, /className="power-symbol"[^>]*style/)
  for (const path of ['../src/usePowerMotion.ts', '../src/components/ConnectionLoader.tsx']) {
    const source = readFileSync(new URL(path, import.meta.url), 'utf8')
    assert.match(source, /prefers-reduced-motion: reduce/)
    assert.match(source, /cancelAnimationFrame/)
    assert.match(source, /removeEventListener/)
  }
})
