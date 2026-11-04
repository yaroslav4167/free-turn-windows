import {test} from 'node:test'
import assert from 'node:assert/strict'
import {formatUpdateBytes, formatUpdateETA, updateProgress} from '../src/updateProgress.ts'

const idle = {busy: '', phase: '', received: 0, total: 0, bytesPerSecond: 0, remainingSeconds: -1}

test('download progress shows bytes, percent, speed and remaining time', () => {
  const progress = updateProgress({...idle, busy: 'preparing', phase: 'downloading', received: 2 * 1024 * 1024,
    total: 8 * 1024 * 1024, bytesPerSecond: 1024 * 1024, remainingSeconds: 6}, 'installing')
  assert.equal(progress.percent, 25)
  assert.equal(progress.transfer, '2,0 МиБ из 8,0 МиБ')
  assert.equal(progress.speed, '1,0 МиБ/с')
  assert.equal(progress.eta, 'Осталось ≈ 6 с')
})

test('verification, extraction and restart never pretend that installation is 100 percent done', () => {
  for (const phase of ['checksum', 'connecting', 'verifying', 'extracting', 'restarting']) {
    const progress = updateProgress({...idle, busy: 'preparing', phase, received: 100, total: 100}, 'installing')
    assert.equal(progress.phase, phase)
    assert.equal(progress.percent, null)
    assert.equal(progress.transfer, '')
    assert.equal(progress.eta, '')
  }
  assert.match(updateProgress({...idle, busy: 'restarting', phase: 'restarting'}, 'installing').detail, /окно закроется/)
})

test('progress appears immediately while waiting for a native dialog and disappears when cancelled', () => {
  assert.equal(updateProgress(idle, null), null)
  assert.equal(updateProgress(idle, 'installing').phase, 'confirming')
  assert.equal(updateProgress(idle, 'downloading').percent, null)
  assert.equal(updateProgress(idle, 'checking').phase, 'checking')
  assert.equal(updateProgress(null, 'installing').phase, 'confirming')
})

test('incomplete and malformed transfer data never shows NaN or premature 100 percent', () => {
  const download = {...idle, busy: 'downloading', phase: 'downloading'}
  assert.equal(updateProgress({...download, received: 999, total: 1000}, null).percent, 99)
  assert.equal(updateProgress({...download, received: 1001, total: 1000}, null).percent, 100)
  const unknown = updateProgress({...download, received: NaN, total: Infinity}, null)
  assert.equal(unknown.percent, null)
  assert.equal(unknown.transfer, '0 Б из …')
  assert.equal(unknown.speed, 'Скорость рассчитывается…')
  assert.equal(unknown.eta, 'Время рассчитывается…')
})

test('sizes and estimates are readable and bounded', () => {
  assert.equal(formatUpdateBytes(-1), '0 Б')
  assert.equal(formatUpdateBytes(Infinity), '0 Б')
  assert.equal(formatUpdateBytes(512), '512 Б')
  assert.equal(formatUpdateETA(-1), 'Время рассчитывается…')
  assert.equal(formatUpdateETA(NaN), 'Время рассчитывается…')
  assert.equal(formatUpdateETA(61), 'Осталось ≈ 2 мин')
  assert.equal(formatUpdateETA(3600), 'Осталось ≈ 1 ч')
})
