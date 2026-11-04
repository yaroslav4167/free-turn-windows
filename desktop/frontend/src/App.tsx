import {useEffect, useMemo, useRef, useState} from 'react'
import {backend, BrowserOpenURL, Connect, CoreVersion, DeleteServer, Disconnect, EventsOn, ListServers, SaveServer, Status} from './api'
import {watchStatus} from './status'
import Sidebar from './components/Sidebar'
import ServerDetail from './components/ServerDetail'
import AddServerModal from './components/AddServerModal'
import {LogEntry} from './components/LogConsole'

const MAX_LOG_LINES = 500

const IDLE_STATUS: backend.Status = {
  state: 'idle', streams: 0, total: 0, errMsg: '', txRate: 0, rxRate: 0, txTotal: 0, rxTotal: 0, serverId: '', captchaUrl: '',
}

function sameServer(a: backend.Server | null, b: backend.Server | null): boolean {
  if (a === b) return true
  if (!a || !b) return false
  return JSON.stringify(a) === JSON.stringify(b)
}

export default function App() {
  const [servers, setServers] = useState<backend.Server[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [draft, setDraft] = useState<backend.Server | null>(null)
  const [status, setStatus] = useState<backend.Status>(IDLE_STATUS)
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [version, setVersion] = useState('')
  const [connectError, setConnectError] = useState('')
  const [pendingServerId, setPendingServerId] = useState('')
  const connectionRequest = useRef(0)
  const [showAdd, setShowAdd] = useState(false)

  useEffect(() => {
    ListServers().then((list) => {
      setServers(list)
      if (list.length > 0) setSelectedId(list[0].id)
    })
    CoreVersion().then(setVersion)
    const stopStatus = watchStatus(Status, (listener) => {
      const offState = EventsOn('core:state', (s: backend.Status) => listener(s))
      const offStats = EventsOn('core:stats', (s: backend.Status) => listener(s, true))
      return () => { offState(); offStats() }
    }, setStatus)
    const offLog = EventsOn('core:log', (l: LogEntry) =>
      setLogs((prev) => {
        const next = [...prev, l]
        return next.length > MAX_LOG_LINES ? next.slice(next.length - MAX_LOG_LINES) : next
      })
    )
    return () => {
      stopStatus()
      offLog()
    }
  }, [])

  useEffect(() => {
    const saved = servers.find((s) => s.id === selectedId) ?? null
    setDraft(saved ? backend.Server.createFrom(saved) : null)
    setConnectError('')
  }, [selectedId, servers])

  const savedSelected = useMemo(() => servers.find((s) => s.id === selectedId) ?? null, [servers, selectedId])
  const dirty = !sameServer(draft, savedSelected)

  const handleSave = async () => {
    if (!draft) return
    const saved = await SaveServer(draft)
    setServers((prev) => prev.map((s) => (s.id === saved.id ? saved : s)))
    setDraft(saved)
  }

  const handleDelete = async () => {
    if (!draft?.id) return
    if (!window.confirm(`Удалить сервер «${draft.name || draft.peer}»?`)) return
    await DeleteServer(draft.id)
    setServers((prev) => prev.filter((s) => s.id !== draft.id))
    setSelectedId(null)
  }

  const handleConnect = async () => {
    if (!draft) return
    const request = ++connectionRequest.current
    setPendingServerId(draft.id)
    setConnectError('')
    try {
      // Connect always force-stops whatever was running first (a different
      // server, or a stuck half-failed attempt on this one), so no need to
      // orchestrate a separate Disconnect from here.
      await Connect(draft.id)
    } catch (e) {
      if (request === connectionRequest.current) setConnectError(String(e))
    } finally {
      if (request === connectionRequest.current) setPendingServerId('')
    }
  }

  const handleDisconnect = () => {
    connectionRequest.current++
    setPendingServerId('')
    void Disconnect().catch(e => setConnectError(String(e)))
  }

  return (
    <div className="app-shell">
      <Sidebar
        servers={servers}
        selectedId={selectedId}
        activeState={status.state}
        activeServerId={status.serverId}
        onSelect={setSelectedId}
        onAdd={() => setShowAdd(true)}
        version={version}
      />
      <div className="main">
        {status.captchaUrl && <div className="notice captcha-notice" role="status" id="captcha-notice">
          <span>{status.state === 'connected'
            ? 'Соединение работает. Для части потоков требуется проверка VK.'
            : 'Для подключения требуется проверка VK в браузере.'}</span>
          <button className="btn small" onClick={() => BrowserOpenURL(status.captchaUrl)}>Открыть проверку в браузере</button>
        </div>}
        {!draft ? (
          <div className="empty-state">
            <p>Выберите сервер слева или добавьте новый, чтобы подключиться через free-turn-proxy.</p>
          </div>
        ) : (
          <ServerDetail
            server={draft}
            dirty={dirty}
            status={status}
            isActive={status.serverId === draft.id}
            pending={pendingServerId === draft.id}
            logs={logs}
            onChange={setDraft}
            onSave={handleSave}
            onDelete={handleDelete}
            onConnect={handleConnect}
            onDisconnect={handleDisconnect}
            onClearLogs={() => setLogs([])}
          />
        )}
        {connectError && (
          <div className="notice error" style={{position: 'absolute', bottom: 240, left: 280, right: 20}}>
            {connectError}
          </div>
        )}
      </div>
      {showAdd && (
        <AddServerModal
          onClose={() => setShowAdd(false)}
          onCreated={(s) => {
            setServers((prev) => [...prev, s])
            setSelectedId(s.id)
            setShowAdd(false)
          }}
        />
      )}
    </div>
  )
}
