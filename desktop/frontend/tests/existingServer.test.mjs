import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'

const wizard = readFileSync(new URL('../src/components/CreateServerWizard.tsx', import.meta.url), 'utf8')
test('existing installations bypass provisioning and save only a local profile', () => {
  assert.match(wizard, /if \(probe.installed \|\| probe.running\)/)
  const existing = wizard.slice(wizard.indexOf("if (step === 'existing')"), wizard.indexOf("if (step === 'config')"))
  assert.match(existing, /SaveServer\(existingDraft\)/)
  assert.doesNotMatch(existing, /SetupInstall|SetupStart|SetupWgSetup|SetupCreateServer/)
  assert.ok(wizard.indexOf('const fresh = await SetupProbe(ssh)') < wizard.indexOf('await SetupInstall(ssh)'))
  assert.match(wizard, /if \(fresh.installed \|\| fresh.running\)/)
})
