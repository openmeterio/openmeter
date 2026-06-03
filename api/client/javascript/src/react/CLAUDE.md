# react

<!-- archie:ai-start -->

> Thin React integration providing a context/provider/hook trio (OpenMeterContext, OpenMeterProvider, useOpenMeter) so React apps can inject and consume a portal OpenMeter client anywhere in the tree. Single file; all SDK logic lives in ../portal/.

## Patterns

**Re-export the portal surface wholesale** — context.tsx does `export * from '../portal/index.js'` so all portal SDK types/functions are available from the react import path. New exports should come via re-export, not local definitions. (`export * from '../portal/index.js'`)
**Null-initialized context with undefined-throws hook** — OpenMeterContext defaults to null (unauthenticated). useOpenMeter throws only when the context is undefined (used outside a provider); callers handle null themselves. (`const ctx = createContext<OpenMeter | null>(null)
if (typeof context === 'undefined') { throw new Error('useOpenMeter must be used within a OpenMeterProvider') }`)
**'use client' directive at top** — The file is marked 'use client' for Next.js App Router compatibility. Any change must preserve this directive as the very first line. (`'use client'`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `context.tsx` | Entire folder. Exports OpenMeterContext, OpenMeterProvider, useOpenMeter and re-exports all portal SDK symbols. | No business logic or API methods here — all SDK behaviour belongs in ../portal/. New exports via re-export, not local definitions. Do not remove 'use client'. |

## Anti-Patterns

- Defining SDK methods or API call logic in context.tsx instead of delegating to ../portal/.
- Removing the 'use client' directive — breaks Next.js App Router.
- Creating a second context/provider for different concerns — extend the OpenMeter type in portal/ instead.
- Throwing inside useOpenMeter when the value is null — null is a valid unauthenticated state, only undefined should throw.

## Decisions

- **Context default is null, not a noop object.** — Forces callers to handle the unauthenticated state explicitly rather than silently calling noop methods that appear to work.
- **All portal exports are re-exported from this entry point.** — Consumers importing from the react path get the full portal SDK without a second import, while ../portal/ remains the single source of truth.

## Example: Wrap the app with the provider and consume the client

```
'use client'
import { OpenMeterProvider, useOpenMeter } from '@openmeter/sdk/react'
import { OpenMeter } from '@openmeter/sdk/portal'

const client = new OpenMeter({ baseUrl: 'https://openmeter.cloud', portalToken: 'tok_...' })
function App() { return (<OpenMeterProvider value={client}><Dashboard /></OpenMeterProvider>) }
```

<!-- archie:ai-end -->
