import "../../shared/index.tsp";
import "../../currencies/index.tsp";

namespace Customers;

/**
 * The type of the credit transaction.
 *
 * - `funded`: Credit granted and available for consumption.
 * - `consumed`: Credit consumed.
 * - `expired`: Credit expired.
 * - `voided`: Credit voided.
 * - `adjusted`: Manual or system adjustment.
 */
@friendlyName("BillingCreditTransactionType")
enum CreditTransactionType {
  // Credit granted and available for consumption.
  Funded: "funded",

  // Credit consumed.
  Consumed: "consumed",

  // Credit expired.
  Expired: "expired",

  // Credit voided.
  Voided: "voided",

  // Manual or system adjustment (e.g., correction grants, balance reconciliation).
  Adjusted: "adjusted",

  // TODO: add other types and do they have additional metadata?
}

/**
 * The type of actor that caused the credit transaction.
 *
 * - `system`: Automated system operations (consumption, expiry).
 * - `customer`: Initiated by the customer.
 * - `admin`: Initiated by the admin.
 * - `payment_provider`: Initiated by the payment provider.
 */
@friendlyName("BillingCreditTransactionActorType")
@summary("Credit transaction actor type")
enum CreditTransactionActorType {
  // Automated system operations (consumption, expiry).
  System: "system",

  // Initiated by the customer.
  Customer: "customer",

  // Initiated by the admin.
  Admin: "admin",

  // Initiated by the payment provider.
  PaymentProvider: "payment_provider",

  // TODO: figure out if this is called actor or source
}

/**
 * The actor that caused the credit transaction.
 */
@friendlyName("BillingCreditTransactionActor")
@summary("Credit transaction actor")
model CreditTransactionActor {
  /**
   * The type of actor.
   */
  @visibility(Lifecycle.Read)
  type: CreditTransactionActorType;

  // TODO: add additional metadata for the actor
}

/**
 * A credit transaction represents a single credit movement on the customer's balance.
 *
 * Credit transactions are immutable.
 */
@friendlyName("BillingCreditTransaction")
@summary("Credit transaction")
model CreditTransaction {
  ...Shared.ResourceImmutable;

  /**
   * The date and time the transaction was booked.
   */
  @visibility(Lifecycle.Read)
  booked_at: Shared.DateTime;

  /**
   * The type of credit transaction.
   */
  @visibility(Lifecycle.Read)
  type: CreditTransactionType;

  /**
   * Currency of the balance affected by the transaction.
   */
  @visibility(Lifecycle.Read)
  currency: Currencies.CurrencyCode;

  /**
   * Signed amount of the credit movement.
   * Positive values add balance, negative values reduce balance.
   */
  @visibility(Lifecycle.Read)
  amount: Shared.Numeric;

  /**
   * The available balance before and after the transaction.
   */
  @visibility(Lifecycle.Read)
  available_balance: {
    #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
    before: Shared.Numeric;
    #suppress "@openmeter/api-spec-aip/doc-decorator" "shared model"
    after: Shared.Numeric;
  };

  /**
   * The actor that caused this credit transaction.
   */
  @visibility(Lifecycle.Read)
  actor: CreditTransactionActor;
}
