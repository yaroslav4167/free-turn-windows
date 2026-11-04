import {useEffect, useState} from 'react'
import {expressiveLoaderFrame} from '../expressiveLoader'

export default function ConnectionLoader({active}: {active: boolean}) {
  const [frame, setFrame] = useState(() => expressiveLoaderFrame(0))
  useEffect(() => {
    if (!active) return
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)')
    let animation = 0
    let previous = performance.now()
    let elapsed = 0
    setFrame(expressiveLoaderFrame(0))
    const tick = (now: number) => {
      elapsed += Math.min(0.1, Math.max(0, (now - previous) / 1000))
      previous = now
      setFrame(expressiveLoaderFrame(elapsed))
      animation = window.requestAnimationFrame(tick)
    }
    const start = () => {
      window.cancelAnimationFrame(animation)
      if (!reduced.matches) {
        previous = performance.now()
        animation = window.requestAnimationFrame(tick)
      }
    }
    start()
    reduced.addEventListener('change', start)
    return () => {
      window.cancelAnimationFrame(animation)
      reduced.removeEventListener('change', start)
    }
  }, [active])
  return <svg className="connection-loader" viewBox="0 0 48 48" width="48" height="48" aria-hidden="true" focusable="false">
    <path d={frame.path} fill="currentColor" transform={`rotate(${frame.angle} 24 24)`} />
  </svg>
}
