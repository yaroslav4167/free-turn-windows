import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'

const css = readFileSync(new URL('../src/style.css', import.meta.url), 'utf8')
const app = readFileSync(new URL('../src/App.tsx', import.meta.url), 'utf8')
const rule = (selector) => css.match(new RegExp(selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '\\s*\\{([^}]+)\\}'))?.[1] || ''

test('notification text is vertically centered with actions and safely wraps', () => {
  assert.match(rule('.app-error'), /align-items:\s*center/)
  assert.match(rule('.app-error'), /flex-wrap:\s*wrap/)
  assert.match(rule('.app-error > span'), /min-width:\s*0/)
  assert.match(rule('.notice'), /line-height:\s*1\.5/)
  assert.match(rule('.notice'), /overflow-wrap:\s*anywhere/)
  assert.match(rule('.notice > :first-child'), /margin-top:\s*0/)
  assert.match(rule('.notice > :last-child'), /margin-bottom:\s*0/)
})

test('global notices share an overlay outside the page layout', () => {
  const stack = app.slice(app.indexOf('<div className="notification-stack">'), app.indexOf("{view === 'connect' && ("))
  assert.match(stack, /<CaptchaNotice/)
  assert.match(stack, /<UpdateNotice/)
  assert.match(stack, /\{connectError &&/)
  assert.match(app, /createPortal\(<div className="notification-stack">/)
  assert.match(stack, /document\.body/)
  assert.match(rule('.notification-stack'), /position:\s*fixed/)
  assert.match(rule('.notification-stack'), /top:\s*16px/)
  assert.match(rule('.notification-stack'), /left:\s*50%/)
  assert.match(rule('.notification-stack'), /transform:\s*translateX\(-50%\)/)
  assert.match(rule('.notification-stack'), /z-index:\s*8/)
  assert.match(rule('.notification-stack'), /gap:\s*10px/)
  assert.match(rule('.app-error'), /margin:\s*0;/)
})

test('notification overlay is bounded and only notices intercept pointer events', () => {
  assert.match(rule('.notification-stack'), /width:\s*min\(460px, calc\(100vw - 32px\)\)/)
  assert.match(rule('.notification-stack'), /max-height:\s*calc\(100dvh - 32px\)/)
  assert.match(rule('.notification-stack'), /overflow-y:\s*auto/)
  assert.match(rule('.notification-stack'), /pointer-events:\s*none/)
  assert.match(rule('.notification-stack > .notice'), /pointer-events:\s*auto/)
  assert.match(rule('.notification-stack > .notice.error'), /background:\s*#[\da-f]{6};/)
  assert.match(rule('.notification-stack > .notice.info:not(.captcha-notice)'), /background:\s*#[\da-f]{6};/)
})
