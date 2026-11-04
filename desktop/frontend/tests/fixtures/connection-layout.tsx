// Serve with Vite. Exercise the real component in an iframe at native client sizes.
import React from 'react'
import {createRoot} from 'react-dom/client'
import Connection from '../../src/components/Connection'
import {configureLanguage} from '../../src/language'
import '../../src/style.css'

const root = createRoot(document.getElementById('root')!)
const server = {id: 'demo', name: 'Windows VPN', connMode: 'vpn', routingLists: []}
window.go = {main: {App: {MeasurePing: async () => ({serverId: 'demo', host: 'example.test', milliseconds: 42})}}} as any

window.renderLayout = (state = 'connected', language = 'ru', captcha = false) => {
  configureLanguage(language)
  root.render(<div className="app-shell">
    <aside className="navigation" />
    <main className="workspace">
      {captcha && <div className="notice captcha-notice">VK: требуется проверка <button className="btn">Открыть браузер</button></div>}
      <Connection servers={[server] as any} selected={server as any}
        status={{state, streams: 2, total: 2, serverId: 'demo', txRate: 1024, rxRate: 2048,
          txTotal: 4096, rxTotal: 8192, captchaUrl: captcha ? 'https://example.test' : '', errMsg: ''} as any}
        pending={false} trafficHistory={{rx: [], tx: []}}
        onSelect={() => {}} onConnect={() => {}} onDisconnect={() => {}} onAdd={() => {}} onEdit={() => {}} />
    </main>
  </div>)
}
window.renderLayout()
