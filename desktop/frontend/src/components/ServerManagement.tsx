import {useState} from 'react'
import {backend, ServerLogs, ServerRemoteStatus, ServerRestart, ServerStop, ServerUninstall, serversetup} from '../api'
import {tr} from '../language'

interface Props {
  server: backend.Server
}

export default function ServerManagement({server}: Props) {
  const [status, setStatus] = useState<serversetup.ProbeResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [logs, setLogs] = useState<string[] | null>(null)

  const run = async (fn: () => Promise<void>) => {
    setBusy(true)
    setError('')
    setInfo('')
    try {
      await fn()
    } catch (e) {
      setError(String(e))
    } finally {
      setBusy(false)
    }
  }

  const refresh = () => run(async () => setStatus(await ServerRemoteStatus(server.id)))
  const restart = () => run(async () => {
    await ServerRestart(server.id)
    setStatus(await ServerRemoteStatus(server.id))
    setInfo(tr('Перезапущено на сервере.', 'Restarted on server.'))
  })
  const stop = () => run(async () => {
    await ServerStop(server.id)
    setStatus(await ServerRemoteStatus(server.id))
    setInfo(tr('Остановлено.', 'Stopped.'))
  })
  const showLogs = () => run(async () => setLogs(await ServerLogs(server.id, 200)))
  const uninstall = () => run(async () => {
    if (!window.confirm(tr('Удалить free-turn-proxy и управляемый WireGuard-интерфейс с этого VPS? Профиль в приложении останется.', 'Remove free-turn-proxy and the managed WireGuard interface from this VPS? The profile in the app will be kept.'))) return
    const res = await ServerUninstall(server.id, false)
    setStatus(null)
    setInfo(
      `${tr('Удалено:', 'Removed:')} ${tr('бинарь', 'binary')}=${res.removed.binary ? tr('да', 'yes') : tr('нет', 'no')}, ${tr('служба', 'service')}=${res.removed.unit ? tr('да', 'yes') : tr('нет', 'no')}, WireGuard=${res.removed.wg_iface ? tr('да', 'yes') : tr('нет', 'no')}` +
        (res.kept.length ? tr('. Не тронуто: {files}', {files: res.kept.join(', ')}) : '')
    )
  })

  const stateLabel = status ? (status.running ? tr('Сервер работает', 'Server running') : status.installed ? tr('Установлен, остановлен', 'Installed, stopped') : tr('Не установлен', 'Not installed')) : tr('Статус неизвестен', 'Status unknown')
  const stateDot = status?.running ? ' connected' : status && !status.installed ? '' : status ? ' captcha' : ''

  return (
    <div className="field-group">
      <h3>{tr('Управление сервером (по SSH)', 'Server management (SSH)')}</h3>
      {error && <div className="notice error">{error}</div>}
      {info && <div className="notice info">{info}</div>}
      <div className="checkbox-row">
        <span className={'dot' + stateDot} />
        {stateLabel}
        <button className="btn small ghost" disabled={busy} onClick={refresh} style={{marginLeft: 'auto'}}>
          {tr('Обновить статус', 'Refresh status')}
        </button>
      </div>
      <div className="inline-actions" style={{marginTop: 8, marginBottom: 8, flexWrap: 'wrap'}}>
        <button className="btn small" disabled={busy} onClick={restart}>
          {tr('Перезапустить на сервере', 'Restart on server')}
        </button>
        <button className="btn small" disabled={busy} onClick={stop}>
          {tr('Остановить', 'Stop')}
        </button>
        <button className="btn small" disabled={busy} onClick={showLogs}>
          {logs ? tr('Обновить логи', 'Refresh logs') : tr('Показать логи сервера', 'Show server logs')}
        </button>
      </div>
      {logs && (
        <pre
          className="mono"
          style={{maxHeight: 220, overflow: 'auto', background: 'var(--bg-input)', padding: 8, borderRadius: 7, border: '1px solid var(--border)', whiteSpace: 'pre-wrap'}}
        >
          {logs.join('\n') || tr('(пусто)', '(empty)')}
        </pre>
      )}
      {server.wgClientConf && (
        <div className="field" style={{marginTop: 12}}>
          <label>{tr('Конфиг WireGuard/AmneziaWG для отдельного клиента', 'WireGuard/AmneziaWG configuration for an external client')}</label>
          <textarea readOnly value={server.wgClientConf} onClick={(e) => (e.target as HTMLTextAreaElement).select()} />
        </div>
      )}
      <div className="inline-actions" style={{marginTop: 12}}>
        <button className="btn small danger" disabled={busy} onClick={uninstall}>
          {tr('Удалить free-turn-proxy с сервера', 'Remove free-turn-proxy from server')}
        </button>
      </div>
    </div>
  )
}
