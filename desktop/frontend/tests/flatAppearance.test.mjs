import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'
import {roundedPowerShape, circlePowerShape} from '../src/powerShape.ts'

const css = readFileSync(new URL('../src/style.css', import.meta.url), 'utf8')

test('interface has no blur, shadows or decorative gradients', () => {
  assert.doesNotMatch(css, /(?:backdrop-filter|\bfilter|box-shadow)\s*:/)
  assert.doesNotMatch(css, /(?:radial|linear|conic)-gradient\s*\(/)
  assert.doesNotMatch(css, /\.app-shell::(?:before|after)/)
})

test('idle screens and traffic charts do not animate', () => {
  const keyframes = [...css.matchAll(/@keyframes\s+([\w-]+)/g)].map((match) => match[1])
  assert.deepEqual(keyframes, ['connection-trace'])
  assert.match(css, /\.power-button\[aria-busy='true'\] \.power-progress\s*\{\s*opacity:\s*1/)
  assert.match(css, /\.power-progress-path\s*\{[^}]*animation-play-state:\s*paused/)
  assert.match(css, /\.power-button\[aria-busy='true'\] \.power-progress-path\s*\{\s*animation-play-state:\s*running/)
  assert.match(css, /prefers-reduced-motion:\s*reduce/)
  assert.doesNotMatch(css, /(?:ambient-drift|page-arrive|panel-arrive|chart-draw|chart-fill)/)
})

test('flat styling retains focus, connection, error and disabled states', () => {
  assert.match(css, /button:focus-visible[\s\S]*?outline:\s*2px solid var\(--accent\)/)
  assert.match(css, /\.connected \.power-button\s*\{[^}]*--power-border:/)
  assert.match(css, /\.error \.power-button\s*\{[^}]*--power-border:/)
  assert.match(css, /\.power-button:disabled\s*\{[^}]*opacity:/)
})

test('rounded star and progress share a contour with fixed state-independent dimensions', () => {
  const component = readFileSync(new URL('../src/components/Connection.tsx', import.meta.url), 'utf8')
  assert.equal([...component.matchAll(/d=\{powerShape\}/g)].length, 2)
  assert.equal([...roundedPowerShape.matchAll(/\bQ\b/g)].length, 20)
  assert.doesNotMatch(roundedPowerShape, /\bL\b|NaN|Infinity/)
  assert.equal([...circlePowerShape.matchAll(/\bQ\b/g)].length, 20)
  assert.match(component, /powerShapeAt\(motion\.morph\)/)
  assert.match(component, /<Icon name="power" size=\{48\}/)
  assert.match(css, /\.power-button\s*\{[^}]*width:\s*150px;[^}]*height:\s*190px;/)
  for (const state of ['connected', 'error']) {
    const rule = css.match(new RegExp('\\.' + state + ' \\.power-button\\s*\\{([^}]+)\\}'))?.[1] || ''
    assert.doesNotMatch(rule, /(?:width|height|margin|padding):/)
  }
})

test('only the contour rotates while connecting; icon and label stay still', () => {
  const component = readFileSync(new URL('../src/components/Connection.tsx', import.meta.url), 'utf8')
  assert.match(component, /usePowerMotion\(connected, presentation\.busy\)/)
  assert.equal([...component.matchAll(/style=\{contourStyle\}/g)].length, 2)
  const symbol = css.match(/\.power-symbol\s*\{([^}]+)\}/)?.[1] || ''
  const label = css.match(/\.power-action\s*\{([^}]+)\}/)?.[1] || ''
  assert.doesNotMatch(symbol + label, /animation:|transform:/)
})
