/** Build an agent-agnostic "fix this" prompt for an architectural finding or
 * pitfall. The output is plain markdown the user pastes into Claude Code,
 * Cursor, Cline, Codex, or any coding agent.
 *
 * The builder resolves linked context from the blueprint — the pitfall a
 * finding stems from, the architectural decision behind that pitfall, the
 * canonical implementation pattern, adopted enforcement rules, the
 * component the affected files belong to, the constraint chain that
 * produced the rule, accepted trade-offs that touch the same area, the
 * schema-level placement/naming rules, and any known semantic
 * duplications — so the receiving agent gets the same structured signal
 * Archie used to flag the problem in the first place.
 *
 * No I/O, no React, no Archie tooling assumed in the resulting prompt: the
 * verification trailer points the agent at the project's own commands
 * (preferring `bp.technology.run_commands` when present), and
 * folder-context instructions name a generic set of agent convention
 * files (CLAUDE.md, AGENTS.md, .cursorrules, ...).
 */

import type { Finding } from './findings'

export type FixItem = Finding & {
  /** Free-form pointer at a class-of-problem entry in `bp.pitfalls`. Findings
   * have a structured `pitfall_id`; pitfalls themselves use this field as a
   * self-id marker so the same builder works for both. */
  __kind?: 'finding' | 'pitfall'
}

interface ResolvedPitfall {
  problem_statement?: string
  root_cause?: string
  stems_from?: string
  fix_direction?: string | string[]
}

interface ResolvedDecision {
  title?: string
  chosen?: string
  rationale?: string
  forced_by?: string
  enables?: string
  alternatives_rejected?: any[]
}

interface ResolvedGuideline {
  capability?: string
  pattern_description?: string
  usage_example?: string
  key_files?: string[]
  applicable_when?: string
  do_not_apply_when?: string | string[]
  scope?: string[]
}

interface ResolvedRule {
  id?: string
  severity_class?: string
  description?: string
  why?: string
  example?: string
}

interface ResolvedComponent {
  name?: string
  location?: string
  platform?: string
  responsibility?: string
  depends_on?: string[]
  exposes_to?: string[]
  key_files?: any[]
}

interface ResolvedTradeoff {
  accept?: string
  benefit?: string
  caused_by?: string
  violation_signals?: string[]
}

interface ResolvedSemanticDup {
  function?: string
  locations?: string[]
  recommendation?: string
}

export interface BuildOpts {
  /** Raw blueprint object from bundle.blueprint. */
  blueprint?: any
  /** rules.json contents — bundle.rules_adopted in share mode. */
  adoptedRules?: any
  /** bundle.semantic_duplications — used to inline dup-specific evidence
   * when the finding matches a known reimplementation. */
  semanticDuplications?: any[]
  /** Override "Archie's deep architectural scan" lead phrasing. */
  leadAttribution?: string
}

/** Public entry point. Returns the full prompt as a single string. */
export function buildFixPrompt(item: FixItem, opts: BuildOpts = {}): string {
  const kind = item.__kind || 'finding'
  const bp = opts.blueprint || {}
  const lead =
    opts.leadAttribution ||
    "Archie's deep architectural scan identified this problem."

  const pitfall = resolvePitfall(item, bp)
  const decision = pitfall?.stems_from ? resolveDecision(pitfall.stems_from, bp) : null
  const guideline = resolveGuideline(item, bp)
  const rules = resolveRules(item, opts.adoptedRules, bp)
  const components = resolveComponents(item, bp)
  const tradeoffs = resolveTradeoffs(item, bp)
  const chainPath = resolveDecisionChain(item, bp)
  const schemaRules = resolveSchemaRules(item, bp)
  const semDup = resolveSemanticDup(item, opts.semanticDuplications)
  const runCmdLines = formatRunCommands(bp)
  const dirs = extractAffectedDirs(item)

  const sections: string[] = []

  // ───── Header ──────────────────────────────────────────────────────────
  sections.push(
    [
      `You are about to fix one ${kind === 'pitfall' ? 'class of architectural problem' : 'architectural problem'} in this repository.`,
      `${lead} All paths below are repo-relative.`,
    ].join('\n'),
  )

  // ───── Severity + confidence ───────────────────────────────────────────
  const metaBits: string[] = []
  if (item.severity) metaBits.push(`**Severity:** ${item.severity}`)
  if (typeof item.confidence === 'number') metaBits.push(`**Confidence:** ${item.confidence}`)
  if (metaBits.length > 0) sections.push(metaBits.join('  ·  '))

  // ───── Problem ─────────────────────────────────────────────────────────
  if (item.title) {
    sections.push(`## Problem\n${item.title}`)
  }
  if (item.description) {
    sections.push(`## Detail\n${item.description}`)
  }

  // ───── Where it fires ──────────────────────────────────────────────────
  if (item.triggering_call_site) {
    sections.push(
      `## Where it fires\n\`\`\`\n${item.triggering_call_site.trim()}\n\`\`\``,
    )
  } else if (kind === 'pitfall') {
    sections.push(
      `## Where it fires\nThis is a *class* of problem — no single triggering site. Locate the instance(s) currently in the codebase that match the problem statement, OR make the preventive change that closes the entire class.`,
    )
  }

  // ───── Other locations ─────────────────────────────────────────────────
  if (Array.isArray(item.applies_to) && item.applies_to.length > 0) {
    sections.push(
      `## Other locations affected\n${item.applies_to.map((p) => `- \`${p}\``).join('\n')}`,
    )
  }

  // ───── Evidence ────────────────────────────────────────────────────────
  if (Array.isArray(item.evidence) && item.evidence.length > 0) {
    sections.push(
      `## Evidence collected during analysis\n${item.evidence.map((e) => `- ${e}`).join('\n')}`,
    )
  }

  // ───── Component context ───────────────────────────────────────────────
  if (components.length > 0) {
    const blocks = components.map((c) => {
      const lines: string[] = []
      const header = c.name
        ? `**${c.name}**${c.platform ? ` *(${c.platform})*` : ''}`
        : `**Component**`
      lines.push(header)
      if (c.location) lines.push(`Location: \`${c.location}\``)
      if (c.responsibility) lines.push(`Responsibility: ${c.responsibility}`)
      if (Array.isArray(c.depends_on) && c.depends_on.length > 0) {
        lines.push(`Depends on: ${c.depends_on.map((d) => `\`${d}\``).join(', ')}`)
      }
      if (Array.isArray(c.exposes_to) && c.exposes_to.length > 0) {
        lines.push(`Exposes to: ${c.exposes_to.map((d) => `\`${d}\``).join(', ')}`)
      }
      return lines.join('\n')
    })
    sections.push(
      `## The component${components.length > 1 ? 's' : ''} this lives in\n${blocks.join('\n\n')}`,
    )
  }

  // ───── Architectural root cause ────────────────────────────────────────
  if (item.root_cause) {
    sections.push(`## Why this is a problem (architectural root cause)\n${item.root_cause}`)
  }

  // ───── Known reimplementation (semantic dup detail) ────────────────────
  if (semDup) {
    const lines: string[] = []
    if (semDup.function) lines.push(`**Function:** \`${semDup.function}\``)
    if (Array.isArray(semDup.locations) && semDup.locations.length > 0) {
      lines.push(`**Locations:**`)
      for (const loc of semDup.locations) lines.push(`- \`${loc}\``)
    }
    if (semDup.recommendation) lines.push(`**Recommendation:** ${semDup.recommendation}`)
    if (lines.length > 0) {
      sections.push(
        `## Known reimplementations of this\nArchie detected this as a near-twin function with the same logic appearing under different names.\n\n${lines.join('\n')}`,
      )
    }
  }

  // ───── Linked pitfall (only when item is a finding) ────────────────────
  if (kind === 'finding' && pitfall) {
    const parts = [`${pitfall.problem_statement || ''}`.trim()]
    if (pitfall.root_cause) parts.push(`\n**Architectural root cause of the class:** ${pitfall.root_cause}`)
    sections.push(`## The class of problem this falls into\n${parts.join('\n')}`)
  }

  // ───── Linked decision (+ rejected alternatives) ───────────────────────
  if (decision) {
    const lines: string[] = []
    if (decision.title && decision.chosen) {
      lines.push(`**Decision:** ${decision.title} — ${decision.chosen}`)
    } else if (decision.title) {
      lines.push(`**Decision:** ${decision.title}`)
    } else if (decision.chosen) {
      lines.push(`**Decision:** ${decision.chosen}`)
    }
    if (decision.rationale) lines.push(`**Rationale:** ${decision.rationale}`)
    if (decision.forced_by) lines.push(`**Forced by:** ${decision.forced_by}`)
    if (decision.enables) lines.push(`**Enables:** ${decision.enables}`)
    const altLines = formatAlternatives(decision.alternatives_rejected)
    if (altLines.length > 0) {
      lines.push('')
      lines.push(`**Rejected alternatives — do NOT reintroduce:**`)
      lines.push(...altLines)
    }
    if (lines.length > 0) {
      sections.push(`## The architectural decision behind it\n${lines.join('\n')}`)
    }
  }

  // ───── Decision chain (root → matching constraint) ─────────────────────
  if (chainPath && chainPath.length > 1) {
    const body = chainPath.map((step, i) => `${i + 1}. ${step}`).join('\n')
    sections.push(
      `## The constraint chain that produced this rule\nThese decisions stack — your fix must keep every link intact.\n\n${body}`,
    )
  }

  // ───── Accepted trade-offs that touch this area ────────────────────────
  if (tradeoffs.length > 0) {
    const blocks = tradeoffs.map((t) => {
      const lines: string[] = []
      if (t.accept) lines.push(`**Accept:** ${t.accept}`)
      if (t.benefit) lines.push(`**Benefit:** ${t.benefit}`)
      if (t.caused_by) lines.push(`**Caused by:** ${t.caused_by}`)
      if (Array.isArray(t.violation_signals) && t.violation_signals.length > 0) {
        lines.push(`**Violation signals:** ${t.violation_signals.map((s) => `\`${s}\``).join(', ')}`)
      }
      return lines.join('\n')
    })
    sections.push(
      `## Accepted trade-offs in this area — don't undo\n${blocks.join('\n\n')}`,
    )
  }

  // ───── Fix direction ───────────────────────────────────────────────────
  const fixSteps =
    normalizeFixDirection(item.fix_direction) ||
    (pitfall ? normalizeFixDirection(pitfall.fix_direction) : null)
  if (fixSteps && fixSteps.length > 0) {
    const body =
      fixSteps.length === 1
        ? fixSteps[0]
        : fixSteps.map((s, i) => `${i + 1}. ${s}`).join('\n')
    sections.push(`## Recommended fix direction\n${body}`)
  }

  // ───── Canonical pattern ───────────────────────────────────────────────
  if (guideline) {
    const lines: string[] = []
    if (guideline.pattern_description) lines.push(guideline.pattern_description)
    if (guideline.usage_example) {
      lines.push('')
      lines.push('**Usage example:**')
      lines.push('```')
      lines.push(guideline.usage_example.trim())
      lines.push('```')
    }
    if (guideline.key_files && guideline.key_files.length > 0) {
      lines.push('')
      lines.push(`**Files that already do this correctly:** ${guideline.key_files.map((f) => `\`${f}\``).join(', ')}`)
    }
    if (guideline.applicable_when) {
      lines.push(`**Apply when:** ${guideline.applicable_when}`)
    }
    const dna = normalizeDoNotApply(guideline.do_not_apply_when)
    if (dna && dna.length > 0) {
      lines.push(`**Do not apply when:** ${dna.join('; ')}`)
    }
    if (lines.length > 0) {
      sections.push(`## The canonical pattern in this codebase\n${lines.join('\n')}`)
    }
  }

  // ───── Schema-level placement / naming rules ───────────────────────────
  if (schemaRules.placement.length > 0 || schemaRules.naming.length > 0) {
    const out: string[] = []
    if (schemaRules.placement.length > 0) {
      out.push('**Placement:**')
      for (const r of schemaRules.placement) {
        const desc = r?.description || r?.rule || r?.title || ''
        const target = r?.applies_to || (Array.isArray(r?.globs) ? r.globs.join(', ') : '')
        out.push(`- ${desc}${target ? ` *(applies to: \`${target}\`)*` : ''}`)
      }
    }
    if (schemaRules.naming.length > 0) {
      if (out.length > 0) out.push('')
      out.push('**Naming:**')
      for (const r of schemaRules.naming) {
        const desc = r?.description || r?.pattern || r?.rule || ''
        const target = r?.applies_to || (Array.isArray(r?.globs) ? r.globs.join(', ') : '')
        out.push(`- ${desc}${target ? ` *(applies to: \`${target}\`)*` : ''}`)
      }
    }
    sections.push(`## File-placement and naming rules that apply\n${out.join('\n')}`)
  }

  // ───── Rules to keep satisfied ─────────────────────────────────────────
  if (rules.length > 0) {
    const lines = rules.map((r) => {
      const sc = r.severity_class ? `[${r.severity_class}] ` : ''
      const head = `- ${sc}${r.description || r.id || 'rule'}`
      const why = r.why ? `\n  **Why:** ${r.why}` : ''
      const ex = r.example
        ? `\n  **Example of the correct shape:**\n  \`\`\`\n  ${r.example.trim().split('\n').join('\n  ')}\n  \`\`\``
        : ''
      return head + why + ex
    })
    sections.push(`## Enforcement rules to keep satisfied\n${lines.join('\n')}`)
  }

  // ───── Working constraints (agent-agnostic + run_commands when known) ──
  const verifyLine =
    runCmdLines.length > 0
      ? `- After editing, run the project's verification commands before reporting done:\n${runCmdLines.join('\n')}`
      : `- After editing, run the project's standard verification (lint, type-check, tests) before reporting done.`
  const constraintLines: string[] = [
    `- If the repository has agent-context files (\`CLAUDE.md\`, \`AGENTS.md\`, \`.cursorrules\`, \`.clinerules\`, \`.windsurfrules\`, or similar), read them before editing${dirs.length > 0 ? ` — especially any in the directories you'll touch: ${dirs.map((d) => `\`${d}\``).join(', ')}` : ''}.`,
    `- Make ONLY the changes needed to fix this specific problem. No surrounding refactors, no new features, no restructuring of unrelated modules.`,
    verifyLine,
  ]
  sections.push(`## Working constraints\n${constraintLines.join('\n')}`)

  // ───── Output expectation ──────────────────────────────────────────────
  const idCite = item.id ? ` Cite id \`${item.id}\` so reviewers can match it back to the original analysis.` : ''
  sections.push(
    `## Output\n- Make the edits.\n- Summarise in one paragraph what you changed and which rule, pitfall, or decision you respected.${idCite}`,
  )

  return sections.join('\n\n')
}

// ── Pitfall / decision resolution ────────────────────────────────────────

function resolvePitfall(item: FixItem, bp: any): ResolvedPitfall | null {
  if (item.__kind === 'pitfall') {
    return {
      problem_statement: item.title,
      root_cause: item.root_cause,
      stems_from: (item as any).stems_from,
      fix_direction: item.fix_direction,
    }
  }
  const id = item.pitfall_id
  if (!id) return null
  const pitfalls = Array.isArray(bp?.pitfalls) ? bp.pitfalls : []
  for (const p of pitfalls) {
    if (p && typeof p === 'object' && p.id === id) {
      return {
        problem_statement: p.problem_statement || p.title || p.area,
        root_cause: p.root_cause,
        stems_from: p.stems_from,
        fix_direction: p.fix_direction || p.recommendation,
      }
    }
  }
  return null
}

function resolveDecision(stemsFrom: string, bp: any): ResolvedDecision | null {
  const needle = stemsFrom.toLowerCase().trim()
  if (!needle) return null
  const decisions = Array.isArray(bp?.decisions?.key_decisions)
    ? bp.decisions.key_decisions
    : []
  for (const d of decisions) {
    const title = String(d?.title || '').toLowerCase()
    if (title && title === needle) return d
  }
  for (const d of decisions) {
    const title = String(d?.title || '').toLowerCase()
    if (title && (title.includes(needle) || needle.includes(title))) return d
  }
  for (const d of decisions) {
    const chosen = String(d?.chosen || '').toLowerCase()
    if (chosen && (chosen.includes(needle) || needle.includes(chosen))) return d
  }
  return null
}

function formatAlternatives(alts: any): string[] {
  if (!Array.isArray(alts) || alts.length === 0) return []
  const out: string[] = []
  for (const a of alts) {
    if (!a) continue
    if (typeof a === 'string') {
      out.push(`- ${a}`)
      continue
    }
    if (typeof a === 'object') {
      const name = a.alternative || a.title || a.name || a.option || ''
      const reason = a.reason || a.rejected_because || a.why || a.rationale || ''
      if (name && reason) out.push(`- **${name}** — rejected because ${reason}`)
      else if (name) out.push(`- ${name}`)
      else if (reason) out.push(`- ${reason}`)
    }
  }
  return out.slice(0, 5)
}

// ── Guideline resolution ─────────────────────────────────────────────────

function collectGuidelines(bp: any): ResolvedGuideline[] {
  const out: any[] = []
  if (Array.isArray(bp?.implementation_guidelines)) out.push(...bp.implementation_guidelines)
  if (Array.isArray(bp?.decisions?.implementation_guidelines)) {
    out.push(...bp.decisions.implementation_guidelines)
  }
  if (Array.isArray(bp?.guidelines)) out.push(...bp.guidelines)
  if (Array.isArray(bp?.architecture_rules?.guidelines)) {
    out.push(...bp.architecture_rules.guidelines)
  }
  return out.filter((g) => g && typeof g === 'object')
}

function resolveGuideline(item: FixItem, bp: any): ResolvedGuideline | null {
  const guidelines = collectGuidelines(bp)
  if (guidelines.length === 0) return null

  const paths = collectItemPaths(item)
  if (paths.length === 0) {
    const titleTokens = tokenize(item.title || '')
    let best: { g: ResolvedGuideline; score: number } | null = null
    for (const g of guidelines) {
      const cap = `${g.capability || ''} ${g.pattern_description || ''}`.toLowerCase()
      const score = titleTokens.reduce((acc, t) => acc + (cap.includes(t) ? 1 : 0), 0)
      if (score > 0 && (!best || score > best.score)) best = { g, score }
    }
    return best?.g || null
  }

  let best: { g: ResolvedGuideline; score: number } | null = null
  for (const g of guidelines) {
    const refs: string[] = []
    if (Array.isArray(g.key_files)) refs.push(...g.key_files.map(String))
    if (Array.isArray(g.scope)) refs.push(...g.scope.map(String))
    if (refs.length === 0) continue
    const score = scorePathOverlap(paths, refs)
    if (score > 0 && (!best || score > best.score)) best = { g, score }
  }
  return best?.g || null
}

// ── Enforcement rules ────────────────────────────────────────────────────

function resolveRules(item: FixItem, adoptedRules: any, _bp: any): ResolvedRule[] {
  const list = extractRulesList(adoptedRules)
  if (list.length === 0) return []

  const paths = collectItemPaths(item)
  const titleHay = (item.title || '').toLowerCase()
  const rootHay = (item.root_cause || '').toLowerCase()

  const scored = list.map((r) => {
    let score = 0
    const aplist: string[] = Array.isArray(r?.applies_to) ? r.applies_to.map(String) : []
    if (paths.length > 0 && aplist.length > 0) {
      score += scorePathOverlap(paths, aplist) * 2
    }
    const forbidden: string[] = Array.isArray(r?.forbidden_patterns) ? r.forbidden_patterns.map(String) : []
    for (const pat of forbidden) {
      try {
        const rx = new RegExp(pat)
        for (const p of paths) {
          if (rx.test(p)) score += 2
        }
        if (rx.test(titleHay) || rx.test(rootHay)) score += 1
      } catch {
        /* invalid regex — skip silently */
      }
    }
    const desc = String(r?.description || '').toLowerCase()
    for (const tok of tokenize(titleHay)) {
      if (desc.includes(tok)) score += 0.5
    }
    return { r, score }
  })

  const SEVERITY_RANK: Record<string, number> = {
    decision_violation: 5,
    pitfall_triggered: 4,
    mechanical_violation: 3,
    tradeoff_undermined: 2,
    pattern_divergence: 1,
  }

  return scored
    .filter((x) => x.score > 0)
    .sort((a, b) => {
      if (b.score !== a.score) return b.score - a.score
      const sa = SEVERITY_RANK[a.r?.severity_class || ''] || 0
      const sb = SEVERITY_RANK[b.r?.severity_class || ''] || 0
      return sb - sa
    })
    .slice(0, 3)
    .map(({ r }) => ({
      id: r?.id,
      severity_class: r?.severity_class || r?.severity,
      description: r?.description || r?.rationale || r?.title,
      why: r?.why,
      example: r?.example,
    }))
}

function extractRulesList(adopted: any): any[] {
  if (!adopted) return []
  if (Array.isArray(adopted)) return adopted.filter((r) => r && typeof r === 'object')
  if (Array.isArray(adopted?.rules)) return adopted.rules.filter((r: any) => r && typeof r === 'object')
  return []
}

// ── Component lookup ────────────────────────────────────────────────────

function resolveComponents(item: FixItem, bp: any): ResolvedComponent[] {
  const components: any[] = Array.isArray(bp?.components?.components)
    ? bp.components.components
    : Array.isArray(bp?.components)
      ? bp.components
      : []
  if (components.length === 0) return []

  const paths = collectItemPaths(item)
  if (paths.length === 0) return []

  const matched: Array<{ c: ResolvedComponent; score: number }> = []
  for (const c of components) {
    if (!c || typeof c !== 'object') continue
    const refs: string[] = []
    if (Array.isArray(c.key_files)) {
      for (const kf of c.key_files) {
        if (typeof kf === 'string') refs.push(kf)
        else if (kf && typeof kf === 'object' && typeof kf.file === 'string') refs.push(kf.file)
      }
    }
    if (typeof c.location === 'string' && c.location) refs.push(c.location)
    if (refs.length === 0) continue
    const score = scorePathOverlap(paths, refs)
    if (score > 0) matched.push({ c, score })
  }

  return matched
    .sort((a, b) => b.score - a.score)
    .slice(0, 2)
    .map(({ c }) => c)
}

// ── Trade-off matching ──────────────────────────────────────────────────

function resolveTradeoffs(item: FixItem, bp: any): ResolvedTradeoff[] {
  const tradeoffs: any[] = Array.isArray(bp?.decisions?.trade_offs)
    ? bp.decisions.trade_offs
    : []
  if (tradeoffs.length === 0) return []

  const hay = [item.title, item.root_cause, item.description]
    .filter(Boolean)
    .join('\n')
    .toLowerCase()
  if (!hay) return []

  const matched: ResolvedTradeoff[] = []
  for (const t of tradeoffs) {
    const signals: string[] = Array.isArray(t?.violation_signals)
      ? t.violation_signals.map(String)
      : []
    if (signals.length === 0) continue
    const hit = signals.some((s) => {
      const lc = s.toLowerCase().trim()
      return lc.length >= 3 && hay.includes(lc)
    })
    if (hit) matched.push(t)
  }
  return matched.slice(0, 2)
}

// ── Decision chain walk ─────────────────────────────────────────────────

function resolveDecisionChain(item: FixItem, bp: any): string[] | null {
  const chain = bp?.decisions?.decision_chain
  if (!chain || typeof chain !== 'object') return null
  if (!Array.isArray(chain.forces) || chain.forces.length === 0) return null

  const hay = [item.title, item.root_cause].filter(Boolean).join('\n').toLowerCase()
  if (!hay) return null

  function walk(nodes: any[], path: string[]): string[] | null {
    for (const n of nodes) {
      if (!n || typeof n !== 'object') continue
      const label = String(n.decision || n.name || '').trim()
      const newPath = label ? [...path, label] : path
      const keywords: string[] = Array.isArray(n.violation_keywords)
        ? n.violation_keywords.map(String)
        : []
      const hit = keywords.some((k) => {
        const lc = k.toLowerCase().trim()
        return lc.length >= 3 && hay.includes(lc)
      })
      if (hit) return newPath
      if (Array.isArray(n.forces) && n.forces.length > 0) {
        const deeper = walk(n.forces, newPath)
        if (deeper) return deeper
      }
    }
    return null
  }

  const tail = walk(chain.forces, [])
  if (!tail || tail.length === 0) return null
  const root = String(chain.root || '').trim() || 'Root constraint'
  return [root, ...tail]
}

// ── Schema-level placement / naming ─────────────────────────────────────

function resolveSchemaRules(
  item: FixItem,
  bp: any,
): { placement: any[]; naming: any[] } {
  const archRules = bp?.architecture_rules || {}
  const placement: any[] = Array.isArray(archRules.file_placement_rules)
    ? archRules.file_placement_rules
    : []
  const naming: any[] = Array.isArray(archRules.naming_conventions)
    ? archRules.naming_conventions
    : []

  const paths = collectItemPaths(item)
  if (paths.length === 0) return { placement: [], naming: [] }

  const matchByPath = (r: any): boolean => {
    const targets: string[] = []
    if (typeof r?.applies_to === 'string' && r.applies_to) targets.push(r.applies_to)
    if (Array.isArray(r?.applies_to)) targets.push(...r.applies_to.map(String))
    if (Array.isArray(r?.globs)) targets.push(...r.globs.map(String))
    if (Array.isArray(r?.paths)) targets.push(...r.paths.map(String))
    if (targets.length === 0) return false
    return scorePathOverlap(paths, targets) > 0
  }

  return {
    placement: placement.filter(matchByPath).slice(0, 3),
    naming: naming.filter(matchByPath).slice(0, 3),
  }
}

// ── Semantic dup lookup ─────────────────────────────────────────────────

function resolveSemanticDup(
  item: FixItem,
  dups: any[] | undefined,
): ResolvedSemanticDup | null {
  if (!Array.isArray(dups) || dups.length === 0) return null
  const titleHay = (item.title || '').toLowerCase()
  const descHay = (item.description || '').toLowerCase()
  const evHay = Array.isArray(item.evidence) ? item.evidence.join('\n').toLowerCase() : ''
  const rootHay = (item.root_cause || '').toLowerCase()
  const hay = `${titleHay}\n${descHay}\n${evHay}\n${rootHay}`
  for (const d of dups) {
    if (!d || typeof d !== 'object') continue
    const fn = d.function
    if (typeof fn !== 'string' || fn.length < 4) continue
    if (hay.includes(fn.toLowerCase())) {
      return {
        function: fn,
        locations: Array.isArray(d.locations) ? d.locations.map(String) : undefined,
        recommendation: d.recommendation,
      }
    }
  }
  return null
}

// ── run_commands extraction ─────────────────────────────────────────────

function formatRunCommands(bp: any): string[] {
  const rc = bp?.technology?.run_commands
  if (!rc || typeof rc !== 'object') return []
  // Preferred order: verify-shaped commands first, so the trailer reads as a
  // verification checklist. Skip install/dev because they aren't verify steps.
  const preferred = [
    'lint', 'format', 'typecheck', 'type_check', 'typescript', 'check',
    'test', 'tests', 'unit_test', 'integration_test', 'build',
  ]
  const seen = new Set<string>()
  const lines: string[] = []
  for (const key of preferred) {
    const val = rc[key]
    if (typeof val === 'string' && val.trim()) {
      lines.push(`    ${key}: ${val.trim()}`)
      seen.add(key)
    }
  }
  // Anything else (custom keys), excluding obvious non-verify commands.
  const skipExtra = new Set(['install', 'dev', 'serve', 'start', 'run', 'clean'])
  for (const [key, val] of Object.entries(rc)) {
    if (seen.has(key) || skipExtra.has(key)) continue
    if (typeof val === 'string' && val.trim()) {
      lines.push(`    ${key}: ${val.trim()}`)
    }
  }
  return lines
}

// ── Path helpers ────────────────────────────────────────────────────────

function collectItemPaths(item: FixItem): string[] {
  const out: string[] = []
  if (Array.isArray(item.applies_to)) out.push(...item.applies_to)
  if (item.triggering_call_site) {
    const first = item.triggering_call_site.split('\n')[0]
    const head = first.split(':')[0].trim()
    if (head) out.push(head)
  }
  return Array.from(new Set(out.filter(Boolean)))
}

function extractAffectedDirs(item: FixItem): string[] {
  const paths = collectItemPaths(item)
  const dirs = new Set<string>()
  for (const p of paths) {
    const slash = p.lastIndexOf('/')
    if (slash > 0) dirs.add(p.slice(0, slash))
    else dirs.add('.')
  }
  return Array.from(dirs)
}

/** Crude overlap score: count how many `refs` share a directory prefix or
 * basename with any `paths`. Returns a non-negative integer. */
function scorePathOverlap(paths: string[], refs: string[]): number {
  let score = 0
  for (const p of paths) {
    for (const r of refs) {
      if (!r) continue
      if (p === r) {
        score += 3
        continue
      }
      const stem = r.replace(/[*?]/g, '').replace(/\/$/, '')
      if (!stem) continue
      if (p.startsWith(stem) || stem.startsWith(p)) {
        score += 2
        continue
      }
      const pBase = p.slice(p.lastIndexOf('/') + 1)
      const rBase = stem.slice(stem.lastIndexOf('/') + 1)
      if (pBase && rBase && pBase === rBase) score += 1
    }
  }
  return score
}

function tokenize(s: string): string[] {
  return s
    .toLowerCase()
    .split(/[^a-z0-9]+/)
    .filter((t) => t.length >= 4)
}

function normalizeFixDirection(fd: string | string[] | undefined): string[] | null {
  if (!fd) return null
  if (Array.isArray(fd)) {
    const cleaned = fd.map((s) => String(s).trim()).filter(Boolean)
    return cleaned.length > 0 ? cleaned : null
  }
  const s = String(fd).trim()
  return s ? [s] : null
}

function normalizeDoNotApply(d: string | string[] | undefined): string[] | null {
  if (!d) return null
  if (Array.isArray(d)) {
    const cleaned = d.map((s) => String(s).trim()).filter(Boolean)
    return cleaned.length > 0 ? cleaned : null
  }
  const s = String(d).trim()
  return s ? [s] : null
}
