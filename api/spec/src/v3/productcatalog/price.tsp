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
}

/**
 * Usage-based price types that can appear on a usage-based rate card.
 *
 * When UnitConfig is present on the rate card, these price types operate on
 * billing units (i.e. post-conversion quantities), not raw metered units.
 */
@friendlyName("BillingPriceUsageBased")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union PriceUsageBased {
  unit: PriceUnit,
  graduated: PriceGraduated,
  volume: PriceVolume,
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
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.Free;
}

/**
 * Flat price.
 */
@friendlyName("BillingPriceFlat")
model PriceFlat {
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.Flat;

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
 *
 * Charges a fixed rate per billing unit.
 * When UnitConfig is present on the rate card, billing units are the
 * converted quantities (e.g. GB instead of bytes).
 */
@friendlyName("BillingPriceUnit")
model PriceUnit {
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.Unit;

  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  amount: Shared.Numeric;
}

/**
 * Graduated tiered price.
 *
 * Each tier's rate applies only to the usage within that tier.
 * Pricing can change as cumulative usage crosses tier boundaries.
 *
 * When UnitConfig is present on the rate card, tier boundaries (up_to_amount)
 * are expressed in converted billing units.
 */
@friendlyName("BillingPriceGraduated")
model PriceGraduated {
  /**
   * The type of the price.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.Graduated;

  /**
   * The tiers of the graduated price.
   * At least one tier is required.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Tiers")
  @minItems(1)
  tiers: PriceTier[];
}

/**
 * Volume tiered price.
 *
 * The maximum quantity within a period determines the per-unit price for
 * all units in that period.
 *
 * When UnitConfig is present on the rate card, tier boundaries (up_to_amount)
 * are expressed in converted billing units.
 */
@friendlyName("BillingPriceVolume")
model PriceVolume {
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.Volume;

  /**
   * The tiers of the volume price.
   * At least one tier is required.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Tiers")
  @minItems(1)
  tiers: PriceTier[];
}

/**
 * A price tier used in graduated and volume pricing.
 *
 * At least one price component (flat_price or unit_price) must be set.
 * When UnitConfig is present on the rate card, up_to_amount is expressed
 * in converted billing units.
 */
@friendlyName("BillingPriceTier")
model PriceTier {
  /**
   * Up to and including this quantity will be contained in the tier.
   * If undefined, the tier is open-ended (the last tier).
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Up to quantity")
  up_to_amount?: Shared.Numeric;

  /**
   * The flat price component of the tier.
   * Charged once when the tier is entered.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Flat price component")
  flat_price?: PriceFlat;

  /**
   * The unit price component of the tier.
   * Charged per billing unit within the tier.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Unit price component")
  unit_price?: PriceUnit;
}

/**
 * Spend commitments for a rate card.
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
