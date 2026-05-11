import "../../shared/index.tsp";
import "../../currencies/index.tsp";

/**
 * A credit adjustment can be used to make manual adjustments to a customer's
 * credit balance.
 *
 * Supported use-cases:
 *
 * - Usage correction
 */
@friendlyName("BillingCreditAdjustment")
@summary("Credit adjustment")
model CreditAdjustment {
  ...PickProperties<Shared.Resource, "name" | "description" | "labels">;

  /**
   * The currency of the granted credits.
   */
  @visibility(Lifecycle.Create)
  currency: Currencies.CurrencyCode;

  /**
   * Granted credit amount.
   */
  @visibility(Lifecycle.Create)
  amount: Shared.Numeric;
}
