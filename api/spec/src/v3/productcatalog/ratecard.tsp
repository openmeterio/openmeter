import "../shared/index.tsp";
import "./price.tsp";
import "../billing/tax.tsp";
import "./feature.tsp";
import "../tax/codes.tsp";

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
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Billing cadence")
  billing_cadence: Shared.ISO8601Duration;

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
   * The commitments of the rate card.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Commitments")
  commitments?: SpendCommitments;

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
