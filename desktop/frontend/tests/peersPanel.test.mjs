import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'

test('late SSH responses cannot replace the next server peer list', () => {
  const source = readFileSync(new URL('../src/components/PeersPanel.tsx', import.meta.url), 'utf8')
  assert.match(source, /const request = \{generation: generation.current\}/)
  assert.match(source, /const result = await ServerPeers\(server.id\)\s+if \(request.generation !== generation.current\) return\s+setList\(result\)/)
  assert.match(source, /catch \(e\) \{\s+if \(request.generation !== generation.current\) return/)
  assert.match(source, /generation.current\+\+/)
  assert.match(source, /if \(inflight.current === request\) inflight.current = null/)
})
