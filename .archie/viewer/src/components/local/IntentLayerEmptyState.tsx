import { Database } from 'lucide-react'

interface Props {
  count: number
}

export default function IntentLayerEmptyState({ count }: Props) {
  return (
    <div className="max-w-3xl mx-auto">
      <div className="bg-white border border-papaya-300/40 rounded-2xl p-10 shadow-sm">
        <div className="w-12 h-12 rounded-2xl bg-teal-500/10 flex items-center justify-center mb-6 ring-1 ring-teal-500/20">
          <Database className="w-6 h-6 text-teal-700" />
        </div>
        <h2 className="text-2xl font-black tracking-tight text-ink mb-3">
          Per-folder context not yet generated
        </h2>
        <p className="text-ink/70 leading-relaxed mb-6">
          Archie can write a CLAUDE.md into each meaningful directory of your repo,
          giving AI agents directory-level architectural context (what this layer
          does, what it depends on, what to avoid here). Without this, agents only
          see the root CLAUDE.md.
        </p>

        <p className="text-[10px] font-black uppercase tracking-[0.2em] text-ink/40 mb-4">
          Two ways to generate
        </p>
        <ul className="list-none space-y-5 mb-8">
          <li className="border-l-2 border-teal-500/40 pl-5">
            <code className="inline-block bg-papaya-100 text-teal-700 px-2 py-0.5 rounded text-sm font-semibold">
              /archie-deep-scan
            </code>
            <p className="text-ink/60 text-sm mt-2">
              Runs the intent layer as Phase 7. Full baseline, ~15-20 min.
            </p>
          </li>
          <li className="border-l-2 border-teal-500/40 pl-5">
            <div className="flex flex-wrap items-center gap-2">
              <code className="inline-block bg-papaya-100 text-teal-700 px-2 py-0.5 rounded text-sm font-semibold">
                /archie-intent-layer prepare
              </code>
              <span className="text-ink/40 text-sm">&amp;&amp;</span>
              <code className="inline-block bg-papaya-100 text-teal-700 px-2 py-0.5 rounded text-sm font-semibold">
                /archie-intent-layer next-ready
              </code>
            </div>
            <p className="text-ink/60 text-sm mt-2">
              Incremental, resumable across sessions. Run next-ready until the queue is empty.
            </p>
          </li>
        </ul>

        <div className="pt-4 border-t border-papaya-300/40">
          <p className="text-xs text-ink/40">
            Detected: <span className="font-bold text-ink/70">{count}</span> per-folder CLAUDE.md file
            {count === 1 ? '' : 's'} outside the repo root.
          </p>
        </div>
      </div>
    </div>
  )
}
