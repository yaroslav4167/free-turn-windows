
export function connectionState(state: string, pending: boolean, vpn: boolean) {
  const labels: Record<string, string> = {
    idle: tr('Отключено', 'Disconnected'),
    connecting: tr('Подключение…', 'Connecting…'),
    captcha: tr('Требуется проверка', 'Verification required'),
    error: tr('Не удалось подключиться', 'Connection failed'),
  }
  const active = state !== 'idle' && state !== 'error'
  const connected = state === 'connected' && !pending
  const visualState = pending && state !== 'captcha' && state !== 'error' ? 'connecting' : state
  return {
    active,
    connected,
    visualState,
    busy: visualState === 'connecting',
    heading: connected ? (vpn ? tr('VPN включён', 'VPN enabled') : tr('Реле запущено', 'Relay running')) : (labels[visualState] ?? visualState),
    action: connected ? tr('Отключить', 'Disconnect') : active ? tr('Отменить', 'Cancel') : tr('Подключить', 'Connect'),
    ariaLabel: connected
      ? vpn
        ? tr('Отключить VPN', 'Disconnect VPN')
        : tr('Остановить реле', 'Stop relay')
      : active
        ? tr('Отменить подключение', 'Cancel connection')
        : tr('Подключиться', 'Connect'),
  }
}
import {tr} from './language.ts'
