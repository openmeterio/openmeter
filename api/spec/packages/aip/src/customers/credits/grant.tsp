import "../../shared/index.tsp";
import "../../currencies/index.tsp";

namespace Customers;

/**
 * The funding method describes how the grant is funded.
 *
 * - `none`: No funding workflow applies, for example promotional grants
 * - `invoice`: The grant is funded by an in-system invoice flow
 * - `external`: The grant is funded outside the system (e.g., wire transfer, external invoice, or manual reconciliation)
 */
@friendlyName("BillingCreditFundingMethod")
@summary("Credit funding method")
enum CreditFundingMethod {
  // No funding workflow applies
  None: "none",

  // The grant is funded by an in-system invoice flow
  Invoice: "invoice",

  // The grant is funded outside the system
  External: "external",
}

/**
 * When credits become available for consumption.
 *
 * - `on_creation`: Credits are available as soon as the grant is created.
 * - `on_authorization`: Credits are available once the payment is authorized.
 * - `on_settlement`: Credits are available once the payment is settled.
 */
@friendlyName("BillingCreditAvailabilityPolicy")
@summary("Credit availability policy")
enum CreditAvailabilityPolicy {
  // Credits become eligible when the grant is created
  OnCreation: "on_creation",

  // // Credits become eligible when payment is authorized
  // OnAuthorization: "on_authorization",

  // // Credits become eligible when payment is settled
  // OnSettlement: "on_settlement",
}

/**
 * Credit purchase payment settlement status.
 *
 * - `pending`: Payment has been initiated and is not yet authorized.
 * - `authorized`: Payment has been authorized.
 * - `settled`: Payment has been settled.
 */
@friendlyName("BillingCreditPurchasePaymentSettlementStatus")
@summary("Credit purchase payment settlement status")
enum CreditPurchasePaymentSettlementStatus {
  // Payment has been initiated and is not yet authorized.
  Pending: "pending",

  // Payment has been authorized.
  Authorized: "authorized",

  // Payment has been settled.
  Settled: "settled",
}

/**
 * Credit grant lifecycle status.
 *
 * - `pending`: The credit block has been created but is not yet valid. (`effective_at` is in the future or availability_policy is not met)
 * - `active`: The credit block is currently valid and eligible for consumption. (`effective_at` is in the past, `expires_at` is in the future and availability_policy is met)
 * - `expired`: The credit block expired with remaining unused balance, `expires_at` time has passed.
 * - `voided`: The credit block was voided. Remaining balance is forfeited.
 */
@friendlyName("BillingCreditGrantStatus")
@summary("Credit grant lifecycle status")
enum CreditGrantStatus {
  // The credit block has been created but is not yet eligible for consumption. (`effective_at` is in the future or availability_policy is not met)
  Pending: "pending",

  // The credit block is currently eligible for consumption. (`effective_at` is in the past, `expires_at` is in the future and availability_policy is met)
  Active: "active",

  // The credit block expired with remaining unused balance, `expires_at` time has passed.
  Expired: "expired",

  // The credit block was voided.
  Voided: "voided",
}

/**
 * Tax configuration for a credit grant.
 *
 * Tax configuration should be provided to ensure correct revenue recognition,
 * including for externally funded grants.
 */
@friendlyName("BillingCreditGrantTaxConfig")
@summary("Tax configuration for a credit grant")
model CreditGrantTaxConfig {
  /**
   * Tax behavior applied to the invoice line item.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  behavior?: Billing.TaxBehavior;

  /**
   * Tax code applied to the invoice line item.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  tax_code?: {
    id: Shared.ULID;
  };
}

/**
 * A credit grant allocates credits to a customer.
 *
 * Credits are drawn down against charges according to the
 * settlement mode configured on the rate card.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "credit grant model"
@summary("Credit grant")
@friendlyName("BillingCreditGrant")
model CreditGrant {
  ...Shared.Resource;

  /**
   * Funding method of the grant.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  funding_method: CreditFundingMethod;

  /**
   * The currency of the granted credits.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  currency: Currencies.CurrencyCode;

  /**
   * Granted credit amount.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  amount: Shared.Numeric;

  /**
   * Purchase and payment terms of the grant.
   * Present when a funding workflow applies (funding_method is not `none`).
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  purchase?: {
    /**
     * Currency of the purchase amount.
     */
    @visibility(Lifecycle.Read, Lifecycle.Create)
    currency: Shared.CurrencyCode;

    /**
     * Cost basis per credit unit used to calculate the purchase amount.
     *
     * If `per_unit_cost_basis` is 0.50 and credit amount is $100.00,
     * the total charge is $50.00.
     * The value must be greater than 0. If the cost basis is 0, use
     * `funding_method=none` instead.
     *
     * Defaults to 1.0.
     */
    @visibility(Lifecycle.Read, Lifecycle.Create)
    per_unit_cost_basis?: Shared.Numeric = "1.0";

    /**
     * The purchase amount.
     * Calculated from `per_unit_cost_basis` and credit `amount`.
     */
    @visibility(Lifecycle.Read)
    amount: Shared.Numeric;

    /**
     * Controls when credits become available for consumption.
     *
     * Defaults to `on_creation`.
     */
    @visibility(Lifecycle.Read, Lifecycle.Create)
    availability_policy?: CreditAvailabilityPolicy = CreditAvailabilityPolicy.OnCreation;

    // TODO: add this option later
    // /**
    //  * The initial payment settlement status when creating a grant with
    //  * `external` funding method.
    //  *
    //  * Governs the payment state of the grant on creation. If the external
    //  * system tracks payment status, synchronize their state with this system
    //  * so that revenue recognition works as expected.
    //  *
    //  * Defaults to `settled` (the external payment is assumed to be already settled on creation).
    //  */
    // @visibility(Lifecycle.Create)
    // external_payment_initial_settlement_status?: CreditPurchasePaymentSettlementStatus = CreditPurchasePaymentSettlementStatus.Settled;

    /**
     * Current payment settlement status.
     */
    @visibility(Lifecycle.Read)
    settlement_status?: CreditPurchasePaymentSettlementStatus;
  };

  /**
   * Tax configuration for the grant.
   *
   * For `invoice` and `external` funding methods, tax configuration should be provided to ensure correct revenue recognition.
   *
   * Defaults to the billing profile's tax configuration when not provided.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  tax_config?: CreditGrantTaxConfig;

  /**
   * Invoice references for the grant.
   * Available when `funding_method` is `invoice`.
   */
  @visibility(Lifecycle.Read)
  invoice?: {
    /**
     * Identifier of the invoice associated with the grant.
     */
    @visibility(Lifecycle.Read)
    id?: Shared.ULID;

    /**
     * Identifier of the invoice line associated with the grant.
     */
    @visibility(Lifecycle.Read)
    line?: {
      id: Shared.ULID;
    };
  };

  /**
   * Filters for the credit grant.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  filters?: {
    /**
     * Limit the credit grant to specific features.
     * If no features are specified, the credit grant can be used for any feature.
     */
    @example(#["input_tokens", "output_tokens"])
    features?: Shared.ResourceKey[];
  };

  /**
   * Draw-down priority of the grant.
   * Lower values have higher priority.
   */
  @visibility(Lifecycle.Create, Lifecycle.Read)
  @minValue(1)
  @maxValue(1000)
  priority?: int16 = 10;

  // /**
  //  * The timestamp when the credit grant becomes effective.
  //  *
  //  * Defaults to the current date and time.
  //  */
  // @visibility(Lifecycle.Create, Lifecycle.Read)
  // effective_at?: Shared.DateTime;

  // /**
  //  * The duration after which the credit grant expires.
  //  *
  //  * Defaults to never expiring.
  //  */
  // @visibility(Lifecycle.Create, Lifecycle.Read)
  // expires_after?: Shared.ISO8601Duration;

  // /**
  //  * The timestamp when the credit grant expires.
  //  * Calculated from `effective_at` and `expires_after` if provided.
  //  */
  // @visibility(Lifecycle.Read)
  // expires_at?: Shared.DateTime;

  /**
   * Timestamp when the grant was voided.
   */
  @visibility(Lifecycle.Read)
  voided_at?: Shared.DateTime;

  /**
   * Current lifecycle status of the grant.
   */
  @visibility(Lifecycle.Read)
  status: CreditGrantStatus;
}
