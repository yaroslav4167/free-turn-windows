import type {backend} from './api'

// State events carry lifecycle changes, not traffic counters. Sample the full
// snapshot separately, including while the window is hidden in the tray.
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
    if (
      !fullSnapshot &&
      current?.state === 'connected' &&
      status.state === 'connected' &&
      current.serverId === status.serverId
    ) {
      status = {
        ...status,
        txRate: current.txRate,
        rxRate: current.rxRate,
        txTotal: current.txTotal,
        rxTotal: current.rxTotal,
      }
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
      // A disconnect or server switch may have happened during this request.
      if (!disposed && requestedRevision === revision) {
        current = status
        publish(status)
      }
    } catch {
      // A transient bridge failure must not stop subsequent samples.
    } finally {
      pending = false
    }
  }

  void poll()
  const timer = setInterval(() => {
    void poll()
  }, 1000)
  return () => {
    if (disposed) return
    disposed = true
    clearInterval(timer)
    off()
  }
}
