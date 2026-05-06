import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../common/error.tsp";
import "../../common/parameters.tsp";
import "../../shared/index.tsp";
import "./grant.tsp";
import "./balance.tsp";
import "./ledger.tsp";
import "./adjustment.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Customers;

/**
 * Filter options for listing credit grants.
 */
@friendlyName("ListCreditGrantsParamsFilter")
model ListCreditGrantsParamsFilter {
  /**
   * Filter credit grants by status.
   */
  status?: CreditGrantStatus;

  /**
   * Filter credit grants by currency.
   */
  currency?: Shared.CurrencyCode;
}

interface CustomerCreditGrantsOperations {
  /**
   * Create a new credit grant.
   * A credit grant represents an allocation of prepaid credits to a customer.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @post
  @operationId("create-credit-grant")
  @summary("Create a new credit grant")
  create(
    @path customerId: Shared.ULID,
    @body credit_grant: Shared.CreateRequestNested<CreditGrant>,
  ): Shared.CreateResponse<CreditGrant> | Common.NotFound | Common.ErrorResponses;

  /**
   * Get a credit grant.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("get-credit-grant")
  @summary("Get a credit grant")
  get(
    @path customerId: Shared.ULID,
    @path creditGrantId: Shared.ULID,
  ): Shared.GetResponse<CreditGrant> | Common.NotFound | Common.ErrorResponses;

  /**
   * List credit grants.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("list-credit-grants")
  @summary("List credit grants")
  list(
    @path customerId: Shared.ULID,
    ...Common.PagePaginationQuery,

    /**
     * Filter credit grants returned in the response.
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListCreditGrantsParamsFilter,
  ): Shared.PagePaginatedResponse<CreditGrant> | Common.NotFound | Common.ErrorResponses;
}

// interface CustomerCreditGrantVoidOperations {
//   /**
//    * Void a credit grant, forfeiting the remaining unused balance.
//    *
//    * Voiding is a forward-looking, irreversible operation. Credits already
//    * consumed by usage remain unaffected — only the remaining balance is
//    * forfeited. The grant transitions to `voided` status and a breakage
//    * ledger entry is recorded for the forfeited amount. Only `pending`
//    * and `active` grants can be voided.
//    */
//   @post
//   @operationId("void-credit-grant")
//   @summary("Void credit grant")
//   voidGrant(
//     @path customerId: Shared.ULID,
//     @path creditGrantId: Shared.ULID,
//   ): Shared.UpdateResponse<CreditGrant> | Common.NotFound | Common.ErrorResponses;
// }

/**
 * Request body for updating the external payment settlement status of a credit grant.
 */
@friendlyName("UpdateCreditGrantExternalSettlementRequest")
model UpdateCreditGrantExternalSettlementRequest {
  /**
   * The new payment settlement status.
   */
  status: CreditPurchasePaymentSettlementStatus;
}

interface CustomerCreditGrantExternalSettlementOperations {
  /**
   * Update the payment settlement status of an externally funded credit grant.
   *
   * Use this endpoint to synchronize the payment state of an external payment
   * with the system so that revenue recognition and credit availability work as expected.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @post
  @operationId("update-credit-grant-external-settlement")
  @summary("Update credit grant external settlement status")
  updateExternalSettlement(
    @path customerId: Shared.ULID,
    @path creditGrantId: Shared.ULID,
    @body body: UpdateCreditGrantExternalSettlementRequest,
  ): Shared.UpdateResponse<CreditGrant> | Common.NotFound | Common.ErrorResponses;
}

/**
 * Filter options for getting a credit balance.
 */
@friendlyName("GetCreditBalanceParamsFilter")
model GetCreditBalanceParamsFilter {
  /**
   * Filter credit balance by currency.
   */
  currency?: Currencies.CurrencyCode;
}

interface CustomerCreditBalancesOperations {
  /**
   * Get a credit balance.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("get-customer-credit-balance")
  @summary("Get a customer's credit balance")
  get(
    @path customerId: Shared.ULID,

    @query(#{ style: "deepObject", explode: true })
    filter?: GetCreditBalanceParamsFilter,
  ): Shared.GetResponse<CreditBalances> | Common.NotFound | Common.ErrorResponses;
}

/**
 * Filter options for listing credit transactions.
 */
@friendlyName("ListCreditTransactionsParamsFilter")
model ListCreditTransactionsParamsFilter {
  /**
   * Filter credit transactions by type.
   */
  type?: CreditTransactionType;

  /**
   * Filter credit transactions by currency.
   */
  currency?: Currencies.CurrencyCode;

  // TODO: add date filters for created_at and booked_at
}

interface CustomerCreditTransactionOperations {
  /**
   * List credit transactions for a customer.
   *
   * Returns an immutable, chronological record of credit movements:
   * funded credits and consumed credits.
   * Transactions are returned in reverse chronological order by default.
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @get
  @operationId("list-credit-transactions")
  @summary("List credit transactions")
  list(
    @path customerId: Shared.ULID,
    ...Common.CursorPaginationQuery,

    /**
     * Filter credit transactions returned in the response.
     */
    @query(#{ style: "deepObject", explode: true })
    filter?: ListCreditTransactionsParamsFilter,
  ): Shared.CursorPaginatedResponse<CreditTransaction> | Common.NotFound | Common.ErrorResponses;
}

interface CustomerCreditAdjustmentsOperations {
  /**
   * A credit adjustment can be used to make manual adjustments to a customer's credit balance.
   *
   * Supported use-cases:
   * - Usage correction
   */
  @extension(Shared.UnstableExtension, true)
  @extension(Shared.InternalExtension, true)
  @post
  @operationId("create-credit-adjustment")
  @summary("Create a credit adjustment")
  create(
    @path customerId: Shared.ULID,
    @body credit_adjustment: Shared.CreateRequest<CreditAdjustment>,
  ): Shared.CreateResponse<CreditAdjustment> | Common.NotFound | Common.ErrorResponses;
}
