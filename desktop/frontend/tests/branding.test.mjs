import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'
import {fileURLToPath} from 'node:url'
import {join} from 'node:path'

const root = fileURLToPath(new URL('../../../', import.meta.url))

test('favicon and frontend icon match the canonical SVG', () => {
  const original = readFileSync(join(root, 'desktop/assets/icon.svg'))
  assert.deepEqual(readFileSync(join(root, 'desktop/frontend/public/freeturn.svg')), original)
  assert.match(readFileSync(join(root, 'desktop/frontend/index.html'), 'utf8'), /href="\/freeturn\.svg"/)
})

test('Wails build uses the generated high-resolution application icon', () => {
  const png = readFileSync(join(root, 'desktop/build/appicon.png'))
  assert.equal(png.subarray(1, 4).toString(), 'PNG')
  assert.equal(png.readUInt32BE(16), 1024)
  assert.equal(png.readUInt32BE(20), 1024)
})

test('README screenshots are present and large enough to read', () => {
  const readme = readFileSync(join(root, 'README.md'), 'utf8')
  const images = [...readme.matchAll(/!\[[^\]]*\]\((docs\/screenshots\/[^)]+\.png)\)/g)]
  assert.equal(images.length, 3)
  for (const [, path] of images) {
    const png = readFileSync(join(root, path))
    assert.equal(png.subarray(1, 4).toString(), 'PNG')
    assert.ok(png.readUInt32BE(16) >= 900, `screenshot is too narrow: ${path}`)
    assert.ok(png.readUInt32BE(20) >= 600, `screenshot is too short: ${path}`)
  }
})
