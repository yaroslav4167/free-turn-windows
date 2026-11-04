import {useEffect, useState} from 'react'
import {desktop} from './desktop'
import {ServerPing, watchServerPing} from './serverPing'

export function useServerPing(serverId: string, connected: boolean): ServerPing {
  const [state, setState] = useState<ServerPing>({serverId: '', host: '', milliseconds: null, measuring: false})
  useEffect(() => {
    if (!connected || !serverId) {
      setState({serverId: '', host: '', milliseconds: null, measuring: false})
      return
    }
    return watchServerPing(serverId, id => desktop().MeasurePing(id), setState)
  }, [serverId, connected])
  return connected && state.serverId === serverId ? state : {serverId: '', host: '', milliseconds: null, measuring: false}
}
