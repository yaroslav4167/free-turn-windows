import type {UpdateState} from './desktop'
import {currentLanguage, tr} from './language.ts'

export type UpdateAction = 'checking' | 'downloading' | 'installing'

function progressPhases(): Record<string, {title: string; detail: string}> { return {
  checking: {title: tr('Проверка обновлений', 'Checking for updates'), detail: tr('Получаем информацию о последнем стабильном релизе.', 'Fetching the latest stable release information.')},
  confirming: {title: tr('Ожидание подтверждения', 'Waiting for confirmation'), detail: tr('Подтвердите установку или выберите место сохранения в открытом диалоге.', 'Confirm installation or select a destination in the open dialog.')},
  preparing: {title: tr('Подготовка обновления', 'Preparing update'), detail: tr('Готовим безопасное скачивание архива.', 'Preparing a verified archive download.')},
  checksum: {title: tr('Получение контрольной суммы', 'Fetching checksum'), detail: tr('Загружаем SHA-256 из релиза.', 'Downloading the release SHA-256 checksum.')},
  connecting: {title: tr('Соединение с сервером', 'Connecting to server'), detail: tr('Ожидаем ответ сервера загрузки GitHub.', 'Waiting for the GitHub download server.')},
  downloading: {title: tr('Скачивание обновления', 'Downloading update'), detail: tr('VPN продолжает работать до завершения подготовки.', 'VPN stays connected until preparation is complete.')},
  verifying: {title: tr('Проверка SHA-256', 'Verifying SHA-256'), detail: tr('Архив скачан. Проверяем целостность и сохраняем файл.', 'Archive downloaded. Verifying integrity and saving the file.')},
  extracting: {title: tr('Распаковка и проверка бинарников', 'Extracting and verifying binaries'), detail: tr('Проверяем состав архива и архитектуру. Установленные файлы пока не меняются.', 'Checking archive contents and architecture. Installed files are not changed yet.')},
  restarting: {title: tr('Подготовка перезапуска', 'Preparing restart'), detail: tr('Сейчас VPN отключится и окно закроется. Установка продолжится отдельно; дождитесь запуска новой версии.', 'VPN will disconnect and the window will close. Installation continues separately; wait for the new version to launch.')},
} }

type ProgressState = Pick<UpdateState, 'busy' | 'phase' | 'received' | 'total' | 'bytesPerSecond' | 'remainingSeconds'>

export function formatUpdateBytes(bytes: number): string {
  let value = Number.isFinite(bytes) ? Math.max(0, bytes) : 0
  const units = [tr('Б', 'B'), tr('КиБ', 'KiB'), tr('МиБ', 'MiB'), tr('ГиБ', 'GiB')]
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) { value /= 1024; unit++ }
  const decimals = unit > 0 && value < 10 ? 1 : 0
  return `${new Intl.NumberFormat(currentLanguage(), {useGrouping: false, minimumFractionDigits: decimals, maximumFractionDigits: decimals}).format(value)} ${units[unit]}`
}

export function formatUpdateETA(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return tr('Время рассчитывается…', 'Estimating time…')
  if (seconds < 60) return tr('Осталось ≈ {count} с', {count: Math.ceil(seconds)})
  if (seconds < 3600) return tr('Осталось ≈ {count} мин', {count: Math.ceil(seconds / 60)})
  return tr('Осталось ≈ {count} ч', {count: Math.ceil(seconds / 3600)})
}

export function updateProgress(state: ProgressState | null, pending: UpdateAction | null) {
  const phases = progressPhases()
  if (!state?.busy && !pending) return null
  const phase = state?.busy ? state.phase || state.busy : pending === 'checking' ? 'checking' : 'confirming'
  const info = phases[phase] || phases.preparing
  const downloading = phase === 'downloading'
  const total = state && Number.isFinite(state.total) ? Math.max(0, state.total) : 0
  const received = state && Number.isFinite(state.received) ? Math.max(0, state.received) : 0
  const percent = downloading && total > 0 ? Math.min(100, Math.floor(received / total * 100)) : null
  const rate = state && Number.isFinite(state.bytesPerSecond) ? Math.max(0, state.bytesPerSecond) : 0
  return {
    phase, ...info, percent,
    transfer: downloading ? `${formatUpdateBytes(Math.min(received, total || received))} ${tr('из', 'of')} ${total ? formatUpdateBytes(total) : '…'}` : '',
    speed: downloading ? rate > 0 ? `${formatUpdateBytes(rate)}${tr('/с', '/s')}` : tr('Скорость рассчитывается…', 'Estimating speed…') : '',
    eta: downloading ? formatUpdateETA(state?.remainingSeconds ?? -1) : '',
  }
}
