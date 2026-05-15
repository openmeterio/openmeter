# react

<!-- archie:ai-start -->

> Thin React integration layer providing a context/provider/hook trio (OpenMeterContext, OpenMeterProvider, useOpenMeter) so React apps can inject and consume a portal OpenMeter client anywhere in the component tree. This is a single file; all SDK logic lives in ../portal/.

## Patterns

**Re-export portal surface wholesale** — context.tsx does `export * from '../portal/index.js'` so all portal SDK types and functions are available from the react import path. Consumers import from '@openmeter/sdk/react' and get everything. (`export * from '../portal/index.js'`)
**Null-initialized context with undefined-throws hook** — OpenMeterContext default is null (unauthenticated state). useOpenMeter throws only if context is undefined (outside provider). Callers check for null themselves to handle unauthenticated state. (`const ctx = createContext<OpenMeter | null>(null)
export function useOpenMeter() {
  const context = useContext(OpenMeterContext)
  if (typeof context === 'undefined') { throw new Error('useOpenMeter must be used within a OpenMeterProvider') }
  return context
}`)
**'use client' directive at top** — File is marked 'use client' for Next.js App Router compatibility. Any addition to this file must preserve this directive at the very first line. (`'use client'`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `context.tsx` | Entire folder. Exports OpenMeterContext, OpenMeterProvider, useOpenMeter, and re-exports all portal SDK symbols. | Do not add business logic or API call methods here. All SDK behaviour belongs in ../portal/. New exports should come via re-export from ../portal/, not defined locally. Do not remove 'use client'. |

## Anti-Patterns

- Defining SDK methods or API call logic directly in context.tsx instead of delegating to ../portal/
- Removing the 'use client' directive — breaks Next.js App Router
- Creating a second context or provider for different API concerns — extend the OpenMeter type in portal/ instead
- Throwing inside useOpenMeter when value is null — null is a valid unauthenticated state, only undefined should throw

## Decisions

- **Context default is null, not a noop object** — Forces callers to handle the unauthenticated state explicitly rather than silently calling noop methods that appear to work but do nothing.
- **All portal exports are re-exported from this entry point** — Consumers importing from the react path get the full portal SDK surface without a second import statement, keeping usage ergonomic while preserving a single source of truth in ../portal/.

## Example: Wrap app with provider and consume client in a component

```
'use client'
import { OpenMeterProvider, useOpenMeter } from '@openmeter/sdk/react'
import { OpenMeter } from '@openmeter/sdk/portal'

const client = new OpenMeter({ baseUrl: 'https://openmeter.cloud', portalToken: 'tok_...' })

function App() {
  return (
    <OpenMeterProvider value={client}>
      <Dashboard />
    </OpenMeterProvider>
  )
}

function Dashboard() {
// ...
```

<!-- archie:ai-end -->
