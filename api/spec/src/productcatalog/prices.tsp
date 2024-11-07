namespace OpenMeter.ProductCatalog;

/**
 * The type of the price.
 */
@friendlyName("PriceType")
enum PriceType {
  flat: "flat",
  unit: "unit",
  tiered: "tiered",
}

/**
 * Price.
 * One of: flat, unit, or tiered.
 */
@friendlyName("Price")
@discriminator("type")
union Price {
  @summary("Flat price")
  flat: FlatPrice,

  @summary("Unit price")
  unit: UnitPrice,

  @summary("Tiered price")
  tiered: TieredPrice,
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
  @visibility("read", "create", "update")
  type: PriceType.flat;

  /**
   * The amount of the flat price.
   */
  @visibility("read", "create", "update")
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
  @visibility("read", "create", "update")
  paymentTerm?: PricePaymentTerm = PricePaymentTerm.inAdvance;
}

/**
 * Unit price.
 */
@friendlyName("UnitPrice")
model UnitPrice {
  @visibility("read", "create", "update")
  type: PriceType.unit;

  /**
   * The amount of the unit price.
   */
  @visibility("read", "create", "update")
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
  @visibility("read", "create", "update")
  type: PriceType.tiered;

  /**
   * Defines if the tiering mode is volume-based or graduated:
   * - In `volume`-based tiering, the maximum quantity within a period determines the per unit price.
   * - In `graduated` tiering, pricing can change as the quantity grows.
   */
  @visibility("read", "create", "update")
  @summary("Mode")
  mode: TieredPriceMode;

  /**
   * The tiers of the tiered price.
   * At least one price component is required in each tier.
   */
  @visibility("read", "create", "update")
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
  @visibility("read", "create", "update")
  @summary("Up to quantity")
  upToAmount: float64 | null;

  /**
   * The flat price component of the tier.
   */
  @visibility("read", "create", "update")
  @summary("Flat price component")
  flatPrice: FlatPrice | null;

  /**
   * The unit price component of the tier.
   */
  @visibility("read", "create", "update")
  @summary("Unit price component")
  unitPrice: UnitPrice | null;
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
  @visibility("read", "create", "update")
  @summary("Minimum amount")
  minimumAmount?: Money;

  /**
   * The customer is limited to spend at most the amount.
   */
  @visibility("read", "create", "update")
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
