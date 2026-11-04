export interface PowerMotion {
  morph: number
  morphVelocity: number
  angle: number
  speed: number
}

// Exponential easing is frame-rate independent; retain the current angle when
// stopping so the silhouette never snaps back to its original orientation.
export function advancePowerMotion(current: PowerMotion, connected: boolean, busy: boolean, seconds: number): PowerMotion {
  const dt = Math.max(0, Math.min(0.1, seconds))
  const targetMorph = connected ? 1 : 0
  const targetSpeed = busy ? 45 : 0
  const speedDamping = 8
  const speedDecay = Math.exp(-speedDamping * dt)
  // Analytic underdamped spring: a small overshoot, independent of frame rate.
  const frequency = 20
  const damping = 0.6 * frequency
  const dampedFrequency = Math.sqrt(frequency * frequency - damping * damping)
  const decay = Math.exp(-damping * dt)
  const sin = Math.sin(dampedFrequency * dt)
  const cos = Math.cos(dampedFrequency * dt)
  const displacement = current.morph - targetMorph
  let morph = targetMorph + decay * (displacement * cos + (current.morphVelocity + damping * displacement) / dampedFrequency * sin)
  let morphVelocity = decay * (current.morphVelocity * cos - (damping * current.morphVelocity + frequency * frequency * displacement) / dampedFrequency * sin)
  let speed = targetSpeed + (current.speed - targetSpeed) * speedDecay
  const angle = current.angle + targetSpeed * dt + (current.speed - targetSpeed) * (1 - speedDecay) / speedDamping
  if (Math.abs(morph - targetMorph) < 0.0001 && Math.abs(morphVelocity) < 0.001) {
    morph = targetMorph
    morphVelocity = 0
  }
  if (Math.abs(speed - targetSpeed) < 0.001) speed = targetSpeed
  return {morph, morphVelocity, speed, angle}
}
