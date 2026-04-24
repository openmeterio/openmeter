# react

<!-- archie:ai-start -->

> Thin React integration layer for the OpenMeter JavaScript SDK — provides a context/provider/hook trio so React apps can inject an OpenMeter client instance and consume it anywhere in the component tree. This is the only file in the folder; all SDK logic lives in ../portal/index.js which is re-exported wholesale.

## Patterns

**Re-export portal surface** — The file does `export * from '../portal/index.js'` so all portal SDK types/functions are available from the react import path without duplication. (`import { OpenMeterContext, useOpenMeter, IngestEvent } from '@openmeter/sdk/react'`)
**Null-initialized context** — OpenMeterContext is created with `null` as default; useOpenMeter throws if context is `undefined` (uninitialised, not null). Callers must check for null (unauthenticated) themselves. (`const ctx = createContext<OpenMeter | null>(null)`)
**'use client' directive** — File is marked 'use client' at the top — required for Next.js App Router compatibility. Any addition to this file must preserve that directive. (`'use client'`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `context.tsx` | Entire folder is this single file. Exports OpenMeterContext, OpenMeterProvider, useOpenMeter, and all portal SDK symbols. | Do not add business logic here. All SDK behaviour belongs in ../portal/. Adding new exports here should come from ../portal/ via re-export, not defined locally. |

## Anti-Patterns

- Defining SDK methods or API call logic directly in this file instead of delegating to ../portal/
- Removing the 'use client' directive — breaks Next.js App Router
- Creating a second context or second provider for different API concerns — extend OpenMeter type in portal instead
- Throwing inside useOpenMeter when value is null (currently it only throws on undefined — null is a valid 'not logged in' state)

## Decisions

- **Context default is null, not a noop object** — Forces callers to handle the unauthenticated state explicitly rather than silently calling noop methods.
- **All portal exports are re-exported from this entry point** — Consumers importing from the react path get the full SDK surface without a second import statement, keeping usage ergonomic.

## Example: Wrap app with provider and consume in a component

```
import { OpenMeterProvider, useOpenMeter } from '@openmeter/sdk/react'
import { OpenMeter } from '@openmeter/sdk/portal'

const client = new OpenMeter({ baseUrl: '...' })

function App() {
  return (
    <OpenMeterProvider value={client}>
      <Dashboard />
    </OpenMeterProvider>
  )
}

function Dashboard() {
  const om = useOpenMeter() // throws if used outside provider
// ...
```

<!-- archie:ai-end -->
