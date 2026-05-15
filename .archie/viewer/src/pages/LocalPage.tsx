import { useEffect, useState, lazy, Suspense } from 'react'
import { Database, FileText } from 'lucide-react'
import ReportPage from './ReportPage'
import type { Bundle } from '@/lib/api'
import { LocalEditContext, type LocalEditCtx } from '@/components/local/context/LocalEditContext'
import Toast from '@/components/local/Toast'

const GeneratedFilesBrowser = lazy(() => import('@/components/local/GeneratedFilesBrowser'))
const FolderClaudeMdsBrowser = lazy(() => import('@/components/local/FolderClaudeMdsBrowser'))

type Tab = 'report' | 'files'
type FilesView = 'folders' | 'generated'

export default function LocalPage() {
  const [bundle, setBundle] = useState<Bundle | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [tab, setTab] = useState<Tab>('report')
  const [filesView, setFilesView] = useState<FilesView>('folders')
  const [toast, setToast] = useState<string | null>(null)
  const [bundleVersion, setBundleVersion] = useState(0)

  const ctx: LocalEditCtx = {
    toggleRule: async (id, action) => {
      const res = await fetch('/api/rules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action, rule_id: id }),
        cache: 'no-store',
      })
      if (!res.ok) {
        const errBody = await res.json().catch(() => ({ error: `HTTP ${res.status}` }))
        setToast(`Failed: ${errBody.error || `HTTP ${res.status}`}`)
        // Still refresh — backend rejected because client thought the rule
        // was somewhere it isn't. Pulling fresh state shows the user what
        // is actually true so the next click works.
        setBundleVersion((v) => v + 1)
        return
      }
      setToast(`Rule ${id} ${action}d.`)
      setBundleVersion((v) => v + 1)
    },
    editRule: async (id, patch) => {
      const res = await fetch('/api/rules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'edit', rule_id: id, patch }),
        cache: 'no-store',
      })
      if (!res.ok) {
        const errBody = await res.json().catch(() => ({ error: `HTTP ${res.status}` }))
        setToast(`Failed: ${errBody.error || `HTTP ${res.status}`}`)
        setBundleVersion((v) => v + 1)
        return
      }
      setToast(`Rule ${id} updated.`)
      setBundleVersion((v) => v + 1)
    },
  }

  const [needsScan, setNeedsScan] = useState(false)

  useEffect(() => {
    fetch('/api/bundle', { cache: 'no-store' })
      .then((r) => {
        // 404 from the sidecar means blueprint.json doesn't exist yet — the
        // user installed Archie but hasn't run a scan. Surface a friendly
        // empty state with the next-step commands instead of a red error.
        if (r.status === 404) {
          setNeedsScan(true)
          return null
        }
        if (!r.ok) throw new Error(`Local bundle fetch failed (HTTP ${r.status}).`)
        return r.json()
      })
      .then((j) => {
        if (j) setBundle(j.bundle)
      })
      .catch((e) => setError(e.message))
  }, [bundleVersion])

  if (error) {
    return (
      <div className="p-8 max-w-2xl mx-auto">
        <h1 className="text-2xl font-semibold mb-2">Local viewer</h1>
        <p className="text-red-600">{error}</p>
      </div>
    )
  }
  if (needsScan) {
    return (
      <div className="min-h-screen flex items-center justify-center p-8">
        <div className="max-w-2xl bg-white border border-papaya-300/40 rounded-2xl shadow-sm p-10">
          <div className="w-12 h-12 rounded-2xl bg-teal-500/10 ring-1 ring-teal-500/20 flex items-center justify-center mb-6">
            <svg className="w-6 h-6 text-teal-700" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
          </div>
          <h1 className="text-2xl font-black tracking-tight text-ink mb-3">
            No blueprint yet
          </h1>
          <p className="text-ink/70 leading-relaxed mb-6">
            Archie is installed and the viewer is running, but there's no
            architectural data to display yet. Run one of the scans below
            inside this project from Claude Code, then refresh this page.
          </p>
          <p className="text-[10px] font-black uppercase tracking-[0.2em] text-ink/40 mb-3">
            Two ways to populate
          </p>
          <ul className="list-none space-y-4 mb-6">
            <li className="border-l-2 border-teal-500/40 pl-5">
              <code className="inline-block bg-papaya-100 text-teal-700 px-2 py-0.5 rounded text-sm font-semibold">
                /archie-scan
              </code>
              <p className="text-ink/60 text-sm mt-1">
                Fast architecture health check, 1-3 min. Produces blueprint
                metrics and proposes initial rules.
              </p>
            </li>
            <li className="border-l-2 border-teal-500/40 pl-5">
              <code className="inline-block bg-papaya-100 text-teal-700 px-2 py-0.5 rounded text-sm font-semibold">
                /archie-deep-scan
              </code>
              <p className="text-ink/60 text-sm mt-1">
                Comprehensive baseline, ~15-20 min. Full architectural
                blueprint plus the kind-tagged rule synthesis you'll curate
                from this viewer.
              </p>
            </li>
          </ul>
          <div className="pt-4 border-t border-papaya-300/40 text-xs text-ink/40">
            This viewer is read-only. It will surface whatever data lives in
            <code className="text-teal-700 mx-1">.archie/</code> after the scan completes.
          </div>
        </div>
      </div>
    )
  }
  if (!bundle) return <div className="p-8 text-ink/60">Loading local bundle…</div>

  // Files tab renders one of two browsers; the sub-nav controls which.
  const filesContent =
    filesView === 'generated' ? (
      <Suspense fallback={<div className="p-12 text-ink/60">Loading…</div>}>
        <GeneratedFilesBrowser />
      </Suspense>
    ) : (
      <Suspense fallback={<div className="p-12 text-ink/60">Loading…</div>}>
        <FolderClaudeMdsBrowser />
      </Suspense>
    )

  const localViewProp =
    tab === 'report'
      ? {
          tab,
          setTab,
          title: 'Blueprint',
        }
      : {
          tab,
          setTab,
          title: 'Files',
          subNav: [
            {
              id: 'folders',
              label: 'Folder Context',
              icon: Database,
              active: filesView === 'folders',
              onClick: () => setFilesView('folders'),
            },
            {
              id: 'generated',
              label: 'Generated Files',
              icon: FileText,
              active: filesView === 'generated',
              onClick: () => setFilesView('generated'),
            },
          ],
        }

  return (
    <LocalEditContext.Provider value={ctx}>
      <ReportPage
        bundle={bundle}
        localView={localViewProp}
        mainContent={tab === 'files' ? filesContent : null}
      />
      <Toast message={toast} onDismiss={() => setToast(null)} />
    </LocalEditContext.Provider>
  )
}
