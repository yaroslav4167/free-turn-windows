import {backend} from '../api'
import Icon from './Icon'
import TrafficSparkline from './TrafficSparkline'
import type {TrafficHistory} from '../trafficHistory'
import {connectionState} from '../connectionState'
import {powerShapeAt} from '../powerShape'
import {usePowerMotion} from '../usePowerMotion'
import ConnectionLoader from './ConnectionLoader'
import {tr} from '../language'
import {useServerPing} from '../useServerPing'
import InfoTip from './InfoTip'

function rate(value: number): string {
  if (value >= 1024 * 1024) return (value / (1024 * 1024)).toFixed(1) + tr(' МБ/с', ' MB/s')
  if (value >= 1024) return (value / 1024).toFixed(1) + tr(' КБ/с', ' KB/s')
  return Math.max(0, value).toFixed(0) + tr(' Б/с', ' B/s')
}

function bytes(value: number): string {
  if (value >= 1024 * 1024 * 1024) return (value / (1024 * 1024 * 1024)).toFixed(2) + tr(' ГБ', ' GB')
  if (value >= 1024 * 1024) return (value / (1024 * 1024)).toFixed(1) + tr(' МБ', ' MB')
  if (value >= 1024) return (value / 1024).toFixed(1) + tr(' КБ', ' KB')
  return Math.max(0, value).toFixed(0) + tr(' Б', ' B')
}

interface Props {
  servers: backend.Server[]
  selected: backend.Server | null
  status: backend.Status
  pending: boolean
  trafficHistory: TrafficHistory
  onSelect: (id: string) => void
  onConnect: () => void
  onDisconnect: () => void
  onAdd: () => void
  onEdit: () => void
}

export default function Connection({
  servers,
  selected,
  status,
  pending,
  trafficHistory,
  onSelect,
  onConnect,
  onDisconnect,
  onAdd,
  onEdit,
}: Props) {
  const active = !['idle', 'error'].includes(status.state)
  const activeServer = servers.find((s) => s.id === status.serverId)
  const vpn = (active ? activeServer : selected)?.connMode === 'vpn'
  const presentation = connectionState(status.state, pending, vpn)
  const connected = presentation.connected
  const verificationRequired = connected && !!status.captchaUrl
  const ping = useServerPing(status.serverId, connected)
  const motion = usePowerMotion(connected, presentation.busy)
  const powerShape = useMemo(() => powerShapeAt(motion.morph), [motion.morph])
  const contourStyle = {transform: `rotate(${motion.angle}deg) scale(${1 + Math.max(0, motion.morph - 1) * 0.4})`}
  return (
    <section className="connection-page">
      <header className="page-heading">
        <div className="connection-brand">
          <h1>FreeTurn</h1>
          <span>{tr('Личное соединение', 'Your private connection')}</span>
        </div>
        <button className="btn icon-button" onClick={onAdd} aria-label={tr('Добавить сервер', 'Add server')} title={tr('Добавить сервер', 'Add server')}>
          <Icon name="plus" size={22} />
        </button>
      </header>
      <div className={'connection-center ' + presentation.visualState + (verificationRequired ? ' verification-required' : '')}>
        <span className={'mode-badge' + (connected ? ' is-connected' : '')}>
          {connected ? <span className="connection-indicator" aria-hidden="true" /> : <Icon name="shield" size={14} />}
          {connected ? tr('Соединение активно', 'Connection active') : vpn ? selected?.routingLists?.some((list) => list.enabled) ? tr('VPN · списки маршрутизации', 'VPN · routing lists') : tr('VPN · маршруты профиля', 'VPN · profile routes') : tr('TURN · режим реле', 'TURN · relay mode')}
        </span>
        <button
          className={'power-button' + (active ? ' is-active' : '')}
          onClick={active ? onDisconnect : onConnect}
          disabled={(!selected && !active) || (pending && !active)}
          aria-label={presentation.ariaLabel}
          aria-busy={presentation.busy}
          aria-pressed={connected}
          aria-describedby={verificationRequired ? 'captcha-notice' : undefined}
        >
          <svg className="power-outline" style={contourStyle} viewBox="0 0 150 150" aria-hidden="true" focusable="false">
            <path d={powerShape} />
          </svg>
          <svg className="power-progress" style={contourStyle} viewBox="0 0 150 150" aria-hidden="true" focusable="false">
            <path
              className="power-progress-path"
              d={powerShape}
              pathLength={1}
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              vectorEffect="non-scaling-stroke"
            />
          </svg>
          <span className="power-symbol" aria-hidden="true">
            <span className="power-idle-icon"><Icon name="power" size={48} /></span>
            <ConnectionLoader active={presentation.busy} />
          </span>
          <span className="power-action">{presentation.action}</span>
        </button>
        <h2 aria-live="polite">{presentation.heading}</h2>
        <p>
          {active
            ? activeServer?.name || activeServer?.peer || tr('Соединение активно', 'Connection active')
            : selected
              ? tr('Нажмите, чтобы подключиться к выбранному серверу', 'Click to connect to the selected server')
              : tr('Добавьте первый сервер, чтобы начать', 'Add your first server to get started')}
        </p>
        {status.state === 'error' && status.errMsg && (
          <div className="notice error connection-error">{status.errMsg}</div>
        )}
        {connected && !vpn && (
          <p className="relay-hint">
            {tr('Endpoint внешнего WireGuard-клиента:', 'External WireGuard client endpoint:')} <span className="mono">{activeServer?.listen}</span>
          </p>
        )}
      </div>
      <div className="connection-bottom">
        <div className="traffic-grid">
          <div className="traffic-card">
            <TrafficSparkline values={trafficHistory.rx} direction="rx" />
            <span>
              <Icon name="download" size={17} /> {tr('Получено', 'Received')}
            </span>
            <strong>{rate(connected ? status.rxRate : 0)}</strong>
            <small className="muted">{tr('Всего:', 'Total:')} {bytes(status.rxTotal)}</small>
          </div>
          <div className="traffic-card">
            <TrafficSparkline values={trafficHistory.tx} direction="tx" />
            <span>
              <Icon name="upload" size={17} /> {tr('Отправлено', 'Sent')}
            </span>
            <strong>{rate(connected ? status.txRate : 0)}</strong>
            <small className="muted">{tr('Всего:', 'Total:')} {bytes(status.txTotal)}</small>
          </div>
          <div className="traffic-card">
            <span>
              <Icon name="servers" size={17} /> {tr('Потоки', 'Streams')}
            </span>
            <strong>{active ? `${status.streams} / ${status.total}` : '—'}</strong>
            <small className="muted connection-ping">
              {tr('Пинг')}: {ping.measuring ? '…' : ping.milliseconds === null ? '—' : `${ping.milliseconds < 1 ? '<1' : Math.round(ping.milliseconds)} ${tr('мс')}`}
              <InfoTip label={tr('О пинге сервера')}>
                {tr('ICMP до адреса сервера, примерно раз в 5 секунд. Это не задержка всего VPN-туннеля. Используется текущая системная маршрутизация. Если ICMP заблокирован или ping недоступен, показано «—».')}
                {ping.host && <p className="mono">{ping.host}</p>}
              </InfoTip>
            </small>
          </div>
        </div>
        <div className="profile-picker">
          <Icon name="servers" size={20} />
          <div className="field">
            <label htmlFor="active-profile" className="sr-only">
              {tr('Сервер подключения', 'Connection server')}
            </label>
            <select
              id="active-profile"
              value={selected?.id ?? ''}
              onChange={(e) => onSelect(e.target.value)}
              disabled={servers.length === 0}
            >
              {servers.length === 0 && <option value="">{tr('Нет серверов', 'No servers')}</option>}
              {servers.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name || s.peer || tr('Без имени', 'Unnamed')}
                </option>
              ))}
            </select>
          </div>
          <button
            className="btn icon-button"
            disabled={!selected}
            onClick={onEdit}
            aria-label={tr('Настроить выбранный сервер', 'Configure selected server')}
            title={tr('Настроить сервер', 'Configure server')}
          >
            <Icon name="settings" size={18} />
          </button>
        </div>
      </div>
    </section>
  )
}
import {useMemo} from 'react'
