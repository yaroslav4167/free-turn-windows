interface Anchor {left: number, right: number, top: number, bottom: number}
interface Size {width: number, height: number}

export function infoTipPosition(anchor: Anchor, popup: Size, viewport: Size) {
  const margin = 12
  const gap = 8
  const left = Math.max(margin, Math.min((anchor.left + anchor.right - popup.width) / 2, viewport.width - popup.width - margin))
  const below = anchor.bottom + gap
  const preferredTop = below + popup.height <= viewport.height - margin ? below : anchor.top - popup.height - gap
  const top = Math.max(margin, Math.min(preferredTop, viewport.height - popup.height - margin))
  return {left, top}
}
