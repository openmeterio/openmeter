namespace OpenMeter.ProductCatalog;

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
   * The rate to apply to the base price to get the dynamic price.
   *
   * Examples:
   * - 0.0: the price is zero
   * - 0.5: the price is 50% of the base price
   * - 1.0: the price is the same as the base price
   * - 1.5: the price is 150% of the base price
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("The rate to apply to the base price to get the dynamic price")
  markupRate?: Numeric = "1";
}

/**
 * Package price.
 *
 * The item is sold in packages. Each package contains quantityPerPackage items, the price of the
 * package is set in amount.
 *
 * For example, if the amount $10, and the quantity per package is 20, then the price of 98 items will
 * be 5*$10.
 *
 * As with tiered prices if there's no usage for a specific period, the user is still expected to buy the
 * first package.
 *
 * Example:
 * If the item is billed monthy, with quantity per pacakge being 20, amount being 10, even if there's no
 * usage for a given month $10 will be billed.
 *
 * Later new pacakge is only billed when the usage is greater than the allowed quantity indicated by the
 * quantityPerPackage.
 *
 * Example:
 * If the item is billed monthy, with quantity per pacakge being 20, amount being 10, and the usage is 20,
 * then $10 will be billed.
 *
 * If the usage increases to 20.1, then $20 will be billed (2 packages).
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
