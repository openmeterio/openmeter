# portal

<!-- archie:ai-start -->

> Restricted portal-scoped JavaScript client for end-customer access. Exposes a single OpenMeter class authenticated via a required portalToken that provides only the portal meter-query endpoint — intentionally a much smaller surface than the full admin client.

## Patterns

**Reuse utilities from the parent client** — All shared logic (RequestOptions, paths/operations types, transformResponse, encodeDates) is imported from ../client/common.js, ../client/schemas.js, and ../client/utils.js. This file adds only portal-specific wiring. (`import type { operations, paths } from '../client/schemas.js'
import { encodeDates, transformResponse } from '../client/utils.js'`)
**portalToken replaces apiKey and is required** — Config requires portalToken: string (not optional). The Authorization header is always `Bearer ${config.portalToken}`. The querySerializer (array form/explode, object deepObject/explode) must stay in sync with the main client. (`headers: { ...config.headers, Authorization: `Bearer ${config.portalToken}` }`)
**Explicit Accept: application/json on the meter query** — The portal meter-query endpoint returns CSV when no Accept header is set. Always pass Accept: application/json and cast the result to the exact operations response type. (`const resp = await this.client.GET('/api/v1/portal/meters/{meterSlug}/query', { headers: { Accept: 'application/json' }, ... }); return transformResponse(resp) as operations['queryPortalMeter']['responses']['200']['content']['application/json']`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `index.ts` | The entire portal SDK: creates an openapi-fetch client, wires Bearer portalToken and the querySerializer, exposes a single query() method for /api/v1/portal/meters/{meterSlug}/query. | Must pass Accept: application/json explicitly or the endpoint returns CSV. querySerializer shape must stay in sync with the main client's index.ts. |

## Anti-Patterns

- Adding admin-level endpoints to the portal client — it is intentionally restricted to portal-scoped access.
- Redefining transformResponse or encodeDates instead of importing from ../client/utils.js.
- Making portalToken optional in Config — portal requests always require authentication.
- Omitting Accept: application/json on the meter query — the endpoint falls back to CSV.

## Decisions

- **The portal client is a separate minimal class, not a subset configuration of the main OpenMeter client.** — Prevents accidentally exposing admin endpoints to end-customer portal tokens and makes the restricted surface explicit at the type level.

## Example: Querying a portal meter with the required Accept header

```
const resp = await this.client.GET('/api/v1/portal/meters/{meterSlug}/query', {
  headers: { Accept: 'application/json' },
  params: { path: { meterSlug }, query },
  ...options,
})
return transformResponse(resp) as operations['queryPortalMeter']['responses']['200']['content']['application/json']
```

<!-- archie:ai-end -->
