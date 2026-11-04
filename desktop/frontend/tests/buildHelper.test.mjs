import {test} from 'node:test'
import assert from 'node:assert/strict'
import {mkdtempSync, mkdirSync, readFileSync, writeFileSync, existsSync, rmSync} from 'node:fs'
import {tmpdir} from 'node:os'
import {join, resolve} from 'node:path'
import {fileURLToPath} from 'node:url'
import {spawnSync} from 'node:child_process'

const root = fileURLToPath(new URL('../../../', import.meta.url))
const builder = join(root, 'packaging/linux/build-vpn-helper.sh')

test('normal Linux Wails builds include the helper post-build hook', () => {
  const config = JSON.parse(readFileSync(join(root, 'desktop/wails.json'), 'utf8'))
  assert.equal(config.postBuildHooks['linux/*'], 'sh ../../../packaging/linux/build-vpn-helper.sh ${platform} ${bin}')
  assert.equal(resolve(root, 'desktop/build/bin', '../../../packaging/linux/build-vpn-helper.sh'), builder)
})

test(
  'helper build matches target architecture and handles output paths with spaces',
  {skip: process.platform !== 'linux'},
  (t) => {
    const temporary = mkdtempSync(join(tmpdir(), 'freeturn-build-'))
    t.after(() => rmSync(temporary, {recursive: true, force: true}))
    const tools = join(temporary, 'tools')
    const output = join(temporary, 'Desktop build with spaces')
    mkdirSync(tools)
    mkdirSync(output)
    const app = join(output, 'freeturn-desktop')
    const log = join(temporary, 'go-arguments.txt')
    writeFileSync(app, '')
    writeFileSync(
      join(tools, 'go'),
      `#!/bin/sh
set -eu
printf '%s\\n' "GOOS=$GOOS" "GOARCH=$GOARCH" "CGO_ENABLED=$CGO_ENABLED" "$@" > "$GO_STUB_LOG"
output=
while [ "$#" -gt 0 ]; do
  if [ "$1" = '-o' ]; then shift; output=$1; fi
  shift
done
: > "$output"
`,
      {mode: 0o755},
    )
    const env = {...process.env, PATH: tools + ':' + process.env.PATH, GO_STUB_LOG: log}
    const result = spawnSync('sh', [builder, 'linux/arm64', app], {env, encoding: 'utf8'})
    assert.equal(result.status, 0, result.stderr)
    const args = readFileSync(log, 'utf8').trim().split('\n')
    assert.deepEqual(args.slice(0, 3), ['GOOS=linux', 'GOARCH=arm64', 'CGO_ENABLED=0'])
    assert.ok(args.includes(join(output, 'freeturn-vpn-helper')))
    assert.ok(existsSync(join(output, 'freeturn-vpn-helper')))
    const invalid = spawnSync('sh', [builder, 'windows/amd64', app], {env, encoding: 'utf8'})
    assert.notEqual(invalid.status, 0)
  },
)
