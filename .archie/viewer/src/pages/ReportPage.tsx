
import { useEffect, useState, useRef, useMemo, useContext } from 'react'
import { useParams, Link } from 'react-router-dom'
import { LocalEditContext } from '@/components/local/context/LocalEditContext'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import { Copy, Check, ExternalLink, ChevronRight, Layout, Github, Menu, X, Info, Activity, Database, Shield, Zap, Rocket, AlertTriangle, Layers, FileText } from 'lucide-react'
import { fetchReport, type Bundle } from '@/lib/api'
import { autoBacktick } from '@/lib/autocode'
import { formatBlueprintTitle } from '@/lib/blueprintTitle'
import { cn } from '@/lib/utils'
import { theme } from '@/lib/theme'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { MermaidDiagram } from '@/components/MermaidDiagram'
import { GhostLogo } from '@/components/GhostLogo'
import * as Sections from '@/components/ReportSections'
import { countSemanticDuplications, extractFindings, normalizeStructuredFinding, rankFindings, scanReportAssertsZeroSemanticDup } from '@/lib/findings'

const INSTALL_CMD = 'npx @bitraptors/archie /path/to/your/project'

export type LocalTab = 'report' | 'files'

interface LocalSubNavItem {
  id: string
  label: string
  icon: any
  active: boolean
  onClick: () => void
}

interface ReportPageProps {
  bundle?: Bundle
  createdAt?: string
  // Local-viewer-only: when provided, ReportPage renders a segmented VIEW
  // toggle at the top of the sidebar that switches between Blueprint and
  // Files modes. Share mode (no localView) keeps the original sidebar
  // unchanged.
  localView?: {
    tab: LocalTab
    setTab: (t: LocalTab) => void
    // Title shown below the segmented control, above the nav items.
    title?: string
    // When set, replaces the blueprint sidebar sections with these items
    // (used by the Files tab to switch between Folder Context / Generated
    // Files in the same sidebar pattern the blueprint sections use).
    subNav?: LocalSubNavItem[]
  }
  // When set (Files mode), replaces the blueprint sections in the main
  // content area. The outer chrome (sidebar margin, container, padding) is
  // preserved.
  mainContent?: React.ReactNode
}

export default function ReportPage({ bundle: bundleProp, createdAt: createdAtProp, localView, mainContent }: ReportPageProps = {}) {
  const { token } = useParams<{ token: string }>()
  const [bundle, setBundle] = useState<Bundle | null>(bundleProp ?? null)
  const [createdAt, setCreatedAt] = useState<string | null>(createdAtProp ?? null)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [activeSection, setActiveSection] = useState('')
  const [isSidebarOpen, setIsSidebarOpen] = useState(false)
  const [ignoredRules, setIgnoredRules] = useState<any[]>([])

  // LocalEditContext is null in share mode and a real ctx in local-viewer mode.
  // We only fetch /api/ignored-rules when localCtx is non-null so share-mode
  // browsers never hit a 404 against archie-viewer.vercel.app.
  const localCtx = useContext(LocalEditContext)

  const contentRef = useRef<HTMLDivElement>(null)
  const scrollingToRef = useRef(false)

  useEffect(() => {
    if (bundleProp) return  // local mode: bundle already provided, skip fetch
    if (!token) return
    fetchReport(token)
      .then((r) => {
        setBundle(r.bundle)
        setCreatedAt(r.created_at)
      })
      .catch((e) => setError(e.message))
  }, [token, bundleProp])

  // Local mode only: when LocalPage refetches /api/bundle after a mutation,
  // it passes the NEW bundle as bundleProp. useState's initial value only
  // applies on first mount, so without this sync the internal `bundle` would
  // stay frozen at the value captured on the first render — making the UI
  // appear stuck on stale rule states. Share mode (bundleProp === undefined)
  // is untouched.
  useEffect(() => {
    if (bundleProp !== undefined) {
      setBundle(bundleProp)
    }
    if (createdAtProp !== undefined) {
      setCreatedAt(createdAtProp)
    }
  }, [bundleProp, createdAtProp])

  // Refetch ignored rules whenever the parent bundle refreshes (LocalPage
  // recreates ctx on every render after a mutation succeeds OR after a stale-
  // state recovery, so keying off bundleProp identity is enough).
  useEffect(() => {
    if (!localCtx) return
    fetch('/api/ignored-rules', { cache: 'no-store' })
      .then((r) => (r.ok ? r.json() : { rules: [] }))
      .then((j) => setIgnoredRules(Array.isArray(j?.rules) ? j.rules : []))
      .catch(() => setIgnoredRules([]))
  }, [localCtx, bundleProp])

  const bp = bundle?.blueprint || {}
  const meta = bp.meta || {}
  const diagram: string = typeof bp.architecture_diagram === 'string' ? bp.architecture_diagram : bp.architecture_diagram?.mermaid || ''
  
  const findings = useMemo(() => {
    // Prefer the structured shared store when present — gives us the 4-field
    // shape (evidence/root_cause/fix_direction). Fall back to parsing the
    // markdown scan_report for older bundles uploaded before findings.json
    // was included.
    if (Array.isArray(bundle?.findings) && bundle!.findings!.length > 0) {
      // Filter to status === "active" only. Verifier-routed entries
      // (status: "demoted" — risk class with no current call-site instance,
      //  status: "dropped" — premise unsound for this codebase) are
      // intentionally hidden from the user-facing Architectural Problems
      // list. status: "resolved" stays hidden too. Older bundles without
      // a status default to "active" so they render unchanged.
      const active = bundle!.findings!.filter((f: any) => (f?.status || 'active') === 'active')
      return rankFindings(active.map(normalizeStructuredFinding))
    }
    if (!bundle?.scan_report) return []
    return rankFindings(extractFindings(bundle.scan_report))
  }, [bundle?.findings, bundle?.scan_report])

  // Semantic duplication — prefer structured field, else honour an explicit
  // zero verdict in the scan report, else fall back to a heuristic regex count
  // over the markdown findings. Older bundles that predate
  // `.archie/semantic_duplications.json` still render a trustworthy number
  // when the AI reported zero via prose ("No semantic duplication detected").
  const semantic = useMemo<{ count: number | null; source: 'structured' | 'heuristic' | 'unknown' }>(() => {
    if (Array.isArray(bundle?.semantic_duplications)) {
      return { count: bundle!.semantic_duplications!.length, source: 'structured' }
    }
    if (bundle?.scan_report && scanReportAssertsZeroSemanticDup(bundle.scan_report)) {
      return { count: 0, source: 'structured' }
    }
    if (bundle?.scan_report && findings.length > 0) {
      return { count: countSemanticDuplications(findings), source: 'heuristic' }
    }
    return { count: null, source: 'unknown' }
  }, [bundle, findings])

  // Scroll sync logic — re-attach after bundle loads so contentRef.current exists
  useEffect(() => {
    if (!bundle) return
    const container = contentRef.current
    if (!container) return

    // Collect top-level tracked IDs: sections directly in content, and the
    // nested `#pitfalls` div inside the Problems section plus `#try-archie`
    // footer. Rebuilt once per load to avoid querySelector churn on scroll.
    const TRACKED_IDS = [
      'summary',
      'health',
      'diagram',
      'workspace-topology',
      'archrules',
      'enforcement-rules',
      'devrules',
      'infrarules',
      'decisions',
      'tradeoffs',
      'guidelines',
      'communications',
      'components',
      'integrations',
      'technology',
      'deployment',
      'problems',
      'pitfalls',
      'try-archie',
    ]

    const handleScroll = () => {
      if (scrollingToRef.current) return
      const offset = 150
      let current = ''
      for (const id of TRACKED_IDS) {
        const el = document.getElementById(id)
        if (!el) continue
        if (el.getBoundingClientRect().top <= offset) current = id
      }
      if (current && current !== activeSection) setActiveSection(current)
    }

    handleScroll()
    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [bundle])

  const scrollToSection = (id: string) => {
    const element = document.getElementById(id)
    if (!element) return
    
    scrollingToRef.current = true
    setActiveSection(id)
    setIsSidebarOpen(false)

    const offset = 100
    const bodyRect = document.body.getBoundingClientRect().top
    const elementRect = element.getBoundingClientRect().top
    const elementPosition = elementRect - bodyRect
    const offsetPosition = elementPosition - offset

    window.scrollTo({
      top: offsetPosition,
      behavior: 'smooth'
    })

    setTimeout(() => {
      scrollingToRef.current = false
    }, 800)
  }

  const copyInstall = async () => {
    await navigator.clipboard.writeText(INSTALL_CMD)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center p-8 bg-gradient-to-br from-papaya-50 to-teal-50/20">
        <Card className="max-w-md border-brandy/20 shadow-2xl shadow-brandy/5 rounded-3xl overflow-hidden">
          <CardHeader className="bg-brandy/5 border-b border-brandy/10 p-8">
            <div className="w-12 h-12 rounded-2xl bg-brandy/10 flex items-center justify-center mb-4">
               <AlertTriangle className="text-brandy w-6 h-6" />
            </div>
            <CardTitle className="text-ink decoration-brandy underline-offset-4 decoration-2">Report Expired or Invalid</CardTitle>
          </CardHeader>
          <CardContent className="p-8">
            <p className="text-ink/60 mb-8 leading-relaxed">
              We couldn't find the blueprint you're looking for. It may have been deleted or the link might be broken.
            </p>
            <Link to="/" className="inline-flex items-center gap-2 font-bold text-teal hover:text-teal-700 transition-colors group">
              <ChevronRight className="w-4 h-4 rotate-180 group-hover:-translate-x-1 transition-transform" />
              Return to Archie
            </Link>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!bundle) {
    return (
      <div className="min-h-screen bg-white">
        <div className="fixed inset-y-0 left-0 w-64 border-r border-papaya-300 hidden lg:block p-8 space-y-6">
           <Skeleton className="h-8 w-32 rounded-lg" />
           <div className="space-y-4 pt-8">
             <Skeleton className="h-4 w-full" />
             <Skeleton className="h-4 w-4/5" />
             <Skeleton className="h-4 w-full" />
             <Skeleton className="h-4 w-3/4" />
           </div>
        </div>
        <div className="lg:ml-64 p-8 md:p-12 lg:p-20 space-y-12 max-w-5xl">
          <header className="space-y-4">
             <Skeleton className="h-4 w-24" />
             <Skeleton className="h-12 w-3/4" />
             <Skeleton className="h-6 w-1/2" />
          </header>
          <Skeleton className="h-[400px] w-full rounded-3xl" />
          <div className="grid grid-cols-3 gap-6">
             <Skeleton className="h-32 rounded-2xl" />
             <Skeleton className="h-32 rounded-2xl" />
             <Skeleton className="h-32 rounded-2xl" />
          </div>
        </div>
      </div>
    )
  }

  const componentsList = bp.components?.components || []
  const keyDecisions = bp.decisions?.key_decisions || []
  const tradeOffs = bp.decisions?.trade_offs || []
  const pitfalls = Array.isArray(bp.pitfalls) ? bp.pitfalls : []
  const archRules = bp.architecture_rules || {}
  const filePlacement = archRules.file_placement_rules || []
  const naming = archRules.naming_conventions || []
  const technology = bp.technology || {}
  const stack = Array.isArray(technology.stack) ? technology.stack : []
  const runCommands = technology.run_commands || {}
  const deployment = bp.deployment || {}
  const implementationGuidelines = [
    ...(bp.implementation_guidelines || []),
    ...(bp.decisions?.implementation_guidelines || []),
    ...(bp.guidelines || []),
    ...(archRules.guidelines || [])
  ]
  const developmentRules = [
    ...(archRules.development_rules || []),
    ...(bp.development_rules || [])
  ]
  // Infrastructure rules — split out from development_rules in the new
  // schema. CI, signing, distribution, secrets, dependency-registry auth.
  // Old shares without infrastructure_rules render unchanged (empty array).
  const infrastructureRules = [
    ...(archRules.infrastructure_rules || []),
    ...(bp.infrastructure_rules || [])
  ]
  // Phase 1 enforcement rules — pulled from rules.json / proposed_rules.json
  // via upload.py's bundle.rules_adopted / .rules_proposed. Both shapes
  // accepted: `{ rules: [...] }` (current) or raw `[...]` (defensive).
  const _rulesArray = (raw: any): any[] => {
    if (!raw) return []
    if (Array.isArray(raw)) return raw
    if (typeof raw === 'object' && Array.isArray(raw.rules)) return raw.rules
    return []
  }
  const enforcementRules = _rulesArray(bundle?.rules_adopted)
  const proposedEnforcementRules = _rulesArray(bundle?.rules_proposed)
  // Blueprint exposes `communication` (singular) as an object with
  // `patterns[]` and `integrations[]`. Patterns flow into the Communications
  // section; integrations get their own inventory section so third-party
  // service wiring is easy to scan.
  const commObj = bp.communication || {}
  const communications = [
    ...(bp.communications || []),
    ...(archRules.communications || []),
    ...((commObj.patterns || []).map((p: any) => ({
      type: p.name || 'Pattern',
      protocol: 'pattern',
      // Legacy shape: some old shares stored a flat `description`. New
      // patterns split it into `how_it_works` (the body) and `when_to_use`
      // (the trigger). Keep description as a fallback for legacy bundles;
      // the section renders how_it_works / when_to_use as their own blocks.
      description: p.description || '',
      how_it_works: p.how_it_works,
      when_to_use: p.when_to_use,
      // Carry the precondition fields through so the Communications card
      // can surface them. Wave 2 grounds these in code citations and
      // they're the load-bearing signal for misapplication detection at
      // edit time — dropping them would silently strip the contract.
      applicable_when: p.applicable_when,
      do_not_apply_when: p.do_not_apply_when,
      scope: p.scope,
    }))),
  ]
  // Integrations: `{service, purpose, integration_point}` in the blueprint.
  // Accept older/alternate shapes too (`name`/`type`/`description`).
  const integrations = (commObj.integrations || []).map((i: any) => ({
    service: i.service || i.name || 'Integration',
    purpose: i.purpose || i.description || '',
    integration_point: i.integration_point || i.file || '',
    type: i.type || '',
  }))

  return (
    <div className="min-h-screen bg-gradient-to-br from-papaya-50 via-white to-teal-50/10 text-ink scroll-smooth">
      {/* Mobile Header */}
      <header className="lg:hidden sticky top-0 z-40 bg-white/80 backdrop-blur-xl border-b border-papaya-300 px-6 py-4 flex items-center justify-between">
        <Link to="/" className="flex items-center gap-2">
          <GhostLogo size={32} className="shrink-0" />
          <span className="font-black tracking-tight text-xl">Archie</span>
        </Link>
        <button onClick={() => setIsSidebarOpen(true)} className="p-2 -mr-2">
          <Menu className="w-6 h-6" />
        </button>
      </header>

      {/* Sidebar Navigation */}
      <aside 
        className={cn(
          "fixed inset-y-0 left-0 z-50 w-72 bg-white/50 backdrop-blur-2xl border-r border-papaya-300 transition-transform duration-300 lg:translate-x-0 overflow-hidden flex flex-col",
          isSidebarOpen ? "translate-x-0" : "-translate-x-full"
        )}
      >
        <div className="p-8 flex items-center justify-between shrink-0">
          <Link to="/" className="flex items-center gap-3">
            <GhostLogo size={40} className="shrink-0 drop-shadow-lg" />
            <div>
              <span className="font-black tracking-tight text-2xl block leading-none">Archie</span>
              <span className="text-[10px] font-black uppercase tracking-[0.2em] text-teal/40 mt-1 block">Blueprint Viewer</span>
            </div>
          </Link>
          <button onClick={() => setIsSidebarOpen(false)} className="lg:hidden p-2 -mr-2 text-ink/40 hover:text-ink">
            <X className="w-6 h-6" />
          </button>
        </div>

        <nav className="flex-1 overflow-y-auto p-6 space-y-8 custom-scrollbar">
          {/* VIEW — local-viewer horizontal segmented toggle. Two tabs:
              Blueprint (the existing scrollable report) and Files (folder
              CLAUDE.mds + generated files, switched via subNav below). */}
          {localView && (
            <div className="mb-6 mt-2 px-3">
              <div className="bg-ink/[0.03] p-1.5 rounded-2xl flex items-center justify-between border border-ink/5 shadow-inner">
                {[
                  { id: 'report' as LocalTab, label: 'Blueprint', icon: Layout },
                  { id: 'files' as LocalTab, label: 'Files', icon: FileText },
                ].map((tab) => {
                  const isActive = localView.tab === tab.id
                  const Icon = tab.icon
                  return (
                    <button
                      key={tab.id}
                      onClick={() => { localView.setTab(tab.id); setIsSidebarOpen(false) }}
                      title={tab.label}
                      className={cn(
                        "relative flex items-center justify-center w-full py-2.5 rounded-xl transition-all duration-500",
                        isActive
                          ? "bg-white shadow-lg shadow-ink/5 border border-papaya-300/40 text-teal z-10"
                          : "text-ink/20 hover:text-ink/40 hover:bg-white/40"
                      )}
                    >
                      <Icon className={cn("w-5 h-5 transition-transform duration-500", isActive && "scale-110")} />
                      {isActive && (
                        <div className="absolute -bottom-1 left-1/2 -translate-x-1/2 w-1.5 h-1.5 rounded-full bg-teal/20 blur-[2px]" />
                      )}
                    </button>
                  )
                })}
              </div>
            </div>
          )}

          {/* Tab title — sits between the segmented control and the nav items.
              Matches the active tab label so the icon-only toggle still gives
              clear context. */}
          {localView?.title && (
            <div className="px-3 mb-6">
              <h2 className="text-xl font-black tracking-tight text-ink">{localView.title}</h2>
            </div>
          )}

          {/* Files-tab sub-nav — when subNav is provided, render these in
              place of the blueprint sections. Same NavButton + section-label
              styling so it reads as native sidebar content. */}
          {localView?.subNav && localView.subNav.length > 0 && (
            <div className="space-y-1">
              <p className="px-3 text-[10px] font-black uppercase tracking-[0.2em] text-ink/20 mb-4">Browse</p>
              {localView.subNav.map((item) => (
                <NavButton
                  key={item.id}
                  active={item.active}
                  onClick={() => { item.onClick(); setIsSidebarOpen(false) }}
                  icon={item.icon}
                  label={item.label}
                />
              ))}
            </div>
          )}

          {/* Blueprint-specific sections — hidden when localView.tab === 'files'
              (the scroll targets don't exist on the Files screen). */}
          {(!localView || localView.tab === 'report') && (
          <>
          {/* Overview */}
          <div className="space-y-1">
            <p className="px-3 text-[10px] font-black uppercase tracking-[0.2em] text-ink/20 mb-4">Overview</p>
            <NavButton
              active={activeSection === 'summary'}
              onClick={() => scrollToSection('summary')}
              icon={Info}
              label="Executive Summary"
            />
            {bundle.health && (
              <NavButton
                active={activeSection === 'health'}
                onClick={() => scrollToSection('health')}
                icon={Activity}
                label="System Health"
              />
            )}
            {diagram && (
              <NavButton
                active={activeSection === 'diagram'}
                onClick={() => scrollToSection('diagram')}
                icon={Layout}
                label="Architecture Diagram"
              />
            )}
            {bp.workspace_topology && (
              <NavButton
                active={activeSection === 'workspace-topology'}
                onClick={() => scrollToSection('workspace-topology')}
                icon={Database}
                label="Workspace Topology"
              />
            )}
          </div>

          {/* Design */}
          {(keyDecisions.length > 0 || tradeOffs.length > 0) && (
            <div className="space-y-1">
              <p className="px-3 text-[10px] font-black uppercase tracking-[0.2em] text-ink/20 mb-4">Design</p>
              {keyDecisions.length > 0 && (
                <NavButton
                  active={activeSection === 'decisions'}
                  onClick={() => scrollToSection('decisions')}
                  icon={Shield}
                  label="Key Decisions"
                />
              )}
              {tradeOffs.length > 0 && (
                <NavButton
                  active={activeSection === 'tradeoffs'}
                  onClick={() => scrollToSection('tradeoffs')}
                  icon={Activity}
                  label="Trade-offs"
                />
              )}
            </div>
          )}

          {/* Practice */}
          {(implementationGuidelines.length > 0 || communications.length > 0) && (
            <div className="space-y-1">
              <p className="px-3 text-[10px] font-black uppercase tracking-[0.2em] text-ink/20 mb-4">Practice</p>
              {implementationGuidelines.length > 0 && (
                <NavButton
                  active={activeSection === 'guidelines'}
                  onClick={() => scrollToSection('guidelines')}
                  icon={Info}
                  label="Implementation Guidelines"
                />
              )}
              {communications.length > 0 && (
                <NavButton
                  active={activeSection === 'communications'}
                  onClick={() => scrollToSection('communications')}
                  icon={Activity}
                  label="Communications"
                />
              )}
            </div>
          )}

          {/* Inventory */}
          {(componentsList.length > 0 || stack.length > 0 || integrations.length > 0) && (
            <div className="space-y-1">
              <p className="px-3 text-[10px] font-black uppercase tracking-[0.2em] text-ink/20 mb-4">Inventory</p>
              {componentsList.length > 0 && (
                <NavButton
                  active={activeSection === 'components'}
                  onClick={() => scrollToSection('components')}
                  icon={Database}
                  label="Components"
                />
              )}
              {integrations.length > 0 && (
                <NavButton
                  active={activeSection === 'integrations'}
                  onClick={() => scrollToSection('integrations')}
                  icon={Zap}
                  label="Integrations"
                />
              )}
              {stack.length > 0 && (
                <NavButton
                  active={activeSection === 'technology'}
                  onClick={() => scrollToSection('technology')}
                  icon={Layers}
                  label="Technology Stack"
                />
              )}
              {(deployment.strategy || deployment.platform || (Array.isArray(deployment.infrastructure) && deployment.infrastructure.length > 0)) && (
                <NavButton
                  active={activeSection === 'deployment'}
                  onClick={() => scrollToSection('deployment')}
                  icon={Rocket}
                  label="Deployment"
                />
              )}
            </div>
          )}

          {/* Risks — merged Findings + Pitfalls */}
          {(findings.length > 0 || pitfalls.length > 0) && (
            <div className="space-y-1">
              <p className="px-3 text-[10px] font-black uppercase tracking-[0.2em] text-ink/20 mb-4">Risks</p>
              <NavButton
                active={activeSection === 'problems'}
                onClick={() => scrollToSection('problems')}
                icon={AlertTriangle}
                label="Architectural Problems"
              />
              {pitfalls.length > 0 && (
                <NavButton
                  active={activeSection === 'pitfalls'}
                  onClick={() => scrollToSection('pitfalls')}
                  icon={Shield}
                  label="Pitfalls"
                />
              )}
            </div>
          )}

          {/* Rules — ONE unified entry in both modes. Backward-compat: when an
              old (pre-3.0) share has no unified rules.json data, fall back to
              scrolling to whichever legacy section (archrules / devrules /
              infrarules) is rendering. The legacy body sections themselves
              stay gated on !localCtx below. */}
          {(() => {
            const hasUnified = enforcementRules.length > 0 || proposedEnforcementRules.length > 0 || ignoredRules.length > 0;
            const hasArch = filePlacement.length > 0 || naming.length > 0;
            const hasDev = developmentRules.length > 0;
            const hasInfra = infrastructureRules.length > 0;
            if (!(hasUnified || hasArch || hasDev || hasInfra)) return null;
            const rulesScrollTarget = hasUnified
              ? 'enforcement-rules'
              : hasArch
                ? 'archrules'
                : hasDev
                  ? 'devrules'
                  : 'infrarules';
            const rulesActive =
              activeSection === 'enforcement-rules' ||
              activeSection === 'archrules' ||
              activeSection === 'devrules' ||
              activeSection === 'infrarules';
            return (
              <div className="space-y-1">
                <p className="px-3 text-[10px] font-black uppercase tracking-[0.2em] text-ink/20 mb-4">Rules</p>
                <NavButton
                  active={rulesActive}
                  onClick={() => scrollToSection(rulesScrollTarget)}
                  icon={Shield}
                  label="Enforcement Rules"
                />
              </div>
            );
          })()}

          {/* Get started — share mode only. Local mode hides the "Try Archie"
              promo because the user is already running Archie. */}
          {!localCtx && (
            <div className="space-y-1">
              <p className="px-3 text-[10px] font-black uppercase tracking-[0.2em] text-ink/20 mb-4">Get Started</p>
              <NavButton
                active={activeSection === 'try-archie'}
                onClick={() => scrollToSection('try-archie')}
                icon={Rocket}
                label="Try Archie"
              />
            </div>
          )}
          </>
          )}
        </nav>

        <div className="p-8 bg-papaya-300/10 border-t border-papaya-300/40">
           <a 
             href="https://github.com/BitRaptors/Archie" 
             target="_blank" 
             className="flex items-center justify-between text-ink/40 hover:text-ink transition-colors group"
           >
             <span className="text-xs font-bold uppercase tracking-widest">Open Source</span>
             <Github className="w-4 h-4 group-hover:rotate-12 transition-transform" />
           </a>
        </div>
      </aside>

      {/* Main Content */}
      <main className="lg:ml-72 flex flex-col min-h-screen">
        {mainContent ? (
          // Files mode — flush against the Archie sidebar (no max-width
          // centering, lean horizontal padding) so the tree doesn't drift
          // away into whitespace on wide screens. Vertical padding stays
          // generous to keep the markdown card from kissing the top edge.
          <div className="flex-1 w-full py-6 lg:py-8 px-4 lg:px-6">
            {mainContent}
          </div>
        ) : (
        <div className="flex-1 p-6 md:p-12 lg:p-20 xl:p-24 space-y-32 max-w-6xl w-full mx-auto" ref={contentRef}>

          {/* Hero Section */}
          <section id="summary" className="space-y-8 animate-in fade-in slide-in-from-bottom-8 duration-700">
            <div className="space-y-4">
              <div className="flex items-center gap-3 text-ink/30 font-black uppercase tracking-[0.3em] text-[10px]">
                <span className="w-8 h-px bg-current" />
                <span>Blueprint Analysis</span>
                {createdAt && (
                  <span className="ml-auto opacity-60 font-mono tracking-tighter normal-case text-[11px]">
                    {new Date(createdAt).toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' })}
                  </span>
                )}
              </div>
              <h1 className="text-5xl md:text-6xl lg:text-7xl font-black tracking-tight leading-[0.95] text-ink">
                {formatBlueprintTitle({ ...meta, executiveSummary: meta.executive_summary })}
              </h1>
              <div className="flex flex-wrap gap-2 pt-2">
                {Array.isArray(meta.platforms) && meta.platforms.map((p: string) => (
                  <Badge key={p} className="bg-white/80 backdrop-blur-sm border-papaya-400 text-ink/60 px-4 py-1.5 rounded-full text-xs font-bold uppercase tracking-widest shadow-sm">
                    {p}
                  </Badge>
                ))}
              </div>
            </div>

            {meta.executive_summary && (
              <div className="relative group">
                <div className="absolute -inset-4 bg-teal/5 rounded-3xl opacity-0 group-hover:opacity-100 transition-opacity duration-500" />
                <div className="relative prose prose-lg max-w-none text-ink/70 leading-relaxed prose-strong:text-ink prose-strong:font-black prose-p:mb-6 first-letter:text-5xl first-letter:font-black first-letter:mr-3 first-letter:float-left first-letter:text-teal font-serif">
                   <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]}>{autoBacktick(meta.executive_summary)}</ReactMarkdown>
                </div>
              </div>
            )}

          </section>

             {/* Health Section */}
          {bundle.health && (
            <section id="health" className="space-y-8 scroll-mt-24">
              <Sections.SectionHeader title="System Health" icon={Activity} />
              <div className={cn("p-10 rounded-3xl border overflow-hidden relative group", theme.surface.panel)}>
                <div className="absolute top-0 right-0 p-8 opacity-[0.03] group-hover:opacity-[0.08] transition-opacity pointer-events-none">
                   <Activity className="w-64 h-64 -mr-20 -mt-20" />
                </div>
                <div className="grid lg:grid-cols-2 gap-16 relative">
                  <div className="space-y-8">
                    <Sections.HealthBar label="Architectural Erosion" value={Math.round((bundle.health.erosion || 0) * 100)} inverted
                      direction="lower" target="<30%"
                      hint="Share of total complexity mass (cc × √sloc) held by functions with CC > 10. High means a few complex functions dominate — hard to test, hard to change." />
                    <Sections.HealthBar label="Logic Concentration (Gini)" value={Math.round((bundle.health.gini || 0) * 100)} inverted
                      direction="lower" target="<40%"
                      hint="How unevenly complexity is spread across functions. 0 = perfectly even, 1 = one function holds everything. High means a few god-functions dominate the codebase." />
                    <Sections.DuplicationCard
                      verbosity={bundle.health.verbosity || 0}
                      totalLoc={bundle.health.total_loc}
                      duplicateLines={bundle.health.duplicate_lines}
                      semanticCount={semantic.count}
                      semanticSource={semantic.source}
                      detailsHref="#problems"
                    />
                  </div>
                  <div className="grid grid-cols-2 gap-8 content-start">
                    <Sections.Stat label="Total LOC" value={bundle.health.total_loc?.toLocaleString() ?? '—'} />
                    <Sections.Stat label="Functions" value={bundle.health.total_functions ?? '—'} />
                    <Sections.Stat label="High Complexity" value={bundle.health.high_cc_functions ?? '—'} />
                    <Sections.Stat label="Duplicate Lines" value={bundle.health.duplicate_lines ?? '—'} />
                  </div>
                </div>
              </div>

              {/* CC distribution + mass concentration + top-N — the 'why is erosion 86%' story */}
              {(bundle.health.cc_distribution || bundle.health.mass || (bundle.health.top_high_cc || []).length > 0) && (
                <div className="grid lg:grid-cols-2 gap-8">
                  {bundle.health.cc_distribution && (
                    <div className={cn("p-8 rounded-3xl border space-y-4", theme.surface.panel)}>
                      <Sections.CCDistribution distribution={bundle.health.cc_distribution} />
                    </div>
                  )}
                  {bundle.health.mass && (
                    <div className={cn("p-8 rounded-3xl border", theme.surface.panel)}>
                      <Sections.MassConcentration
                        mass={bundle.health.mass}
                        totalFunctions={bundle.health.total_functions}
                        highCcFunctions={bundle.health.high_cc_functions}
                        distribution={bundle.health.cc_distribution}
                      />
                    </div>
                  )}
                </div>
              )}

              {Array.isArray(bundle.health.top_high_cc) && bundle.health.top_high_cc.length > 0 && (
                <div className={cn("p-8 rounded-3xl border", theme.surface.panel)}>
                  <Sections.TopHighCCList
                    items={bundle.health.top_high_cc}
                    totalMass={bundle.health.mass?.total}
                  />
                </div>
              )}
            </section>
          )}

          {/* Architecture Diagram */}
          {diagram && (
            <section id="diagram" className="space-y-8 scroll-mt-24">
              <Sections.SectionHeader title="Architecture Diagram" icon={Layout} />
              <div className={cn("p-10 rounded-3xl border shadow-2xl shadow-ink/5 bg-white/50 backdrop-blur-md overflow-hidden relative", theme.surface.panel)}>
                <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,rgba(45,161,176,0.03),transparent)] pointer-events-none" />
                <div className="relative">
                  <MermaidDiagram chart={diagram} />
                </div>
                <details className="mt-12 group overflow-hidden">
                  <summary className="list-none cursor-pointer inline-flex items-center gap-2 px-4 py-2 bg-ink/5 rounded-xl text-[10px] font-black uppercase tracking-widest text-ink/40 hover:text-ink hover:bg-ink/10 transition-all">
                    <Database className="w-3.5 h-3.5" />
                    <span>Scale Logic (Mermaid Source)</span>
                  </summary>
                  <div className="mt-4 p-8 rounded-2xl font-mono text-xs overflow-x-auto ring-1 ring-white/10 shadow-inner bg-ink text-papaya-300">
                    <pre>{diagram}</pre>
                  </div>
                </details>
              </div>
            </section>
          )}

          {/* 3b. Workspace Topology (monorepo whole-mode blueprints only) */}
          {bp.workspace_topology && (
            <section id="workspace-topology" className="scroll-mt-24">
              <Sections.WorkspaceTopologySection topology={bp.workspace_topology} />
            </section>
          )}

          {/* 6. Key Decisions */}
          {keyDecisions.length > 0 && (
            <section id="decisions" className="scroll-mt-24">
              <Sections.KeyDecisionsSection decisions={keyDecisions} />
            </section>
          )}

          {/* 7. Trade-offs */}
          {tradeOffs.length > 0 && (
            <section id="tradeoffs" className="scroll-mt-24">
              <Sections.TradeOffsSection tradeoffs={tradeOffs} />
            </section>
          )}

          {/* 8. Implementation Guidelines */}
          {implementationGuidelines.length > 0 && (
            <section id="guidelines" className="scroll-mt-24">
              <Sections.ImplementationGuidelinesSection items={implementationGuidelines} />
            </section>
          )}

          {/* 9. Communications */}
          {communications.length > 0 && (
            <section id="communications" className="scroll-mt-24">
              <Sections.CommunicationsSection communications={communications} />
            </section>
          )}

          {/* 10. Components */}
          {componentsList.length > 0 && (
            <section id="components" className="scroll-mt-24">
              <Sections.ComponentsSection components={componentsList} />
            </section>
          )}

          {/* 10b. Integrations — third-party services wired into the app */}
          {integrations.length > 0 && (
            <section id="integrations" className="scroll-mt-24">
              <Sections.IntegrationsSection integrations={integrations} />
            </section>
          )}

          {/* 11. Technology Stack */}
          {stack.length > 0 && (
            <section id="technology" className="scroll-mt-24">
              <Sections.TechnologySection stack={stack} runCommands={runCommands} />
            </section>
          )}

          {/* Deployment (kept — not in user's spec but still useful if present) */}
          {Object.keys(deployment).length > 0 && (deployment.strategy || deployment.platform || (Array.isArray(deployment.infrastructure) && deployment.infrastructure.length > 0)) && (
            <section id="deployment" className="scroll-mt-24">
              <Sections.DeploymentSection deployment={deployment} />
            </section>
          )}

          {/* 12. Architectural Problems + Pitfalls — merged, end of page */}
          {(findings.length > 0 || pitfalls.length > 0) && (
            <section id="problems" className="space-y-12 scroll-mt-24">
              <Sections.SectionHeader
                title="Architectural Problems"
                icon={AlertTriangle}
                hint="Concrete problems observed in specific files."
              />

              {findings.length > 0 && (
                <Sections.FindingsList
                  findings={findings}
                  semanticFunctionNames={
                    Array.isArray(bundle.semantic_duplications)
                      ? bundle.semantic_duplications
                          .map((d: any) => d.function)
                          .filter((x: any): x is string => typeof x === 'string')
                      : undefined
                  }
                  semanticDuplications={
                    Array.isArray(bundle.semantic_duplications)
                      ? bundle.semantic_duplications
                      : undefined
                  }
                  blueprint={bp}
                  adoptedRules={bundle?.rules_adopted}
                />
              )}

              {pitfalls.length > 0 && (
                <div id="pitfalls" className="scroll-mt-24">
                  <Sections.PitfallsSection
                    pitfalls={pitfalls}
                    blueprint={bp}
                    adoptedRules={bundle?.rules_adopted}
                    semanticDuplications={
                      Array.isArray(bundle.semantic_duplications)
                        ? bundle.semantic_duplications
                        : undefined
                    }
                  />
                </div>
              )}
            </section>
          )}

          {/* Rules — moved to render directly after Pitfalls so the body order
              mirrors the sidebar (Risks → Rules). Order within the block stays
              archrules → enforcement-rules → devrules → infrarules. The three
              legacy sections remain gated on !localCtx so pre-3.0 shared
              blueprints still render their data; the unified RulesSection is
              the canonical surface for modern blueprints in both modes. */}
          {!localCtx && (filePlacement.length > 0 || naming.length > 0) && (
            <section id="archrules" className="scroll-mt-24">
              <Sections.ArchRulesSection filePlacement={filePlacement} naming={naming} />
            </section>
          )}

          {(enforcementRules.length > 0 || proposedEnforcementRules.length > 0 || ignoredRules.length > 0) && (
            <section id="enforcement-rules" className="scroll-mt-24">
              <Sections.RulesSection adopted={enforcementRules} proposed={proposedEnforcementRules} ignored={ignoredRules} />
            </section>
          )}

          {!localCtx && developmentRules.length > 0 && (
            <section id="devrules" className="scroll-mt-24">
              <Sections.DevelopmentRulesSection rules={developmentRules} />
            </section>
          )}

          {!localCtx && infrastructureRules.length > 0 && (
            <section id="infrarules" className="scroll-mt-24">
              <Sections.InfrastructureRulesSection rules={infrastructureRules} />
            </section>
          )}

          {/* Conversion Footer — share mode only. Hiding this in local mode
              both matches the sidebar (which omits Get Started locally) and
              avoids pitching Archie to someone already running Archie. */}
          {!localCtx && (
          <footer id="try-archie" className="pt-20 pb-32 scroll-mt-24">
             <div className="relative group">
                <div className="absolute -inset-1 bg-gradient-to-r from-teal to-tangerine rounded-[40px] blur opacity-10 group-hover:opacity-20 transition-opacity duration-1000" />
                <div className="relative p-12 md:p-20 rounded-[38px] bg-white border border-papaya-400 shadow-2xl shadow-ink/5 text-center space-y-8 overflow-hidden">
                  <div className="absolute top-0 right-0 p-12 opacity-5 pointer-events-none">
                    <Activity className="w-96 h-96 -mr-48 -mt-48" />
                  </div>
                  
                  <div className="space-y-4 relative">
                    <div className="w-20 h-20 rounded-3xl bg-teal/10 flex items-center justify-center mx-auto mb-10 shadow-inner border border-teal/20">
                       <Activity className="text-teal w-10 h-10" />
                    </div>
                    <h3 className="text-4xl md:text-5xl font-black tracking-tighter text-ink leading-tight">
                      Archie knows your <br className="hidden md:block" /> codebase like a Senior Architect.
                    </h3>
                    <p className="text-xl text-ink/60 max-w-2xl mx-auto font-medium">
                      Understand complexity, enforce standards, and guide AI agents with precision.
                      Get started in 3 minutes.
                    </p>
                  </div>

                  <div className="relative pt-8 max-w-lg mx-auto">
                    <div className={cn(
                      'rounded-2xl p-6 font-mono text-sm flex items-center justify-between gap-4 shadow-2xl transition-all group/cmd',
                      theme.console.bg,
                      theme.console.text
                    )}>
                      <code className="truncate text-teal-100">{INSTALL_CMD}</code>
                      <button 
                        onClick={copyInstall} 
                        className="p-3 rounded-xl bg-white/10 hover:bg-white/20 text-white transition-all shrink-0 active:scale-90"
                        title="Copy"
                      >
                        {copied ? <Check className="w-5 h-5 text-teal" /> : <Copy className="w-5 h-5" />}
                      </button>
                    </div>
                    {copied && (
                      <div className="absolute -top-4 left-1/2 -translate-x-1/2 px-3 py-1 bg-teal text-white text-[10px] font-black uppercase tracking-widest rounded-full shadow-lg animate-in fade-in zoom-in duration-300">
                         Copied to Keyboard
                      </div>
                    )}
                  </div>

                  <div className="pt-12 flex flex-col md:flex-row items-center justify-center gap-8">
                    <a
                      href="https://github.com/BitRaptors/Archie"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-2 text-sm font-black uppercase tracking-[0.2em] text-ink/40 hover:text-teal transition-all group"
                    >
                      <span>Explore GitHub</span>
                      <ExternalLink className="w-4 h-4 group-hover:-translate-y-1 group-hover:translate-x-1 transition-transform" />
                    </a>
                    <div className="w-1 h-1 rounded-full bg-ink/10 hidden md:block" />
                    <Link
                      to="/"
                      className="inline-flex items-center gap-2 text-sm font-black uppercase tracking-[0.2em] text-ink/40 hover:text-ink transition-all"
                    >
                      Documentation
                    </Link>
                  </div>
                </div>
             </div>
          </footer>
          )}
        </div>
        )}
      </main>
    </div>
  )
}

function NavButton({ active, onClick, icon: Icon, label }: { active: boolean; onClick: () => void; icon: any; label: string }) {
  return (
    <button 
      onClick={onClick}
      className={cn(
        "flex items-center gap-4 w-full px-4 py-3 rounded-2xl text-sm font-bold transition-all duration-300 group",
        active 
          ? "bg-teal/10 text-teal shadow-inner ring-1 ring-teal/20" 
          : "text-ink/60 hover:text-ink hover:bg-papaya-300/30"
      )}
    >
      <div className={cn(
        "p-2 rounded-xl transition-all duration-500",
        active ? "bg-teal text-white shadow-lg shadow-teal/30 scale-110" : "bg-ink/5 group-hover:bg-ink/10 text-ink/30 group-hover:text-ink/60"
      )}>
        <Icon className="w-4 h-4" />
      </div>
      <span className="truncate">{label}</span>
      {active && (
        <div className="ml-auto w-1.5 h-1.5 rounded-full bg-teal animate-pulse shadow-[0_0_8px_rgba(45,161,176,0.5)]" />
      )}
    </button>
  )
}
