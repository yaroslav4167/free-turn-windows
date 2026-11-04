import {useEffect, useRef, useState} from 'react'
import {advancePowerMotion, type PowerMotion} from './powerMotion'

export function usePowerMotion(connected: boolean, busy: boolean) {
  const [motion, setMotion] = useState<PowerMotion>({morph: connected ? 1 : 0, morphVelocity: 0, angle: 0, speed: 0})
  const current = useRef(motion)
  useEffect(() => {
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)')
    let frame = 0
    let last = performance.now()
    const publish = (next: PowerMotion) => {
      current.current = next
      setMotion(next)
    }
    const tick = (now: number) => {
      const next = advancePowerMotion(current.current, connected, busy, (now - last) / 1000)
      last = now
      publish(next)
      if (busy || next.speed !== 0 || next.morph !== (connected ? 1 : 0) || next.morphVelocity !== 0) {
        frame = window.requestAnimationFrame(tick)
      }
    }
    const start = () => {
      window.cancelAnimationFrame(frame)
      if (reduced.matches) {
        publish({morph: connected ? 1 : 0, morphVelocity: 0, angle: current.current.angle, speed: 0})
      } else {
        last = performance.now()
        frame = window.requestAnimationFrame(tick)
      }
    }
    start()
    reduced.addEventListener('change', start)
    return () => {
      window.cancelAnimationFrame(frame)
      reduced.removeEventListener('change', start)
    }
  }, [connected, busy])
  return motion
}
