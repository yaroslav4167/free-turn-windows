import {ReactNode, useEffect, useId, useLayoutEffect, useRef, useState} from 'react'
import {createPortal} from 'react-dom'
import {infoTipPosition} from '../infoTipPosition'

export default function InfoTip({label, children}: {label: string, children: ReactNode}) {
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState<{left: number, top: number} | null>(null)
  const button = useRef<HTMLButtonElement>(null)
  const popup = useRef<HTMLDivElement>(null)
  const closeTimer = useRef<number | undefined>(undefined)
  const id = useId()

  const cancelClose = () => window.clearTimeout(closeTimer.current)
  const show = () => {
    cancelClose()
    setOpen(true)
  }
  // The short delay lets the pointer cross the gap into the portal popup.
  const scheduleClose = () => {
    cancelClose()
    closeTimer.current = window.setTimeout(() => setOpen(false), 160)
  }
  useEffect(() => () => window.clearTimeout(closeTimer.current), [])

  useLayoutEffect(() => {
    if (!open) return
    const place = () => {
      if (!button.current || !popup.current) return
      setPosition(infoTipPosition(button.current.getBoundingClientRect(), popup.current.getBoundingClientRect(), {
        width: window.innerWidth, height: window.innerHeight,
      }))
    }
    place()
    window.addEventListener('resize', place)
    window.addEventListener('scroll', place, true)
    return () => {
      window.removeEventListener('resize', place)
      window.removeEventListener('scroll', place, true)
    }
  }, [open, children])

  useEffect(() => {
    if (!open) return
    const outside = (event: PointerEvent) => {
      if (event.target instanceof Node && !button.current?.contains(event.target) && !popup.current?.contains(event.target)) {
        setOpen(false)
      }
    }
    const escape = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return
      event.stopPropagation()
      if (popup.current?.contains(document.activeElement)) button.current?.focus({preventScroll: true})
      setOpen(false)
    }
    document.addEventListener('pointerdown', outside, true)
    document.addEventListener('keydown', escape)
    return () => {
      document.removeEventListener('pointerdown', outside, true)
      document.removeEventListener('keydown', escape)
    }
  }, [open])

  return <span className="info-tip">
    <button ref={button} type="button" className="info-tip-button" aria-label={label}
      aria-expanded={open} aria-controls={open ? id : undefined} aria-describedby={open ? id : undefined}
      onPointerEnter={(event) => { if (event.pointerType !== 'touch') show() }}
      onPointerLeave={(event) => { if (event.pointerType !== 'touch') scheduleClose() }}
      onFocus={(event) => { if (event.currentTarget.matches(':focus-visible')) show() }}
      onBlur={scheduleClose}
      onClick={() => {
        cancelClose()
        setPosition(null)
        setOpen(!open)
      }}>ⓘ</button>
    {open && createPortal(<div ref={popup} id={id} role="tooltip" className="info-tip-content"
      onPointerEnter={cancelClose} onPointerLeave={scheduleClose}
      onFocus={cancelClose} onBlur={scheduleClose}
      style={{left: position?.left ?? 0, top: position?.top ?? 0, visibility: position ? 'visible' : 'hidden'}}>
      {children}
    </div>, document.body)}
  </span>
}
