/**
 * Auto-wrap code-like tokens in markdown backticks (for ReactMarkdown)
 * and provide a React component for plain text rendering.
 */

/** Patterns that look like code — order matters (longer/more specific first) */
const CODE_PATTERNS = [
  // Route-like paths: /localization
  /\/[a-zA-Z0-9_.-]+(?:\/[a-zA-Z0-9_.-]+)*\/?/,
  // Slash-separated paths: common/domain/api, util/services/
  /[a-zA-Z_][\w]*(?:\/[\w.*]+){1,}\/?/,
  // Dotted identifiers (3+ segments): com.bitraptors.babyweather
  /[a-zA-Z_][\w]*(?:\.[a-zA-Z_][\w]*){2,}/,
  // Generic types: BaseDataSource<T>, List<String>
  /[A-Z][\w]*<[\w?,\s]*>/,
  // Wildcard identifiers: page_*, *_impl, *Impl
  /\w+_\*|\*_\w+|\*[A-Z][A-Za-z0-9_]*/,
  // snake_case identifiers (2+ segments): page_dashboard
  /[a-z][a-z0-9]*(?:_[a-z0-9]+)+/,
  // PascalCase compound names: MainActivity, SharedPreferences
  /[A-Z][a-z]+(?:[A-Z][a-z]+){1,}/,
]

/** Combined pattern matching any code-like token */
const COMBINED_RE = new RegExp(
  '(' + CODE_PATTERNS.map(p => p.source).join('|') + ')',
  'g'
)

// ── Markdown helper (for ReactMarkdown) ──

export function autoBacktick(text: string): string {
  if (!text) return text

  const BACKTICK_RE = /`[^`]+`/g
  const parts: string[] = []
  let last = 0
  for (const m of text.matchAll(BACKTICK_RE)) {
    if (m.index! > last) parts.push(wrapPlain(text.slice(last, m.index!)))
    parts.push(m[0])
    last = m.index! + m[0].length
  }
  if (last < text.length) parts.push(wrapPlain(text.slice(last)))
  return parts.join('')
}

function wrapPlain(segment: string): string {
  return segment.replace(COMBINED_RE, '`$1`')
}

// ── React helper (for plain text in JSX) ──

import { createElement, Fragment, useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

/** Loud chip-shaped inline code. Used for explicit `<code>` slots in card
 *  chrome (scope badges, location cells, rule patterns) where the chip is the
 *  whole content of its container. Dense in prose — see `codeInlineSubtleClassName`. */
export const codeInlineClassName =
  'inline rounded-md bg-[#e4f1f5] px-1.5 py-0.5 font-mono text-[0.92em] font-semibold text-[#4b98ad] box-decoration-clone'

/** Quiet inline code for *prose*. AutoCode produces many of these per
 *  paragraph — a Kotlin description can have 15+ PascalCase identifiers — so
 *  the loud teal pill turns descriptions into a wall of chips. We keep
 *  monospace + accent color (still distinguishes from prose) but drop the
 *  background pill, padding, and rounded corners. */
export const codeInlineSubtleClassName =
  'inline font-mono text-[0.92em] font-semibold text-[#1a7d95]'

/** Trigger chip — keeps the loud pill so the dotted underline + chip together
 *  advertise that this token has a tooltip the user can hover. Tooltips are
 *  rare enough per paragraph (1-3) that the loud styling reads as a signal,
 *  not noise. */
const codeWithTooltipTriggerClassName =
  codeInlineClassName +
  ' cursor-help underline decoration-dotted decoration-[#4b98ad]/40 underline-offset-2'

/** Popover bubble rendered through a portal at document.body. Fixed
 *  positioning + portal escape every overflow-hidden ancestor and every
 *  ancestor `.group:hover` selector that previously triggered every chip's
 *  popover at once. */
const tooltipPopoverPortalClassName =
  'pointer-events-none fixed z-[1000] ' +
  'max-w-[min(36rem,calc(100vw-1rem))] break-all whitespace-normal ' +
  'rounded-md bg-ink/95 px-2.5 py-1.5 ' +
  'font-mono text-[11px] leading-snug text-papaya-50 shadow-lg'

/**
 * Matches the "<Identifier> (<path-with-/>)" shape — accepts both file paths
 * (with extension and optional :line) AND directory paths (trailing slash):
 *   BabyWeatherAnalyticsManager (app/src/.../BabyWeatherAnalyticsManager.kt)
 *   LocationDataSource (app/src/.../LocationDataSource.kt:80)
 *   FooBar (lib/foo/bar.go:18-21)
 *   AppStartService (app/src/main/.../appstart/)        ← directory ref
 *   LocalisationManager (app/src/main/.../localisation/) ← directory ref
 *
 * Captured groups: 1 = identifier (rendered visibly), 2 = full path
 * (rendered into the hover popover, hidden inline).
 *
 * Constraint: the parenthesised content must contain at least one `/` and
 * no spaces — keeps prose like "(no slashes here)" or "(bar) baz" out.
 */
const OBJECT_WITH_PATH_RE =
  /\b([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)\s]*\/[^)\s]*)\)/g

/**
 * Matches a standalone slash path in prose (no parens). Any path with at
 * least one directory segment collapses — both long file paths and short
 * directory references like `sdk/location` or `app/CLAUDE.md`. The user's
 * directive is "show only the object name", so short paths collapse too.
 *
 * Captured groups:
 *   1 = directory portion (rendered as the tooltip's prefix)
 *   2 = basename + optional :line[-line], OR empty string for trailing-slash dirs
 *       (rendered visibly as the chip text)
 *
 * Examples that match:
 *   app/src/main/java/com/.../FooViewModel.kt           → chip "FooViewModel.kt"
 *   common/.../SubscriptionRepositoryImpl.kt:32          → chip "...kt:32"
 *   app/src/main/java/com/bitraptors/babyweather/util/   → chip "util/"
 *   sdk/location                                         → chip "location"
 *   app/CLAUDE.md                                        → chip "CLAUDE.md"
 */
const STANDALONE_LONG_PATH_RE =
  /\b((?:[A-Za-z0-9_.\-]+\/)+)([A-Za-z0-9_.\-]+(?:\.[a-z]{1,5}(?::\d+(?:-\d+)?)?)?\/?)/g

/**
 * Matches a dotted package — three or more all-lowercase segments separated
 * by dots, e.g. `com.bitraptors.babyweather.util` or `hu.bitraptors.login`.
 * The leaf segment is shown as the chip and the full package goes into the
 * hover popover, mirroring how slash paths collapse. The all-lowercase
 * constraint avoids eating `Foo.kt`-style filenames or `Map.Entry`-style
 * type references.
 *
 * Captured groups:
 *   1 = the entire dotted package (used as both match value and tooltip)
 */
const DOTTED_PACKAGE_RE =
  /\b([a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*){2,})\b/g

/**
 * Chip-styled trigger that shows ``visible`` inline and reveals ``hover`` in
 * a portaled popover on mouse-enter / focus. The popover is rendered at
 * document.body via createPortal — that escapes every `overflow-hidden`
 * ancestor (so edge-of-cell chips don't get clipped) and avoids the
 * accidental "all popovers on" behavior that came from sharing the
 * `.group:hover .group-hover\:opacity-100` selector with ancestor cards
 * that themselves use `group`.
 *
 * Position is computed from the trigger's bounding rect, then clamped to
 * the viewport with a small margin so the popover always lands on screen.
 */
function TooltipChip({ visible, hover, className }: { visible: string; hover: string; className?: string }): ReactNode {
  const triggerRef = useRef<HTMLSpanElement | null>(null)
  const popRef = useRef<HTMLSpanElement | null>(null)
  const [open, setOpen] = useState(false)
  const [pos, setPos] = useState<{ left: number; top: number } | null>(null)

  // Measure & clamp once the popover has rendered so we know its size.
  useLayoutEffect(() => {
    if (!open || !triggerRef.current || !popRef.current) return
    const trig = triggerRef.current.getBoundingClientRect()
    const pop = popRef.current.getBoundingClientRect()
    const margin = 8
    const vw = window.innerWidth
    const vh = window.innerHeight

    let left = trig.left
    let top = trig.bottom + 4
    // Horizontal clamp: keep within viewport.
    if (left + pop.width + margin > vw) left = Math.max(margin, vw - pop.width - margin)
    if (left < margin) left = margin
    // Vertical flip: if no room below, place above.
    if (top + pop.height + margin > vh && trig.top - pop.height - 4 > margin) {
      top = trig.top - pop.height - 4
    }
    // If neither fits cleanly, keep below but clamp to viewport.
    if (top + pop.height + margin > vh) top = Math.max(margin, vh - pop.height - margin)

    setPos({ left, top })
  }, [open, hover])

  const show = () => setOpen(true)
  const hide = () => {
    setOpen(false)
    setPos(null)
  }

  const trigger = createElement(
    'span',
    {
      ref: triggerRef,
      className: className
        ? `${codeWithTooltipTriggerClassName} ${className}`
        : codeWithTooltipTriggerClassName,
      tabIndex: 0,
      title: hover,
      onMouseEnter: show,
      onMouseLeave: hide,
      onFocus: show,
      onBlur: hide,
    },
    visible,
  )

  const popover =
    open && typeof document !== 'undefined'
      ? createPortal(
          createElement(
            'span',
            {
              ref: popRef,
              role: 'tooltip',
              className: tooltipPopoverPortalClassName,
              // Hide visually until measured so we don't flash at (0,0).
              style: pos
                ? { left: pos.left, top: pos.top }
                : { left: -9999, top: -9999, opacity: 0 },
            },
            hover,
          ),
          document.body,
        )
      : null

  return createElement(Fragment, null, trigger, popover)
}

/**
 * Compute the "leaf" of a path or dotted package — the part a reader cares
 * about (file/object name) when the rest is bookkeeping (directory chain or
 * package qualifier).
 *
 *   app/src/main/foo/Bar.kt           → Bar.kt
 *   app/src/main/foo/Bar.kt:32        → Bar.kt:32
 *   app/src/.../localisation/         → localisation/
 *   com.bitraptors.babyweather.util   → util
 *   CLAUDE.md                         → CLAUDE.md   (already a leaf)
 *
 * Returns the input unchanged when there's nothing to strip.
 */
function pathLeaf(path: string): string {
  const trimmed = path.trim()
  if (!trimmed) return trimmed

  if (trimmed.includes('/')) {
    const hadTrailingSlash = trimmed.endsWith('/')
    const body = hadTrailingSlash ? trimmed.slice(0, -1) : trimmed
    const last = body.slice(body.lastIndexOf('/') + 1)
    if (!last) return trimmed
    return hadTrailingSlash ? last + '/' : last
  }

  // Dotted package — only collapse when there are 2+ dots (3+ segments) so
  // ordinary `Foo.kt`-style filenames are left alone.
  const parts = trimmed.split('.')
  if (parts.length >= 3) {
    return parts[parts.length - 1] || trimmed
  }
  return trimmed
}

/**
 * Reusable view: render a single path or package as just its leaf, with the
 * full string revealed on hover. The atomic primitive every section that
 * shows file/object identifiers should reach for.
 *
 *   <PathChip path="app/src/main/foo/Bar.kt" />
 *     → chip "Bar.kt", hover popover "app/src/main/foo/Bar.kt"
 *
 * When the input has no qualifier to strip (already a leaf), renders as a
 * plain code chip with no tooltip — keeps short tokens unobtrusive.
 */
export function PathChip({ path, className }: { path: string; className?: string }): ReactNode {
  if (!path) return null
  const leaf = pathLeaf(path)
  if (leaf === path) {
    return createElement(
      'code',
      { className: className ? `${codeInlineClassName} ${className}` : codeInlineClassName },
      path,
    )
  }
  return createElement(TooltipChip, { visible: leaf, hover: path, className })
}

/** A merged regex that finds whichever path-collapse pattern matches first
 *  in a left-to-right scan. We tag the alternatives by named groups so the
 *  caller can dispatch on which one fired. */
const COLLAPSE_RE = new RegExp(
  '(?<objWithPath>' + OBJECT_WITH_PATH_RE.source + ')' +
    '|(?<longPath>' + STANDALONE_LONG_PATH_RE.source + ')' +
    '|(?<dottedPkg>' + DOTTED_PACKAGE_RE.source + ')',
  'g',
)

/**
 * Split text into plain spans, <code> elements, and tooltip-spans.
 *
 * Tooltip-span fires for two shapes — both render only a chip with hover-on
 * popover carrying the long stuff:
 *   (1) ``<Identifier> (<long/path/file.ext>)`` — show identifier, hide parens+path
 *   (2) Standalone path with ≥3 directory segments — show basename, hide directory
 *
 * Use: <AutoCode text={someString} />
 */
export function AutoCode({ text }: { text: string }): ReactNode {
  if (!text) return null
  // Wave-2 AI output occasionally drifts from schema (e.g. a string field
  // returning an array). Coerce here so one stray field can't unmount the
  // whole report tree via text.matchAll() throwing.
  if (typeof text !== 'string') text = String(text)

  const parts: ReactNode[] = []
  let key = 0
  const getKey = () => key++
  let last = 0

  // First pass: walk the COLLAPSE_RE matches; tokenize in-between text via CODE_PATTERNS.
  for (const m of text.matchAll(COLLAPSE_RE)) {
    if (m.index! > last) {
      const between = text.slice(last, m.index!)
      // Tokenize the in-between text through the existing PascalCase / path / etc. patterns.
      let bLast = 0
      for (const cm of between.matchAll(COMBINED_RE)) {
        if (cm.index! > bLast) parts.push(between.slice(bLast, cm.index!))
        parts.push(createElement('code', { key: getKey(), className: codeInlineSubtleClassName }, cm[0]))
        bLast = cm.index! + cm[0].length
      }
      if (bLast < between.length) parts.push(between.slice(bLast))
    }

    const groups = m.groups || {}
    if (groups.objWithPath) {
      // OBJECT_WITH_PATH_RE inside a named group: identifier+path captures
      // are at m[2] and m[3]. Visible identifier comes from prose, so we
      // can't ask PathChip to derive it — we keep the explicit pairing.
      parts.push(createElement(TooltipChip, { key: getKey(), visible: m[2], hover: m[3] }))
    } else if (groups.longPath) {
      // STANDALONE_LONG_PATH_RE inside a named group: m[5] = directory
      // prefix, m[6] = basename. Delegate to PathChip so every collapse
      // call site (prose paths, explicit lists) shares one component.
      const dir = m[5] ?? ''
      const basename = m[6] ?? ''
      parts.push(createElement(PathChip, { key: getKey(), path: dir + basename }))
    } else {
      // DOTTED_PACKAGE_RE inside a named group. PathChip handles the leaf
      // computation for dotted packages too.
      parts.push(createElement(PathChip, { key: getKey(), path: groups.dottedPkg! }))
    }
    last = m.index! + m[0].length
  }
  if (last < text.length) {
    const tail = text.slice(last)
    let bLast = 0
    for (const cm of tail.matchAll(COMBINED_RE)) {
      if (cm.index! > bLast) parts.push(tail.slice(bLast, cm.index!))
      parts.push(createElement('code', { key: getKey(), className: codeInlineSubtleClassName }, cm[0]))
      bLast = cm.index! + cm[0].length
    }
    if (bLast < tail.length) parts.push(tail.slice(bLast))
  }

  return createElement(Fragment, null, ...parts)
}

/**
 * Render a schema-string-or-array field as a paragraph or bullet list.
 *
 * Wave-2 AI sometimes returns an array where the schema declares a string
 * (e.g. decisions[].enables, communication.patterns[].how_it_works). Every
 * standalone-paragraph render of a blueprint string field should go through
 * this component so a shape drift in any single field can't take down the
 * page. Inline sites (where the field is composed mid-sentence with prose
 * around it) should keep <AutoCode/> directly — AutoCode's String() coercion
 * keeps those crash-safe even if the value drifts to non-string.
 *
 *   <Prose value={d.rationale} className="text-sm text-ink/70" />
 *
 * Single string → <p className={...}><AutoCode text=...></p>
 * Array of strings → <ul className={...}>... bulleted items ...</ul>
 * Empty/null/non-{string|array} → null (with one round-trip through String() for
 * primitive coercions like number).
 */
export function Prose({
  value,
  className,
  bulletClassName,
}: {
  value: unknown
  className?: string
  bulletClassName?: string
}): ReactNode {
  if (value == null) return null
  if (Array.isArray(value)) {
    const items = value
      .map((v) => (typeof v === 'string' ? v : v == null ? '' : String(v)))
      .filter((s) => s.length > 0)
    if (items.length === 0) return null
    if (items.length === 1)
      return createElement('p', { className }, createElement(AutoCode, { text: items[0] }))
    return createElement(
      'ul',
      { className: className ? `${className} space-y-1.5` : 'space-y-1.5' },
      items.map((s, i) =>
        createElement(
          'li',
          { key: i, className: 'flex items-start gap-2' },
          createElement(
            'span',
            { className: bulletClassName ?? 'text-ink/30 mt-0.5 shrink-0' },
            '•',
          ),
          createElement(
            'span',
            { className: 'flex-1 min-w-0 break-words' },
            createElement(AutoCode, { text: s }),
          ),
        ),
      ),
    )
  }
  const text = typeof value === 'string' ? value : String(value)
  if (!text) return null
  return createElement('p', { className }, createElement(AutoCode, { text }))
}
