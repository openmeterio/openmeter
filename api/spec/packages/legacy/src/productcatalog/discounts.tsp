namespace OpenMeter;

/**
 * Percentage discount.
 */
@friendlyName("DiscountPercentage")
model DiscountPercentage {
  /**
   * The percentage of the discount.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Percentage")
  @minValue(0)
  @maxValue(100)
  percentage: Percentage;
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

/**
 * Usage discount.
 *
 * Usage discount means that the first N items are free. From billing perspective
 * this means that any usage on a specific feature is considered 0 until this discount
 * is exhausted.
 */
@friendlyName("DiscountUsage")
model DiscountUsage {
  /**
   * The quantity of the usage discount.
   *
   * Must be positive.
   */
  @summary("Usage")
  quantity: Numeric;
}

/**
 * Discount by type on a price
 */
@friendlyName("Discounts")
model Discounts {
  /**
   * The percentage discount.
   */
  percentage?: DiscountPercentage;

  /**
   * The usage discount.
   */
  usage?: DiscountUsage;

  // amount?: DiscountAmount,
}
