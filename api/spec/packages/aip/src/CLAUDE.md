# src

<!-- archie:ai-start -->

> Root composition layer for the v3 AIP TypeSpec spec: openmeter.tsp and konnect.tsp each declare a @service namespace and bind every domain operation interface to a @route path, while main.tsp imports both (plus test.tsp) to produce two distinct OpenAPI outputs (OpenMeter self-hosted and Kong Konnect) from one shared domain library. No model/operation definitions live here — only routing, tag metadata, and security scheme declarations.

## Patterns

**Route and tag binding exclusively at root** — Domain operation interfaces (Meters.MetersOperations, Billing.BillingProfilesOperations, ...) carry no @route or @tag in their own .tsp. @route/@tag are applied only in openmeter.tsp and konnect.tsp via interface extension. (`@route("/openmeter/profiles") @tag(Shared.BillingTag)
interface BillingProfilesEndpoints extends Billing.BillingProfilesOperations {}`)
**Tag metadata declared once per root namespace file** — Every tag used by any domain is registered with @tagMetadata(Shared.<X>Tag, #{description: Shared.<X>Description}) at the @service level in BOTH openmeter.tsp and konnect.tsp. Omitting it drops the tag description from the generated OpenAPI. (`@tagMetadata(Shared.BillingTag, #{ description: Shared.BillingDescription })`)
**Domain sub-operations imported via index.tsp barrels** — Each sub-domain folder exposes its types via an index.tsp barrel. Root files import the barrel (e.g. import "./billing/index.tsp") and reference namespace-qualified interfaces. Never import individual .tsp from the root. (`import "./billing/index.tsp";
interface BillingProfilesEndpoints extends Billing.BillingProfilesOperations {}`)
**Two-namespace compilation (OpenMeter vs Konnect)** — openmeter.tsp declares namespace OpenMeter (self-hosted), konnect.tsp declares namespace MeteringAndBilling (Kong). Both import the same sub-folders; feature differences (e.g. konnect lacks currencies/features/llmcost) are expressed by including/excluding imports and interface extensions. (`// konnect.tsp adds @useAuth + multi-region @server; openmeter.tsp imports currencies/features/llmcost that konnect omits`)
**Security schemes via @useRef to external YAML** — Konnect auth schemes (systemAccountAccessToken, personalAccessToken, konnectAccessToken) are TypeSpec model stubs spreading Http.BearerAuth with @useRef pointing at a shared security.yaml. Credential details never live in TypeSpec source. (`@useRef("../../../../common/definitions/security.yaml#/.../systemAccountAccessToken")
model systemAccountAccessToken { ...Http.BearerAuth; }`)
**Selective operation mounting via inline interface body** — When only a subset of operations from a domain interface is needed, declare an inline interface body referencing selected operations instead of extending the full interface. (`interface CustomerCreditGrantEndpoints {
  @route("/settlement/external")
  updateExternalSettlement is Customers.CustomerCreditGrantExternalSettlementOperations.updateExternalSettlement;
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.tsp` | Single compiler entry point — imports openmeter.tsp, konnect.tsp, and test.tsp. Pure import aggregator. | Never add route or model definitions here; domain logic here would appear in all namespace outputs. |
| `openmeter.tsp` | Self-hosted OpenMeter namespace: @service, @server, all @tagMetadata, and all @route/@tag interface extensions for OpenMeter endpoints. | Every new sub-folder needs both an import and a @route interface extension here; forgetting one silently drops endpoints. Add matching @tagMetadata or the tag description disappears. |
| `konnect.tsp` | Kong Konnect variant (namespace MeteringAndBilling): different @server URLs, multi-region servers, @useAuth, and Konnect security schemes. Omits currencies/features/llmcost domains. | When adding a domain, decide explicitly whether it belongs in Konnect too; missing domains are intentional only when omitted by design — mismatches elsewhere are bugs. |
| `test.tsp` | Standalone test namespace (namespace Test) exercising Common filter types via a /field-filters endpoint. Not compiled into production OpenAPI outputs. | Do not add production operations here; it exists only to verify filter-type codegen and OpenAPI shape. |

## Anti-Patterns

- Declaring @route or @tag inside a domain sub-folder's operations.tsp — routing is bound exclusively in openmeter.tsp and konnect.tsp
- Adding a new domain import in openmeter.tsp but forgetting the matching @tagMetadata — silently drops the tag description from generated OpenAPI
- Defining security scheme bodies inline instead of @useRef to external security.yaml — duplicates credential config outside the source of truth
- Importing a domain twice (once directly, once via its index.tsp) — causes duplicate symbol compilation errors
- Hand-editing api/v3/openapi.yaml or api/v3/api.gen.go — always regenerate via make gen-api then make generate after editing a .tsp here

## Decisions

- **Two root namespace files (openmeter.tsp, konnect.tsp) rather than one parameterized file** — OpenMeter and Konnect have different server URLs, security schemes, and feature sets; a single parameterized file would require conditional compilation TypeSpec doesn't support cleanly.
- **@route and @tag bound only at root, not in domain operation interfaces** — Lets the same domain operation interface be mounted at different paths or omitted entirely per root namespace without modifying the domain definition — enabling per-deployment feature gating.
- **Security schemes as TypeSpec model stubs with @useRef to external YAML** — Keeps Konnect OAuth/PAT scheme definitions in a shared security.yaml controlled by Kong infra, avoiding duplication and keeping TypeSpec free of secrets.

## Example: Add a new domain 'reports' to the OpenMeter namespace

```
// openmeter.tsp
import "./reports/index.tsp";
@tagMetadata(Shared.ReportsTag, #{ description: Shared.ReportsDescription })
// ...
@route("/openmeter/reports")
@tag(Shared.ReportsTag)
interface ReportsEndpoints extends Reports.ReportsOperations {}
// repeat in konnect.tsp only if Konnect needs it, then run: make gen-api && make generate
```

<!-- archie:ai-end -->
