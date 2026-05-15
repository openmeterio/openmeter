import assert from 'node:assert/strict'
import { test } from 'node:test'

import { calculateEyeOffset } from './ghostEyes'

test('keeps pupils centered when cursor is at the eye center', () => {
  assert.deepEqual(calculateEyeOffset({ eyeX: 100, eyeY: 100, cursorX: 100, cursorY: 100, maxOffset: 6 }), {
    x: 0,
    y: 0,
  })
})

test('caps pupil movement to the configured maximum distance', () => {
  assert.deepEqual(calculateEyeOffset({ eyeX: 100, eyeY: 100, cursorX: 200, cursorY: 100, maxOffset: 6 }), {
    x: 6,
    y: 0,
  })
})

test('preserves diagonal direction within the movement radius', () => {
  const offset = calculateEyeOffset({ eyeX: 100, eyeY: 100, cursorX: 103, cursorY: 104, maxOffset: 6 })

  assert.equal(offset.x, 3)
  assert.equal(offset.y, 4)
})
