---
paths:
  - **/*.jsx
  - **/*.tsx
---

## Frontend Architecture

**Framework:** none (generated TypeScript SDK only)

**Styling:** Not applicable — no UI, no stylesheets, no component library, no design tokens. The api/client/javascript package uses Biome (biome.json) solely as its linter/formatter (single quotes, no semicolons, useConst, useAsConstAssertion); it is not a styling system.

**State management:** Not applicable — there is no application UI. The only client surface is the generated @openmeter/sdk npm package (api/client/javascript), an OpenAPI-derived API client. The optional React sub-export (api/client/javascript/src/react/context.tsx) provides only a context/provider/hook to inject an OpenMeter client instance; it holds no application state of its own.
  - Server state: No client-side server-state library (no React Query/SWR/Apollo). Each SDK resource method calls openapi-fetch (this.client.GET/POST/PUT/DELETE/PATCH) and pipes the raw result through transformResponse (api/client/javascript/src/client/utils.ts), which throws HTTPError on resp.error or status >= 400 and recursively decodes ISO date strings to JS Date objects. Callers own any caching/state.
  - Local state: Not applicable — no UI components with local state exist.

**Conventions:**
- No application UI exists: OpenMeter is a Go usage-metering backend (frontend_ratio = 0.0). The only client-facing surface is the generated @openmeter/sdk npm package under api/client/javascript.
- SDK is generated-first, wrapper-second: src/client/schemas.ts (openapi-typescript) and src/zod/index.ts (orval) are generated from api/openapi.cloud.yaml via scripts/generate.ts / orval.config.ts and must never be hand-edited (api/client/javascript/CLAUDE.md).
- Four named sub-package exports in package.json: '.' (admin client -> dist/index.js), './portal' (portal-token client), './react' (React context/provider/hook), './zod' (orval-generated Zod schemas). New resource classes must be registered as public fields on the root OpenMeter class in src/client/index.ts.
- One file per domain resource under src/client/ (addons.ts, apps.ts, billing.ts, customers.ts, events.ts, meters.ts, etc.); each exports a class taking Client<paths> and returning transformResponse(resp) — callers never read resp.data directly.
- Centralized date handling: encodeDates (Date -> ISO string) is applied in the querySerializer; decodeDates (ISO string -> Date) runs inside transformResponse (api/client/javascript/src/client/utils.ts).
- Error handling: transformResponse throws HTTPError (api/client/javascript/src/client/common.ts) on resp.error or HTTP status >= 400; HTTPError.fromResponse extracts problem.detail when Content-Type is application/problem+json (RFC 7807); isHTTPError is the type guard.
- React integration (src/react/context.tsx) carries the 'use client' directive (first line) for Next.js App Router compatibility, re-exports the portal surface wholesale (export * from '../portal/index.js'), and distinguishes null (valid unauthenticated state) from undefined (throws in useOpenMeter).
- Tooling: pnpm package manager (pinned in package.json), dual ESM/CJS build via duel, Node >= 22 engine, Vitest tests colocated as *.spec.ts (e.g. src/client/events.spec.ts) using @fetch-mock/vitest.