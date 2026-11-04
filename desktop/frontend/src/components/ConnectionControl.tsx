import {useMemo} from 'react'
import {powerShapeAt} from '../powerShape'
import {usePowerMotion} from '../usePowerMotion'
import {connectionState} from '../connectionState'
import ConnectionLoader from './ConnectionLoader'

interface Props {
  state: string
  pending: boolean
  vpn: boolean
  canConnect: boolean
  captcha: boolean
  onConnect: () => void
  onDisconnect: () => void
}

export default function ConnectionControl({state, pending, vpn, canConnect, captcha, onConnect, onDisconnect}: Props) {
  const presentation = connectionState(state, pending, vpn)
  const motion = usePowerMotion(presentation.connected, presentation.busy)
  const shape = useMemo(() => powerShapeAt(motion.morph), [motion.morph])
  const contour = {transform: `rotate(${motion.angle}deg) scale(${1 + Math.max(0, motion.morph - 1) * 0.4})`}
  return <section className={'connection-control' + (presentation.connected ? ' connected' : '') + (captcha && presentation.connected ? ' verification-required' : '')}>
    <button className="power-button" onClick={presentation.active ? onDisconnect : onConnect}
      disabled={!canConnect && !presentation.active} aria-label={presentation.action}
      aria-pressed={presentation.connected} aria-busy={presentation.busy}
      aria-describedby={captcha ? 'captcha-notice' : undefined}>
      <svg className="power-outline" style={contour} viewBox="0 0 150 150" aria-hidden="true" focusable="false"><path d={shape} /></svg>
      <span className="power-symbol" aria-hidden="true">
        <svg className="power-idle-icon" width="48" height="48" viewBox="0 0 48 48" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round">
          <path d="M24 5v18M13 11a18 18 0 1 0 22 0" />
        </svg>
        <ConnectionLoader active={presentation.busy} />
      </span>
      <span className="power-action">{presentation.action}</span>
    </button>
    <h3 aria-live="polite">{presentation.heading}</h3>
  </section>
}
