import {backend, DNS_MODES, GenerateClientID, GenerateObfKey, MODES, OBF_PROFILES, TRANSPORTS} from '../api'
import {tr} from '../language'

interface Props {
  value: backend.Server
  onChange: (s: backend.Server) => void
}

export default function ServerForm({value: s, onChange}: Props) {
  const patch = (p: Partial<backend.Server>) => onChange({...s, ...p} as backend.Server)
  const patchKcp = (p: Partial<backend.KCPProfile>) => patch({kcp: {...s.kcp, ...p} as backend.KCPProfile})

  return (
    <>
      <div className="field-group">
        <h3>{tr('Подключение', 'Connection')}</h3>
        <div className="field-row">
          <div className="field">
            <label>{tr('Имя', 'Name')}</label>
            <input type="text" value={s.name} onChange={(e) => patch({name: e.target.value})} placeholder={tr('Мой сервер', 'My server')} />
          </div>
          <div className="field">
            <label>{tr('Адрес сервера (peer), host:port', 'Server address (peer), host:port')}</label>
            <input type="text" value={s.peer} onChange={(e) => patch({peer: e.target.value})} placeholder="1.2.3.4:56000" />
          </div>
        </div>
        <div className="field">
          <label>{tr('Ссылки на звонки VK Calls (по одной в строке)', 'VK Calls links (one per line)')}</label>
          <textarea
            value={(s.vkLinks ?? []).join('\n')}
            onChange={(e) => patch({vkLinks: e.target.value.split('\n')})}
            placeholder="https://vk.ru/call/join/..."
          />
        </div>
      </div>

      <div className="field-group">
        <h3>{tr('Обфускация', 'Obfuscation')}</h3>
        <div className="field-row">
          <div className="field">
            <label>{tr('Профиль', 'Profile')}</label>
            <select value={s.obfProfile} onChange={(e) => patch({obfProfile: e.target.value})}>
              {OBF_PROFILES.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </div>
          <div className="field" style={{flex: 2}}>
            <label>{tr('Ключ (64 hex)', 'Key (64 hex)')}</label>
            <div className="inline-actions">
              <input
                type="text"
                className="mono"
                value={s.obfKey}
                onChange={(e) => patch({obfKey: e.target.value.trim()})}
                placeholder="d823fa..."
                disabled={s.obfProfile === 'none'}
              />
              <button
                className="btn small"
                disabled={s.obfProfile === 'none'}
                onClick={() => GenerateObfKey().then((key) => patch({obfKey: key}))}
              >
                {tr('Сгенерировать', 'Generate')}
              </button>
            </div>
          </div>
        </div>
        <div className="field-row">
          <div className="field">
            <label>{tr('Пейсинг обфускации, мс (0 = выкл)', 'Obfuscation pacing, ms (0 = off)')}</label>
            <input
              type="number"
              min={0}
              max={60}
              value={s.obfTimingMs}
              onChange={(e) => patch({obfTimingMs: Number(e.target.value)})}
              disabled={s.obfProfile === 'none'}
            />
          </div>
        </div>
      </div>

      <div className="field-group">
        <h3>{tr('Транспорт и потоки', 'Transport and streams')}</h3>
        <div className="field-row">
          <div className="field">
            <label>{tr('Транспорт до TURN', 'TURN transport')}</label>
            <select value={s.transport} onChange={(e) => patch({transport: e.target.value})}>
              {TRANSPORTS.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </select>
          </div>
          <div className="field">
            <label>{tr('Режим туннеля', 'Tunnel mode')}</label>
            <select value={s.mode} onChange={(e) => patch({mode: e.target.value})}>
              {MODES.map((m) => (
                <option key={m} value={m}>
                  {m}
                </option>
              ))}
            </select>
          </div>
          <div className="field">
            <label>{tr('Локальный listen', 'Local listen address')}</label>
            <input type="text" value={s.listen} onChange={(e) => patch({listen: e.target.value})} />
          </div>
        </div>
        <div className="field-row">
          <div className="field">
            <label>{tr('TURN-потоков (-n)', 'TURN streams (-n)')}</label>
            <input type="number" min={1} value={s.n} onChange={(e) => patch({n: Number(e.target.value)})} />
          </div>
          <div className="field">
            <label>{tr('Потоков на VK-креды', 'Streams per VK credential')}</label>
            <input
              type="number"
              min={1}
              value={s.streamsPerCred}
              onChange={(e) => patch({streamsPerCred: Number(e.target.value)})}
            />
          </div>
        </div>
      </div>

      {s.mode === 'tcp' && (
        <div className="field-group">
          <h3>{tr('KCP (ARQ для tcp-режима)', 'KCP (ARQ for TCP mode)')}</h3>
          <div className="field-row">
            <div className="field">
              <label>nodelay</label>
              <input type="number" value={s.kcp.noDelay} onChange={(e) => patchKcp({noDelay: Number(e.target.value)})} />
            </div>
            <div className="field">
              <label>interval</label>
              <input type="number" value={s.kcp.interval} onChange={(e) => patchKcp({interval: Number(e.target.value)})} />
            </div>
            <div className="field">
              <label>resend</label>
              <input type="number" value={s.kcp.resend} onChange={(e) => patchKcp({resend: Number(e.target.value)})} />
            </div>
            <div className="field">
              <label>nc</label>
              <input type="number" value={s.kcp.nc} onChange={(e) => patchKcp({nc: Number(e.target.value)})} />
            </div>
          </div>
          <div className="field-row">
            <div className="field">
              <label>sndwnd</label>
              <input type="number" value={s.kcp.sndWnd} onChange={(e) => patchKcp({sndWnd: Number(e.target.value)})} />
            </div>
            <div className="field">
              <label>rcvwnd</label>
              <input type="number" value={s.kcp.rcvWnd} onChange={(e) => patchKcp({rcvWnd: Number(e.target.value)})} />
            </div>
            <div className="field">
              <label>mtu</label>
              <input type="number" value={s.kcp.mtu} onChange={(e) => patchKcp({mtu: Number(e.target.value)})} />
            </div>
          </div>
          <label className="checkbox-row">
            <input
              type="checkbox"
              checked={s.kcp.ackNoDelay}
              onChange={(e) => patchKcp({ackNoDelay: e.target.checked})}
            />
            ack no delay
          </label>
        </div>
      )}

      <div className="field-group">
        <h3>DNS</h3>
        <div className="field-row">
          <div className="field">
            <label>{tr('Резолвер клиента', 'Client resolver')}</label>
            <select value={s.dnsMode} onChange={(e) => patch({dnsMode: e.target.value})}>
              {DNS_MODES.map((m) => (
                <option key={m} value={m}>
                  {m}
                </option>
              ))}
            </select>
          </div>
          <div className="field" style={{flex: 2}}>
            <label>{tr('Свои DNS-серверы (через запятую)', 'Custom DNS servers (comma-separated)')}</label>
            <input
              type="text"
              value={(s.dnsServers ?? []).join(', ')}
              onChange={(e) => patch({dnsServers: e.target.value.split(',').map((v) => v.trim()).filter(Boolean)})}
              placeholder="192.168.31.1"
            />
          </div>
        </div>
      </div>

      <div className="field-group">
        <h3>{tr('Дополнительно', 'Advanced')}</h3>
        <div className="field-row">
          <div className="field" style={{flex: 2}}>
            <label>Client ID</label>
            <div className="inline-actions">
              <input type="text" className="mono" value={s.clientId} onChange={(e) => patch({clientId: e.target.value.trim()})} />
              <button className="btn small" onClick={() => GenerateClientID().then((id) => patch({clientId: id}))}>
                {tr('Сгенерировать', 'Generate')}
              </button>
            </div>
          </div>
        </div>
        <label className="checkbox-row">
          <input type="checkbox" checked={s.manualCaptcha} onChange={(e) => patch({manualCaptcha: e.target.checked})} />
          {tr('Ручная VK captcha', 'Manual VK captcha')}
        </label>
        <label className="checkbox-row">
          <input type="checkbox" checked={s.routes} onChange={(e) => patch({routes: e.target.checked})} />
          {tr('Автоматические маршруты к TURN-серверам (нужны права администратора)', 'Automatic TURN server routes (administrator authorization required)')}
        </label>
        <label className="checkbox-row">
          <input type="checkbox" checked={s.debug} onChange={(e) => patch({debug: e.target.checked})} />
          {tr('Подробные debug-логи', 'Verbose debug logs')}
        </label>
      </div>
    </>
  )
}
