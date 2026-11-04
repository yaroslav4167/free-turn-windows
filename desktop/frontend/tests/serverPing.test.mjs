import {test} from 'node:test'
import assert from 'node:assert/strict'
import {watchServerPing} from '../src/serverPing.ts'

const tick = () => new Promise(resolve => setImmediate(resolve))

test('ping polls every five seconds without overlapping and stops on disposal', async () => {
  let resolve
  let callback
  let reads = 0
  let clears = 0
  const samples = []
  const stop = watchServerPing('demo', () => { reads++; return new Promise(done => { resolve = done }) }, state => samples.push(state), {
    set: (fn, ms) => { assert.equal(ms, 5000); callback = fn; return 7 },
    clear: id => { assert.equal(id, 7); clears++ },
  })
  assert.equal(reads, 1)
  assert.equal(callback, undefined)
  assert.equal(samples[0].measuring, true)
  resolve({serverId: 'demo', host: 'example.com', milliseconds: 42.7})
  await tick()
  assert.equal(samples.at(-1).milliseconds, 42.7)
  callback()
  assert.equal(reads, 2)
  stop()
  resolve({serverId: 'demo', host: 'example.com', milliseconds: 1})
  await tick()
  assert.equal(samples.at(-1).milliseconds, 42.7)
  assert.equal(clears, 1)
})

test('failed, malformed and wrong-profile ping results never appear as zero or stale latency', async () => {
  for (const result of [null, {serverId: 'other', host: 'other', milliseconds: 10}, {serverId: 'demo', milliseconds: NaN}, {serverId: 'demo', milliseconds: -1}]) {
    const samples = []
    const stop = watchServerPing('demo', async () => { if (!result) throw new Error('blocked'); return result }, state => samples.push(state), {set: () => 1, clear: () => {}})
    await tick()
    assert.equal(samples.at(-1).milliseconds, null)
    assert.equal(samples.at(-1).measuring, false)
    stop()
  }
})
