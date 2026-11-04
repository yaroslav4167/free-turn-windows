import type {backend} from './api'

// Ignore stale bridge responses after a newer lifecycle event or snapshot.
export function watchStatus(
  readStatus: () => Promise<backend.Status>,
  subscribe: (listener: (status: backend.Status, fullSnapshot?: boolean) => void) => () => void,
  publish: (status: backend.Status) => void,
): () => void {
  let disposed = false
  let pending = false
  let revision = 0
  let current: backend.Status | undefined
  const off = subscribe((status, fullSnapshot = false) => {
    if (disposed) return
    revision++
    if (!fullSnapshot && current?.state === 'connected' && status.state === 'connected' && current.serverId === status.serverId) {
      status = {...status, txRate: current.txRate, rxRate: current.rxRate, txTotal: current.txTotal, rxTotal: current.rxTotal}
    }
    current = status
    publish(status)
  })
  const poll = async () => {
    if (disposed || pending) return
    pending = true
    const requestedRevision = revision
    try {
      const status = await readStatus()
      if (!disposed && requestedRevision === revision) {
        current = status
        publish(status)
      }
    } catch {
      // Retry a transient bridge failure on the next sample.
    } finally {
      pending = false
    }
  }
  void poll()
  const timer = setInterval(() => { void poll() }, 1000)
  return () => {
    if (disposed) return
    disposed = true
    clearInterval(timer)
    off()
  }
}
