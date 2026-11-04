import {test} from 'node:test'
import assert from 'node:assert/strict'
import {setImmediate} from 'node:timers/promises'
import {watchStatus} from '../src/status.ts'

const snapshot = (overrides = {}) => ({
  state: 'connected',
  serverId: 'server-a',
  streams: 12,
  total: 12,
  errMsg: '',
  txRate: 0,
  rxRate: 0,
  txTotal: 0,
  rxTotal: 0,
  ...overrides,
})

function setup(t, read) {
  t.mock.timers.enable({apis: ['setInterval']})
  let listener
  let unsubscribed = false
  const values = []
  const stop = watchStatus(
    read,
    (callback) => {
      listener = callback
      return () => {
        unsubscribed = true
      }
    },
    (status) => values.push(status),
  )
  t.after(stop)
  return {
    values,
    stop,
    emit: (status) => listener(status),
    emitSnapshot: (status) => listener(status, true),
    unsubscribed: () => unsubscribed,
  }
}

test('accepts backend traffic pushes without preserving stale zero rates', async (t) => {
  const monitor = setup(t, async () => snapshot())
  await setImmediate()
  monitor.emitSnapshot(snapshot({txRate: 1024, rxRate: 2048, txTotal: 4096, rxTotal: 8192}))
  assert.equal(monitor.values.at(-1).txRate, 1024)
  assert.equal(monitor.values.at(-1).rxTotal, 8192)
  monitor.emit(snapshot({streams: 11}))
  assert.equal(monitor.values.at(-1).rxRate, 2048)
  monitor.emitSnapshot(snapshot({txTotal: 4096, rxTotal: 8192}))
  assert.equal(monitor.values.at(-1).rxRate, 0, 'an idle full snapshot must reset the speed')
})

test('a newer backend push wins over an in-flight poll', async (t) => {
  let resolve
  const monitor = setup(
    t,
    () =>
      new Promise((done) => {
        resolve = done
      }),
  )
  monitor.emitSnapshot(snapshot({rxRate: 2048}))
  resolve(snapshot())
  await setImmediate()
  assert.equal(monitor.values.at(-1).rxRate, 2048)
})

test('refreshes traffic every second without another state event', async (t) => {
  let calls = 0
  const monitor = setup(t, async () => snapshot({txRate: calls++ * 1024, rxRate: calls * 2048}))
  await setImmediate()
  assert.equal(monitor.values.at(-1).txRate, 0)
  t.mock.timers.tick(1000)
  await setImmediate()
  assert.equal(monitor.values.at(-1).txRate, 1024)
  assert.equal(monitor.values.at(-1).rxRate, 4096)
  monitor.emit(snapshot({streams: 11}))
  assert.equal(monitor.values.at(-1).txRate, 1024, 'lifecycle event must not erase traffic')
  assert.equal(monitor.values.at(-1).streams, 11)
})

test('does not restore a stale connected snapshot after disconnect', async (t) => {
  let resolve
  const monitor = setup(
    t,
    () =>
      new Promise((done) => {
        resolve = done
      }),
  )
  t.mock.timers.tick(3000)
  monitor.emit(snapshot({state: 'idle'}))
  resolve(snapshot({txRate: 1024}))
  await setImmediate()
  assert.equal(monitor.values.length, 1)
  assert.equal(monitor.values.at(-1).state, 'idle')
})

test('clears traffic on disconnect and server switch', async (t) => {
  const monitor = setup(t, async () => snapshot({txRate: 1024, rxRate: 2048}))
  await setImmediate()
  monitor.emit(snapshot({serverId: 'server-b'}))
  assert.equal(monitor.values.at(-1).txRate, 0)
  monitor.emit(snapshot({state: 'idle'}))
  assert.equal(monitor.values.at(-1).rxRate, 0)
})

test('retries after a failed sample and cancels polling on disposal', async (t) => {
  let calls = 0
  const monitor = setup(t, async () => {
    if (++calls === 1) throw new Error('bridge unavailable')
    return snapshot({rxRate: 2048})
  })
  await setImmediate()
  t.mock.timers.tick(1000)
  await setImmediate()
  assert.equal(monitor.values.at(-1).rxRate, 2048)
  monitor.stop()
  assert.equal(monitor.unsubscribed(), true)
  t.mock.timers.tick(3000)
  await setImmediate()
  assert.equal(calls, 2)
})

test('ignores an in-flight response after disposal', async (t) => {
  let resolve
  const monitor = setup(
    t,
    () =>
      new Promise((done) => {
        resolve = done
      }),
  )
  monitor.stop()
  resolve(snapshot({rxRate: 2048}))
  await setImmediate()
  assert.equal(monitor.values.length, 0)
})

test('pending captcha is recovered from snapshots without marking working streams disconnected', async (t) => {
  const monitor = setup(t, async () => snapshot({streams: 1, total: 2, captchaUrl: 'http://localhost:8765', rxRate: 2048}))
  await setImmediate()
  const current = monitor.values.at(-1)
  assert.equal(current.state, 'connected')
  assert.equal(current.captchaUrl, 'http://localhost:8765')
  assert.equal(current.rxRate, 2048)
  monitor.emitSnapshot(snapshot({streams: 1, total: 2, captchaUrl: '', rxRate: 4096}))
  assert.equal(monitor.values.at(-1).captchaUrl, '')
  assert.equal(monitor.values.at(-1).state, 'connected')
  assert.equal(monitor.values.at(-1).rxRate, 4096)
})
