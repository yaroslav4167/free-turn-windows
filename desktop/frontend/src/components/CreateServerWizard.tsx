import {useEffect, useState} from 'react'
import {tr} from '../language'
import ServerForm from './ServerForm'
import {
  backend,
  EventsOn,
  GenerateClientID,
  GenerateObfKey,
  OBF_PROFILES,
  serversetup,
  SetupCreateServer,
  SetupDetectRootMode,
  SetupInstall,
  SetupProbe,
  SetupStart,
  SetupWgSetup,
  NewServer,
  SaveServer,
} from '../api'

interface Props {
  onCreated: (s: backend.Server) => void
}

type Step = 'ssh' | 'existing' | 'config' | 'progress' | 'done'
type TaskState = 'pending' | 'running' | 'done' | 'error'


// Must match backend.DefaultServer()'s Listen default.
const DEFAULT_RELAY_LISTEN = '127.0.0.1:9000'

function randomPort(min: number, max: number): number {
  return Math.floor(min + Math.random() * (max - min))
}

function newSshDraft(): serversetup.SSHConfig {
  return {
    ip: '', port: 22, username: 'root', password: '', authType: 'PASSWORD', sshKey: '',
    hostFingerprint: '', rootMode: 'ROOT', sudoPassword: '',
  }
}

export default function CreateServerWizard({onCreated}: Props) {
  const TASK_LABEL: Record<string, string> = {
    install: tr('Установка free-turn-proxy', 'Installing free-turn-proxy'),
    wgSetup: tr('Настройка WireGuard', 'Configuring WireGuard'),
    start: tr('Запуск сервера', 'Starting server'), save: tr('Сохранение профиля', 'Saving profile'),
  }
  const [step, setStep] = useState<Step>('ssh')
  const [ssh, setSsh] = useState<serversetup.SSHConfig>(newSshDraft())
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [existing, setExisting] = useState<serversetup.ProbeResult | null>(null)
  const [existingDraft, setExistingDraft] = useState<backend.Server | null>(null)

  const [name, setName] = useState('')
  const [vkLink, setVkLink] = useState('')
  const [obfProfile, setObfProfile] = useState('rtpopus3')
  const [obfKey, setObfKey] = useState('')
  const [listenPort, setListenPort] = useState(() => randomPort(56000, 57000))
  const [wgPort, setWgPort] = useState(() => randomPort(51000, 52000))
  const [wgPortExisting, setWgPortExisting] = useState(false)
  const [clientId, setClientId] = useState('')

  const [tasks, setTasks] = useState<Record<string, {state: TaskState; msg?: string}>>({})
  const [result, setResult] = useState<{server: backend.Server; wgConf: string} | null>(null)
  // What the server-side script is doing right now (apt busy, downloading, ...):
  // without it a slow step looks like a hang.
  const [progressLines, setProgressLines] = useState<string[]>([])

  useEffect(() => {
    if (step !== 'progress') return
    return EventsOn('setup:progress', (line: string) =>
      setProgressLines((prev) => [...prev.slice(-40), line])
    )
  }, [step])

  const submitSsh = async () => {
    setBusy(true)
    setError('')
    try {
      const withRoot = await SetupDetectRootMode(ssh)
      const probe = await SetupProbe(withRoot)
      const pinned: serversetup.SSHConfig = {...withRoot, hostFingerprint: probe.hostFingerprint || withRoot.hostFingerprint}
      setSsh(pinned)
      setExisting(probe)
      setWgPortExisting(probe.wgPort > 0)
      if (probe.installed || probe.running) {
        const draft = await NewServer()
        draft.name = ssh.ip
        // Do not attach SSH management to an unknown installation layout.
        setExistingDraft(draft)
        setStep('existing')
        return
      }
      if (probe.wgPort > 0) {
        setWgPort(probe.wgPort)
        setWgPortExisting(true)
      }
      const [key, cid] = await Promise.all([GenerateObfKey(), GenerateClientID()])
      setObfKey(key)
      setClientId(cid)
      setStep('config')
    } catch (e) {
      setError(String(e))
    } finally {
      setBusy(false)
    }
  }

  const runProvision = async () => {
    if (existing?.installed || existing?.running) return
    setStep('progress')
    setError('')
    setProgressLines([])
    const localTasks: Record<string, {state: TaskState; msg?: string}> = {
      install: {state: 'pending'}, wgSetup: {state: 'pending'}, start: {state: 'pending'}, save: {state: 'pending'},
    }
    setTasks({...localTasks})
    const mark = (key: string, state: TaskState, msg?: string) => {
      localTasks[key] = {state, msg}
      setTasks({...localTasks})
    }

    try {
      mark('install', 'running')
      // Recheck just before the first remote mutation (also on retries).
      const fresh = await SetupProbe(ssh)
      if (fresh.installed || fresh.running) {
        setExisting(fresh)
        setExistingDraft(await NewServer())
        setStep('existing')
        return
      }
      await SetupInstall(ssh)
      mark('install', 'done')

      mark('wgSetup', 'running')
      // Baked into the generated conf's Endpoint - must match the new
      // server's default `listen` (backend.DefaultServer's 127.0.0.1:9000),
      // since that's the local relay port the WG/AmneziaWG client should hit.
      const wg = await SetupWgSetup(ssh, wgPort, DEFAULT_RELAY_LISTEN)
      mark('wgSetup', 'done')

      mark('start', 'running')
      let usedListenPort = listenPort
      try {
        await SetupStart(ssh, {
          listen: `0.0.0.0:${usedListenPort}`,
          connect: `127.0.0.1:${wg.port}`,
          obfProfile, obfKey, obfTimingMs: 0, clientId,
        })
      } catch (e) {
        // The random port collided with something else already listening on
        // the VPS - pick a fresh one and retry once rather than making the
        // user notice and redo the whole wizard by hand.
        if (!String(e).includes('listen_port_busy')) throw e
        usedListenPort = randomPort(56000, 57000)
        setListenPort(usedListenPort)
        await SetupStart(ssh, {
          listen: `0.0.0.0:${usedListenPort}`,
          connect: `127.0.0.1:${wg.port}`,
          obfProfile, obfKey, obfTimingMs: 0, clientId,
        })
      }
      mark('start', 'done')

      mark('save', 'running')
      const draft: backend.ServerSetupDraft = {
        name: name.trim() || ssh.ip, vkLink: vkLink.trim(), obfProfile, obfKey, obfTimingMs: 0,
        listenPort: usedListenPort, clientId,
      }
      const saved = await SetupCreateServer(ssh, draft, wg)
      mark('save', 'done')

      setResult({server: saved, wgConf: wg.clientConf})
      setStep('done')
    } catch (e) {
      const failed = Object.entries(localTasks).find(([, v]) => v.state === 'running')?.[0]
      if (failed) mark(failed, 'error', String(e))
      setError(String(e))
    }
  }

  if (step === 'ssh') {
    return (
      <>
        {error && <div className="notice error">{error}</div>}
        <div className="field-row">
          <div className="field" style={{flex: 2}}>
            <label>{tr('IP адрес сервера', 'Server IP address')}</label>
            <input type="text" value={ssh.ip} onChange={(e) => setSsh({...ssh, ip: e.target.value.trim()})} placeholder="1.2.3.4" />
          </div>
          <div className="field">
            <label>{tr('SSH порт', 'SSH port')}</label>
            <input type="number" value={ssh.port} onChange={(e) => setSsh({...ssh, port: Number(e.target.value)})} />
          </div>
        </div>
        <div className="field-row">
          <div className="field">
            <label>{tr('Пользователь', 'Username')}</label>
            <input type="text" value={ssh.username} onChange={(e) => setSsh({...ssh, username: e.target.value})} />
          </div>
          <div className="field">
            <label>{tr('Способ входа', 'Authentication method')}</label>
            <select value={ssh.authType} onChange={(e) => setSsh({...ssh, authType: e.target.value})}>
              <option value="PASSWORD">{tr('Пароль', 'Password')}</option>
              <option value="SSH_KEY">{tr('SSH-ключ', 'SSH key')}</option>
            </select>
          </div>
        </div>
        {ssh.authType === 'SSH_KEY' ? (
          <>
            <div className="field">
              <label>{tr('Приватный ключ (OpenSSH)', 'Private key (OpenSSH)')}</label>
              <textarea value={ssh.sshKey} onChange={(e) => setSsh({...ssh, sshKey: e.target.value})} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
            </div>
            <div className="field">
              <label>{tr('Passphrase ключа (если есть)', 'Key passphrase (if any)')}</label>
              <input type="password" value={ssh.password} onChange={(e) => setSsh({...ssh, password: e.target.value})} />
            </div>
          </>
        ) : (
          <div className="field">
            <label>{tr('Пароль', 'Password')}</label>
            <input type="password" value={ssh.password} onChange={(e) => setSsh({...ssh, password: e.target.value})} />
          </div>
        )}
        <div className="field">
          <label>{tr('Пароль sudo (если отличается от пароля входа)', 'Sudo password (if different from login password)')}</label>
          <input type="password" value={ssh.sudoPassword} onChange={(e) => setSsh({...ssh, sudoPassword: e.target.value})} />
        </div>
        <div className="modal-actions">
          <button
            className="btn primary"
            disabled={busy || !ssh.ip || (ssh.authType === 'SSH_KEY' ? !ssh.sshKey : !ssh.password)}
            onClick={submitSsh}
          >
            {busy ? tr('Подключение…', 'Connecting…') : tr('Далее', 'Next')}
          </button>
        </div>
      </>
    )
  }

  if (step === 'existing') {
    return <>
      <div className="notice info">
        {tr('Обнаружен установленный сервер. Повторная установка и перенастройка отключены.', 'An existing server was detected. Reinstallation and reconfiguration are disabled.')}
        {' '}{existing?.existingService} {existing?.version}
        {' · '}{existing?.running ? tr('Запущен', 'Running') : tr('Остановлен', 'Stopped')}
      </div>
      <p>{tr('Укажите параметры существующего подключения и его клиентский конфиг. Сохранение добавит только локальный профиль: настройки сервера и список клиентов не изменятся.', 'Enter the existing connection settings and its client configuration. Saving only adds a local profile: server settings and authorized clients will not change.')}</p>
      {error && <div className="notice error">{error}</div>}
      {existingDraft && <ServerForm value={existingDraft} onChange={setExistingDraft} />}
      {existingDraft && <div className="field">
        <label>{tr('Клиентский конфиг WireGuard/AmneziaWG (для режима VPN)', 'WireGuard/AmneziaWG client configuration (for VPN mode)')}</label>
        <textarea value={existingDraft.wgClientConf || ''} onChange={e => setExistingDraft({
          ...existingDraft, wgClientConf: e.target.value, connMode: e.target.value.trim() ? 'vpn' : 'relay',
        } as backend.Server)} placeholder="[Interface]" />
      </div>}
      <div className="modal-actions">
        <button className="btn" disabled={busy} onClick={() => setStep('ssh')}>{tr('Назад', 'Back')}</button>
        <button className="btn primary" disabled={busy || !existingDraft?.peer} onClick={async () => {
          if (!existingDraft) return
          setBusy(true)
          setError('')
          try { onCreated(await SaveServer(existingDraft)) }
          catch (e) { setError(String(e)) }
          finally { setBusy(false) }
        }}>{tr('Сохранить', 'Save')}</button>
      </div>
    </>
  }

  if (step === 'config') {
    return (
      <>
        {error && <div className="notice error">{error}</div>}
        <div className="field">
          <label>{tr('Имя сервера', 'Server name')}</label>
          <input type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder={ssh.ip} />
        </div>
        <div className="field">
          <label>{tr('Ваша ссылка на звонок VK Calls (можно добавить позже)', 'Your VK Calls link (can be added later)')}</label>
          <input type="text" value={vkLink} onChange={(e) => setVkLink(e.target.value)} placeholder="https://vk.ru/call/join/..." />
        </div>
        <div className="field-row">
          <div className="field">
            <label>{tr('Профиль обфускации', 'Obfuscation profile')}</label>
            <select value={obfProfile} onChange={(e) => setObfProfile(e.target.value)}>
              {OBF_PROFILES.filter((p) => p !== 'none').map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </div>
          <div className="field">
            <label>{tr('Внешний порт (-listen)', 'External port (-listen)')}</label>
            <input type="number" value={listenPort} onChange={(e) => setListenPort(Number(e.target.value))} />
          </div>
          <div className="field">
            <label>{tr('Порт WireGuard', 'WireGuard port')}{wgPortExisting ? tr(' (уже настроен)', ' (already configured)') : ''}</label>
            <input type="number" value={wgPort} disabled={wgPortExisting} onChange={(e) => setWgPort(Number(e.target.value))} />
          </div>
        </div>
        <div className="notice info">
          {tr('Развернём free-turn-proxy и интерфейс WireGuard (ft-wg0) на сервере, запустим их с обфускацией и добавим вас первым разрешённым клиентом.', 'We will deploy free-turn-proxy and the WireGuard interface (ft-wg0), start them with obfuscation and add you as the first authorized client.')}
        </div>
        <div className="modal-actions">
          <button className="btn" onClick={() => setStep('ssh')}>
            {tr('Назад', 'Back')}
          </button>
          <button className="btn primary" onClick={runProvision}>
            {tr('Развернуть', 'Deploy')}
          </button>
        </div>
      </>
    )
  }

  if (step === 'progress') {
    return (
      <>
        {Object.entries(tasks).map(([key, t]) => (
          <div key={key} className="checkbox-row">
            <span
              className={'dot' + (t.state === 'done' ? ' connected' : t.state === 'error' ? ' error' : t.state === 'running' ? ' connecting' : '')}
            />
            {TASK_LABEL[key]}
            {t.state === 'error' && t.msg && <span className="task-error">{t.msg}</span>}
          </div>
        ))}
        {progressLines.length > 0 && (
          <pre
            className="setup-log"
            ref={(el) => {
              if (el) el.scrollTop = el.scrollHeight
            }}
          >
            {progressLines.join('\n')}
          </pre>
        )}
        {error && (
          <div className="modal-actions">
            <button className="btn" onClick={() => setStep('config')}>
              {tr('Назад', 'Back')}
            </button>
            <button className="btn primary" onClick={runProvision}>
              {tr('Повторить', 'Retry')}
            </button>
          </div>
        )}
      </>
    )
  }

  // done
  return (
    <>
      <div className="notice info">{tr('Сервер развёрнут и запущен.', 'Server deployed and running.')}</div>
      {result && (
        <>
          <div className="field">
            <label>{tr('Конфиг WireGuard/AmneziaWG для отдельного клиента (Endpoint уже указывает на локальный релей)', 'WireGuard/AmneziaWG configuration for an external client (Endpoint already points to the local relay)')}</label>
            <textarea readOnly value={result.wgConf} onClick={(e) => (e.target as HTMLTextAreaElement).select()} />
          </div>
        </>
      )}
      <div className="modal-actions">
        <button className="btn primary" onClick={() => result && onCreated(result.server)}>
          {tr('Готово', 'Done')}
        </button>
      </div>
    </>
  )
}
