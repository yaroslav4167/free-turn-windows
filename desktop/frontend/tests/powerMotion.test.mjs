import {test} from 'node:test'
import assert from 'node:assert/strict'
import {advancePowerMotion} from '../src/powerMotion.ts'
import {powerShapeAt, roundedPowerShape, circlePowerShape} from '../src/powerShape.ts'

test('morph has matching curve topology and real intermediate contours', () => {
  assert.equal(powerShapeAt(0), roundedPowerShape)
  assert.equal(powerShapeAt(1), circlePowerShape)
  const middle = powerShapeAt(0.5)
  assert.notEqual(middle, roundedPowerShape)
  assert.notEqual(middle, circlePowerShape)
  for (const shape of [roundedPowerShape, middle, circlePowerShape]) {
    assert.equal([...shape.matchAll(/\bQ\b/g)].length, 20)
    assert.doesNotMatch(shape, /NaN|Infinity/)
    for (const coordinate of shape.matchAll(/[\d.]+/g)) {
      assert.ok(Number(coordinate[0]) >= 0 && Number(coordinate[0]) <= 150)
    }
  }
})

test('connecting eases into rotation and connection eases into a circle without snapping', () => {
  let motion = {morph: 0, morphVelocity: 0, angle: 0, speed: 0}
  motion = advancePowerMotion(motion, false, true, 1 / 60)
  assert.ok(motion.speed > 0 && motion.speed < 45)
  for (let i = 0; i < 120; i++) motion = advancePowerMotion(motion, false, true, 1 / 60)
  const before = motion
  motion = advancePowerMotion(motion, true, false, 1 / 60)
  assert.ok(motion.morph > 0 && motion.morph < 1)
  assert.ok(motion.speed > 0 && motion.speed < before.speed)
  assert.ok(motion.angle > before.angle)
  assert.ok(motion.angle - before.angle < 0.76)
  for (let i = 0; i < 180; i++) motion = advancePowerMotion(motion, true, false, 1 / 60)
  assert.equal(motion.morph, 1)
  assert.equal(motion.speed, 0)
  const stopped = motion.angle
  motion = advancePowerMotion(motion, true, false, 1 / 60)
  assert.equal(motion.angle, stopped)
  motion = advancePowerMotion(motion, false, false, 1 / 60)
  assert.ok(motion.morph > 0 && motion.morph < 1)
  assert.equal(motion.angle, stopped)
})

test('reversing a transition starts from its current shape and is frame-rate independent', () => {
  const initial = {morph: 0.4, morphVelocity: 0, angle: 123, speed: 12}
  const reversed = advancePowerMotion(initial, false, true, 1 / 60)
  assert.ok(reversed.morph < 0.4 && reversed.morph > 0)
  assert.ok(reversed.angle > 123)
  const coarse = advancePowerMotion(initial, true, false, 1 / 30)
  let fine = advancePowerMotion(initial, true, false, 1 / 60)
  fine = advancePowerMotion(fine, true, false, 1 / 60)
  for (const field of ['morph', 'morphVelocity', 'angle', 'speed']) assert.ok(Math.abs(coarse[field] - fine[field]) < 1e-10)
})

test('circle transition has one gentle spring overshoot and then settles', () => {
  let motion = {morph: 0, morphVelocity: 0, angle: 80, speed: 30}
  let maximum = 0
  let changes = 0
  let previousVelocity = 0
  for (let frame = 0; frame < 180; frame++) {
    motion = advancePowerMotion(motion, true, false, 1 / 60)
    maximum = Math.max(maximum, motion.morph)
    if (previousVelocity > 0 && motion.morphVelocity < 0) changes++
    previousVelocity = motion.morphVelocity
  }
  assert.ok(maximum > 1.08 && maximum < 1.11, `overshoot too strong or absent: ${maximum}`)
  assert.ok(changes >= 1)
  assert.equal(motion.morph, 1)
  assert.equal(motion.morphVelocity, 0)
  assert.notEqual(powerShapeAt(maximum), circlePowerShape)
  assert.ok(1 + (maximum - 1) * 0.4 < 1.045)
})
