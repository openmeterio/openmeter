import { useState } from 'react'
import { Check, X, Pencil, PowerOff, Power } from 'lucide-react'
import RuleEditModal from './RuleEditModal'

interface Props {
  rule: { id: string; description?: string; why?: string; example?: string; severity_class?: string }
  state: 'active' | 'proposed' | 'ignored'
  onAction: (action: 'adopt' | 'reject' | 'disable' | 'enable') => Promise<void>
  onEdit: (patch: Record<string, string>) => Promise<void>
}

// Pill-shaped action buttons matching the existing severity badge style
// (text-[9px] font-black uppercase tracking-widest px-2 py-0.5 rounded
// border + accent color/10 fill + accent border/20). Each button has an
// icon AND a label so the affordance is obvious without a tooltip.
//
// Color mapping aligns to the existing palette:
//   adopt / enable  → teal     (info-style — positive intent)
//   disable / reject → tangerine (warn-style — pulls back something live)
//   edit            → papaya/ink — neutral, unobtrusive
const PILL =
  'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[9px] font-black uppercase tracking-widest border transition-colors'

const TEAL = 'bg-teal/10 text-teal border-teal/20 hover:bg-teal/20'
const TANGERINE = 'bg-tangerine/10 text-tangerine border-tangerine/30 hover:bg-tangerine/20'
const NEUTRAL = 'bg-ink/[0.04] text-ink/60 border-ink/10 hover:bg-ink/[0.08] hover:text-ink'

export default function RuleControls({ rule, state, onAction, onEdit }: Props) {
  const [editing, setEditing] = useState(false)
  const [pending, setPending] = useState<string | null>(null)

  const run = async (label: string, fn: () => Promise<void>) => {
    setPending(label)
    try {
      await fn()
    } finally {
      setPending(null)
    }
  }

  return (
    // Stop click propagation so action buttons don't also toggle the parent
    // rule card open/closed (the card header has its own onClick).
    <div className="flex flex-wrap gap-1.5" onClick={(e) => e.stopPropagation()}>
      {state === 'proposed' && (
        <>
          <button
            onClick={() => run('adopt', () => onAction('adopt'))}
            disabled={pending !== null}
            className={`${PILL} ${TEAL} disabled:opacity-50`}
          >
            <Check className="w-3 h-3" />
            <span>{pending === 'adopt' ? 'Adopting…' : 'Adopt'}</span>
          </button>
          <button
            onClick={() => run('reject', () => onAction('reject'))}
            disabled={pending !== null}
            className={`${PILL} ${NEUTRAL} disabled:opacity-50`}
          >
            <X className="w-3 h-3" />
            <span>{pending === 'reject' ? 'Rejecting…' : 'Reject'}</span>
          </button>
        </>
      )}
      {state === 'active' && (
        <>
          <button
            onClick={() => setEditing(true)}
            disabled={pending !== null}
            className={`${PILL} ${NEUTRAL} disabled:opacity-50`}
          >
            <Pencil className="w-3 h-3" />
            <span>Edit</span>
          </button>
          <button
            onClick={() => run('disable', () => onAction('disable'))}
            disabled={pending !== null}
            className={`${PILL} ${TANGERINE} disabled:opacity-50`}
          >
            <PowerOff className="w-3 h-3" />
            <span>{pending === 'disable' ? 'Disabling…' : 'Disable'}</span>
          </button>
        </>
      )}
      {state === 'ignored' && (
        <button
          onClick={() => run('enable', () => onAction('enable'))}
          disabled={pending !== null}
          className={`${PILL} ${TEAL} disabled:opacity-50`}
        >
          <Power className="w-3 h-3" />
          <span>{pending === 'enable' ? 'Enabling…' : 'Enable'}</span>
        </button>
      )}
      {editing && (
        <RuleEditModal
          rule={rule}
          onSave={(patch) => onEdit(patch).finally(() => setEditing(false))}
          onCancel={() => setEditing(false)}
        />
      )}
    </div>
  )
}
