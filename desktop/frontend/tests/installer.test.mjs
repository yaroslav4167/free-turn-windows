import {test} from 'node:test'
import assert from 'node:assert/strict'
import {mkdtempSync, mkdirSync, writeFileSync, cpSync, readFileSync, rmSync} from 'node:fs'
import {tmpdir} from 'node:os'
import {join} from 'node:path'
import {spawnSync} from 'node:child_process'

test('installer configures helper and asks about shortcut; uninstaller requires consent', {skip: process.platform !== 'linux'}, (t) => {
  const dir = mkdtempSync(join(tmpdir(), 'freeturn-install-'))
  t.after(() => rmSync(dir, {recursive: true, force: true}))
  const bin = join(dir, 'tools')
  mkdirSync(bin)
  writeFileSync(join(bin, 'id'), '#!/bin/sh\necho 1000\n', {mode: 0o755})
  writeFileSync(join(dir, 'freeturn-desktop'), '#!/bin/sh\necho "$*" >> "$LOG"\nif [ "$1" = --configure-vpn-helper ] && [ "${FAIL_HELPER:-}" = 1 ]; then exit 1; fi\n', {mode: 0o755})
  for (const script of ['install-desktop.sh', 'uninstall-desktop.sh']) {
    cpSync(new URL('../../../packaging/linux/' + script, import.meta.url), join(dir, script))
  }
  const log = join(dir, 'log')
  const run = (script, input, extra = {}) => {
    writeFileSync(log, '')
    const result = spawnSync('sh', [join(dir, script)], {input, encoding: 'utf8', env: {...process.env, PATH: bin + ':' + process.env.PATH, LOG: log, ...extra}})
    return {...result, calls: readFileSync(log, 'utf8')}
  }
  let result = run('install-desktop.sh', 'да\n')
  assert.equal(result.status, 0, result.stderr)
  assert.equal(result.calls, '--install-desktop\n--configure-vpn-helper\n--create-desktop-shortcut\n')
  result = run('install-desktop.sh', '')
  assert.equal(result.calls, '--install-desktop\n--configure-vpn-helper\n')
  result = run('install-desktop.sh', 'нет\n', {FAIL_HELPER: '1'})
  assert.equal(result.status, 1)
  assert.match(result.stderr, /VPN-helper не настроен/)
  result = run('uninstall-desktop.sh', '')
  assert.equal(result.status, 0)
  assert.equal(result.calls, '')
  result = run('uninstall-desktop.sh', 'y\n')
  assert.equal(result.status, 0)
  assert.equal(result.calls, '--uninstall-desktop\n')
})

test('settings keep updates last and hide persistent installation details', () => {
  const read = (path) => readFileSync(new URL('../src/components/' + path, import.meta.url), 'utf8')
  const settings = read('DesktopPreferences.tsx')
  assert.ok(settings.indexOf('<UpdatePreferences') > settings.lastIndexOf('settings-card'))
  assert.doesNotMatch(settings, /InstallVPNHelper|InstallDesktop|CreateDesktopShortcut|renderingNote/)
  const updates = read('UpdatePreferences.tsx')
  assert.doesNotMatch(updates, /installationMessage|backupPath/)
  assert.match(updates, /InfoTip/)
  assert.doesNotMatch(updates, /«Скачать и установить» запрашивает/)
})
