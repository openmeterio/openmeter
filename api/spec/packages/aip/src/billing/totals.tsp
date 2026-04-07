import "../shared/index.tsp";

namespace Billing;

/**
 * Totals contains the summaries of all calculations for a billing resource.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "taxes_* totals are intentionally kept flat for API compatibility"
@friendlyName("BillingTotals")
model BillingTotals {
  /**
   * The total value of the resource before taxes, discounts and commitments.
   */
  @visibility(Lifecycle.Read)
  @summary("Amount")
  amount: Shared.Numeric;

  /**
   * The total tax amount applied to the resource.
   */
  @visibility(Lifecycle.Read)
  @summary("Taxes total")
  taxes_total: Shared.Numeric;

  /**
   * The total tax amount already included in the resource amount.
   */
  @visibility(Lifecycle.Read)
  @summary("Inclusive taxes total")
  taxes_inclusive_total: Shared.Numeric;

  /**
   * The total tax amount added on top of the resource amount.
   */
  @visibility(Lifecycle.Read)
  @summary("Exclusive taxes total")
  taxes_exclusive_total: Shared.Numeric;

  /**
   * The total amount contributed by additional charges.
   */
  @visibility(Lifecycle.Read)
  @summary("Charges total")
  charges_total: Shared.Numeric;

  /**
   * The total amount deducted through discounts.
   */
  @visibility(Lifecycle.Read)
  @summary("Discounts total")
  discounts_total: Shared.Numeric;

  /**
   * The total amount deducted through credits before taxes are applied.
   */
  @visibility(Lifecycle.Read)
  @summary("Credits total")
  credits_total: Shared.Numeric;

  /**
   * The final total value of the resource after taxes, discounts and commitments.
   */
  @visibility(Lifecycle.Read)
  @summary("Total")
  total: Shared.Numeric;
}
