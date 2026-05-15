import { useEffect, useMemo, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import {
  Activity,
  AlertTriangle,
  ChevronRight,
  FileText,
  Layout,
} from 'lucide-react'
import { fetchReport, type Bundle } from '@/lib/api'
import { autoBacktick, AutoCode } from '@/lib/autocode'
import { formatBlueprintTitle } from '@/lib/blueprintTitle'
import { cn } from '@/lib/utils'
import { theme } from '@/lib/theme'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { GhostLogo } from '@/components/GhostLogo'
import * as Sections from '@/components/ReportSections'
import {
  countSemanticDuplications,
  extractFindings,
  normalizePitfall,
  normalizeStructuredFinding,
  pickTopFindings,
  rankFindings,
  scanReportAssertsZeroSemanticDup,
  type Finding,
} from '@/lib/findings'

const TOTAL_FINDINGS = 6
const MIN_ERRORS = 4

export default function CoverPage() {
  const { token } = useParams<{ token: string }>()
  const [bundle, setBundle] = useState<Bundle | null>(null)
  const [createdAt, setCreatedAt] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    fetchReport(token)
      .then((r) => {
        setBundle(r.bundle)
        setCreatedAt(r.created_at)
      })
      .catch((e) => setError(e.message))
  }, [token])

  const findings: Finding[] = useMemo(() => {
    if (!bundle) return []
    // Prefer the structured findings store (4-field shape).
    if (Array.isArray(bundle.findings) && bundle.findings.length > 0) {
      const active = bundle.findings.filter((f: any) => (f?.status || 'active') !== 'resolved')
      return rankFindings(active.map(normalizeStructuredFinding))
    }
    const fromReport = extractFindings(bundle.scan_report || '')
    if (fromReport.length > 0) return rankFindings(fromReport)
    // Last-resort fallback: synthesize from blueprint pitfalls.
    const pitfalls = Array.isArray(bundle.blueprint?.pitfalls) ? bundle.blueprint.pitfalls : []
    return rankFindings(pitfalls.map(normalizePitfall))
  }, [bundle])

  // Prefer structured semantic_duplications; fall back to heuristic over findings;
  // if no scan_report was included at all, we can't compute this at all.
  const { semanticCount, semanticSource } = useMemo<{
    semanticCount: number | null
    semanticSource: 'structured' | 'heuristic' | 'unknown'
  }>(() => {
    if (Array.isArray(bundle?.semantic_duplications)) {
      return { semanticCount: bundle!.semantic_duplications!.length, semanticSource: 'structured' }
    }
    if (bundle?.scan_report && scanReportAssertsZeroSemanticDup(bundle.scan_report)) {
      return { semanticCount: 0, semanticSource: 'structured' }
    }
    if (bundle?.scan_report && findings.length > 0) {
      return { semanticCount: countSemanticDuplications(findings), semanticSource: 'heuristic' }
    }
    return { semanticCount: null, semanticSource: 'unknown' }
  }, [bundle, findings])

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center p-8 bg-gradient-to-br from-papaya-50 to-teal-50/20">
        <Card className="max-w-md border-brandy/20 shadow-2xl shadow-brandy/5 rounded-3xl overflow-hidden">
          <CardContent className="p-8">
            <div className="w-12 h-12 rounded-2xl bg-brandy/10 flex items-center justify-center mb-4">
              <AlertTriangle className="text-brandy w-6 h-6" />
            </div>
            <h2 className="font-black text-xl text-ink mb-2">Report not found</h2>
            <p className="text-ink/60 mb-6 leading-relaxed">
              This shared blueprint may have been removed, or the URL is incorrect.
            </p>
            <Link to="/" className="inline-flex items-center gap-2 font-bold text-teal hover:text-teal-700">
              <ChevronRight className="w-4 h-4 rotate-180" />
              Return to Archie
            </Link>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!bundle) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-papaya-50 via-white to-teal-50/10 p-8 md:p-16 lg:p-24 max-w-5xl mx-auto space-y-12">
        <Skeleton className="h-4 w-32" />
        <Skeleton className="h-16 w-2/3" />
        <Skeleton className="h-32 w-full rounded-3xl" />
        <div className="grid grid-cols-3 gap-6">
          <Skeleton className="h-32 rounded-2xl" />
          <Skeleton className="h-32 rounded-2xl" />
          <Skeleton className="h-32 rounded-2xl" />
        </div>
      </div>
    )
  }

  const bp = bundle.blueprint || {}
  const meta = bp.meta || {}
  const health = bundle.health
  const scanMeta = bundle.scan_meta
  const structureType = bp.components?.structure_type
  const archStyle = bp.decisions?.architectural_style
  const topFindings = pickTopFindings(findings, TOTAL_FINDINGS, MIN_ERRORS)

  return (
    <div className="min-h-screen bg-gradient-to-br from-papaya-50 via-white to-teal-50/10 text-ink">
      {/* Top bar */}
      <header className="sticky top-0 z-40 bg-white/60 backdrop-blur-xl border-b border-papaya-300 px-6 py-4">
        <div className="max-w-6xl mx-auto flex items-center justify-between">
          <Link to="/" className="flex items-center gap-2">
            <GhostLogo size={32} className="shrink-0" />
            <span className="font-black tracking-tight text-xl">Archie</span>
          </Link>
          {createdAt && (
            <span className="text-xs text-ink/40 font-mono">
              {new Date(createdAt).toLocaleDateString(undefined, {
                year: 'numeric',
                month: 'long',
                day: 'numeric',
              })}
            </span>
          )}
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-6 md:px-12 py-16 md:py-24 space-y-24">
        {/* Overview */}
        <section className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
          <div className="space-y-4">
            <div className="flex items-center gap-3 text-ink/30 font-black uppercase tracking-[0.3em] text-[10px]">
              <span className="w-8 h-px bg-current" />
              <span>Architecture Blueprint</span>
              <Badge className="bg-tangerine/10 border-tangerine/30 text-tangerine-800 px-2 py-0.5 text-[9px] rounded-full tracking-widest">
                Preview
              </Badge>
            </div>
            <h1 className="text-5xl md:text-6xl lg:text-7xl font-black tracking-tight leading-[0.95] text-ink">
              {formatBlueprintTitle({ ...meta, executiveSummary: meta.executive_summary })}
            </h1>
            <div className="flex flex-wrap gap-2 pt-2">
              {Array.isArray(meta.platforms) &&
                meta.platforms.map((p: string) => (
                  <Badge
                    key={p}
                    className="bg-white/80 backdrop-blur-sm border-papaya-400 text-ink/60 px-4 py-1.5 rounded-full text-xs font-bold uppercase tracking-widest shadow-sm"
                  >
                    {p}
                  </Badge>
                ))}
              {structureType && (
                <Badge className="bg-teal/10 border-teal/20 text-teal px-4 py-1.5 rounded-full text-xs font-bold uppercase tracking-widest">
                  {structureType}
                </Badge>
              )}
            </div>
          </div>

          {meta.executive_summary && (
            <div className="prose prose-lg max-w-none text-ink/70 leading-relaxed prose-strong:text-ink prose-strong:font-black prose-p:mb-6 first-letter:text-5xl first-letter:font-black first-letter:mr-3 first-letter:float-left first-letter:text-teal font-serif">
              <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]}>{autoBacktick(meta.executive_summary)}</ReactMarkdown>
            </div>
          )}
        </section>

        {/* Metrics */}
        {(health || scanMeta) && (
          <section className="space-y-8">
            <Sections.SectionHeader title="Metrics" icon={Activity} />
            <div className={cn('p-10 rounded-3xl border', theme.surface.panel)}>
              <div className="grid lg:grid-cols-2 gap-12">
                {health && (
                  <div className="space-y-6">
                    <Sections.HealthBar
                      label="Architectural Erosion"
                      value={Math.round((health.erosion || 0) * 100)}
                      inverted
                      direction="lower"
                      target="<30%"
                      hint="Share of total complexity mass (cc × √sloc) held by functions with CC > 10. High means a few complex functions dominate — hard to test, hard to change."
                    />
                    <Sections.HealthBar
                      label="Logic Concentration (Gini)"
                      value={Math.round((health.gini || 0) * 100)}
                      inverted
                      direction="lower"
                      target="<40%"
                      hint="How unevenly complexity is spread across functions. 0 = perfectly even, 1 = one function holds everything. High means a few god-functions dominate the codebase."
                    />
                    <Sections.HealthBar
                      label="Top-20% Share"
                      value={Math.round((health.top20_share || 0) * 100)}
                      inverted
                      direction="lower"
                      target="<50%"
                      hint="Complexity mass held by the biggest fifth of functions. 20% = perfectly balanced; 90% means that fifth carries almost all the work."
                    />
                    <Sections.DuplicationCard
                      verbosity={health.verbosity || 0}
                      totalLoc={health.total_loc}
                      duplicateLines={health.duplicate_lines}
                      semanticCount={semanticCount}
                      semanticSource={semanticSource}
                      /* Cover stays compact — details list only appears on the detail page */
                    />
                  </div>
                )}
                <div className="grid grid-cols-2 gap-6 content-start">
                  {health && (
                    <>
                      <Sections.Stat label="Total LOC" value={health.total_loc?.toLocaleString() ?? '—'} />
                      <Sections.Stat label="Functions" value={health.total_functions ?? '—'} />
                      <Sections.Stat label="High Complexity" value={health.high_cc_functions ?? '—'} />
                      <Sections.Stat label="Duplicate Lines" value={health.duplicate_lines ?? '—'} />
                    </>
                  )}
                  {scanMeta && (
                    <>
                      <Sections.Stat
                        label="Files"
                        value={scanMeta.total_files?.toLocaleString?.() ?? scanMeta.total_files ?? '—'}
                      />
                      {bp.workspace_topology ? (
                        <Sections.Stat
                          label={`Workspaces (${bp.workspace_topology.type || 'monorepo'})`}
                          value={bp.workspace_topology.members?.length ?? scanMeta.subprojects?.length ?? 0}
                        />
                      ) : (
                        <Sections.Stat label="Subprojects" value={scanMeta.subprojects?.length ?? 0} />
                      )}
                    </>
                  )}
                </div>
              </div>
              {(health?.cc_distribution || health?.mass) && (
                <div className="mt-8 pt-8 border-t border-papaya-400/30 grid md:grid-cols-2 gap-6">
                  {health?.cc_distribution && (
                    <Sections.CCDistribution distribution={health.cc_distribution} compact />
                  )}
                  {health?.mass && (
                    <Sections.MassConcentration
                      mass={health.mass}
                      totalFunctions={health.total_functions}
                      highCcFunctions={health.high_cc_functions}
                      distribution={health.cc_distribution}
                    />
                  )}
                </div>
              )}

              {Array.isArray(scanMeta?.frameworks) && scanMeta.frameworks.length > 0 && (
                <div className="mt-8 pt-8 border-t border-papaya-400/30">
                  <div className="text-[10px] font-black uppercase tracking-[0.2em] text-ink/30 mb-3">
                    Frameworks
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {scanMeta.frameworks.map((f: any, i: number) => (
                      <Badge key={i} variant="outline" className="text-xs border-papaya-400">
                        {f.name}
                        {f.version ? ` ${f.version}` : ''}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </section>
        )}

        {/* Architecture */}
        {(archStyle || meta.architecture_style) && (
          <section className="space-y-8">
            <Sections.SectionHeader title="Architecture" icon={Layout} />
            <div className={cn('p-10 rounded-3xl border space-y-6', theme.surface.panel)}>
              {archStyle?.title && (
                <h3 className="text-2xl font-black text-ink leading-tight">
                  <AutoCode text={archStyle.title} />
                </h3>
              )}
              {archStyle?.chosen && (
                <p className="text-ink/70 leading-relaxed"><AutoCode text={archStyle.chosen} /></p>
              )}
              {!archStyle && meta.architecture_style && (
                <p className="text-ink/70 leading-relaxed"><AutoCode text={meta.architecture_style} /></p>
              )}
              {archStyle?.rationale && (
                <div className="pt-4 border-t border-papaya-400/20">
                  <div className="text-[10px] font-black uppercase tracking-[0.2em] text-ink/30 mb-2">
                    Rationale
                  </div>
                  <p className="text-sm text-ink/60 leading-relaxed"><AutoCode text={archStyle.rationale} /></p>
                </div>
              )}
            </div>
          </section>
        )}

        {/* Found architectural problems */}
        {topFindings.length > 0 && (
          <section className="space-y-8">
            <Sections.SectionHeader
              title="Architectural Problems"
              icon={AlertTriangle}
              hint="Concrete problems observed in specific files."
            />
            <Sections.FindingsList findings={topFindings} truncate />
            {findings.length > topFindings.length && (
              <p className="text-sm text-ink/40 text-center pt-2">
                Showing {topFindings.length} of {findings.length} findings. Full list in the detailed
                report.
              </p>
            )}
          </section>
        )}

        {/* CTA — view full report */}
        <section className="pt-8 space-y-6">
          <p className="text-center text-sm text-ink/40">
            This is a preview. The detailed report includes every component, rule, decision, and
            finding in the blueprint.
          </p>
          <Link
            to={`/r/${token}/details`}
            className={cn(
              'group block p-12 rounded-3xl text-center transition-all hover:shadow-2xl hover:-translate-y-1',
              theme.console.bg,
              theme.console.text
            )}
          >
            <FileText className="w-10 h-10 mx-auto mb-4 opacity-60 group-hover:opacity-100 transition-opacity" />
            <h3 className="text-2xl md:text-3xl font-black mb-2">View full report</h3>
            <p className="opacity-60 mb-6 max-w-lg mx-auto">
              Components, key decisions, trade-offs, rules, technology stack, communication patterns,
              guidelines, and the full scan report.
            </p>
            <span className="inline-flex items-center gap-2 font-bold border-b-2 border-current pb-1 group-hover:gap-3 transition-all">
              Open detailed report
              <ChevronRight className="w-5 h-5" />
            </span>
          </Link>
        </section>
      </main>
    </div>
  )
}
