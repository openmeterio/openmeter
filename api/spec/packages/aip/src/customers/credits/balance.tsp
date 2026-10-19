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
   * Managed currency reference. Present only for custom currencies.
   */
  @visibility(Lifecycle.Read)
  custom_currency?: Shared.ResourceReference<Currencies.CurrencyCustom>;

  /**
   * Spendable balance after applying currently live, unbooked charge impacts. Can be
   * negative when usage exceeds available credit.
   *
   * Always zero for historical balance queries using the `timestamp` parameter
   * because live charge impacts cannot be reconstructed historically.
   */
  @visibility(Lifecycle.Read)
  @example("200.00")
  live: Shared.Numeric;

  /**
   * Credits that have been booked on the ledger as of the balance timestamp,
   * including credit-only advance debt. Can be negative.
   */
  @visibility(Lifecycle.Read)
  @example("150.00")
  booked: Shared.Numeric;

  /**
   * Deprecated alias for `booked`. Always returns the same value.
   *
   * @deprecated Use `booked` instead.
   */
  #deprecated "Use booked instead"
  @visibility(Lifecycle.Read)
  @example("150.00")
  settled: Shared.Numeric;

  /**
   * Spendable credit at the balance timestamp, including any credit-only advance
   * debt. Preserves credit already committed to later usage. Future grants cannot be
   * spent early; scheduled expiry reduces availability when it takes effect. Can be
   * negative when advance debt exceeds spendable credit.
   */
  @visibility(Lifecycle.Read)
  @example("120.00")
  spendable: Shared.Numeric;

  /**
   * Credits that have been granted but are not yet written to the ledger, or are
   * written to the ledger with a future booked time.
   */
  @visibility(Lifecycle.Read)
  @example("50.00")
  pending: Shared.Numeric;
}
