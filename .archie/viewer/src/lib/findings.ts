/** Extract ranked findings from a scan_report.md string.
 *
 * Scan reports follow this shape:
 *
 *   ## Findings
 *   <preamble>
 *   ### RECURRING (previously documented, still present)
 *   1. **[error] Title text.** Description text spanning one or more lines.
 *   2. **[warn] Another title.** More description.
 *   ### NEW (first observed in scan #N)
 *   ### RESOLVED
 */

export type FindingSeverity = 'error' | 'warn' | 'info'
export type FindingGroup = 'RECURRING' | 'NEW' | 'RESOLVED'

export interface Finding {
  severity: FindingSeverity
  title: string
  description: string
  group?: FindingGroup
  // Rich 4-field shape (present when sourced from findings.json / blueprint
  // pitfalls rather than from scan_report.md regex parsing).
  id?: string
  evidence?: string[]
  root_cause?: string
  fix_direction?: string | string[]
  applies_to?: string[]
  confidence?: number
  status?: string
  pitfall_id?: string
  // Verifier-pipeline fields. `triggering_call_site` is the verbatim caller
  // quote the synthesizer is required to provide — surfacing it tells the
  // user exactly where the failure fires. The verdict_* fields are the
  // backward-check audit trail; `pending_*` flags surface borderline
  // entries the hysteresis layer is tracking but hasn't transitioned yet.
  triggering_call_site?: string
  verdict_history?: string[]
  last_verdict_reason?: string
  last_verdict_confidence?: number
  pending_demotion?: boolean
  pending_promotion?: boolean
  demoted_at?: string
  dropped_at?: string
}

/** Normalize a structured finding (from findings.json) into the Finding shape
 * the UI components expect. Keeps the 4-field data available for rich rendering
 * while preserving legacy `description` (if present) so old bundles still
 * show their prose body. `description` is NOT auto-filled from root_cause /
 * evidence — doing so would duplicate content when rendered in rich mode. */
export function normalizeStructuredFinding(raw: any): Finding {
  const sev = (raw?.severity || 'warn').toLowerCase() as FindingSeverity
  const problem = raw?.problem_statement || raw?.title || raw?.area || ''
  return {
    severity: (['error', 'warn', 'info'] as const).includes(sev) ? sev : 'warn',
    title: String(problem),
    description: raw?.description ? String(raw.description) : '',
    id: raw?.id,
    evidence: Array.isArray(raw?.evidence) ? raw.evidence.map(String) : undefined,
    root_cause: raw?.root_cause || undefined,
    fix_direction: raw?.fix_direction || undefined,
    applies_to: Array.isArray(raw?.applies_to) ? raw.applies_to.map(String) : undefined,
    confidence: typeof raw?.confidence === 'number' ? raw.confidence : undefined,
    status: raw?.status || undefined,
    pitfall_id: raw?.pitfall_id || undefined,
    triggering_call_site: raw?.triggering_call_site
      ? String(raw.triggering_call_site)
      : undefined,
    verdict_history: Array.isArray(raw?.verdict_history) ? raw.verdict_history.map(String) : undefined,
    last_verdict_reason: raw?.last_verdict_reason ? String(raw.last_verdict_reason) : undefined,
    last_verdict_confidence: typeof raw?.last_verdict_confidence === 'number'
      ? raw.last_verdict_confidence
      : undefined,
    pending_demotion: raw?.pending_demotion === true,
    pending_promotion: raw?.pending_promotion === true,
    demoted_at: raw?.demoted_at ? String(raw.demoted_at) : undefined,
    dropped_at: raw?.dropped_at ? String(raw.dropped_at) : undefined,
  }
}

/** Normalize a pitfall (from blueprint.pitfalls) into the Finding shape. Same
 * 4-field schema as findings, but kept separate so callers can label the
 * section differently. Falls back to the legacy
 * {title/area, description, recommendation} shape for old bundles. */
export function normalizePitfall(raw: any): Finding {
  const sev = (raw?.severity || 'warn').toLowerCase() as FindingSeverity
  const problem = raw?.problem_statement || raw?.title || raw?.area || 'Pitfall'
  return {
    severity: (['error', 'warn', 'info'] as const).includes(sev) ? sev : 'warn',
    title: String(problem),
    // Preserve legacy prose body. Don't synthesize from 4-field data — that
    // would double-render alongside Evidence / Root Cause in the rich view.
    description: raw?.description ? String(raw.description) : '',
    id: raw?.id,
    evidence: Array.isArray(raw?.evidence) ? raw.evidence.map(String) : undefined,
    root_cause: raw?.root_cause || undefined,
    // Bridge legacy pitfalls: recommendation becomes a single-step fix_direction
    // so it still renders in the new "Fix Direction" block.
    fix_direction: raw?.fix_direction || raw?.recommendation || undefined,
    applies_to: Array.isArray(raw?.applies_to) ? raw.applies_to.map(String) : undefined,
    confidence: typeof raw?.confidence === 'number' ? raw.confidence : undefined,
    status: raw?.status || undefined,
  }
}

const SEVERITY_RANK: Record<FindingSeverity, number> = { error: 0, warn: 1, info: 2 }
const GROUP_RANK: Record<FindingGroup, number> = { NEW: 0, RECURRING: 1, RESOLVED: 2 }

export function extractFindings(scanReport: string): Finding[] {
  if (!scanReport) return []

  const lines = scanReport.split('\n')
  let inFindings = false
  let currentGroup: FindingGroup | undefined

  // Find the ## Findings heading, capture until next ## heading
  const findingsLines: Array<{ text: string; group?: FindingGroup }> = []
  for (const line of lines) {
    if (!inFindings) {
      if (/^##\s+Findings\b/i.test(line)) inFindings = true
      continue
    }
    if (/^##\s+/.test(line) && !/^###/.test(line)) break

    const groupMatch = line.match(/^###\s+(RECURRING|NEW|RESOLVED)\b/i)
    if (groupMatch) {
      currentGroup = groupMatch[1].toUpperCase() as FindingGroup
      continue
    }
    findingsLines.push({ text: line, group: currentGroup })
  }

  // Now walk collected lines; a finding starts with `N. **[sev] Title.**` and
  // continues on subsequent non-numbered, non-empty lines.
  const findings: Finding[] = []
  let current: Finding | null = null

  for (const { text, group } of findingsLines) {
    const startMatch = text.match(/^\s*\d+\.\s*\*\*\[(\w+)\]\s*([^*]+?)\*\*\s*(.*)$/)
    if (startMatch) {
      if (current) findings.push(current)
      const sev = startMatch[1].toLowerCase() as FindingSeverity
      const title = startMatch[2].trim().replace(/[.:]+$/, '')
      const description = startMatch[3].trim()
      current = {
        severity: ['error', 'warn', 'info'].includes(sev) ? sev : 'warn',
        title,
        description,
        group,
      }
    } else if (current && text.trim() && !/^#/.test(text)) {
      current.description = (current.description + ' ' + text.trim()).trim()
    } else if (!text.trim()) {
      if (current) {
        findings.push(current)
        current = null
      }
    }
  }
  if (current) findings.push(current)

  return findings
}

/** Sort findings by severity error>warn>info first, then NEW>RECURRING>RESOLVED. */
export function rankFindings(findings: Finding[]): Finding[] {
  return [...findings].sort((a, b) => {
    const sa = SEVERITY_RANK[a.severity]
    const sb = SEVERITY_RANK[b.severity]
    if (sa !== sb) return sa - sb
    const ga = a.group ? GROUP_RANK[a.group] : 3
    const gb = b.group ? GROUP_RANK[b.group] : 3
    return ga - gb
  })
}

/** Pick up to `total` findings, reserving up to `minErrors` slots for errors.
 *
 * Behavior:
 * - If ≥ minErrors errors exist: show `minErrors` errors + fill remaining (total - minErrors) with warns/info.
 * - If fewer than minErrors errors exist: show all errors + fill the rest with warns/info.
 * - Non-error slots are filled from warns first, then info. */
export function pickTopFindings(
  findings: Finding[],
  total = 6,
  minErrors = 4,
): Finding[] {
  const ranked = rankFindings(findings)
  const errors = ranked.filter((f) => f.severity === 'error')
  const nonErrors = ranked.filter((f) => f.severity !== 'error')
  const errorSlots = Math.min(errors.length, Math.max(minErrors, 0), total)
  const fillSlots = Math.max(0, total - errorSlots)
  return [...errors.slice(0, errorSlots), ...nonErrors.slice(0, fillSlots)]
}

/** Heuristic keyword match for findings that describe semantic duplication /
 * reimplementation. Used for count (fallback when no structured data) and
 * for tagging matching findings visibly in the UI.
 *
 * Captures:
 *   - duplicat(ed|ion), reimplement(ation), near-dup, near-twin, similar function
 *   - "N separate <X> implementations"   (e.g., "3 separate generateSlug implementations")
 *   - "duplicate <X> of", "copies of <X>"
 */
// Word-start boundary only. Word-END boundary is deliberately omitted so
// "duplicat" matches "duplicated", "duplication", "duplicates"; "reimplement"
// matches "reimplementation", etc.
export const SEMANTIC_DUPE_RX =
  /\b(duplicat|reimplement|near[- ]?dup|near[- ]?twin|similar function|\d+\s+separate\s+\S+\s+implementations?|multiple\s+\S+\s+implementations?|copies\s+of)/i

function _escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function _matchesFunctionName(haystack: string, rawName: string): boolean {
  // Split the entry on separators, keep only identifier-looking tokens with a
  // camelCase transition (lowercase followed by uppercase). This filters out
  // generic words like "violation", "package", "import" that would cause
  // false positives on unrelated findings.
  const tokens = rawName
    .split(/[\s/()]+/)
    .map((t) => t.trim())
    .filter((t) => t.length >= 5 && /[a-z][A-Z]/.test(t))

  for (const tok of tokens) {
    // (1) Exact match — `\bgenerateMessageId\b` in the text.
    const exactRx = new RegExp(`\\b${_escapeRegex(tok)}\\b`)
    if (exactRx.test(haystack)) return true

    // (2) Relaxed match — allow whitespace between camelCase parts so that
    // "generate MessageId" in the prose still matches "generateMessageId".
    const parts = tok.split(/(?<=[a-z])(?=[A-Z])/).map(_escapeRegex)
    if (parts.length >= 2) {
      const relaxRx = new RegExp(`\\b${parts.join('\\s*')}\\b`)
      if (relaxRx.test(haystack)) return true
    }
  }
  return false
}

export function isSemanticDupFinding(
  f: Finding,
  opts?: { functionNames?: string[] },
): boolean {
  if (SEMANTIC_DUPE_RX.test(f.title) || SEMANTIC_DUPE_RX.test(f.description)) {
    return true
  }
  // Structured hint: the bundle may carry a list of function names known to be
  // reimplementations (Agent C's duplications output). Match conservatively on
  // camelCase identifiers only, exact or separated by whitespace.
  const names = opts?.functionNames
  if (names && names.length > 0) {
    const haystack = `${f.title}\n${f.description}`
    for (const raw of names) {
      if (_matchesFunctionName(haystack, raw)) return true
    }
  }
  return false
}

export function countSemanticDuplications(findings: Finding[]): number {
  return findings.filter((f) => isSemanticDupFinding(f)).length
}

/** Parse the scan report for an explicit "no semantic duplication" verdict.
 *
 * Older bundles (and bundles written before `semantic_duplications.json` was
 * a deterministic pipeline artifact) often carry the AI's zero verdict only
 * in the scan report prose — e.g. `## Part 6: Semantic Duplication` followed
 * by "No semantic duplication detected after AI analysis." Returns `true`
 * when the report explicitly asserts a zero count, so the viewer can treat
 * this as a structured 0 rather than a heuristic fallback. */
export function scanReportAssertsZeroSemanticDup(scanReport: string): boolean {
  if (!scanReport) return false
  // Anchor on a "Semantic Duplication" heading (Part 6 in the current template,
  // but keep it tolerant of future section numbering). Then look for an
  // explicit zero assertion within a reasonable slice after the heading.
  const headingRx = /^#{1,4}\s+[^\n]*semantic\s+duplication[^\n]*$/im
  const m = headingRx.exec(scanReport)
  if (!m) return false
  const after = scanReport.slice(m.index + m[0].length, m.index + m[0].length + 1200)
  return /\b(no|zero)\s+(semantic\s+duplicat\w*|near[- ]?twin\s+functions?)\b/i.test(after)
    || /\b(semantic\s+duplicat\w*|near[- ]?twin\s+functions?)\b[^.\n]{0,40}\bnot\s+detected\b/i.test(after)
    || /\bclean\b[^.\n]{0,40}\b(semantic|duplicat\w*)\b/i.test(after)
}

export function severityColor(sev: FindingSeverity): string {
  if (sev === 'error') return 'text-brandy border-brandy/30 bg-brandy/5'
  if (sev === 'warn') return 'text-tangerine-800 border-tangerine/30 bg-tangerine/5'
  return 'text-ink/60 border-ink/10 bg-ink/5'
}
