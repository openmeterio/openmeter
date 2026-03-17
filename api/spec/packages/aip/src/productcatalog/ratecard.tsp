import "../shared/index.tsp";
import "../billing/tax.tsp";
import "../features/feature.tsp";
import "../tax/codes.tsp";
import "./price.tsp";
import "./unitconfig.tsp";

namespace ProductCatalog;

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
  feature?: Shared.ResourceKeyReference<Feature>;

  /**
   * The billing cadence of the rate card.
   * When null, the charge is one-time (non-recurring). Only valid for flat prices.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Billing cadence")
  billing_cadence: Shared.ISO8601Duration | null;

  /**
   * The price of the rate card.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Price")
  price: Price;

  /**
   * The payment term of the rate card.
   * In advance payment term can only be used for flat prices.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Payment term")
  payment_term?: PricePaymentTerm = PricePaymentTerm.InArrears;

  /**
   * Unit conversion configuration applied before pricing.
   *
   * When set, raw metered quantities are transformed into billing units
   * before being passed to the price model and entitlement checks.
   * Tier boundaries (up_to_amount) are also expressed in billing units.
   * Only applicable to usage-based prices (unit, graduated, volume).
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Unit conversion config")
  unit_config?: UnitConfig;

  /**
   * Spend commitments for this rate card.
   * Only applicable to usage-based prices (unit, graduated, volume).
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
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  behavior?: Billing.TaxBehavior;
  #suppress "@openmeter/api-spec/doc-decorator" "shared model"
  code: Shared.ResourceWithKeyReference<Tax.TaxCode>;
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
  percentage?: Shared.Percentage;

  /**
   * Number of usage units granted free before billing starts.
   * Only applies to usage-based lines (not flat fees). Usage is treated as zero
   * until this amount is exhausted.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  usage?: Shared.Numeric;
}

