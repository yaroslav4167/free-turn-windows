import {test} from 'node:test'
import assert from 'node:assert/strict'
import {EMPTY_TRAFFIC_HISTORY, recordTraffic, sparklinePaths} from '../src/trafficHistory.ts'

const status = (rxRate, txRate, overrides = {}) => ({state: 'connected', serverId: 'a', rxRate, txRate, ...overrides})

test('keeps separate receive/send histories and samples once per second', () => {
  let history = recordTraffic(EMPTY_TRAFFIC_HISTORY, status(1024, 2048), 1000)
  assert.deepEqual(history.rx, [1024])
  assert.deepEqual(history.tx, [2048])
  assert.equal(recordTraffic(history, status(5000, 6000), 1100), history)
  history = recordTraffic(history, status(3000, 4000), 2000)
  assert.deepEqual(history.rx, [1024, 3000])
  assert.deepEqual(history.tx, [2048, 4000])
})

test('history is bounded and cleared on server switch or disconnect', () => {
  let history = EMPTY_TRAFFIC_HISTORY
  for (let i = 0; i < 100; i++) history = recordTraffic(history, status(i, i * 2), i * 1000)
  assert.equal(history.rx.length, 60)
  assert.equal(history.rx[0], 40)
  history = recordTraffic(history, status(1, 2, {serverId: 'b'}), 100000)
  assert.deepEqual(history.rx, [1])
  history = recordTraffic(history, status(0, 0, {state: 'idle'}), 101000)
  assert.equal(history.rx.length, 0)
})

test('sparkline produces a bounded area with no NaN on idle data', () => {
  assert.deepEqual(sparklinePaths([]), {line: '', area: ''})
  assert.deepEqual(sparklinePaths([1]), {line: '', area: ''})
  const {line, area} = sparklinePaths([0, 10, 0, 20])
  assert.equal(line, 'M0.00,39.00 L33.33,22.00 L66.67,39.00 L100.00,5.00')
  assert.ok(area.endsWith('L100,40 L0,40 Z'))
  assert.ok(!sparklinePaths([0, 0]).line.includes('NaN'))
})
