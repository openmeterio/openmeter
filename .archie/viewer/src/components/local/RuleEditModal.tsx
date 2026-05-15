import { useState } from 'react'

const SEVERITIES = [
  'decision_violation',
  'pitfall_triggered',
  'tradeoff_undermined',
  'pattern_divergence',
  'mechanical_violation',
]

interface Props {
  rule: {
    description?: string
    why?: string
    forced_by?: string
    enables?: string
    alternative?: string
    example?: string
    severity_class?: string
  }
  onSave: (patch: Record<string, string>) => Promise<void>
  onCancel: () => void
}

const FIELD_LABEL = 'block mb-2 text-[10px] font-black uppercase tracking-[0.2em] text-ink/40'
const TEXTAREA =
  'w-full bg-papaya-50 border border-papaya-300/40 rounded-lg px-3 py-2 mb-4 text-ink placeholder:text-ink/30 focus:outline-none focus:ring-2 focus:ring-teal-500/30 focus:border-teal-500/40 transition'

export default function RuleEditModal({ rule, onSave, onCancel }: Props) {
  const [description, setDescription] = useState(rule.description || '')
  const [why, setWhy] = useState(rule.why || '')
  const [forcedBy, setForcedBy] = useState(rule.forced_by || '')
  const [enables, setEnables] = useState(rule.enables || '')
  const [alternative, setAlternative] = useState(rule.alternative || '')
  const [example, setExample] = useState(rule.example || '')
  const [severity, setSeverity] = useState(rule.severity_class || 'pattern_divergence')

  return (
    <div
      className="fixed inset-0 bg-ink/40 backdrop-blur-sm flex items-center justify-center z-50 p-6"
      onClick={onCancel}
    >
      <div
        className="bg-white border border-papaya-300 rounded-2xl shadow-2xl shadow-ink/20 p-8 w-full max-w-2xl max-h-[85vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="text-xl font-black tracking-tight text-ink mb-1">Edit rule</h3>
        <p className="text-sm text-ink/50 mb-6">
          Refine the description, reasoning, or example shown to the enforcement hook.
        </p>

        <label className={FIELD_LABEL}>Description</label>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          className={TEXTAREA}
          rows={2}
        />

        <label className={FIELD_LABEL}>Why</label>
        <textarea
          value={why}
          onChange={(e) => setWhy(e.target.value)}
          className={TEXTAREA}
          rows={4}
        />

        <label className={FIELD_LABEL}>Forced by</label>
        <textarea
          value={forcedBy}
          onChange={(e) => setForcedBy(e.target.value)}
          className={TEXTAREA}
          rows={2}
          placeholder="The constraint that drove this decision (one sentence)"
        />

        <label className={FIELD_LABEL}>Enables</label>
        <textarea
          value={enables}
          onChange={(e) => setEnables(e.target.value)}
          className={TEXTAREA}
          rows={2}
          placeholder="What capability this preserves (one sentence)"
        />

        <label className={FIELD_LABEL}>Do this instead</label>
        <textarea
          value={alternative}
          onChange={(e) => setAlternative(e.target.value)}
          className={TEXTAREA}
          rows={2}
          placeholder="The correct path the agent should take (one imperative sentence)"
        />

        <label className={FIELD_LABEL}>Example</label>
        <textarea
          value={example}
          onChange={(e) => setExample(e.target.value)}
          className={`${TEXTAREA} font-mono text-sm`}
          rows={5}
        />

        <label className={FIELD_LABEL}>Severity class</label>
        <select
          value={severity}
          onChange={(e) => setSeverity(e.target.value)}
          className={TEXTAREA}
        >
          {SEVERITIES.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>

        <div className="flex gap-3 justify-end pt-2">
          <button
            onClick={onCancel}
            className="px-4 py-2 text-sm font-bold text-ink/60 hover:text-ink transition-colors rounded-lg"
          >
            Cancel
          </button>
          <button
            onClick={() =>
              onSave({
                description,
                why,
                forced_by: forcedBy,
                enables,
                alternative,
                example,
                severity_class: severity,
              })
            }
            className="px-5 py-2 text-sm font-bold bg-teal-500 text-white rounded-lg hover:bg-teal-600 transition-colors shadow-sm shadow-teal-500/30"
          >
            Save
          </button>
        </div>
      </div>
    </div>
  )
}
