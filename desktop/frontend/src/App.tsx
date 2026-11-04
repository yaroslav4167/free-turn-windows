import {useEffect, useMemo, useRef, useState} from 'react'
import {createPortal} from 'react-dom'
import {
  backend,
  Connect,
  CoreVersion,
  DeleteServer,
  Disconnect,
  EventsOff,
  EventsOn,
  ListServers,
  SaveServer,
  Status,
} from './api'
import Sidebar from './components/Sidebar'
import ServerDetail from './components/ServerDetail'
import AddServerModal from './components/AddServerModal'
import LogConsole, {LogEntry} from './components/LogConsole'
import Navigation, {View} from './components/Navigation'
import Connection from './components/Connection'
import DesktopPreferences from './components/DesktopPreferences'
import {UpdateNotice} from './components/UpdatePreferences'
import {desktop} from './desktop'
import {watchStatus} from './status'
import {EMPTY_TRAFFIC_HISTORY, recordTraffic} from './trafficHistory'
import {tr} from './language'
import CaptchaNotice from './components/CaptchaNotice'

const MAX_LOG_LINES = 500

const IDLE_STATUS: backend.Status = {
  state: 'idle',
  streams: 0,
  total: 0,
  errMsg: '',
  txRate: 0,
  rxRate: 0,
  txTotal: 0,
  rxTotal: 0,
  serverId: '',
  captchaUrl: '',
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
  const connectionRequest = useRef(0)
  const [showAdd, setShowAdd] = useState(false)
  const [view, setView] = useState<View>('connect')
  const [settingsOpened, setSettingsOpened] = useState(false)
  const [pending, setPending] = useState(false)
  const [trafficHistory, setTrafficHistory] = useState(EMPTY_TRAFFIC_HISTORY)

  useEffect(() => {
    setTrafficHistory((history) => recordTraffic(history, status, performance.now()))
  }, [status])

  useEffect(() => {
    Promise.all([ListServers(), desktop().DesktopSettings().catch(() => null)])
      .then(([list, settings]) => {
        setServers(list)
        const last = settings?.autoConnect ? settings.autoConnectServerId : localStorage.getItem('freeturn:selected-server')
        if (list.length > 0) setSelectedId(list.find((s) => s.id === last)?.id ?? list[0].id)
        if (settings?.autoConnectError) setConnectError(settings.autoConnectError)
      })
      .catch((e) => setConnectError(String(e)))
    CoreVersion().then(setVersion)
    const offState = watchStatus(
      Status,
      (listener) => {
        const offLifecycle = EventsOn('core:state', (status: backend.Status) => listener(status))
        const offStats = EventsOn('core:stats', (status: backend.Status) => listener(status, true))
        return () => {
          offLifecycle()
          offStats()
        }
      },
      setStatus,
    )
    const offLog = EventsOn('core:log', (l: LogEntry) =>
      setLogs((prev) => {
        const next = [...prev, l]
        return next.length > MAX_LOG_LINES ? next.slice(next.length - MAX_LOG_LINES) : next
      }),
    )
    return () => {
      offState()
      offLog()
      EventsOff('core:state', 'core:stats', 'core:log', 'core:captcha')
    }
  }, [])

  useEffect(() => {
    const saved = servers.find((s) => s.id === selectedId) ?? null
    setDraft(saved ? backend.Server.createFrom(saved) : null)
    setConnectError('')
  }, [selectedId, servers])

  const savedSelected = useMemo(() => servers.find((s) => s.id === selectedId) ?? null, [servers, selectedId])
  const dirty = !sameServer(draft, savedSelected)

  const selectServer = (id: string) => {
    if (id === selectedId) return
    if (dirty && !window.confirm(tr('Несохранённые изменения будут потеряны. Продолжить?', 'Unsaved changes will be lost. Continue?'))) return
    setSelectedId(id)
    localStorage.setItem('freeturn:selected-server', id)
  }

  const handleSave = async () => {
    if (!draft) return
    const saved = await SaveServer(draft)
    setServers((prev) => prev.map((s) => (s.id === saved.id ? saved : s)))
    setDraft(saved)
  }

  const handleDelete = async () => {
    if (!draft?.id) return
    if (!window.confirm(tr('Удалить сервер «{name}»?', {name: draft.name || draft.peer}))) return
    await DeleteServer(draft.id)
    setServers((prev) => prev.filter((s) => s.id !== draft.id))
    setSelectedId(null)
  }

  const handleConnect = async () => {
    if (!draft || pending) return
    const request = ++connectionRequest.current
    setConnectError('')
    setPending(true)
    try {
      if (dirty) await handleSave()
      // Connect always force-stops whatever was running first (a different
      // server, or a stuck half-failed attempt on this one), so no need to
      // orchestrate a separate Disconnect from here.
      await Connect(draft.id)
    } catch (e) {
      if (request === connectionRequest.current) setConnectError(String(e))
    } finally {
      if (request === connectionRequest.current) setPending(false)
    }
  }

  const handleDisconnect = () => {
    connectionRequest.current++
    setPending(false)
    void Disconnect().catch(e => setConnectError(String(e)))
  }

  const changeView = (next: View) => {
    if (next === 'settings') setSettingsOpened(true)
    setView(next)
  }

  return (
    <div className="app-shell">
      <Navigation view={view} onChange={changeView} version={version} />
      <main className="workspace">
        {createPortal(<div className="notification-stack">
          <CaptchaNotice url={status.captchaUrl || ''} connected={status.state === 'connected' && !pending} />
          {view !== 'settings' && <UpdateNotice onOpen={() => changeView('settings')} />}
          {connectError && (
            <div className="app-error notice error" role="alert">
              <span>{connectError}</span>
              <button className="btn small ghost" onClick={() => setConnectError('')} aria-label={tr('Закрыть сообщение', 'Dismiss message')}>
                ×
              </button>
            </div>
          )}
        </div>, document.body)}
        {view === 'connect' && (
          <Connection
            servers={servers}
            selected={savedSelected}
            status={status}
            pending={pending}
            trafficHistory={trafficHistory}
            onSelect={selectServer}
            onConnect={handleConnect}
            onDisconnect={handleDisconnect}
            onAdd={() => setShowAdd(true)}
            onEdit={() => setView('servers')}
          />
        )}
        {settingsOpened && <DesktopPreferences version={version} active={view === 'settings'} servers={servers} />}
        {view === 'logs' && (
          <section className="logs-page">
            <header className="page-heading">
              <div>
                <span className="eyebrow">{tr('ДИАГНОСТИКА', 'DIAGNOSTICS')}</span>
                <h1>{tr('Журнал событий', 'Event log')}</h1>
              </div>
              <span className="muted">{tr('Последние {count} записей', {count: MAX_LOG_LINES})}</span>
            </header>
            <LogConsole lines={logs} onClear={() => setLogs([])} />
          </section>
        )}
        {view === 'servers' && (
          <div className="profiles-page">
            <Sidebar
              servers={servers}
              selectedId={selectedId}
              activeState={status.state}
              activeServerId={status.serverId}
              onSelect={selectServer}
              onAdd={() => setShowAdd(true)}
            />
            <div className="main">
              {!draft ? (
                <div className="empty-state">
                  <p>{tr('Выберите сервер слева или добавьте новый, чтобы подключиться через free-turn-proxy.', 'Select a server on the left or add a new one to connect via free-turn-proxy.')}</p>
                </div>
              ) : (
                <ServerDetail
                  server={draft}
                  dirty={dirty}
                  status={status}
                  isActive={status.serverId === draft.id}
                  logs={logs}
                  onChange={setDraft}
                  onSave={() => {
                    handleSave().catch((e) => setConnectError(String(e)))
                  }}
                  onDelete={() => {
                    handleDelete().catch((e) => setConnectError(String(e)))
                  }}
                  onConnect={handleConnect}
                  onDisconnect={handleDisconnect}
                  onClearLogs={() => setLogs([])}
                />
              )}
            </div>
          </div>
        )}
      </main>
      {showAdd && (
        <AddServerModal
          onClose={() => setShowAdd(false)}
          onCreated={(s) => {
            setServers((prev) => [...prev, s])
            setSelectedId(s.id)
            localStorage.setItem('freeturn:selected-server', s.id)
            setShowAdd(false)
          }}
        />
      )}
    </div>
  )
}
