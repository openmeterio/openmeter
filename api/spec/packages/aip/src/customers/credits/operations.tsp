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

  /**
   * Filter credit grants by key.
   */
  key?: Common.StringFieldFilter;
}

namespace Credits.Grants {
  interface CustomerCreditGrantsOperations {
    /**
     * Create a new credit grant. A credit grant represents an allocation of prepaid
     * credits to a customer.
     */
    @extension(Shared.UnstableExtension, true)
    @post
    @operationId("create-credit-grant")
    @summary("Create a new credit grant")
    create(
      @path customerId: Shared.ULID,
      @body credit_grant: Shared.CreateRequestNested<CreditGrant>,
    ):
      | Shared.CreateResponse<CreditGrant>
      | Common.NotFound
      | Common.Conflict
      | Common.ErrorResponses;

    /**
     * Get a credit grant.
     */
    @extension(Shared.UnstableExtension, true)
    @get
    @operationId("get-credit-grant")
    @summary("Get a credit grant")
    get(@path customerId: Shared.ULID, @path creditGrantId: Shared.ULID):
      | Shared.GetResponse<CreditGrant>
      | Common.NotFound
      | Common.ErrorResponses;

    /**
     * List credit grants.
     */
    @extension(Shared.UnstableExtension, true)
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
    ):
      | Shared.PagePaginatedResponse<CreditGrant>
      | Common.NotFound
      | Common.ErrorResponses;
  }
}

namespace Credits.Grants {
  /**
   * Request body for voiding a credit grant.
   */
  @friendlyName("VoidCreditGrantRequest")
  model VoidCreditGrantRequest {
    /**
     * How voiding adjusts payment state related to the grant.
     *
     * Currently only `none` is supported: voiding does not adjust invoices, payment
     * authorization, settlement, payment intents, or external collection state. If
     * payment later completes, the original invoiced amount may still be collected.
     */
    payment_adjustment?: CreditGrantVoidPaymentAdjustment = CreditGrantVoidPaymentAdjustment.None;
  }

  interface CustomerCreditGrantVoidOperations {
    /**
     * Void a credit grant, forfeiting the remaining unused balance.
     *
     * Voiding is a forward-looking, irreversible operation. Credits already consumed
     * by usage remain unaffected — only the remaining balance is forfeited. The grant
     * reads as `voided` status afterwards. Payment state is not adjusted when
     * `payment_adjustment` is `none`, so invoice-backed or externally collected
     * payments may still collect the original amount. Only `active` grants can be
     * voided; voiding a pending, expired, or fully consumed grant returns a conflict.
     * Retrying a successful void is an idempotent success.
     */
    @extension(Shared.UnstableExtension, true)
    @post
    @operationId("void-credit-grant")
    @summary("Void credit grant")
    voidGrant(
      @path customerId: Shared.ULID,
      @path creditGrantId: Shared.ULID,
      @body request?: VoidCreditGrantRequest,
    ):
      | Shared.UpdateResponse<CreditGrant>
      | Common.NotFound
      | Common.Conflict
      | Common.ErrorResponses;
  }
}

/**
 * Request body for updating the external payment settlement status of a credit
 * grant.
 */
@friendlyName("UpdateCreditGrantExternalSettlementRequest")
model UpdateCreditGrantExternalSettlementRequest {
  /**
   * The new payment settlement status.
   */
  status: CreditPurchasePaymentSettlementStatus;
}

namespace Credits.Grants {
  interface CustomerCreditGrantExternalSettlementOperations {
    /**
     * Update the payment settlement status of an externally funded credit grant.
     *
     * Use this endpoint to synchronize the payment state of an external payment with
     * the system so that revenue recognition and credit availability work as expected.
     */
    @extension(Shared.UnstableExtension, true)
    @post
    @operationId("update-credit-grant-external-settlement")
    @summary("Update credit grant external settlement status")
    updateExternalSettlement(
      @path customerId: Shared.ULID,
      @path creditGrantId: Shared.ULID,
      @body body: UpdateCreditGrantExternalSettlementRequest,
    ):
      | Shared.UpdateResponse<CreditGrant>
      | Common.NotFound
      | Common.ErrorResponses;
  }
}

/**
 * Filter options for getting a credit balance.
 */
@friendlyName("GetCreditBalanceParamsFilter")
model GetCreditBalanceParamsFilter {
  /**
   * Filter credit balance by currency.
   */
  currency?: Common.StringFieldFilterExact;

  /**
   * Filter credit balance by feature key. Omit to return the total portfolio value.
   * Use `exists=false` to return only unrestricted balance.
   */
  feature_key?: Common.StringFieldFilter;
}

namespace Credits.Balance {
  interface CustomerCreditBalancesOperations {
    /**
     * Get a credit balance.
     */
    @extension(Shared.UnstableExtension, true)
    @get
    @operationId("get-customer-credit-balance")
    @summary("Get a customer's credit balance")
    get(
      @path customerId: Shared.ULID,

      /**
       * Return the credit balance as of this timestamp.
       *
       * Defaults to the current time. Historical responses return `live` as zero because
       * live charge impacts are only available for current balances.
       */
      @query
      timestamp?: Shared.DateTime,

      @query(#{ style: "deepObject", explode: true })
      filter?: GetCreditBalanceParamsFilter,
    ):
      | Shared.GetResponse<CreditBalances>
      | Common.NotFound
      | Common.ErrorResponses;
  }
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

  /**
   * Filter credit transactions by feature key. Omit to return all credit
   * transactions. Use `exists=false` to return only unrestricted credit
   * transactions.
   */
  feature_key?: Common.StringFieldFilter;

  // TODO: add date filters for created_at and booked_at
}

namespace Credits.Transactions {
  interface CustomerCreditTransactionOperations {
    /**
     * List credit transactions for a customer.
     *
     * Returns an immutable, chronological record of credit movements: funded credits
     * and consumed credits. Transactions are returned in reverse chronological order
     * by default.
     */
    @extension(Shared.UnstableExtension, true)
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
    ):
      | Shared.CursorPaginatedResponse<CreditTransaction>
      | Common.NotFound
      | Common.ErrorResponses;
  }
}

namespace Credits.Adjustments {
  interface CustomerCreditAdjustmentsOperations {
    /**
     * A credit adjustment can be used to make manual adjustments to a customer's
     * credit balance.
     *
     * Supported use-cases:
     *
     * - Usage correction
     */
    @extension(Shared.UnstableExtension, true)
    @post
    @operationId("create-credit-adjustment")
    @summary("Create a credit adjustment")
    create(
      @path customerId: Shared.ULID,
      @body credit_adjustment: Shared.CreateRequest<CreditAdjustment>,
    ):
      | Shared.CreateResponse<CreditAdjustment>
      | Common.NotFound
      | Common.ErrorResponses;
  }
}
