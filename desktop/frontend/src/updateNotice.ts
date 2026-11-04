import type {UpdateState} from './desktop'

const key = 'freeturn:dismissed-update'

// Include the source so dismissing a release from one fork cannot hide another.
export function updateNoticeId(state: Pick<UpdateState, 'repository' | 'latestVersion'>): string {
  return JSON.stringify([state.repository, state.latestVersion])
}

export function readDismissedUpdate(storage?: Pick<Storage, 'getItem'>): string {
  try {
    return (storage ?? localStorage).getItem(key) || ''
  } catch {
    return ''
  }
}

export function saveDismissedUpdate(id: string, storage?: Pick<Storage, 'setItem'>): void {
  try {
    (storage ?? localStorage).setItem(key, id)
  } catch {
    // In-memory dismissal still works if persistent storage is unavailable.
  }
}
