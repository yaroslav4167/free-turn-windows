import {useState} from 'react'
import {backend, ConfigCommand} from '../api'
import ServerForm from './ServerForm'
import ServerManagement from './ServerManagement'
import PeersPanel from './PeersPanel'
import LogConsole, {LogEntry} from './LogConsole'
import {formatBytes} from '../format'

const STATE_LABEL: Record<string, string> = {
  idle: 'Отключено',
  connecting: 'Подключение…',
  connected: 'Подключено',
  captcha: 'Требуется captcha',
  error: 'Ошибка',
}

interface Props {
  server: backend.Server
  dirty: boolean
  status: backend.Status
  isActive: boolean
  logs: LogEntry[]
  onChange: (s: backend.Server) => void
  onSave: () => void
  onDelete: () => void
  onConnect: () => void
  onDisconnect: () => void
  onClearLogs: () => void
}

export default function ServerDetail({
  server,
  dirty,
  status,
  isActive,
  logs,
  onChange,
  onSave,
  onDelete,
  onConnect,
  onDisconnect,
  onClearLogs,
}: Props) {
  const [command, setCommand] = useState('')

  const state = isActive ? status.state : 'idle'
  const busy = state === 'connecting'
  const otherActive = !isActive && status.state !== 'idle' && status.state !== 'error'

  const toggleCommand = () => {
    if (command) {
      setCommand('')
      return
    }
    ConfigCommand(server.id).then(setCommand).catch((e) => setCommand('Ошибка: ' + e))
  }

  const isVPN = server.connMode === 'vpn'

  return (
    <div className="detail">
      <div className="detail-header">
        <h2>{server.name || server.peer || 'Новый сервер'}</h2>
        <span className={'status-pill ' + state + (isActive && status.captchaUrl ? ' verification-required' : '')}>{STATE_LABEL[state] ?? state}</span>
        {isActive && state !== 'idle' && (
          <button className="btn danger" onClick={onDisconnect}>
            Отключить
          </button>
        )}
        {!(isActive && (state === 'connecting' || state === 'connected' || state === 'captcha')) && (
          <button className="btn primary" disabled={busy || !server.peer} onClick={onConnect}>
            {otherActive ? 'Переключиться' : isVPN ? 'Включить VPN' : 'Подключиться'}
          </button>
        )}
        <button className="btn" disabled={!dirty} onClick={onSave}>
          Сохранить
        </button>
        <button className="btn danger" onClick={onDelete}>
          Удалить
        </button>
      </div>

      <div className="detail-body">
        {isActive && state === 'connected' && <div className="traffic-grid" aria-label="Трафик соединения">
          <div><span>Получено</span><strong>{formatBytes(status.rxRate)}/с</strong><small>Всего: {formatBytes(status.rxTotal)}</small></div>
          <div><span>Отправлено</span><strong>{formatBytes(status.txRate)}/с</strong><small>Всего: {formatBytes(status.txTotal)}</small></div>
          <div><span>Потоки</span><strong>{status.streams} / {status.total}</strong></div>
        </div>}
        <div className="field-group">
          <h3>Тип подключения</h3>
          {server.wgClientConf ? (
            <div className="inline-actions">
              <button
                className={'btn small' + (isVPN ? ' primary' : '')}
                disabled={isActive && (state === 'connected' || state === 'connecting')}
                onClick={() => onChange({...server, connMode: 'vpn'} as backend.Server)}
              >
                VPN (одна кнопка)
              </button>
              <button
                className={'btn small' + (!isVPN ? ' primary' : '')}
                disabled={isActive && (state === 'connected' || state === 'connecting')}
                onClick={() => onChange({...server, connMode: 'relay'} as backend.Server)}
              >
                Реле (свой WireGuard-клиент)
              </button>
            </div>
          ) : (
            <div className="notice info">
              VPN-режим в одну кнопку доступен только для серверов, развёрнутых через «Новый VPS» — там генерируется
              конфиг WireGuard. Для этого сервера доступен только режим реле ниже.
            </div>
          )}
        </div>

        {otherActive && (
          <div className="notice info">
            Сейчас подключён другой сервер — переключение сначала отключит его.
          </div>
        )}
        {isActive && state === 'error' && status.errMsg && <div className="notice error">{status.errMsg}</div>}
        {isActive && state === 'connected' && isVPN && (
          <div className="notice info">VPN включён — весь трафик идёт через {server.peer}.</div>
        )}
        {isActive && state === 'connected' && !isVPN && (
          <div className="notice info">
            Реле поднято на <span className="mono">{server.listen}</span>. Направьте на этот адрес Endpoint в
            вашем WireGuard/AmneziaWG-клиенте (релейный конфиг, MTU 1280) и включайте VPN.
          </div>
        )}

        {server.ssh && <ServerManagement server={server} />}
        {server.ssh && <PeersPanel server={server} />}

        <ServerForm value={server} onChange={onChange} />

        <div className="field-group">
          <div className="inline-actions">
            <button className="btn small" onClick={toggleCommand}>
              {command ? 'Скрыть команду' : 'Показать эквивалент CLI'}
            </button>
          </div>
          {command && <div className="mono" style={{marginTop: 8, whiteSpace: 'pre-wrap', color: 'var(--text-dim)'}}>{command}</div>}
        </div>
      </div>

      <LogConsole lines={logs} onClear={onClearLogs} />
    </div>
  )
}
