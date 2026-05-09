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
   * Credits that have been granted but cannot yet be consumed. Includes grants
   * awaiting payment clearance or with a future effective date.
   */
  @visibility(Lifecycle.Read)
  @example("200.00")
  pending: Shared.Numeric;

  /**
   * Credits that can be consumed right now. Derived from cleared grants after
   * applying eligibility and restriction rules.
   */
  @visibility(Lifecycle.Read)
  @example("150.00")
  available: Shared.Numeric;
}
