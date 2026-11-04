import {test} from 'node:test'
import assert from 'node:assert/strict'
import {setImmediate} from 'node:timers/promises'
import {readFileSync} from 'node:fs'
import {watchStatus} from '../src/status.ts'

const snapshot = (change = {}) => ({state: 'connected', serverId: 'a', streams: 1, total: 2,
  txRate: 0, rxRate: 2048, txTotal: 0, rxTotal: 8192, captchaUrl: '', errMsg: '', ...change})

test('snapshots recover captcha and retain a working connection', async t => {
  let emit
  const values = []
  const stop = watchStatus(async () => snapshot({captchaUrl: 'https://example.test/check'}), callback => {
    emit = callback
    return () => {}
  }, s => values.push(s))
  t.after(stop)
  await setImmediate()
  assert.equal(values.at(-1).state, 'connected')
  assert.equal(values.at(-1).captchaUrl, 'https://example.test/check')
  emit(snapshot({rxRate: 0}), false)
  assert.equal(values.at(-1).rxRate, 2048)
  emit(snapshot({rxRate: 0}), true)
  assert.equal(values.at(-1).rxRate, 0)
  assert.equal(values.at(-1).captchaUrl, '')
})

test('disconnect wins over an older in-flight poll', async t => {
  let resolve, emit
  const values = []
  const stop = watchStatus(() => new Promise(r => { resolve = r }), callback => {
    emit = callback
    return () => {}
  }, s => values.push(s))
  t.after(stop)
  emit(snapshot({state: 'idle', captchaUrl: ''}))
  resolve(snapshot({captchaUrl: 'https://example.test/stale'}))
  await setImmediate()
  assert.equal(values.length, 1)
  assert.equal(values[0].state, 'idle')
})

test('polling retries failures and stops on disposal', async t => {
  t.mock.timers.enable({apis: ['setInterval']})
  let calls = 0, off = false
  const values = []
  const stop = watchStatus(async () => {
    if (++calls === 1) throw Error('bridge offline')
    return snapshot()
  }, () => () => { off = true }, s => values.push(s))
  t.after(stop)
  await setImmediate()
  t.mock.timers.tick(1000)
  await setImmediate()
  assert.equal(values.at(-1).rxRate, 2048)
  stop()
  t.mock.timers.tick(3000)
  assert.equal(calls, 2)
  assert.equal(off, true)
})

test('VK editing preserves blank lines and captcha action is not tied to phase', () => {
  const form = readFileSync(new URL('../src/components/ServerForm.tsx', import.meta.url), 'utf8')
  assert.match(form, /vkLinks: e.target.value.split\('\\n'\)/)
  const app = readFileSync(new URL('../src/App.tsx', import.meta.url), 'utf8')
  assert.match(app, /status.captchaUrl &&/)
  assert.match(app, /BrowserOpenURL\(status.captchaUrl\)/)
})

test('existing server saves a local profile without running remote mutation commands', () => {
  const wizard = readFileSync(new URL('../src/components/CreateServerWizard.tsx', import.meta.url), 'utf8')
  assert.match(wizard, /if \(probe.installed \|\| probe.running\)/)
  assert.match(wizard, /if \(fresh.installed \|\| fresh.running\)/)
  const existing = wizard.slice(wizard.indexOf("if (step === 'existing')"), wizard.indexOf("if (step === 'config')"))
  assert.match(existing, /SaveServer\(existingDraft\)/)
  assert.doesNotMatch(existing, /SetupInstall|SetupStart|SetupWgSetup|SetupCreateServer/)
  assert.ok(wizard.indexOf('const fresh = await SetupProbe(ssh)') < wizard.indexOf('await SetupInstall(ssh)'))
})
