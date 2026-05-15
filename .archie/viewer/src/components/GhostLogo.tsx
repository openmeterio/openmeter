import { useEffect, useId, useRef, useState } from 'react'
import { calculateEyeOffset } from '@/lib/ghostEyes'

interface GhostLogoProps {
  className?: string
  size?: number
}

const VIEWBOX_SIZE = 200
const LEFT_EYE = { x: 79, y: 88 }
const RIGHT_EYE = { x: 121, y: 88 }

export function GhostLogo({ className = '', size = 32 }: GhostLogoProps) {
  const ref = useRef<SVGSVGElement>(null)
  const id = useId().replace(/:/g, '')
  const gradientId = `archieGhostGrad-${id}`
  const glowId = `archieGhostGlow-${id}`
  const [cursor, setCursor] = useState({ x: VIEWBOX_SIZE / 2, y: VIEWBOX_SIZE / 2 })

  useEffect(() => {
    const handlePointerMove = (event: PointerEvent) => {
      const rect = ref.current?.getBoundingClientRect()
      if (!rect) return

      setCursor({
        x: ((event.clientX - rect.left) / rect.width) * VIEWBOX_SIZE,
        y: ((event.clientY - rect.top) / rect.height) * VIEWBOX_SIZE,
      })
    }

    window.addEventListener('pointermove', handlePointerMove, { passive: true })
    return () => window.removeEventListener('pointermove', handlePointerMove)
  }, [])

  const leftPupil = calculateEyeOffset({
    eyeX: LEFT_EYE.x,
    eyeY: LEFT_EYE.y,
    cursorX: cursor.x,
    cursorY: cursor.y,
    maxOffset: 6,
  })
  const rightPupil = calculateEyeOffset({
    eyeX: RIGHT_EYE.x,
    eyeY: RIGHT_EYE.y,
    cursorX: cursor.x,
    cursorY: cursor.y,
    maxOffset: 6,
  })

  return (
    <svg
      ref={ref}
      aria-hidden="true"
      className={className}
      viewBox="0 0 200 200"
      width={size}
      height={size}
      xmlns="http://www.w3.org/2000/svg"
    >
      <defs>
        <linearGradient id={gradientId} x1="20%" y1="0%" x2="80%" y2="100%">
          <stop offset="0%" stopColor="#8ED8F5" />
          <stop offset="100%" stopColor="#3AACE0" />
        </linearGradient>
        <filter id={glowId}>
          <feGaussianBlur stdDeviation="4" result="coloredBlur" />
          <feMerge>
            <feMergeNode in="coloredBlur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>

      <path
        d="M 40 105 C 40 63, 68 32, 100 32 C 132 32, 160 63, 160 105 L 160 162 C 151 155, 142 148, 133 155 C 124 162, 115 155, 100 155 C 85 155, 76 162, 67 155 C 58 148, 49 155, 40 162 Z"
        fill={`url(#${gradientId})`}
        filter={`url(#${glowId})`}
      />

      <g opacity="0.15" stroke="#ffffff" strokeWidth="1.2" strokeDasharray="4,3">
        <line x1="60" y1="85" x2="140" y2="85" />
        <line x1="55" y1="110" x2="145" y2="110" />
        <line x1="60" y1="135" x2="140" y2="135" />
        <line x1="78" y1="55" x2="78" y2="150" />
        <line x1="100" y1="42" x2="100" y2="152" />
        <line x1="122" y1="55" x2="122" y2="150" />
      </g>

      <ellipse cx={LEFT_EYE.x} cy={LEFT_EYE.y} rx="14" ry="15" fill="white" />
      <ellipse cx={RIGHT_EYE.x} cy={RIGHT_EYE.y} rx="14" ry="15" fill="white" />

      <g style={{ transition: 'transform 120ms ease-out' }}>
        <ellipse cx={83 + leftPupil.x} cy={91 + leftPupil.y} rx="7" ry="8" fill="#1A2E4A" />
        <circle cx={86 + leftPupil.x} cy={87 + leftPupil.y} r="2.5" fill="white" />
      </g>
      <g style={{ transition: 'transform 120ms ease-out' }}>
        <ellipse cx={125 + rightPupil.x} cy={91 + rightPupil.y} rx="7" ry="8" fill="#1A2E4A" />
        <circle cx={128 + rightPupil.x} cy={87 + rightPupil.y} r="2.5" fill="white" />
      </g>
    </svg>
  )
}
