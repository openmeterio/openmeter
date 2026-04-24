# portal

<!-- archie:ai-start -->

> Lightweight portal-scoped client for end-customer access. Exposes a single OpenMeter class that authenticates via portalToken (Bearer) and provides only the portal meter query endpoint — intentionally restricted surface compared to the full admin client.

## Patterns

**Portal-only client reuses client utilities from parent** — Imports RequestOptions, operations/paths (schemas), and transformResponse/encodeDates from '../client/*'. All utility logic is shared; this file adds only portal-specific wiring. (`import type { RequestOptions } from '../client/common.js'
import type { operations, paths } from '../client/schemas.js'
import { encodeDates, transformResponse } from '../client/utils.js'`)
**portalToken replaces apiKey in Config** — Config type requires portalToken: string instead of apiKey. Auth header is always 'Bearer <portalToken>'. No optional key here — portal access always requires a token. (`export type Config = Pick<ClientOptions, 'baseUrl' | 'headers' | 'fetch' | 'Request' | 'requestInitExt'> & { portalToken: string }`)
**Explicit return type cast for meter query** — The portal query method casts the return value to the exact operations response type to provide proper TypeScript types to callers, since the generic transformResponse return type may be too broad. (`return transformResponse(resp) as operations['queryPortalMeter']['responses']['200']['content']['application/json']`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `index.ts` | Entire portal SDK in one file. Creates openapi-fetch client with same date-encoding querySerializer as main client, wires Bearer portalToken, exposes single query() method for /api/v1/portal/meters/{meterSlug}/query with Accept: application/json header. | Must pass Accept: application/json explicitly on the meter query — the endpoint can return CSV otherwise. Same querySerializer shape (form explode, deepObject) as the main client index.ts must be kept in sync. |

## Anti-Patterns

- Adding admin-level endpoints to the portal client — this client is intentionally restricted to portal-scoped access
- Redefining transformResponse or encodeDates instead of importing from '../client/utils.js'
- Making portalToken optional in Config — portal requests always require authentication

## Decisions

- **Portal client is a separate minimal class rather than a subset configuration of the main OpenMeter client** — Prevents accidentally exposing admin endpoints to end-customer portal tokens and makes the restricted surface explicit at the type level.

<!-- archie:ai-end -->
