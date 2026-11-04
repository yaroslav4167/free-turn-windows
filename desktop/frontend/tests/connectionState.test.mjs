import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'
import {connectionState} from '../src/connectionState.ts'

test('connected VPN has an explicit on state and disconnect action', () => {
  const state = connectionState('connected', false, true)
  assert.equal(state.connected, true)
  assert.equal(state.heading, 'VPN включён')
  assert.equal(state.action, 'Отключить')
  assert.equal(state.ariaLabel, 'Отключить VPN')
  assert.equal(state.busy, false)
})

test('TURN connection is not labelled as a VPN', () => {
  assert.equal(connectionState('connected', false, false).heading, 'Реле запущено')
})

test('finishing VPN setup does not prematurely show enabled', () => {
  const state = connectionState('connected', true, true)
  assert.equal(state.connected, false)
  assert.equal(state.visualState, 'connecting')
  assert.equal(state.busy, true)
  assert.equal(state.action, 'Отменить')
})

test('idle, captcha and errors do not show an enabled state', () => {
  for (const phase of ['idle', 'captcha', 'error']) assert.equal(connectionState(phase, false, true).connected, false)
  assert.equal(connectionState('idle', false, true).busy, false)
  assert.equal(connectionState('captcha', true, true).heading, 'Требуется проверка')
  assert.equal(connectionState('captcha', true, true).busy, false)
  assert.equal(connectionState('error', true, true).busy, false)
})

test('connecting and captcha use the main button to cancel without an extra button', () => {
  for (const phase of ['connecting', 'captcha']) {
    const state = connectionState(phase, false, true)
    assert.equal(state.active, true)
    assert.equal(state.action, 'Отменить')
    assert.equal(state.ariaLabel, 'Отменить подключение')
  }
  const component = readFileSync(new URL('../src/components/Connection.tsx', import.meta.url), 'utf8')
  assert.match(component, /onClick=\{active \? onDisconnect : onConnect\}/)
  assert.doesNotMatch(component, /Отменить подключение/)
})
