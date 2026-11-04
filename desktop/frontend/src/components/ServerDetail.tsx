import {useState} from 'react'
import {backend, ConfigCommand} from '../api'
import ServerForm from './ServerForm'
import ServerManagement from './ServerManagement'
import PeersPanel from './PeersPanel'
import LogConsole, {LogEntry} from './LogConsole'
import VPNRoutingLists from './VPNRoutingLists'
import {tr} from '../language'


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
  const STATE_LABEL: Record<string, string> = {
    idle: tr('Отключено', 'Disconnected'), connecting: tr('Подключение…', 'Connecting…'),
    connected: tr('Подключено', 'Connected'), captcha: tr('Требуется captcha', 'Captcha required'), error: tr('Ошибка', 'Error'),
  }
  const [command, setCommand] = useState('')

  const state = isActive ? status.state : 'idle'
  const busy = state === 'connecting'
  const otherActive = !isActive && status.state !== 'idle' && status.state !== 'error'

  const toggleCommand = () => {
    if (command) {
      setCommand('')
      return
    }
    ConfigCommand(server.id).then(setCommand).catch((e) => setCommand(tr('Ошибка: ', 'Error: ') + e))
  }

  const isVPN = server.connMode === 'vpn'

  return (
    <div className="detail">
      <div className="detail-header">
        <h2>{server.name || server.peer || tr('Новый сервер', 'New server')}</h2>
        <span className={'status-pill ' + state}>{STATE_LABEL[state] ?? state}</span>
        {isActive && state !== 'idle' && (
          <button className="btn danger" onClick={onDisconnect}>
            {tr('Отключить', 'Disconnect')}
          </button>
        )}
        {!(isActive && (state === 'connecting' || state === 'connected' || state === 'captcha')) && (
          <button className="btn primary" disabled={busy || !server.peer} onClick={onConnect}>
            {otherActive ? tr('Переключиться', 'Switch') : isVPN ? tr('Включить VPN', 'Enable VPN') : tr('Подключиться', 'Connect')}
          </button>
        )}
        <button className="btn" disabled={!dirty} onClick={onSave}>
          {tr('Сохранить', 'Save')}
        </button>
        <button className="btn danger" onClick={onDelete}>
          {tr('Удалить', 'Delete')}
        </button>
      </div>

      <div className="detail-body">
        <div className="field-group">
          <h3>{tr('Тип подключения', 'Connection type')}</h3>
          {server.wgClientConf ? (
            <div className="inline-actions">
              <button
                className={'btn small' + (isVPN ? ' primary' : '')}
                disabled={isActive && (state === 'connected' || state === 'connecting')}
                onClick={() => onChange({...server, connMode: 'vpn'} as backend.Server)}
              >
                {tr('VPN (одна кнопка)', 'VPN (one click)')}
              </button>
              <button
                className={'btn small' + (!isVPN ? ' primary' : '')}
                disabled={isActive && (state === 'connected' || state === 'connecting')}
                onClick={() => onChange({...server, connMode: 'relay'} as backend.Server)}
              >
                {tr('Реле (свой WireGuard-клиент)', 'Relay (external WireGuard client)')}
              </button>
            </div>
          ) : (
            <div className="notice info">
              {tr('VPN-режим в одну кнопку доступен только для серверов, развёрнутых через «Новый VPS» — там генерируется конфиг WireGuard. Для этого сервера доступен только режим реле ниже.', 'One-click VPN is available for servers deployed with “New VPS”, which generates a WireGuard configuration. Only relay mode is available for this server.')}
            </div>
          )}
        </div>

        {otherActive && (
          <div className="notice info">
            {tr('Сейчас подключён другой сервер — переключение сначала отключит его.', 'Another server is connected — switching will disconnect it first.')}
          </div>
        )}
        {isActive && state === 'error' && status.errMsg && <div className="notice error">{status.errMsg}</div>}
        {isActive && state === 'connected' && isVPN && (
          <div className="notice info">{tr('VPN включён через', 'VPN enabled via')} {server.peer}{server.routingLists?.some((list) => list.enabled) ? tr(' · применены списки маршрутизации', ' · routing lists applied') : ''}.</div>
        )}
        {isActive && state === 'connected' && !isVPN && (
          <div className="notice info">
            {tr('Реле поднято на', 'Relay listening on')} <span className="mono">{server.listen}</span>. {tr('Направьте на этот адрес Endpoint в вашем WireGuard/AmneziaWG-клиенте (релейный конфиг, MTU 1280) и включайте VPN.', 'Set this as the Endpoint in your WireGuard/AmneziaWG client (relay configuration, MTU 1280), then enable VPN.')}
          </div>
        )}

        {server.ssh && <ServerManagement server={server} />}
        {server.ssh && <PeersPanel server={server} />}

        <ServerForm value={server} onChange={onChange} />
        {isVPN && <VPNRoutingLists server={server} onChange={onChange} disabled={isActive && state !== 'idle' && state !== 'error'} />}

        <div className="field-group">
          <div className="inline-actions">
            <button className="btn small" onClick={toggleCommand}>
              {command ? tr('Скрыть команду', 'Hide command') : tr('Показать эквивалент CLI', 'Show equivalent CLI command')}
            </button>
          </div>
          {command && <div className="mono" style={{marginTop: 8, whiteSpace: 'pre-wrap', color: 'var(--text-dim)'}}>{command}</div>}
        </div>
      </div>

      <LogConsole lines={logs} onClear={onClearLogs} />
    </div>
  )
}
