namespace OpenMeter;

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
  /**
   * The type of the RateCard.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("RateCard type")
  type: T;

  /**
   * A semi-unique identifier for the resource.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Key")
  key: Key;

  /**
   * Human-readable name for the resource. Between 1 and 256 characters.
   */
  @summary("Display name")
  @minLength(1)
  @maxLength(256)
  name: string;

  /**
   * Optional description of the resource. Maximum 1024 characters.
   */
  @maxLength(1024)
  @summary("Description")
  description?: string;

  /**
   * Additional metadata for the resource.
   */
  @summary("Metadata")
  metadata?: Metadata | null;

  /**
   * The feature the customer is entitled to use.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Feature key")
  featureKey?: Key;

  /**
   * The entitlement of the rate card.
   * Only available when featureKey is set.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  entitlementTemplate?: RateCardEntitlement;

  /**
   * The tax config of the rate card.
   * When undefined, the tax config of the feature or the default tax config of the plan is used.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Tax config")
  taxConfig?: TaxConfig;
}

/**
 * A rate card defines the pricing and entitlement of a feature or service.
 */
@friendlyName("RateCard")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union RateCard {
  flat_fee: RateCardFlatFee,
  usage_based: RateCardUsageBased,
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
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Billing cadence")
  @encode(DurationKnownEncoding.ISO8601)
  billingCadence: duration | null;

  /**
   * The price of the rate card.
   * When null, the feature or service is free.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Price")
  @example(#{ type: PriceType.flat, amount: "100", paymentTerm: "in_arrears" })
  price: FlatPriceWithPaymentTerm | null;

  /**
   * The discount of the rate card. For flat fee rate cards only percentage discounts are supported.
   * Only available when price is set.
   */
  @summary("Discounts")
  discounts?: Discounts;
}

/**
 * The price of the usage based rate card.
 */
@friendlyName("RateCardUsageBasedPrice")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union RateCardUsageBasedPrice {
  flat: FlatPriceWithPaymentTerm,
  unit: UnitPriceWithCommitments,
  tiered: TieredPriceWithCommitments,
  dynamic: DynamicPriceWithCommitments,
  package: PackagePriceWithCommitments,
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
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Billing cadence")
  @encode(DurationKnownEncoding.ISO8601)
  billingCadence: duration;

  /**
   * The price of the rate card.
   * When null, the feature or service is free.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  price: RateCardUsageBasedPrice | null;

  /**
   * The discounts of the rate card.
   *
   * Flat fee rate cards only support percentage discounts.
   */
  @summary("Discounts")
  discounts?: Discounts;
}

/**
 * Entitlement templates are used to define the entitlements of a plan.
 * Features are omitted from the entitlement template, as they are defined in the rate card.
 */
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
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
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
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
