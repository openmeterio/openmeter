import "../types.tsp";

namespace OpenMeter.ProductCatalog;

/**
 * The type of the commitment.
 */
@friendlyName("CommitmentType")
enum CommitmentType {
  spend: "spend",
  // usage: "usage",
}

/**
 * A commitment on a plan phase.
 * One of: spend.
 */
@discriminator("type")
@friendlyName("Commitment")
union Commitment {
  @summary("Spend commitment")
  spend: CommitmentSpend,

  // @summary("Usage commitment")
  // usage: CommitmentUsage,
}

/**
 * Spend commitment.
 */
@friendlyName("CommitmentSpend")
model CommitmentSpend {
  /**
   * The type of the commitment.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Type")
  type: CommitmentType.spend;

  /**
   * The customer is committed to spend at least the amount.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Minimum amount")
  minimumAmount?: Money;

  /**
   * The customer is limited to spend at most the amount.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Maximum amount")
  maximumAmount?: Money;

  /**
   * The rate cards that the commitment applies to.
   * When not specified, the commitment applies to all rate cards.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  @summary("Committed RateCards")
  rateCards?: Key[];
}
