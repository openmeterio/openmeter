import "@typespec/openapi3";
import "../shared/index.tsp";

using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * Usage discount.
 *
 * Usage discount means that the first N items are free. From billing perspective
 * this means that any usage on a specific feature is considered 0 until this discount
 * is exhausted.
 */
@friendlyName("DiscountUsage")
model DiscountUsage {
  /**
   * The quantity of the usage discount.
   *
   * Must be positive.
   */
  @summary("Usage")
  quantity: Shared.Numeric;
}

/**
 * Discount by type on a price
 */
@friendlyName("Discounts")
model Discounts {
  /**
   * The percentage discount.
   */
  percentage?: Shared.Percentage;

  /**
   * The usage discount.
   */
  usage?: Shared.Numeric;
}
