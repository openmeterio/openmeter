import { createContext } from 'react'

export interface LocalEditCtx {
  toggleRule: (id: string, action: 'adopt' | 'reject' | 'disable' | 'enable') => Promise<void>
  editRule: (id: string, patch: Record<string, string>) => Promise<void>
}

// LocalPage wraps ReportPage with a Provider whose value is `null` until
// V2-i3 lands the inline rule editor. ReportPage's RuleControls injection
// will short-circuit when ctx is null, keeping the share-mode build
// completely free of editor code.
export const LocalEditContext = createContext<LocalEditCtx | null>(null)
