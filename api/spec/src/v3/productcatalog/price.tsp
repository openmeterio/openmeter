import "../shared/index.tsp";

namespace ProductCatalog;


/**
 * The type of the price.
 */
@friendlyName("PriceType")
enum PriceType {
  flat: "flat",
  unit: "unit",
  tiered_graduated: "tiered_graduated",
  tiered_volume: "tiered_volume",
  dynamic: "dynamic",
  package: "package",
}

/**
 * Price.
 * One of: flat, unit, or tiered.
 */
@friendlyName("Price")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union Price {
  @summary("Flat price")
  flat: PriceFlat,

  @summary("Unit price")
  unit: PriceUnit,

  @summary("Graduated tiered price")
  tiered_graduated: PriceTieredGraduated,

  @summary("Volume tiered price")
  tiered_volume: PriceTieredVolume,

  @summary("Dynamic price")
  dynamic: PriceDynamic,

  @summary("Package price")
  package: PricePackage,
}

/**
 * The payment term of a flat price.
 * One of: in_advance or in_arrears.
 */
@friendlyName("PricePaymentTerm")
union PricePaymentTerm {
  /**
   * If in_advance, the rate card will be invoiced in the previous billing cycle.
   */
  inAdvance: "in_advance",

  /**
   * If in_arrears, the rate card will be invoiced in the current billing cycle.
   */
  inArrears: "in_arrears",

  // NOTE(chrisgacsal): prepaid might be implemented in future releases
  // /**
  //  * If prepaid, entitlements will be granted only after invoice is paid
  //  */
  // prepaid: "prepaid",
}

/**
 * Flat price.
 */
@friendlyName("PriceFlat")
model PriceFlat {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.flat;

  /**
   * The amount of the flat price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  amount: Shared.Numeric;
}

/**
 * Flat price with payment term.
 */
@friendlyName("PriceFlatWithPaymentTerm")
model PriceFlatWithPaymentTerm {
  ...PriceFlat;

  /**
   * The payment term of the flat price.
   * Defaults to in advance.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  payment_term?: PricePaymentTerm = PricePaymentTerm.inAdvance;
}

/**
 * Unit price.
 */
@friendlyName("PriceUnit")
model PriceUnit {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.unit;

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
@friendlyName("PriceTieredGraduated")
model PriceTieredGraduated {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.tiered_graduated;

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
@friendlyName("PriceTieredVolume")
model PriceTieredVolume {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.tiered_volume;

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
@friendlyName("PriceTier")
model PriceTier {
  /**
   * Up to and including to this quantity will be contained in the tier.
   * If null, the tier is open-ended.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Up to quantity")
  up_to_amount?: Shared.Numeric;

  /**
   * The flat price component of the tier.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Flat price component")
  flat_price: PriceFlat | null;

  /**
   * The usage based price component of the tier.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Usage based price component")
  usage_price: PriceUnit | PriceDynamic | PricePackage | null;
}

/**
 * Dynamic price.
 *
 * The underlying meter's value is considered the base price in the
 * customer's currency.
 *
 * The rate specifies the markup over the price.
 */
@friendlyName("PriceDynamic")
model PriceDynamic {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.dynamic;

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
@friendlyName("PricePackage")
model PricePackage {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.package;

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
@friendlyName("SpendCommitments")
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
@friendlyName("PriceUnitWithCommitments")
model PriceUnitWithCommitments {
  ...PriceUnit;
  ...SpendCommitments;
}

/**
 * Graduated tiered price with spend commitments.
 */
@friendlyName("PriceTieredGraduatedWithCommitments")
model PriceTieredGraduatedWithCommitments {
  ...PriceTieredGraduated;
  ...SpendCommitments;
}

/**
 * Volume tiered price with spend commitments.
 */
@friendlyName("PriceTieredVolumeWithCommitments")
model PriceTieredVolumeWithCommitments {
  ...PriceTieredVolume;
  ...SpendCommitments;
}

/**
 * Dynamic price with spend commitments.
 */
@friendlyName("PriceDynamicWithCommitments")
model PriceDynamicWithCommitments {
  ...PriceDynamic;
  ...SpendCommitments;
}

/**
 * Package price with spend commitments.
 */
@friendlyName("PricePackageWithCommitments")
model PricePackageWithCommitments {
  ...PricePackage;
  ...SpendCommitments;
}
