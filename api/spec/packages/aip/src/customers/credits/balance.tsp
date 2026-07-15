import "../../shared/index.tsp";
import "../../currencies/index.tsp";

namespace Customers;

/**
 * The balances of the credits of a customer.
 */
@friendlyName("BillingCreditBalances")
@summary("Credit balances")
model CreditBalances {
  /**
   * The timestamp of the balance retrieval.
   */
  @visibility(Lifecycle.Read)
  retrieved_at: Shared.DateTime;

  /**
   * The balances by currencies.
   */
  @visibility(Lifecycle.Read)
  balances: CreditBalance[];
}

/**
 * The credit balance by currency.
 */
@friendlyName("CreditBalance")
@summary("Credit balance")
model CreditBalance {
  #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
  @visibility(Lifecycle.Read)
  currency: Currencies.CurrencyCode;

  /**
   * Credits available after applying currently live charge impacts.
   */
  @visibility(Lifecycle.Read)
  @example("200.00")
  live: Shared.Numeric;

  /**
   * Credits that have been booked on the ledger as of the balance timestamp.
   */
  @visibility(Lifecycle.Read)
  @example("150.00")
  settled: Shared.Numeric;

  /**
   * Credits that have been granted but are not yet written to the ledger, or are
   * written to the ledger with a future booked time.
   */
  @visibility(Lifecycle.Read)
  @example("50.00")
  pending: Shared.Numeric;

  /**
   * Present only when `group_by=feature` is requested. Buckets partition this
   * currency's balance by feature restriction; bucket amounts sum to the top-level
   * live, settled and pending values.
   */
  @visibility(Lifecycle.Read)
  by_feature?: CreditBalanceFeatureBucket[];
}

/**
 * Dimension to group the credit balance breakdown by.
 */
@friendlyName("BillingCreditBalanceGroupByDimension")
@summary("Credit balance group by dimension")
enum CreditBalanceGroupByDimension {
  // Partition the balance by feature restriction
  Feature: "feature",
}

/**
 * A slice of the credit balance scoped to one feature restriction set.
 */
@friendlyName("BillingCreditBalanceFeatureBucket")
@summary("Credit balance feature bucket")
model CreditBalanceFeatureBucket {
  /**
   * The features this bucket is restricted to. Omitted for the unrestricted bucket
   * (credit usable by any feature). A bucket represents an exact restriction set: a
   * grant restricted to more than one feature gets its own bucket rather than being
   * split across each feature's bucket, so bucket amounts sum to the top-level
   * totals without double counting.
   */
  @visibility(Lifecycle.Read)
  @example(#["input_tokens", "output_tokens"])
  features?: Shared.ResourceKey[];

  /**
   * Credits available after applying currently live charge impacts.
   */
  @visibility(Lifecycle.Read)
  @example("200.00")
  live: Shared.Numeric;

  /**
   * Credits that have been booked on the ledger as of the balance timestamp.
   */
  @visibility(Lifecycle.Read)
  @example("150.00")
  settled: Shared.Numeric;

  /**
   * Credits that have been granted but are not yet written to the ledger, or are
   * written to the ledger with a future booked time.
   */
  @visibility(Lifecycle.Read)
  @example("50.00")
  pending: Shared.Numeric;
}
