# src

<!-- archie:ai-start -->

> Root composition layer for the v3 AIP TypeSpec spec: openmeter.tsp and konnect.tsp declare @service namespaces and bind all domain operation interfaces to @route paths, while main.tsp imports both to produce two distinct OpenAPI outputs (OpenMeter self-hosted and Kong Konnect) from one shared domain library. No model or operation definitions live here — only routing, tag metadata, and security scheme declarations.

## Patterns

**Route and tag binding exclusively at root** — Domain operation interfaces (e.g. Meters.MetersOperations, Billing.BillingProfilesOperations) carry no @route or @tag in their own .tsp files. @route and @tag are applied only in openmeter.tsp and konnect.tsp via interface extension syntax: `@route('/openmeter/meters') @tag(Shared.MetersTag) interface MetersEndpoints extends Meters.MetersOperations {}` (`@route("/openmeter/profiles") @tag(Shared.BillingTag)
interface BillingProfilesEndpoints extends Billing.BillingProfilesOperations {}`)
**Tag metadata declared once per root namespace file** — Every tag used by any domain must be registered with @tagMetadata(Shared.<X>Tag, #{description: Shared.<X>Description}) at the @service level in both openmeter.tsp and konnect.tsp. Omitting a tag here silently drops its description from the generated OpenAPI. (`@tagMetadata(Shared.BillingTag, #{ description: Shared.BillingDescription })`)
**Domain sub-operations imported via index.tsp barrels** — Each sub-domain folder exposes its types via an index.tsp barrel. Root files import these barrels (e.g. `import "./billing/index.tsp"`) and reference namespace-qualified interfaces. Never import individual .tsp files directly from the root namespace files. (`import "./billing/index.tsp";
// then: interface BillingProfilesEndpoints extends Billing.BillingProfilesOperations {}`)
**Security schemes via @useRef to external YAML** — Auth schemes (systemAccountAccessToken, personalAccessToken, konnectAccessToken) are declared as TypeSpec model stubs spreading Http.BearerAuth with @useRef pointing to a shared security.yaml. Credential details never live in TypeSpec source. (`@useRef("../../../../common/definitions/security.yaml#/components/securitySchemes/systemAccountAccessToken")
model systemAccountAccessToken { ...Http.BearerAuth; }`)
**Two-namespace compilation (openmeter vs konnect)** — openmeter.tsp declares namespace OpenMeter (self-hosted), konnect.tsp declares namespace MeteringAndBilling (Kong Konnect). Both import the same domain sub-folders; feature differences (e.g. productcatalog only in OpenMeter) are expressed by including/excluding the relevant import and interface extension. (`// openmeter.tsp only:
import "./productcatalog/index.tsp";
@route("/openmeter/plans") @tag(Shared.ProductCatalogTag)
interface PlansEndpoints extends ProductCatalog.PlanOperations {}`)
**Selective operation mounting via inline interface body** — When only a subset of operations from a domain interface is needed, declare an inline interface body with selected operation references instead of extending the full interface. (`interface CustomerCreditGrantEndpoints {
  @route("/settlement/external")
  updateExternalSettlement is Customers.CustomerCreditGrantExternalSettlementOperations.updateExternalSettlement;
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.tsp` | Single compiler entry point — imports openmeter.tsp, konnect.tsp, and test.tsp. TypeSpec compilation starts here. | Never add route or model definitions here; it is a pure import aggregator. Adding domain logic here causes it to appear in all three namespace outputs. |
| `openmeter.tsp` | Self-hosted OpenMeter namespace: declares @service, @server, all @tagMetadata entries, and all @route/@tag interface extensions for OpenMeter endpoints. | Every new domain sub-folder must add both an import and a @route interface extension here; forgetting one silently drops the endpoints from the OpenMeter OpenAPI output. Also add matching @tagMetadata or tag descriptions disappear. |
| `konnect.tsp` | Kong Konnect variant: same pattern as openmeter.tsp but with different @server URLs, multi-region servers, Konnect-specific security schemes, and namespace MeteringAndBilling. Does not include productcatalog routes. | When adding a new domain, decide explicitly whether it belongs in Konnect too. Missing productcatalog is intentional; mismatches for other domains are bugs. |
| `test.tsp` | Standalone test namespace (namespace Test) exercising Common filter types. Not compiled into production OpenAPI outputs. | Do not add production operations here. Its @route /field-filters endpoint exists only to verify filter type codegen and OpenAPI output shape. |

## Anti-Patterns

- Declaring @route or @tag inside a domain sub-folder's operations.tsp — routing is exclusively bound in openmeter.tsp and konnect.tsp
- Adding a new domain import in openmeter.tsp but forgetting the matching @tagMetadata declaration — silently drops the tag description from generated OpenAPI
- Defining security scheme bodies inline in konnect.tsp instead of @useRef to external YAML — duplicates credential config outside the security.yaml source of truth
- Adding a domain import twice (once directly, once via its index.tsp) — causes duplicate symbol compilation errors
- Hand-editing api/v3/openapi.yaml or api/v3/api.gen.go — always regenerate via `make gen-api` then `make generate` after modifying any .tsp file here

## Decisions

- **Two root namespace files (openmeter.tsp, konnect.tsp) rather than one parameterized file** — OpenMeter self-hosted and Kong Konnect have different server URLs, security schemes, and feature sets (productcatalog only in OpenMeter); a single parameterized file would require conditional compilation not supported cleanly in TypeSpec.
- **@route and @tag bound only at root level, not in domain operation interfaces** — Allows the same domain operation interface to be mounted at different paths or omitted entirely in each root namespace without modifying the domain definition — enabling per-deployment feature gating.
- **Security schemes declared as TypeSpec model stubs with @useRef to external YAML** — Keeps Konnect-specific OAuth/PAT scheme definitions in a shared security.yaml controlled by Kong infra, avoiding duplication and preventing TypeSpec compilation from depending on secrets.

## Example: Adding a new domain 'reports' to the OpenMeter namespace

```
// 1. Create api/spec/packages/aip/src/reports/index.tsp importing all reports .tsp files
// 2. In openmeter.tsp:
import "./reports/index.tsp";
// At @service level, register tag:
@tagMetadata(Shared.ReportsTag, #{ description: Shared.ReportsDescription })
// Add route binding:
@route("/openmeter/reports")
@tag(Shared.ReportsTag)
interface ReportsEndpoints extends Reports.ReportsOperations {}
// 3. If also needed in Konnect, repeat steps 2-3 in konnect.tsp
// 4. Run: make gen-api && make generate
```

<!-- archie:ai-end -->
