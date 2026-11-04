// Quadratic curves through alternating radial control points make ten soft
// hills. Midpoints join the curves tangentially, with no sharp tips or valleys.
const points = Array.from({length: 20}, (_, index) => {
  const angle = -Math.PI / 2 + index * Math.PI / 10
  const radius = index % 2 === 0 ? 74 : 55
  return {x: 75 + radius * Math.cos(angle), y: 75 + radius * Math.sin(angle)}
})
const midpoint = (a: typeof points[number], b: typeof points[number]) =>
  `${((a.x + b.x) / 2).toFixed(3)} ${((a.y + b.y) / 2).toFixed(3)}`

export const roundedPowerShape = `M ${midpoint(points[19], points[0])} ` + points.map((point, index) =>
  `Q ${point.x.toFixed(3)} ${point.y.toFixed(3)} ${midpoint(point, points[(index + 1) % points.length])}`,
).join(' ') + ' Z'

// Matching quadratic segments allow a real contour morph, not a cross-fade.
// Tangent circle segments differ from a perfect circle by less than 0.01px.
export function powerShapeAt(progress: number): string {
  // Leave room for the spring's gentle overshoot instead of clipping it.
  const blend = Math.max(-0.125, Math.min(1.125, progress))
  const coordinate = (star: number, circle: number) => (star + (circle - star) * blend).toFixed(3)
  const endpoint = (index: number) => {
    const next = (index + 1) % points.length
    const angle = -Math.PI / 2 + (index + 0.5) * Math.PI / 10
    return `${coordinate((points[index].x + points[next].x) / 2, 75 + 69 * Math.cos(angle))} ${coordinate((points[index].y + points[next].y) / 2, 75 + 69 * Math.sin(angle))}`
  }
  return `M ${endpoint(19)} ` + points.map((point, index) => {
    const angle = -Math.PI / 2 + index * Math.PI / 10
    const radius = 69 / Math.cos(Math.PI / 20)
    return `Q ${coordinate(point.x, 75 + radius * Math.cos(angle))} ${coordinate(point.y, 75 + radius * Math.sin(angle))} ${endpoint(index)}`
  }).join(' ') + ' Z'
}

export const circlePowerShape = powerShapeAt(1)
