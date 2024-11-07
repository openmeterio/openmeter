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

  /**
   * The rate cards that the discount applies to.
   * When not specified, the discount applies to all rate cards.
   */
  @visibility("read", "create", "update")
  @summary("Discounted RateCards")
  rateCards?: Key[];
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
