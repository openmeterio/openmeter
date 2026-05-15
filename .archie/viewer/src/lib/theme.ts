/**
 * Semantic theme constants — single source of truth for the custom color palette.
 *
 * Every leaf is a plain Tailwind class string. Components import `theme` and
 * compose with `cn()`. Swapping the palette means editing only this file.
 */
export const theme = {
  /* ── Interactive / CTA ──────────────────────────────────────────────── */
  interactive: {
    cta: "bg-tangerine hover:bg-tangerine-600 shadow-lg shadow-tangerine/20 text-white font-semibold",
    ctaLarge: "bg-tangerine hover:bg-tangerine-600 text-white font-bold shadow-xl shadow-tangerine/20",
    strategyActive: "bg-teal text-white shadow-md",
    strategyInactive: "text-foreground/70 border-papaya-400 hover:border-teal-400",
    ghostBrand: "hover:text-teal hover:bg-teal-50",
    focusRing: "focus:ring-2 focus:ring-teal",
  },

  /* ── Active / Selected ──────────────────────────────────────────────── */
  active: {
    card: "border-teal ring-1 ring-teal shadow-md bg-teal-50/5",
    badge: "bg-teal-50 text-teal border-teal-200",
    sidebarItem: "bg-teal-50/30 text-teal hover:bg-teal-50/50 border border-teal-50/50",
    sidebarContext: "bg-teal-50/50 border border-teal-50 hover:bg-teal-50/80",
    sidebarContextLabel: "text-teal",
    phasePill: "bg-teal text-white shadow-md shadow-teal-50 border-transparent",
    blueprintBtn: "bg-teal-50 text-teal hover:bg-teal-50",
    iconColor: "text-teal",
    checkBtn: "bg-teal-50 text-teal",
  },

  /* ── Status indicators ──────────────────────────────────────────────── */
  status: {
    successPanel: "bg-teal-50 border border-teal-200",
    successText: "font-medium text-teal-800",
    successSubtext: "text-teal-600",
    successBtn: "bg-teal-600 hover:bg-teal-700 text-white shadow-lg shadow-teal/20",
    successHud: "bg-teal-50/50 border-teal-100/50",
    successHudIcon: "text-teal-600",
    successHudSubtext: "text-teal-600/70",
    errorPanel: "bg-brandy-50/50 border border-brandy-50",
    errorTitle: "font-medium text-brandy",
    errorText: "text-brandy-300",
    errorIcon: "text-brandy",
    warningPanel: "bg-amber-50/50 border border-amber-200/50",
    warningText: "text-amber-900",
    warningSubtext: "text-amber-700",
    warningIcon: "text-amber-400",
    syncRequired: "bg-amber-500 hover:bg-amber-600 shadow-amber-500/30 animate-pulse",
  },

  /* ── Surfaces / panels ──────────────────────────────────────────────── */
  surface: {
    panel: "bg-papaya-300/50 border-papaya-400/50",
    panelStrong: "bg-papaya-300/50 border-papaya-400",
    sectionHeader: "bg-papaya-300/80 hover:bg-papaya-400/80",
    sectionHeaderIcon: "bg-white border border-papaya-400 shadow-sm",
    chip: "bg-papaya-300/50 border border-papaya-400",
    chipHover: "hover:border-teal-200",
    divider: "border-papaya-300/50",
    dividerStrong: "border-papaya-400/50",
    cardBorder: "border-papaya-400",
    inputBorder: "border-papaya-400",
    inactivePhase: "bg-white text-foreground/70 hover:bg-papaya-300/50 border-papaya-400 shadow-sm",
    pageGradient: "from-papaya-50 via-white to-teal-50/30",
    authCard: "border-papaya-400/70",
    overlay: "bg-ink/20 backdrop-blur-sm",
    promptHeader: "bg-papaya-300 border-papaya-400",
    config: "bg-papaya-300/50 border border-papaya-400/50 shadow-inner",
    markdown: "bg-papaya-300/50 border border-papaya-400/60 shadow-sm",
    footer: "bg-papaya-300/50",
    emptyState: "bg-papaya-300/50 border-dashed border-papaya-400",
    cardRing: "ring-1 ring-ink/5",
  },

  /* ── Console / terminal ─────────────────────────────────────────────── */
  console: {
    bg: "bg-ink border-ink-400 shadow-inner",
    text: "text-papaya-400",
    timestamp: "text-ink-400",
    waiting: "text-ink-50",
    separator: "border-ink-400",
    directives: "bg-ink text-ink-50 border border-ink-400 shadow-inner",
  },

  /* ── Console event types ────────────────────────────────────────────── */
  consoleEvent: {
    phaseStart: "text-teal-300",
    phaseEnd: "text-green-400",
    error: "text-brandy-200",
    warning: "text-amber-400",
    phaseEndAlt: "text-tangerine-300", // Optional: if we want to use tangerine for specific phase ends
  },

  /* ── Brand / identity ───────────────────────────────────────────────── */
  brand: {
    icon: "text-teal",
    iconBg: "bg-teal shadow-xl shadow-teal-500/25",
    title: "text-ink",
    stepCircle: "bg-teal-50 text-teal",
    scopeBadge: "bg-teal-50 text-teal border-teal-50",
    syncIcon: "bg-tangerine rounded-lg shadow-lg shadow-tangerine/20",
    languageDot: "bg-tangerine",
    link: "text-teal hover:underline",
    ragBadge: "bg-teal-50 text-teal border-teal-200 hover:bg-teal-50",
    sectionDot: "bg-teal border-2 border-white shadow-sm",
    statTeal: "text-teal",
    statTangerine: "text-tangerine",
    statPurple: "text-purple-600",
    statEmerald: "text-teal-600",
    phaseOutcomeIcon: "bg-teal-50",
  },

  /* ── Feature tab accents ────────────────────────────────────────────── */
  featureTab: {
    claude: "text-green-600 data-[active=true]:border-green-600 data-[active=true]:bg-green-50/50",
    cursor: "text-purple-600 data-[active=true]:border-purple-600 data-[active=true]:bg-purple-50/50",
    debug: "text-amber-600 data-[active=true]:border-amber-600 data-[active=true]:bg-amber-50/50",
    debugBadge: "bg-amber-100 text-amber-700 border-amber-200",
  },

  /* ── Truncation overlay ─────────────────────────────────────────────── */
  truncation: {
    gradient: "from-brandy/10",
  },
} as const
