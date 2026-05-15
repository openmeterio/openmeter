/** FixThisButton — copies an agent-agnostic "fix this" prompt to the
 * clipboard. Renders in every finding / pitfall card in both the share
 * viewer and the local viewer; the prompt itself names no Archie tooling
 * so the receiving agent (Claude Code, Cursor, Codex, …) can act on it
 * directly.
 *
 * Mobile = icon only; sm+ = icon + label. Clipboard write tries the secure
 * navigator.clipboard API first and falls back to a modal with a selectable
 * textarea when the API is unavailable (non-HTTPS or sandboxed iframes).
 *
 * Shift+Click always opens the modal so power users can review the prompt
 * before pasting.
 */

import { useEffect, useRef, useState } from 'react'
import { Wand2, Check, X, Clipboard } from 'lucide-react'
import { buildFixPrompt, type FixItem, type BuildOpts } from '@/lib/fixPrompt'

interface Props {
  item: FixItem
  blueprint?: any
  adoptedRules?: any
  semanticDuplications?: any[]
  className?: string
}

type CopyState = 'idle' | 'copied' | 'error'

export default function FixThisButton({
  item,
  blueprint,
  adoptedRules,
  semanticDuplications,
  className,
}: Props) {
  const [state, setState] = useState<CopyState>('idle')
  const [showModal, setShowModal] = useState(false)
  const resetRef = useRef<number | null>(null)

  // Avoid building the prompt on every render — only when we actually copy
  // or open the modal. This keeps long lists of findings cheap.
  const buildOpts: BuildOpts = { blueprint, adoptedRules, semanticDuplications }

  const flash = (next: CopyState) => {
    setState(next)
    if (resetRef.current !== null) window.clearTimeout(resetRef.current)
    resetRef.current = window.setTimeout(() => setState('idle'), 2200)
  }

  useEffect(
    () => () => {
      if (resetRef.current !== null) window.clearTimeout(resetRef.current)
    },
    [],
  )

  const handleClick = async (e: React.MouseEvent) => {
    // Shift+Click → modal escape hatch (review-before-paste flow).
    if (e.shiftKey) {
      e.preventDefault()
      setShowModal(true)
      return
    }
    const prompt = buildFixPrompt(item, buildOpts)
    const ok = await tryCopy(prompt)
    if (ok) {
      flash('copied')
    } else {
      // Clipboard API unavailable — open the modal as a manual-copy fallback.
      setShowModal(true)
    }
  }

  const label =
    state === 'copied' ? 'Copied' : state === 'error' ? 'Copy failed' : 'Fix this'
  const Icon = state === 'copied' ? Check : Wand2

  return (
    <>
      <button
        type="button"
        onClick={handleClick}
        aria-label="Fix this — copy a prompt for a coding agent"
        title={'Copy a fix prompt for any coding agent.\nShift+Click to review before copying.'}
        className={
          'inline-flex items-center gap-1.5 shrink-0 self-start ' +
          'h-8 px-2.5 sm:px-3 rounded-full border text-[11px] font-bold ' +
          'transition-all ' +
          (state === 'copied'
            ? 'bg-teal/10 text-teal border-teal/30'
            : 'bg-white/70 text-ink/70 border-papaya-400/60 hover:bg-teal/5 hover:text-teal hover:border-teal/30') +
          ' focus:outline-none focus:ring-2 focus:ring-teal/30 ' +
          (className || '')
        }
      >
        <Icon className="w-3.5 h-3.5" />
        <span className="hidden sm:inline">{label}</span>
      </button>
      {showModal && (
        <FixPromptModal
          item={item}
          buildOpts={buildOpts}
          onClose={() => setShowModal(false)}
          onCopied={() => flash('copied')}
        />
      )}
    </>
  )
}

// ── Modal ───────────────────────────────────────────────────────────────

function FixPromptModal({
  item,
  buildOpts,
  onClose,
  onCopied,
}: {
  item: FixItem
  buildOpts: BuildOpts
  onClose: () => void
  onCopied: () => void
}) {
  const [prompt] = useState(() => buildFixPrompt(item, buildOpts))
  const [copyOk, setCopyOk] = useState<null | boolean>(null)
  const taRef = useRef<HTMLTextAreaElement>(null)

  // Pre-select the textarea content so Cmd/Ctrl+C works without the user
  // having to click into the textarea first.
  useEffect(() => {
    const ta = taRef.current
    if (!ta) return
    ta.focus()
    ta.select()
  }, [])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  const copy = async () => {
    const ok = await tryCopy(prompt)
    setCopyOk(ok)
    if (ok) {
      onCopied()
      // Brief delay so the user sees the green state before the modal closes.
      window.setTimeout(onClose, 400)
    }
  }

  return (
    <div
      className="fixed inset-0 bg-ink/40 backdrop-blur-sm flex items-center justify-center z-50 p-6"
      onClick={onClose}
    >
      <div
        className="bg-white border border-papaya-300 rounded-2xl shadow-2xl shadow-ink/20 w-full max-w-3xl max-h-[85vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-6 pt-6 pb-4 border-b border-papaya-300/40 flex items-start justify-between gap-4">
          <div>
            <h3 className="text-xl font-black tracking-tight text-ink mb-1 inline-flex items-center gap-2">
              <Wand2 className="w-4 h-4 text-teal" />
              Fix-this prompt
            </h3>
            <p className="text-sm text-ink/50">
              Agent-agnostic. Paste into Claude Code, Cursor, Codex, or any
              coding agent that operates on this repository.
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="text-ink/40 hover:text-ink/70 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        <div className="px-6 py-4 flex-1 overflow-hidden flex flex-col">
          <textarea
            ref={taRef}
            value={prompt}
            readOnly
            className="w-full flex-1 min-h-[300px] font-mono text-[12px] leading-relaxed bg-papaya-50 border border-papaya-300/40 rounded-xl px-4 py-3 text-ink/80 focus:outline-none focus:ring-2 focus:ring-teal/30 resize-none"
          />
        </div>
        <div className="px-6 pb-6 pt-2 flex items-center justify-between gap-3">
          <span className="text-[11px] text-ink/40">
            {prompt.length.toLocaleString()} chars
            {copyOk === false && (
              <span className="ml-2 text-brandy">
                — Clipboard blocked. Select the text and copy manually.
              </span>
            )}
          </span>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={onClose}
              className="h-9 px-4 rounded-lg border border-papaya-300 text-ink/70 text-sm font-semibold hover:bg-papaya-50 transition-colors"
            >
              Close
            </button>
            <button
              type="button"
              onClick={copy}
              className={
                'h-9 px-4 rounded-lg text-sm font-bold inline-flex items-center gap-2 transition-colors ' +
                (copyOk
                  ? 'bg-teal/10 text-teal border border-teal/30'
                  : 'bg-teal text-white hover:bg-teal-700')
              }
            >
              {copyOk ? <Check className="w-3.5 h-3.5" /> : <Clipboard className="w-3.5 h-3.5" />}
              {copyOk ? 'Copied' : 'Copy to clipboard'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

// ── Clipboard helper ─────────────────────────────────────────────────────

async function tryCopy(text: string): Promise<boolean> {
  // Preferred path — secure context, modern API.
  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      /* fall through to legacy path */
    }
  }
  // Legacy fallback — only works while a user gesture is on the stack, but
  // it's reachable from non-HTTPS contexts and sandboxed iframes where the
  // Clipboard API throws or is undefined.
  try {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.top = '-1000px'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.focus()
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  } catch {
    return false
  }
}
