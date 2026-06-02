import "../shared/index.tsp";
import "../billing/tax.tsp";
import "../features/index.tsp";
import "../tax/codes.tsp";
import "./price.tsp";

namespace ProductCatalog;

/**
 * Settlement mode for billing.
 *
 * Values:
 *
 * - `credit_then_invoice`: Credits are applied first, then any remainder is
 *   invoiced.
 * - `credit_only`: Usage is settled exclusively against credits.
 */
@friendlyName("BillingSettlementMode")
@summary("Settlement mode")
enum SettlementMode {
  CreditThenInvoice: "credit_then_invoice",
  CreditOnly: "credit_only",
}

/**
 * A rate card defines the pricing and entitlement of a feature or service.
 */
@friendlyName("BillingRateCard")
model RateCard {
  ...PickProperties<
    Shared.ResourceWithKey,
    "key" | "name" | "description" | "labels"
  >;

  /**
   * The feature associated with the rate card.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Feature reference")
  feature?: Shared.ResourceReference<Features.Feature>;

  /**
   * The billing cadence of the rate card. When null, the charge is one-time
   * (non-recurring). Only valid for flat prices.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Billing cadence")
  billing_cadence?: Shared.ISO8601Duration;

  /**
   * The price of the rate card.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Price")
  price: Price;

  /**
   * The payment term of the rate card. In advance payment term can only be used for
   * flat prices.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Payment term")
  payment_term?: PricePaymentTerm = PricePaymentTerm.InArrears;

  /**
   * Spend commitments for this rate card. Only applicable to usage-based prices
   * (unit, graduated, volume).
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Commitments")
  commitments?: SpendCommitments;

  /**
   * The discounts of the rate card.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Discounts")
  discounts?: Discounts;

  /**
   * The tax config of the rate card.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Tax config")
  tax_config?: RateCardTaxConfig;
}

/**
 * The tax config of the rate card.
 */
@friendlyName("BillingRateCardTaxConfig")
model RateCardTaxConfig {
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  behavior?: Billing.TaxBehavior;
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  code: Shared.ResourceReference<Tax.TaxCode>;
}

/**
 * Discount configuration for a rate card.
 */
@friendlyName("BillingRateCardDiscounts")
model Discounts {
  /**
   * Percentage discount applied to the price (0–100).
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @minValue(0)
  @maxValue(100)
  percentage?: float;

  /**
   * Number of usage units granted free before billing starts. Only applies to
   * usage-based lines (not flat fees). Usage is treated as zero until this amount is
   * exhausted.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  usage?: Shared.Numeric;
}

/**
 * The proration configuration of the rate card.
 */
@friendlyName("BillingRateCardProrationConfiguration")
model ProrationConfiguration {
  /**
   * The proration mode of the rate card.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Proration mode")
  mode: ProrationMode;
}

/**
 * The proration mode of the rate card.
 *
 * Values:
 *
 * - `no_proration`: No proration.
 * - `prorate_prices`: Prorate the price based on the time remaining in the billing
 *   period.
 */
@friendlyName("BillingRateCardProrationMode")
enum ProrationMode {
  NoProration: "no_proration",
  ProratePrices: "prorate_prices",
}
