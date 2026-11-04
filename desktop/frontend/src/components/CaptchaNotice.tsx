import {BrowserOpenURL} from '../api'
import {tr} from '../language'

export default function CaptchaNotice({url, connected}: {url: string, connected: boolean}) {
  if (!url) return null
  return <div id="captcha-notice" className="notice info app-error captcha-notice" role="status">
    <span>{connected
      ? tr('Соединение работает. Для части потоков требуется проверка VK.')
      : tr('Для подключения требуется проверка VK в браузере.')}</span>
    <button className="btn small primary" onClick={() => BrowserOpenURL(url)}>
      {tr('Открыть проверку в браузере')}
    </button>
  </div>
}
