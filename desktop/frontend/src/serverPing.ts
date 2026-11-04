export interface ServerPingResult {serverId: string, host: string, milliseconds: number}
export interface ServerPing {serverId: string, host: string, milliseconds: number | null, measuring: boolean}
interface Clock {set: (callback: () => void, delay: number) => unknown, clear: (handle: unknown) => void}

export function watchServerPing(serverId: string, read: (id: string) => Promise<ServerPingResult>, publish: (state: ServerPing) => void,
  clock: Clock = {set: (fn, ms) => setTimeout(fn, ms), clear: handle => clearTimeout(handle as ReturnType<typeof setTimeout>)}) {
  let active = true
  let timer: unknown
  publish({serverId, host: '', milliseconds: null, measuring: true})
  const measure = async () => {
    let sample: ServerPing = {serverId, host: '', milliseconds: null, measuring: false}
    try {
      const result = await read(serverId)
      if (result.serverId === serverId && Number.isFinite(result.milliseconds) && result.milliseconds >= 0) {
        sample = {serverId, host: result.host, milliseconds: result.milliseconds, measuring: false}
      }
    } catch { /* ICMP may be unavailable even while VPN traffic works. */ }
    if (!active) return
    publish(sample)
    timer = clock.set(measure, 5000)
  }
  measure()
  return () => { active = false; if (timer !== undefined) clock.clear(timer) }
}
