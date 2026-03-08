namespace OpenMeter;

/**
 * The type of the price.
 */
@friendlyName("PriceType")
enum PriceType {
  flat: "flat",
  unit: "unit",
  tiered: "tiered",
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
  flat: FlatPrice,

  @summary("Unit price")
  unit: UnitPrice,

  @summary("Tiered price")
  tiered: TieredPrice,

  @summary("Dynamic price")
  dynamic: DynamicPrice,

  @summary("Package price")
  package: PackagePrice,
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
@friendlyName("FlatPrice")
model FlatPrice {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.flat;

  /**
   * The amount of the flat price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  amount: Money;
}

/**
 * Flat price with payment term.
 */
@friendlyName("FlatPriceWithPaymentTerm")
model FlatPriceWithPaymentTerm {
  ...FlatPrice;

  /**
   * The payment term of the flat price.
   * Defaults to in advance.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  paymentTerm?: PricePaymentTerm = PricePaymentTerm.inAdvance;
}

/**
 * Unit price.
 */
@friendlyName("UnitPrice")
model UnitPrice {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.unit;

  /**
   * The amount of the unit price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  amount: Money;
}

/**
 * The mode of the tiered price.
 */
@friendlyName("TieredPriceMode")
enum TieredPriceMode {
  volume: "volume",
  graduated: "graduated",
}

/**
 * Tiered price.
 */
@friendlyName("TieredPrice")
model TieredPrice {
  /**
   * The type of the price.
   *
   * One of: flat, unit, or tiered.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.tiered;

  /**
   * Defines if the tiering mode is volume-based or graduated:
   * - In `volume`-based tiering, the maximum quantity within a period determines the per unit price.
   * - In `graduated` tiering, pricing can change as the quantity grows.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Mode")
  mode: TieredPriceMode;

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
  upToAmount?: Numeric;

  /**
   * The flat price component of the tier.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Flat price component")
  flatPrice: FlatPrice | null;

  /**
   * The unit price component of the tier.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Unit price component")
  unitPrice: UnitPrice | null;
}

/**
 * Dynamic price.
 *
 * The underlying meter's value is considered the base price in the
 * customer's currency.
 *
 * The rate specifies the markup over the price.
 */
@friendlyName("DynamicPrice")
model DynamicPrice {
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
  multiplier?: Numeric = "1";
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
@friendlyName("PackagePrice")
model PackagePrice {
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
  amount: Money;

  /**
   * The quantity per package.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Quantity per package")
  quantityPerPackage: Numeric;
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
  minimumAmount?: Money;

  /**
   * The customer is limited to spend at most the amount.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Maximum amount")
  maximumAmount?: Money;
}

/**
 * Unit price with spend commitments.
 */
@friendlyName("UnitPriceWithCommitments")
model UnitPriceWithCommitments {
  ...UnitPrice;
  ...SpendCommitments;
}

/**
 * Tiered price with spend commitments.
 */
@friendlyName("TieredPriceWithCommitments")
model TieredPriceWithCommitments {
  ...TieredPrice;
  ...SpendCommitments;
}

/**
 * Dynamic price with spend commitments.
 */
@friendlyName("DynamicPriceWithCommitments")
model DynamicPriceWithCommitments {
  ...DynamicPrice;
  ...SpendCommitments;
}

/**
 * Package price with spend commitments.
 */
@friendlyName("PackagePriceWithCommitments")
model PackagePriceWithCommitments {
  ...PackagePrice;
  ...SpendCommitments;
}
