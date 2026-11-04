import {test} from 'node:test'
import assert from 'node:assert/strict'
import {updateNoticeId, readDismissedUpdate, saveDismissedUpdate} from '../src/updateNotice.ts'

test('dismissed update stays hidden only for the same repository and version', () => {
  const values = new Map()
  const storage = {getItem: (key) => values.get(key) ?? null, setItem: (key, value) => values.set(key, value)}
  const release = {repository: 'example/fork', latestVersion: 'v2.1.0'}
  const id = updateNoticeId(release)
  assert.equal(readDismissedUpdate(storage), '')
  saveDismissedUpdate(id, storage)
  assert.equal(readDismissedUpdate(storage), id)
  assert.notEqual(updateNoticeId({...release, latestVersion: 'v2.2.0'}), id)
  assert.notEqual(updateNoticeId({...release, repository: 'other/fork'}), id)
})

test('unavailable storage does not break the update notification', () => {
  const storage = {getItem: () => {throw new Error('denied')}, setItem: () => {throw new Error('full')}}
  assert.equal(readDismissedUpdate(storage), '')
  assert.doesNotThrow(() => saveDismissedUpdate('release', storage))
})
