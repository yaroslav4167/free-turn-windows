export function connectionState(state: string, pending: boolean, vpn: boolean) {
  const connected = state === 'connected' && !pending
  const active = pending || ['connecting', 'connected', 'captcha'].includes(state)
  const busy = pending || state === 'connecting' || state === 'captcha'
  const heading = connected ? vpn ? 'VPN включён' : 'Реле подключено'
    : state === 'captcha' ? 'Требуется проверка VK'
    : busy ? state === 'connected' ? 'Завершение настройки…' : 'Подключение…'
    : state === 'error' ? 'Ошибка подключения' : vpn ? 'VPN выключен' : 'Реле отключено'
  const action = connected ? 'Отключить' : active ? 'Отменить' : vpn ? 'Включить VPN' : 'Подключиться'
  return {connected, active, busy, heading, action}
}
