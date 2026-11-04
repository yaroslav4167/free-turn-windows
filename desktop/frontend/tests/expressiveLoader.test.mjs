import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'
import {expressiveLoaderFrame} from '../src/expressiveLoader.ts'

test('expressive loader continuously changes shape inside its fixed icon slot', () => {
  const shapes = new Set()
  for (const time of [0, 0.2, 0.4, 0.8, 1.6, 2.4, 3.2]) {
    const frame = expressiveLoaderFrame(time)
    shapes.add(frame.path)
    assert.equal([...frame.path.matchAll(/\bQ\b/g)].length, 64)
    assert.doesNotMatch(frame.path, /NaN|Infinity/)
    for (const value of frame.path.matchAll(/[\d.]+/g)) assert.ok(Number(value[0]) >= 5 && Number(value[0]) <= 43)
  }
  assert.ok(shapes.size >= 5)
  assert.equal(expressiveLoaderFrame(1).angle, 180)
  assert.equal(expressiveLoaderFrame(NaN).path, expressiveLoaderFrame(0).path)
})

test('loader morph is continuous at shape boundaries', () => {
  const numbers = (time) => [...expressiveLoaderFrame(time).path.matchAll(/[\d.]+/g)].map(value => Number(value[0]))
  for (const boundary of [0.8, 1.6, 2.4, 3.2, 4]) {
    const before = numbers(boundary - 0.00001)
    const after = numbers(boundary + 0.00001)
    assert.ok(before.every((value, index) => Math.abs(value - after[index]) <= 0.011))
  }
})

test('each cycle begins with a horizontal oval', () => {
  const coordinates = [...expressiveLoaderFrame(0).path.matchAll(/[\d.]+/g)].map(value => Number(value[0]))
  const xs = coordinates.filter((_, index) => index % 2 === 0)
  const ys = coordinates.filter((_, index) => index % 2 === 1)
  assert.ok(Math.max(...xs) - Math.min(...xs) > 1.5 * (Math.max(...ys) - Math.min(...ys)))
  assert.equal(expressiveLoaderFrame(4).path, expressiveLoaderFrame(0).path)
})

test('central loader is connection-only and respects reduced motion', () => {
  const read = (file) => readFileSync(new URL('../src/' + file, import.meta.url), 'utf8')
  assert.match(read('components/Connection.tsx'), /<ConnectionLoader active=\{presentation\.busy\}/)
  const loader = read('components/ConnectionLoader.tsx')
  assert.match(loader, /if \(!active\) return/)
  assert.match(loader, /prefers-reduced-motion/)
  assert.match(loader, /cancelAnimationFrame/)
})
