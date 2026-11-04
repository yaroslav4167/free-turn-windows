import type {backend} from './api'

export interface TrafficHistory {
  serverId: string
  sampledAt: number
  rx: number[]
  tx: number[]
}

export const EMPTY_TRAFFIC_HISTORY: TrafficHistory = {serverId: '', sampledAt: -Infinity, rx: [], tx: []}

// Push at most one point per second, regardless of how many lifecycle events,
// backend pushes or polling responses arrive during that interval.
export function recordTraffic(history: TrafficHistory, status: backend.Status, now: number): TrafficHistory {
  if (status.state === 'idle' || status.state === 'error') return EMPTY_TRAFFIC_HISTORY
  if (history.serverId !== status.serverId) history = {...EMPTY_TRAFFIC_HISTORY, serverId: status.serverId}
  if (status.state !== 'connected' || now - history.sampledAt < 1000) return history
  return {
    serverId: status.serverId,
    sampledAt: now,
    rx: [...history.rx.slice(-59), Math.max(0, status.rxRate)],
    tx: [...history.tx.slice(-59), Math.max(0, status.txRate)],
  }
}

export function sparklinePaths(values: number[]): {line: string; area: string} {
  if (values.length < 2) return {line: '', area: ''}
  const peak = Math.max(1, ...values)
  const points = values.map((value, index) => {
    const x = (index * 100) / (values.length - 1)
    const y = 39 - (Math.max(0, value) / peak) * 34
    return `${x.toFixed(2)},${y.toFixed(2)}`
  })
  const line = 'M' + points.join(' L')
  return {line, area: line + ' L100,40 L0,40 Z'}
}
