import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'
import {connectionState} from '../src/connectionState.ts'

const read = file => readFileSync(new URL('../src/' + file, import.meta.url), 'utf8')

test('captcha action is independent of connection phase and visible on every screen', () => {
  const notice = read('components/CaptchaNotice.tsx')
  assert.match(notice, /if \(!url\) return null/)
  assert.match(notice, /BrowserOpenURL\(url\)/)
  assert.doesNotMatch(notice, /state === 'captcha'/)
  const app = read('App.tsx')
  assert.match(app, /<CaptchaNotice url=\{status\.captchaUrl \|\| ''\}/)
  assert.ok(app.indexOf('<CaptchaNotice') < app.indexOf("{view !== 'settings'"))
})

test('connected captcha warning keeps disconnect action and uses amber rather than a disconnected state', () => {
  const state = connectionState('connected', false, true)
  assert.equal(state.connected, true)
  assert.equal(state.action, 'Отключить')
  assert.equal(state.heading, 'VPN включён')
  const component = read('components/Connection.tsx')
  assert.match(component, /verificationRequired = connected && !!status\.captchaUrl/)
  assert.match(component, /verification-required/)
  const css = read('style.css')
  assert.match(css, /\.verification-required \.power-button\s*\{[^}]*--power-fill:\s*#493f26/)
})
