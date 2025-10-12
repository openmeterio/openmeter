import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

namespace OpenMeter.ProductCatalog;

/**
 * Cost
 */
@friendlyName("Cost")
model Cost {
  /**
   * The cost type
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  kind: CostKind;

  /**
   * Provider ID
   *
   * Only if type "provider"
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  providerId?: string;

  /**
   * Currency for cost
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  currency: CurrencyCode;

  /**
   * The per unit amount
   *
   * Writeable only if type "manual"
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  perUnitAmount: Numeric;
}

/**
 * Cost create input
 */
@friendlyName("CostCreateInput")
model CostCreateInput {
  ...TypeSpec.Rest.Resource.ResourceCreateModel<OmitProperties<
    Cost,
    "perUnitAmount"
  >>;

  /**
   * The per unit amount
   *
   * Writeable only if type "manual"
   */
  @visibility(Lifecycle.Create)
  perUnitAmount?: Numeric;
}

/**
 * The cost kind
 */
@friendlyName("CostKind")
enum CostKind {
  manual: "manual",
  provider: "provider",
}
