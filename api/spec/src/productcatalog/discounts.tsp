namespace OpenMeter.ProductCatalog;

/**
 * The type of the discount.
 */
@friendlyName("DiscountType")
enum DiscountType {
  percentage: "percentage",
  // amount: "amount",
  // usage: "usage",
}

/**
 * A discount on a price.
 * One of: percentage, amount, or usage.
 */
@discriminator("type")
@friendlyName("Discount")
union Discount {
  @summary("Percentage discount")
  percentage: DiscountPercentage,

  // @summary("Amount discount")
  // amount: DiscountAmount,

  // @summary("Usage discount")
  // usage: DiscountUsage,
}

/**
 * Percentage discount.
 */
@friendlyName("DiscountPercentage")
model DiscountPercentage {
  /**
   * The type of the discount.
   */
  @visibility("read", "create")
  @summary("Type")
  type: DiscountType.percentage;

  /**
   * The percentage of the discount.
   */
  @visibility("read", "create", "update")
  @summary("Percentage")
  @minValue(0)
  @maxValue(100)
  percentage: float;
}

// NOTE(chrisgacsal): amount discount is going to be implemented in future releases
// /**
//  * Amount discount.
//  */
// @friendlyName("DiscountAmount")
// model DiscountAmount {
//   /**
//    * The type of the discount.
//    */
//   @summary("Type")
//   type: DiscountType.amount;

//   /**
//    * The amount of the discount.
//    */
//   @summary("Amount")
//   amount: Money;
// }

// NOTE(chrisgacsal): usage discount is going to be implemented in future releases
// /**
//  * Usage discount.
//  */
// @friendlyName("UsageDiscount")
// model DiscountUsage {
//   /**
//    * The type of the discount.
//    */
//   @summary("Type")
//   type: DiscountType.usage;

//   /**
//    * The usage discount.
//    */
//   @summary("Usage")
//   usage: float64;
// }

/**
 * A discount on plan. One of: percentage or amount.
 */
@friendlyName("AppliedDiscount")
@discriminator("type")
union AppliedDiscount {
  @summary("Percentage discount")
  percentage: AppliedDiscountPercentage,

  // @summary("Amount discount")
  // amount: AppliedDiscountAmount,

  // @summary("Usage discount")
  // usage: AppliedDiscountUsage,
}

alias RateCardKey = Key;

/**
 * Percentage discount on plan.
 */
@friendlyName("AppliedDiscountPercentage")
model AppliedDiscountPercentage {
  ...DiscountPercentage;

  /**
   * The rate cards that the discount applies to.
   * When not specified, the discount applies to all rate cards.
   */
  @summary("Applies To")
  appliesTo?: RateCardKey[];
}

// /**
//  * Percentage discount on plan.
//  */
// @friendlyName("AppliedDiscountUsage")
// model AppliedDiscountUsage {
//   ...DiscountUsage;

//   /**
//    * The rate cards that the discount applies to.
//    * When not specified, the discount applies to all rate cards.
//    */
//   @summary("Applies To")
//   appliesTo?: RateCardKey[];
// }

// /**
//  * Amount discount on plan.
//  */
// @friendlyName("AppliedDiscountAmount")
// model AppliedDiscountAmount {
//   ...DiscountAmount;

//   /**
//    * The rate cards that the discount applies to.
//    * When not specified, the discount applies to all rate cards.
//    */
//   @summary("Applies To")
//   appliesTo?: RateCardKey[];
// }
