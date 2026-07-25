import "@typespec/openapi";
import "@typespec/openapi3";

using TypeSpec.OpenAPI;

namespace Common;

/**
 * Alias to the top-level `Billing` namespace.
 *
 * Nesting operation interfaces under `Customers.Billing` (so the SDK generator
 * emits `customers.billing.*`) shadows the global `Billing` namespace for
 * unqualified `Billing.*` references inside `Customers`-scoped files. Use
 * `Common.BillingRoot.*` from those files to reach the top-level `Billing`
 * models unambiguously. An alias emits nothing, so the OpenAPI output is
 * unchanged.
 */
alias BillingRoot = Billing;

/**
 * Labels store metadata of an entity that can be used for filtering an entity list
 * or for searching across entity types.
 *
 * Keys must be of length 1-63 characters, and cannot start with "kong", "konnect",
 * "mesh", "kic", or "\_".
 */
@useRef("../../../../common/definitions/konnect_properties.yaml#/components/schemas/Labels")
@friendlyName("Labels")
model Labels is Record<string>;

/**
 * Public labels store information about an entity that can be used for filtering a
 * list of objects.
 */
@useRef("../../../../common/definitions/konnect_properties.yaml#/components/schemas/PublicLabels")
@friendlyName("PublicLabels")
model PublicLabels is Record<string>;
