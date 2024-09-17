import "./types.tsp";
import "./billing.tsp";
import "./discounts.tsp";
import "./prices.tsp";
import "./entitlements.tsp";

namespace OpenMeter;

using OpenMeter.Entitlements;

/**
 * The type of the rate card.
 */
@friendlyName("RateCardType")
enum RateCardType {
  flatFee: "flat_fee",
  usageBased: "usage_based",
}

/**
 * Common fields of the rate card.
 */
@friendlyName("RateCardBase")
model RateCardBase extends Resource {
  /**
   * The feature the customer is entitled to use.
   */
  @summary("Feature key")
  featureKey?: Key;

  /**
   * The entitlement of the rate card.
   * Only available when featureKey is set.
   */
  @summary("Entitlement template")
  entitlementTemplate?: EntitlementTemplate;

  /**
   * The tax config of the rate card.
   * When undefined, the tax config of the feature or the default tax config of the plan is used.
   */
  @summary("Tax config")
  taxConfig?: TaxConfig;
}

/**
 * A rate card defines the pricing and entitlement of a feature or service.
 */
@friendlyName("RateCard")
@discriminator("type")
union RateCard {
  flatFee: RateCardFlatFee,
  usageBased: RateCardUsageBased,
}

/**
 * A flat fee rate card defines a one-time purchase or a recurring fee.
 */
@friendlyName("RateCardFlatFee")
model RateCardFlatFee extends RateCardBase {
  type: RateCardType.flatFee;

  /**
   * The billing cadence of the rate card.
   * When null, the rate card is a one-time purchase.
   */
  @summary("Billing cadence")
  billingCandence: Interval | null;

  /**
   * The price of the rate card.
   * When null, the feature or service is free.
   */
  @summary("Price")
  @example(#{ type: PriceType.flat, amount: "100", paymentTerm: "in_arrears" })
  price: FlatPriceWithPaymentTerm | null;

  /**
   * The discount of the rate card. Either a percentage or an amount.
   * Only available when price is set.
   */
  @summary("Discount")
  discount?: DiscountPercentage | DiscountAmount;
}

/**
 * A usage-based rate card defines a price based on usage.
 */
@friendlyName("RateCardUsageBased")
model RateCardUsageBased extends RateCardBase {
  type: RateCardType.usageBased;

  /**
   * The billing cadence of the rate card.
   */
  @summary("Billing cadence")
  billingCandence: Interval;

  /**
   * The price of the rate card.
   * When null, the feature or service is free.
   */
  @summary("Price")
  @example(#{ type: PriceType.unit, amount: "0.01", minimumAmount: "100" })
  price: UnitPriceWithCommitments | TieredPriceWithCommitments | FlatPriceWithPaymentTerm | null;

  /**
   * The discount of the rate card.
   */
  @summary("Discount")
  discount?: Discount;
}

/**
 * Rate card override.
 */
@friendlyName("RateCardOverride")
@discriminator("type")
union RateCardOverride {
  flatFee: RateCardOverrideFlatFee,
  usageBased: RateCardOverrideUsageBased,
}

/**
 * Flat fee rate card override.
 */
@friendlyName("RateCardOverrideFlatFee")
model RateCardOverrideFlatFee {
  ...PickProperties<RateCardFlatFee, "type" | "price" | "discount">;
}

/**
 * Usage-based rate card override.
 */
@friendlyName("RateCardOverrideUsageBased")
model RateCardOverrideUsageBased {
  ...PickProperties<RateCardUsageBased, "type" | "price" | "discount">;
}

// /**
//  * Addons are additional services that can be added to a subscription.
//  */
// model AddonRateCard extends RateCard {
//   /**
//    * The maximum quantity of the addon that can be added to a subscription.
//    */
//   maxQuantity?: integer = 1;

//   /**
//    * The strategy for an existing entitlement of the addon.
//    * Without entitlement, the strategy must be null.
//    */
//   entitlementStrategy: AddonEntitlementStrategy | null;
// }

// /**
//  * The strategy for an existing entitlement of the addon.
//  */
// enum AddonEntitlementStrategy {
//   replace: "replace",
//   accumulate: "accumulate",
// }
