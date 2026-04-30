# src

<!-- archie:ai-start -->

> Root composition layer for the v3 AIP TypeSpec spec: openmeter.tsp and konnect.tsp each declare a @service namespace and wire all domain operation interfaces to @route paths. main.tsp imports both variants, producing two distinct OpenAPI outputs from one shared domain library.

## Patterns

**Two-namespace compilation** — openmeter.tsp declares namespace OpenMeter (self-hosted), konnect.tsp declares namespace MeteringAndBilling (Kong Konnect). Both import the same domain sub-folders under src/; main.tsp imports both. Each produces its own OpenAPI YAML. (`// openmeter.tsp
namespace OpenMeter;
@route("/openmeter/meters") @tag(Shared.MetersTag)
interface MetersEndpoints extends Meters.MetersOperations {}`)
**Route+tag binding at root, not in domain folders** — Domain operation interfaces (Meters.MetersOperations, Billing.BillingProfilesOperations, etc.) carry no @route or @tag in their own .tsp files. @route and @tag are applied only here in the root namespace files via interface extension. (`@route("/openmeter/profiles") @tag(Shared.BillingTag)
interface BillingProfilesEndpoints extends Billing.BillingProfilesOperations {}`)
**Shared tag metadata declared once per root file** — Every tag used by any domain must be registered with @tagMetadata(Shared.<X>Tag, #{description: Shared.<X>Description}) at the @service level in both openmeter.tsp and konnect.tsp; omitting a tag here silently drops its description from the generated OpenAPI. (`@tagMetadata(Shared.BillingTag, #{ description: Shared.BillingDescription })`)
**Security schemes via @useRef to external YAML** — Auth schemes (systemAccountAccessToken, personalAccessToken, konnectAccessToken) are declared as TypeSpec model stubs that spread Http.BearerAuth and carry @useRef pointing to a shared security.yaml, keeping credential details out of TypeSpec. (`@useRef("../../../../common/definitions/security.yaml#/components/securitySchemes/systemAccountAccessToken")
model systemAccountAccessToken { ...Http.BearerAuth; }`)
**Domain sub-operation interfaces imported via index.tsp barrels** — Each sub-domain folder exposes its types via an index.tsp barrel. Root files import these barrels (e.g. import "./billing/index.tsp") and then reference namespace-qualified interfaces (Billing.BillingProfilesOperations). Never import individual .tsp files directly from root. (`import "./billing/index.tsp";
// then: interface BillingProfilesEndpoints extends Billing.BillingProfilesOperations {}`)
**Sub-route overrides via inline interface body** — When only a subset of operations from a domain interface is needed (or a non-standard sub-route is required), declare an inline interface body with selected operation references instead of extending the full interface. (`interface CustomerCreditGrantEndpoints {
  @route("/settlement/external")
  updateExternalSettlement is Customers.CustomerCreditGrantExternalSettlementOperations.updateExternalSettlement;
}`)

## Key Files

| File            | Role                                                                                                                                                                                                               | Watch For                                                                                                                                                                  |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `main.tsp`      | Single entry point that imports both openmeter.tsp and konnect.tsp; the TypeSpec compiler starts here.                                                                                                             | Never add route or model definitions here; it is a pure import aggregator.                                                                                                 |
| `openmeter.tsp` | Self-hosted OpenMeter namespace: declares @service, @server, @tagMetadata, @useAuth (implicit for v3), and all @route/@tag interface extensions.                                                                   | Every new domain sub-folder must add both an import and a @route interface extension here; forgetting one silently drops the endpoints from the OpenMeter OpenAPI output.  |
| `konnect.tsp`   | Kong Konnect variant: same pattern as openmeter.tsp but different @server URLs, multi-region servers, Konnect-specific security schemes, and namespace MeteringAndBilling. Does not include productcatalog routes. | When adding a new domain, decide whether it belongs in Konnect too; tax/ is present in konnect.tsp but productcatalog/ is not — mismatches are intentional feature-gating. |

## Anti-Patterns

- Declaring @route or @tag inside a domain sub-folder's operations.tsp — routing is exclusively bound in openmeter.tsp and konnect.tsp
- Adding a new domain import in openmeter.tsp but forgetting the matching @tagMetadata declaration — drops the tag description from generated OpenAPI
- Defining security scheme bodies inline in konnect.tsp instead of @useRef to external YAML — duplicates credential config
- Adding the same sub-domain import twice (once directly, once via index.tsp) — causes duplicate symbol errors in the compiler
- Hand-editing api/v3/openapi.yaml — always regenerate via `make gen-api` after modifying any .tsp file here

## Decisions

- **Two root namespace files (openmeter.tsp, konnect.tsp) rather than one parameterized file** — OpenMeter self-hosted and Kong Konnect have different server URLs, security schemes, and feature sets (e.g. productcatalog only in OpenMeter); a single parameterized file would require conditional compilation not supported cleanly in TypeSpec.
- **@route and @tag are bound only at the root level, not in domain operation interfaces** — Allows the same domain operation interface to be mounted at different paths or omitted entirely in each root namespace without modifying the domain definition.
- **Security schemes declared as TypeSpec model stubs with @useRef to external YAML** — Keeps Konnect-specific OAuth/PAT scheme definitions in a shared security.yaml controlled by Kong infra, avoiding duplication and preventing TypeSpec compilation from depending on secrets.

## Example: Adding a new domain (e.g. 'reports') to the OpenMeter namespace

```
// 1. Create api/spec/packages/aip/src/reports/index.tsp importing all reports .tsp files
// 2. In openmeter.tsp:
import "./reports/index.tsp";
// Add tagMetadata at @service level:
@tagMetadata(Shared.ReportsTag, #{ description: Shared.ReportsDescription })
// Add route binding:
@route("/openmeter/reports")
@tag(Shared.ReportsTag)
interface ReportsEndpoints extends Reports.ReportsOperations {}
// 3. Run: make gen-api
```

<!-- archie:ai-end -->
