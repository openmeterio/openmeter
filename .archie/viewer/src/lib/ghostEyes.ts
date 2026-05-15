interface EyeOffsetInput {
  eyeX: number
  eyeY: number
  cursorX: number
  cursorY: number
  maxOffset: number
}

export function calculateEyeOffset({ eyeX, eyeY, cursorX, cursorY, maxOffset }: EyeOffsetInput) {
  const dx = cursorX - eyeX
  const dy = cursorY - eyeY
  const distance = Math.hypot(dx, dy)

  if (distance === 0) return { x: 0, y: 0 }

  const scale = Math.min(maxOffset, distance) / distance

  return {
    x: Math.round(dx * scale * 100) / 100,
    y: Math.round(dy * scale * 100) / 100,
  }
}
