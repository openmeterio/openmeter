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
   *
   * Always zero for historical balance queries using the `timestamp` parameter
   * because live charge impacts cannot be reconstructed historically.
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
