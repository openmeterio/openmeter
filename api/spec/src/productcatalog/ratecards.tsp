namespace OpenMeter.ProductCatalog;

using TypeSpec.OpenAPI;
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
model RateCardBase<T extends RateCardType> {
  ...OmitProperties<UniqueResource, "id">;

  /**
   * The type of the RateCard.
   */
  @visibility("read", "create", "update")
  @summary("RateCard type")
  type: T;

  /**
   * The feature the customer is entitled to use.
   */
  @visibility("read", "create", "update")
  @summary("Feature key")
  featureKey?: Key;

  /*
   * The entitlement of the rate card.
   * Only available when featureKey is set.
   */
  @visibility("read", "create", "update")
  entitlementTemplate?: RateCardEntitlement;

  /**
   * The tax config of the rate card.
   * When undefined, the tax config of the feature or the default tax config of the plan is used.
   */
  @visibility("read", "create", "update")
  @summary("Tax config")
  taxConfig?: TaxConfig;
}

/**
 * A rate card defines the pricing and entitlement of a feature or service.
 */
@friendlyName("RateCard")
@discriminator("type")
@oneOf
union RateCard {
  flatFee: RateCardFlatFee,
  usageBased: RateCardUsageBased,
}

/**
 * A flat fee rate card defines a one-time purchase or a recurring fee.
 */
@friendlyName("RateCardFlatFee")
model RateCardFlatFee {
  ...RateCardBase<RateCardType.flatFee>;

  /**
   * The billing cadence of the rate card.
   * When null it means it is a one time fee.
   */
  @visibility("read", "create", "update")
  @summary("Billing cadence")
  @encode(DurationKnownEncoding.ISO8601)
  billingCadence: duration | null;

  /**
   * The price of the rate card.
   * When null, the feature or service is free.
   */
  @visibility("read", "create", "update")
  @summary("Price")
  @example(#{ type: PriceType.flat, amount: "100", paymentTerm: "in_arrears" })
  price: FlatPriceWithPaymentTerm | null;

  // NOTE(chrisgacsal): discount on RateCard level going to be implemented in upcoming releases
  // /**
  //  * The discount of the rate card. Either a percentage or an amount.
  //  * Only available when price is set.
  //  */
  // @summary("Discount")
  // discount?: DiscountPercentage | DiscountAmount;
}

/**
 * The price of the usage based rate card.
 */
@friendlyName("RateCardUsageBasedPrice")
@discriminator("type")
@oneOf
union RateCardUsageBasedPrice {
  flat: FlatPriceWithPaymentTerm,
  unit: UnitPriceWithCommitments,
  tiered: TieredPriceWithCommitments,
}

/**
 * A usage-based rate card defines a price based on usage.
 */
@friendlyName("RateCardUsageBased")
model RateCardUsageBased {
  ...RateCardBase<RateCardType.usageBased>;

  /**
   * The billing cadence of the rate card.
   */
  @visibility("read", "create", "update")
  @summary("Billing cadence")
  @encode(DurationKnownEncoding.ISO8601)
  billingCadence: duration;

  /*
   * The price of the rate card.
   * When null, the feature or service is free.
   */
  @visibility("read", "create", "update")
  price: RateCardUsageBasedPrice | null;

  // NOTE(chrisgacsal): discount on RateCard level going to be implemented in upcoming releases
  // /**
  //  * The discount of the rate card.
  //  */
  // @summary("Discount")
  // discount?: Discount;
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
  ...PickProperties<RateCardFlatFee, "type" | "price">;
}

/**
 * Usage-based rate card override.
 */
@friendlyName("RateCardOverrideUsageBased")
model RateCardOverrideUsageBased {
  ...PickProperties<RateCardUsageBased, "type" | "price">;
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
    OpenMeter.Entitlements.EntitlementMeteredCreateInputs,

      | "featureKey"
      | "featureId"
      | "usagePeriod"
      | "measureUsageFrom"
      | "isUnlimited"
  >;

  /**
   * The interval of the metered entitlement.
   * Defaults to the billing cadence of the rate card.
   */
  @visibility("read", "create", "update")
  @summary("Usage Period")
  @encode(DurationKnownEncoding.ISO8601)
  usagePeriod?: duration;
}

/**
 * Entitlement template of a static entitlement.
 */
@friendlyName("RateCardStaticEntitlement")
model RateCardStaticEntitlement {
  ...OmitProperties<
    EntitlementStaticCreateInputs,
    "featureKey" | "featureId" | "usagePeriod"
  >;
}

/**
 * Entitlement template of a boolean entitlement.
 */
@friendlyName("RateCardBooleanEntitlement")
model RateCardBooleanEntitlement {
  ...OmitProperties<
    EntitlementBooleanCreateInputs,
    "featureKey" | "featureId" | "usagePeriod"
  >;
}
