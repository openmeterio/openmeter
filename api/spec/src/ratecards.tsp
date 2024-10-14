import "./types.tsp";
import "./billing.tsp";
import "./discounts.tsp";
import "./prices.tsp";
import "./entitlements/main.tsp";

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
 * A rate card defines the pricing and entitlement of a feature or service.
 */
@friendlyName("RateCardResource")
model RateCardResource {
  ...Resource;
  ...RateCard;
}

/**
 * A rate card defines the pricing and entitlement of a feature or service.
 */
@friendlyName("RateCard")
model RateCard {
  /**
   * The identifier of the RateCard.
   * SubscriptionItem/RateCard can be identified, it has a reference:
   *
   * 1. If a Feature is associated with the SubscriptionItem, it is identified by the Feature
   *  1.1 It can be an ID reference, for an exact version of the Feature (Features can change across versions)
   *  1.2 It can be a Key reference, which always refers to the latest (active or inactive) version of a Feature
   *
   * 2. If a Feature is not associated with the SubscriptionItem, it is referenced by the Price
   *
   * We say “referenced by the Price” regardless of how a price itself is referenced, it colloquially makes sense to say “paying the same price for the same thing”. In practice this should be derived from what's printed on the invoice line-item.
   */
  key: Key;

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
  entitlementTemplate?: RateCardEntitlement;

  /**
   * The tax config of the rate card.
   * When undefined, the tax config of the feature or the default tax config of the plan is used.
   */
  @summary("Tax config")
  taxConfig?: TaxConfig;

  /**
   * The billing cadence of the rate card.
   * When null, the rate card is a one-time purchase.
   */
  @summary("Billing cadence")
  @encode(DurationKnownEncoding.ISO8601)
  billingCandence: duration | null;

  /**
   * The price of the rate card.
   * When null, the feature or service is free.
   */
  @summary("Price")
  @example(#{ type: PriceType.flat, amount: "100", paymentTerm: "in_arrears" })
  price:
    | FlatPriceWithPaymentTerm
    | UnitPriceWithCommitments
    | TieredPriceWithCommitments
    | FlatPriceWithPaymentTerm
    | null;
}

/**
 * Entitlement templates are used to define the entitlements of a plan.
 * Features are omitted from the entitlement template, as they are defined in the rate card.
 */
@discriminator("type")
@friendlyName("RateCardEntitlement")
union RateCardEntitlement {
  metered: RateCardMeteredEntitlement,
  static: RateCardStaticEntitlement,
  boolean: RateCardBooleanEntitlement,
}

/**
 * The entitlement template with a metered entitlement.
 */
@friendlyName("RateCardMeteredEntitlement")
model RateCardMeteredEntitlement {
  ...OmitProperties<
    EntitlementMeteredCreateInputs,
    "featureKey" | "featureId" | "usagePeriod" | "measureUsageFrom" | "isUnlimited"
  >;

  /**
   * The interval of the metered entitlement.
   * Defaults to the billing cadence of the rate card.
   */
  @summary("Interval")
  @encode(DurationKnownEncoding.ISO8601)
  usagePeriod?: duration;
}

/**
 * Entitlement template of a static entitlement.
 */
@friendlyName("RateCardStaticEntitlement")
model RateCardStaticEntitlement {
  ...OmitProperties<EntitlementStaticCreateInputs, "featureKey" | "featureId" | "usagePeriod">;
}

/**
 * Entitlement template of a boolean entitlement.
 */
@friendlyName("RateCardBooleanEntitlement")
model RateCardBooleanEntitlement {
  ...OmitProperties<EntitlementBooleanCreateInputs, "featureKey" | "featureId" | "usagePeriod">;
}
