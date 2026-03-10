import "../shared/index.tsp";

namespace ProductCatalog;

/**
 * The type of the price.
 */
@friendlyName("BillingPriceType")
enum PriceType {
  Free: "free",
  Flat: "flat",
  Unit: "unit",
  Graduated: "graduated",
  Volume: "volume",
  Dynamic: "dynamic",
  Package: "package",
}

/**
 * Price.
 */
@friendlyName("BillingPrice")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union Price {
  @summary("Free price")
  free: PriceFree,

  @summary("Flat price")
  flat: PriceFlat,

  @summary("Unit price")
  unit: PriceUnit,

  @summary("Graduated price")
  graduated: PriceGraduated,

  @summary("Volume price")
  volume: PriceVolume,

  @summary("Dynamic price")
  dynamic: PriceDynamic,

  @summary("Package price")
  package: PricePackage,
}

/**
 * Usage based price.
 */
@friendlyName("BillingPriceUsageBased")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union PriceUsageBased {
  unit: PriceUnit,
  dynamic: PriceDynamic,
  package: PricePackage,
}

/**
 * The payment term of a flat price.
 */
@friendlyName("BillingPricePaymentTerm")
union PricePaymentTerm {
  /**
   * If in_advance, the rate card will be invoiced in the previous billing cycle.
   */
  InAdvance: "in_advance",

  /**
   * If in_arrears, the rate card will be invoiced in the current billing cycle.
   */
  InArrears: "in_arrears",
}

/**
 * Free price.
 */
@friendlyName("BillingPriceFree")
model PriceFree {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.Free;
}

/**
 * Flat price.
 */
@friendlyName("BillingPriceFlat")
model PriceFlat {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.Flat;

  /**
   * The amount of the flat price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  amount: Shared.Numeric;
}

/**
 * Flat price with payment term.
 */
@friendlyName("BillingPriceFlatWithPaymentTerm")
model PriceFlatWithPaymentTerm {
  ...PriceFlat;

  /**
   * The payment term of the flat price.
   * Defaults to in advance.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  payment_term?: PricePaymentTerm = PricePaymentTerm.InAdvance;
}

/**
 * Unit price.
 */
@friendlyName("BillingPriceUnit")
model PriceUnit {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.Unit;

  /**
   * The amount of the unit price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  amount: Shared.Numeric;
}

/**
 * Graduated tiered price.
 * Pricing can change as the quantity grows.
 */
@friendlyName("BillingPriceGraduated")
model PriceGraduated {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.Graduated;

  /**
   * The tiers of the tiered price.
   * At least one price component is required in each tier.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Tiers")
  @minItems(1)
  tiers: PriceTier[];
}

/**
 * Volume tiered price.
 * The maximum quantity within a period determines the per unit price.
 */
@friendlyName("BillingPriceVolume")
model PriceVolume {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.Volume;

  /**
   * The tiers of the tiered price.
   * At least one price component is required in each tier.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Tiers")
  @minItems(1)
  tiers: PriceTier[];
}

/**
 * A price tier.
 * At least one price component is required in each tier.
 */
@friendlyName("BillingPriceTier")
model PriceTier {
  /**
   * Up to and including to this quantity will be contained in the tier.
   * If undefined, the tier is open-ended.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Up to quantity")
  up_to_amount?: Shared.Numeric;

  /**
   * The flat price component of the tier.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Flat price component")
  flat_price?: PriceFlat;

  /**
   * The usage based price component of the tier.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Usage based price component")
  usage_price?: PriceUsageBased;
}

/**
 * Dynamic price.
 *
 * The underlying meter's value is considered the base price in the
 * customer's currency.
 *
 * The rate specifies the markup over the price.
 */
@friendlyName("BillingPriceDynamic")
model PriceDynamic {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.Dynamic;

  /**
   * The multiplier to apply to the base price to get the dynamic price.
   *
   * Examples:
   * - 0.0: the price is zero
   * - 0.5: the price is 50% of the base price
   * - 1.0: the price is the same as the base price
   * - 1.5: the price is 150% of the base price
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("The multiplier to apply to the base price to get the dynamic price")
  multiplier?: Shared.Numeric = "1";
}

/**
 * Package price.
 *
 * The item is sold in packages. Each package contains quantityPerPackage items, the price of the
 * package is set in amount.
 *
 * The total price of the usage will be enough packages that can accomodate all the usage.
 *
 * Examples (given a package size of 20, and an amount of $10):
 * - if the quantity is 98, the price will be 5*$10=$50.
 * - if the quantity is zero, the price will be 0*$10=$0, as even the first package is not purchased.
 * - if the quantity is 20, the price will be 1*$10=$10, as the usage fits into the first package.
 * - if the quantity is 20.1, the price will be 2*$10=$20, as the additional 0.1 usage (compared to the
 *   previous example) requires a new package.
 */
@friendlyName("BillingPricePackage")
model PricePackage {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.Package;

  /**
   * The price of one package.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Amount")
  amount: Shared.Numeric;

  /**
   * The quantity per package.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Quantity per package")
  quantity_per_package: Shared.Numeric;
}

/**
 * Spending commitments.
 * The customer is committed to spend at least the minimum amount and at most the maximum amount.
 */
@friendlyName("BillingSpendCommitments")
model SpendCommitments {
  /**
   * The customer is committed to spend at least the amount.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Minimum amount")
  minimum_amount?: Shared.Numeric;

  /**
   * The customer is limited to spend at most the amount.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Maximum amount")
  maximum_amount?: Shared.Numeric;
}

/**
 * Unit price with spend commitments.
 */
@friendlyName("BillingPriceUnitWithCommitments")
model PriceUnitWithCommitments {
  ...PriceUnit;
  ...SpendCommitments;
}

/**
 * Graduated price with spend commitments.
 */
@friendlyName("BillingPriceGraduatedWithCommitments")
model PriceGraduatedWithCommitments {
  ...PriceGraduated;
  ...SpendCommitments;
}

/**
 * Volume price with spend commitments.
 */
@friendlyName("BillingPriceVolumeWithCommitments")
model PriceVolumeWithCommitments {
  ...PriceVolume;
  ...SpendCommitments;
}

/**
 * Dynamic price with spend commitments.
 */
@friendlyName("BillingPriceDynamicWithCommitments")
model PriceDynamicWithCommitments {
  ...PriceDynamic;
  ...SpendCommitments;
}

/**
 * Package price with spend commitments.
 */
@friendlyName("BillingPricePackageWithCommitments")
model PricePackageWithCommitments {
  ...PricePackage;
  ...SpendCommitments;
}
