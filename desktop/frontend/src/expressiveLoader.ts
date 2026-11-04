// An original, Material Expressive-inspired indicator: an oval and soft three-,
// four-, five- and six-lobed shapes continuously blend while rotating.
export function expressiveLoaderFrame(seconds: number) {
  const time = Number.isFinite(seconds) ? Math.max(0, seconds) : 0
  const shapes = [0, 3, 4, 5, 6]
  const phase = time / 0.8
  const index = Math.floor(phase)
  const fraction = phase - index
  const blend = fraction * fraction * (3 - 2 * fraction)
  const from = shapes[index % shapes.length]
  const to = shapes[(index + 1) % shapes.length]
  const points = Array.from({length: 64}, (_, i) => {
    const angle = i * Math.PI / 32 - Math.PI / 2
    const radiusFor = (shape: number) => shape === 0
      ? 1 / Math.sqrt((Math.cos(angle) / 18) ** 2 + (Math.sin(angle) / 11) ** 2)
      : 14.5 + 3.5 * Math.cos(shape * angle)
    const radius = (1 - blend) * radiusFor(from) + blend * radiusFor(to)
    return {x: 24 + radius * Math.cos(angle), y: 24 + radius * Math.sin(angle)}
  })
  const midpoint = (index: number) => {
    const a = points[index]
    const b = points[(index + 1) % points.length]
    return `${((a.x + b.x) / 2).toFixed(2)} ${((a.y + b.y) / 2).toFixed(2)}`
  }
  const path = `M ${midpoint(63)} ` + points.map((point, index) =>
    `Q ${point.x.toFixed(2)} ${point.y.toFixed(2)} ${midpoint(index)}`,
  ).join(' ') + ' Z'
  return {path, angle: time * 180}
}
