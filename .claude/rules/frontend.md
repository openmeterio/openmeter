---
paths:
  - **/*
---

## Frontend Architecture

**Framework:** none (generated TypeScript SDK only)

**Styling:** Not applicable — no UI, no stylesheets, no component library, no design tokens. The api/client/javascript package uses Biome (biome.json) solely as its linter/formatter; it is not a styling system.

**State management:** Not applicable — there is no application UI. The only client surface is the generated @openmeter/sdk npm package (api/client/javascript), an OpenAPI-derived API client. The optional React sub-export (api/client/javascript/src/react/context.tsx) provides only a context/provider/hook to inject an OpenMeter client instance; it holds no application state of its own.
  - Server state: No client-side server-state library (no React Query/SWR/Apollo). Each SDK resource method calls openapi-fetch (this.client.GET/POST/PUT/DELETE/PATCH) and pipes the raw result through transformResponse (api/client/javascript/src/client/utils.ts), which throws HTTPError on resp.error or status >= 400 and recursively decodes ISO date strings to JS Date objects via decodeDates. Callers own any caching/state.
  - Local state: Not applicable — no UI components with local state exist.

**Conventions:**
- No application UI exists: OpenMeter is a Go usage-metering backend (frontend_ratio = 0.0). The only client-facing surface is the generated @openmeter/sdk npm package under api/client/javascript.
- SDK is generated-first, wrapper-second: src/client/schemas.ts (openapi-typescript via scripts/generate.ts) and src/zod/index.ts (orval, per orval.config.ts) are generated from the OpenAPI spec and carry 'Do not edit manually' headers — never hand-edited.
- Four named sub-package exports in package.json: '.' (admin client -> dist/index.js), './portal' (portal-token client), './react' (React context/provider/hook), './zod' (orval-generated Zod schemas). New resource classes are registered as public fields on the root OpenMeter class in src/client/index.ts.
- One file per domain resource under src/client/ (addons.ts, apps.ts, billing.ts, customers.ts, events.ts, meters.ts, notifications.ts, plans.ts, subscriptions.ts, etc.); each exports a class taking Client<paths> and returning transformResponse(resp) — callers never read resp.data directly.
- Centralized date handling (api/client/javascript/src/client/utils.ts): encodeDates (Date -> ISO string) is applied in the querySerializer; decodeDates (ISO string -> Date, matched by ISODateFormat regex) runs inside transformResponse.
- Error handling: transformResponse throws HTTPError (api/client/javascript/src/client/common.ts) on resp.error or HTTP status >= 400; HTTPError.fromResponse extracts problem.detail/type/title/status when Content-Type is application/problem+json (RFC 7807); isHTTPError is the type guard.
- React integration (src/react/context.tsx) is the only .tsx file; it carries the 'use client' directive (first line) for Next.js App Router compatibility, re-exports the portal surface wholesale (export * from '../portal/index.js'), and provides only Context plumbing (OpenMeterContext / OpenMeterProvider / useOpenMeter) — no rendered UI.
- Tooling: pnpm package manager (pinned in package.json), dual ESM/CJS build via @knighted/duel, Node >= 22 engine, Vitest tests colocated as *.spec.ts (e.g. src/client/events.spec.ts) using @fetch-mock/vitest. Published library supports React >=18 via peerDependency.
- The quickstart/ directory is a Go + docker-compose deployment example (quickstart/quickstart_test.go, docker-compose.yaml, config.yaml), not a web UI.