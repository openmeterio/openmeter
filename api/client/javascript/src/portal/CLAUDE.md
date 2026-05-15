# portal

<!-- archie:ai-start -->

> Restricted portal-scoped client for end-customer access. Exposes a single OpenMeter class authenticated via portalToken (required, not optional) that provides only the portal meter query endpoint — intentionally a much smaller surface than the full admin client.

## Patterns

**Imports utilities from parent client, not redefined** — All utility logic (RequestOptions, paths/operations types, transformResponse, encodeDates) is imported from '../client/common.js', '../client/schemas.js', and '../client/utils.js'. This file adds only portal-specific wiring. (`import type { RequestOptions } from '../client/common.js'
import type { operations, paths } from '../client/schemas.js'
import { encodeDates, transformResponse } from '../client/utils.js'`)
**portalToken replaces apiKey in Config** — Config type requires portalToken: string (not optional). Auth header is always `Bearer ${config.portalToken}`. The same querySerializer shape (form explode, deepObject) as the main client must be kept in sync. (`export type Config = Pick<ClientOptions, 'baseUrl' | 'headers' | 'fetch' | 'Request' | 'requestInitExt'> & { portalToken: string }`)
**Accept: application/json header on meter query** — The portal meter query endpoint can return CSV if Accept header is not set. Always pass Accept: application/json explicitly, and cast the return type to the exact operations response type. (`const resp = await this.client.GET('/api/v1/portal/meters/{meterSlug}/query', { headers: { Accept: 'application/json' }, params: { path: { meterSlug }, query }, ...options })
return transformResponse(resp) as operations['queryPortalMeter']['responses']['200']['content']['application/json']`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `index.ts` | Entire portal SDK in one file. Creates openapi-fetch client, wires Bearer portalToken, exposes single query() method for /api/v1/portal/meters/{meterSlug}/query. | Must pass Accept: application/json explicitly on the meter query — endpoint returns CSV otherwise. querySerializer shape must stay in sync with main client index.ts. |

## Anti-Patterns

- Adding admin-level endpoints to the portal client — this client is intentionally restricted to portal-scoped access only
- Redefining transformResponse or encodeDates instead of importing from '../client/utils.js'
- Making portalToken optional in Config — portal requests always require authentication
- Omitting Accept: application/json on the meter query — endpoint falls back to CSV response

## Decisions

- **Portal client is a separate minimal class rather than a subset configuration of the main OpenMeter client** — Prevents accidentally exposing admin endpoints to end-customer portal tokens and makes the restricted surface explicit at the type level with no risk of scope creep.

<!-- archie:ai-end -->
