import "../../shared/index.tsp";
import "../../currencies/index.tsp";

namespace Customers;

/**
 * The type of the credit transaction.
 *
 * - `funded`: Credit granted and available for consumption.
 * - `consumed`: Credit consumed by usage or fees.
 */
@friendlyName("BillingCreditTransactionType")
enum CreditTransactionType {
  // Credit granted and available for consumption.
  Funded: "funded",

  // Credit consumed by usage or fees.
  Consumed: "consumed",
}

/**
 * Cursor pagination metadata for credit transactions.
 *
 * NOTE: We intentionally return opaque cursor tokens (not URI links) in
 * `first`, `last`, `next`, and `previous`. This matches existing OpenMeter
 * cursor implementations and avoids coupling response construction to
 * externally visible URL shape behind proxies/gateways.
 */
@friendlyName("BillingCreditTransactionCursorMetaPage")
model CreditTransactionCursorMetaPage {
  /**
   * Opaque cursor for loading the next page (older records).
   */
  next?: string;

  /**
   * Opaque cursor for loading the previous page (newer records).
   */
  previous?: string;

  /**
   * Requested page size.
   */
  size: integer;
}

/**
 * Cursor metadata container for credit transactions.
 */
@friendlyName("BillingCreditTransactionCursorMeta")
model CreditTransactionCursorMeta {
  /**
   * Cursor page metadata.
   */
  page: CreditTransactionCursorMetaPage;
}

/**
 * Cursor paginated response for credit transactions.
 */
@friendlyName("CreditTransactionPaginatedResponse")
model CreditTransactionPaginatedResponse {
  @pageItems
  data: CreditTransaction[];

  meta: CreditTransactionCursorMeta;
}

// /**
//  * The type of actor that caused the credit transaction.
//  *
//  * - `system`: Automated system operations (consumption, expiry).
//  * - `customer`: Initiated by the customer.
//  * - `admin`: Initiated by the admin.
//  * - `payment_provider`: Initiated by the payment provider.
//  */
// @friendlyName("BillingCreditTransactionActorType")
// @summary("Credit transaction actor type")
// enum CreditTransactionActorType {
//   // Automated system operations (consumption, expiry).
//   System: "system",

//   // Initiated by the customer.
//   Customer: "customer",

//   // Initiated by the admin.
//   Admin: "admin",

//   // Initiated by the payment provider.
//   PaymentProvider: "payment_provider",

//   // TODO: figure out if this is called actor or source
// }

// /**
//  * The actor that caused the credit transaction.
//  */
// @friendlyName("BillingCreditTransactionActor")
// @summary("Credit transaction actor")
// model CreditTransactionActor {
//   /**
//    * The type of actor.
//    */
//   @visibility(Lifecycle.Read)
//   type: CreditTransactionActorType;

//   // TODO: add additional metadata for the actor
// }

/**
 * A credit transaction represents a single credit movement on the customer's balance.
 *
 * Credit transactions are immutable.
 */
@friendlyName("BillingCreditTransaction")
@summary("Credit transaction")
model CreditTransaction {
  // Add references in labels:
  // https://kong-aip.netlify.app/aip/129/
  // - charge_id
  // - subscription_id
  // - subscription_phase_id
  // - subscription_item_id
  // - feature_id
  // and later add references: {} to the type
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

  // /**
  //  * The actor that caused this credit transaction.
  //  */
  // @visibility(Lifecycle.Read)
  // actor: CreditTransactionActor;
}
