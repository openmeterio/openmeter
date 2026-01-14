import "../shared/index.tsp";
import "./app.tsp";
import "../invoices/index.tsp";

namespace Apps;

/**
 * Custom Invoicing app enables integration with any external invoicing or payment system.
 *
 * The app supports a bi-directional synchronization pattern where the billing system manages
 * the invoice lifecycle while the external system handles invoice presentation and payment collection.
 *
 * Integration workflow:
 * 1. The billing system creates invoices and transitions them through lifecycle states (draft → issuing → issued)
 * 2. The integration receives webhook notifications about invoice state changes
 * 3. The integration calls back to provide external system IDs and metadata
 * 4. The integration reports payment events back via the payment status API
 *
 * State synchronization is controlled by hooks that pause invoice progression until the
 * external system confirms synchronization via API callbacks.
 */
#suppress "@openmeter/api-spec/repeated-prefix-grouping" "Hooks are conceptually separate settings"
@friendlyName("BillingAppCustomInvoicing")
model AppCustomInvoicing {
  ...AppBase<AppType.CustomInvoicing>;

  /**
   * Enable draft synchronization hook.
   *
   * When enabled, invoices will pause at the draft state and wait for the integration
   * to call the draft synchronized endpoint before progressing to the issuing state.
   * This allows the external system to validate and prepare the invoice data.
   *
   * When disabled, invoices automatically progress through the draft state based on
   * the configured workflow timing.
   */
  enable_draft_sync_hook: boolean;

  /**
   * Enable issuing synchronization hook.
   *
   * When enabled, invoices will pause at the issuing state and wait for the integration
   * to call the issuing synchronized endpoint before progressing to the issued state.
   * This ensures the external invoicing system has successfully created and finalized
   * the invoice before it is marked as issued.
   *
   * When disabled, invoices automatically progress through the issuing state and are
   * immediately marked as issued.
   */
  enable_issuing_sync_hook: boolean;
}

/**
 * Invoice synchronization result containing external system identifiers.
 *
 * This structure allows the integration to map invoice entities to their
 * corresponding entities in the external invoicing system. These external IDs are
 * stored with the invoice and can be used for future reconciliation, debugging,
 * and cross-system references.
 */
#suppress "@openmeter/api-spec/repeated-prefix-grouping" "line_external_ids and line_discount_external_ids are distinct entity types"
@friendlyName("BillingAppCustomInvoicingSyncResult")
model CustomInvoicingSyncResult {
  /**
   * The invoice number assigned by the external invoicing system.
   *
   * If provided, this will replace the auto-generated invoice number.
   * The invoice number becomes permanent once the invoice reaches the issued state.
   */
  invoice_number?: Invoices.InvoiceNumber;

  /**
   * The external invoicing system's unique identifier for this invoice.
   *
   * This ID is stored as the invoicing external ID on the invoice and can be used
   * to link back to the corresponding invoice in the external system (e.g., QuickBooks
   * invoice ID, Xero invoice ID).
   */
  external_id?: string;

  /**
   * Mappings between invoice line item IDs and external system line item IDs.
   *
   * Use this to establish a bidirectional reference between invoice line items
   * and their corresponding entities in the external invoicing system.
   * This is useful for reconciliation and when updating or querying specific line items.
   */
  line_external_ids?: CustomInvoicingLineExternalIdMapping[];

  /**
   * Mappings between line discount IDs and external system discount IDs.
   *
   * Use this to link line-level discounts to their corresponding entities in the
   * external system for accurate financial reconciliation and reporting.
   */
  line_discount_external_ids?: CustomInvoicingLineDiscountExternalIdMapping[];
}

/**
 * Maps an invoice line item to its external system counterpart.
 *
 * This mapping enables bidirectional references between line items across systems,
 * facilitating updates, queries, and reconciliation operations.
 */
@friendlyName("BillingAppCustomInvoicingLineExternalIDMapping")
model CustomInvoicingLineExternalIdMapping {
  /**
   * The invoice line item ID (ULID format).
   *
   * This is the unique identifier for the line item in the billing system.
   */
  line_id: Shared.ULID;

  /**
   * The external invoicing system's unique identifier for this line item.
   *
   * Examples: QuickBooks line item ID, Xero line item ID, or any custom system's identifier.
   * This value is stored as the invoicing external ID on the line item.
   */
  external_id: string;
}

/**
 * Maps an invoice line discount to its external system counterpart.
 *
 * Line discounts represent reductions applied to specific invoice line items.
 * This mapping allows tracking of discount entities across systems for accurate
 * financial reporting and reconciliation.
 */
@friendlyName("BillingAppCustomInvoicingLineDiscountExternalIdMapping")
model CustomInvoicingLineDiscountExternalIdMapping {
  /**
   * The line discount ID (ULID format).
   *
   * This is the unique identifier for the discount in the billing system.
   */
  line_discount_id: Shared.ULID;

  /**
   * The external invoicing system's unique identifier for this discount.
   *
   * This allows tracking how discounts are represented in the external system.
   */
  external_id: string;
}

/**
 * Payment synchronization details for an invoice.
 *
 * Contains external system identifiers related to payment processing.
 */
@friendlyName("BillingAppCustomInvoicingFinalizedPaymentRequest")
model CustomInvoicingFinalizedPaymentRequest {
  /**
   * The external payment system's unique identifier for this invoice's payment.
   *
   * This ID links the invoice to its payment record in the external payment system
   * (e.g., Stripe payment intent ID, external payment processor transaction ID).
   * Stored as the payment external ID on the invoice.
   */
  external_id?: string;
}

/**
 * Request payload for confirming draft invoice synchronization.
 *
 * Called by the integration when draft synchronization is complete and the
 * invoice can progress from draft to issuing state. This is only required
 * when `enable_draft_sync_hook` is enabled on the app.
 */
@friendlyName("BillingAppCustomInvoicingDraftSynchronizedRequest")
model CustomInvoicingDraftSynchronizedRequest {
  /**
   * Synchronization results containing external system identifiers.
   *
   * Provides mappings between invoice entities and their corresponding
   * external system entities after the draft has been validated/prepared
   * in the external system.
   */
  invoicing?: CustomInvoicingSyncResult;
}

/**
 * Invoicing finalization details for an invoice.
 *
 * Contains information about the issued invoice in the external system.
 */
@friendlyName("BillingAppCustomInvoicingFinalizedInvoicingRequest")
model CustomInvoicingFinalizedInvoicingRequest {
  /**
   * The final invoice number assigned by the external invoicing system.
   *
   * If not provided, an invoice number will be auto-generated with
   * the "INV-" prefix. Once set at the issued state, the invoice number
   * becomes immutable.
   */
  invoice_number?: Invoices.InvoiceNumber;

  /**
   * The timestamp when the invoice was delivered to the customer.
   *
   * This represents when the external system sent the invoice to the customer
   * (e.g., via email, portal, or other delivery method). Used for tracking
   * invoice delivery and payment term calculations.
   */
  sent_to_customer_at?: Shared.DateTime;
}

/**
 * Request payload for confirming invoice issuance synchronization.
 *
 * Called by the integration when the invoice has been successfully issued in the
 * external invoicing system. This callback allows the invoice to progress from
 * the issuing to issued state. This is only required when `enable_issuing_sync_hook`
 * is enabled on the app.
 *
 * The request can include both invoicing details (invoice number, delivery timestamp)
 * and payment setup details (payment processor identifiers).
 */
@friendlyName("BillingAppCustomInvoicingFinalizedRequest")
model CustomInvoicingFinalizedRequest {
  /**
   * Invoicing finalization details from the external invoicing system.
   *
   * Includes the final invoice number and delivery confirmation. If the invoice
   * number is not provided, one will be auto-generated with the "INV-" prefix.
   */
  invoicing?: CustomInvoicingFinalizedInvoicingRequest;

  /**
   * Payment synchronization details from the external payment system.
   *
   * Provides external payment system identifiers for tracking and reconciliation.
   */
  payment?: CustomInvoicingFinalizedPaymentRequest;
}

/**
 * Payment status triggers for invoice state transitions.
 *
 * These triggers allow the external payment system to update the invoice's
 * payment status based on events occurring in the external system.
 * Each trigger transitions the invoice to a corresponding payment state.
 */
@friendlyName("BillingAppCustomInvoicingPaymentTrigger")
enum CustomInvoicingPaymentTrigger {
  /**
   * Payment received successfully.
   *
   * Transitions the invoice to the "paid" status, indicating that the full
   * payment has been collected. This is the successful terminal state for
   * invoice payment.
   */
  Paid: "paid",

  /**
   * Payment attempt failed.
   *
   * Indicates that a payment attempt was made but failed (e.g., insufficient
   * funds, card declined). The invoice may be retried or may require customer
   * action to resolve.
   */
  PaymentFailed: "payment_failed",

  /**
   * Payment is uncollectible.
   *
   * Marks the invoice as uncollectible after repeated collection failures or
   * when the business determines the debt is unlikely to be recovered. This is
   * typically used after exhausting collection attempts.
   */
  PaymentUncollectible: "payment_uncollectible",

  /**
   * Payment is past due.
   *
   * Transitions the invoice to the "overdue" status when payment has not been
   * received by the due date. Used for tracking aging receivables and triggering
   * collection workflows.
   */
  PaymentOverdue: "payment_overdue",

  /**
   * Customer action required.
   *
   * Indicates that payment cannot proceed without customer intervention (e.g.,
   * authentication required, payment method expired, additional information needed).
   * The customer must take action before payment can be processed.
   */
  ActionRequired: "action_required",

  /**
   * Void the invoice.
   *
   * Cancels the invoice permanently. Voided invoices cannot be paid and are
   * typically used for canceled orders, duplicate invoices, or other scenarios
   * where the invoice should not be collected.
   */
  Void: "void",
}

/**
 * Request to update an invoice's payment status based on external payment events.
 *
 * This endpoint allows the external payment system to report payment-related events
 * back to the billing system, keeping the invoice payment status synchronized across systems.
 * Typically called in response to payment processor webhooks or polling mechanisms.
 *
 * Only applicable for invoices that have reached the issued state and are managed
 * by a custom invoicing app.
 */
@friendlyName("BillingAppCustomInvoicingUpdatePaymentStatusRequest")
model CustomInvoicingUpdatePaymentStatusRequest {
  /**
   * The payment status trigger to execute.
   *
   * Specifies the payment event that occurred in the external system, which will
   * transition the invoice to the corresponding payment state.
   */
  trigger: CustomInvoicingPaymentTrigger;
}
