import "../shared/index.tsp";
import "../invoices/index.tsp";
import "../apps/index.tsp";
import "./tax.tsp";

namespace Billing;

/**
 * Billing profiles contain the settings for billing and controls invoice generation.
 */
@friendlyName("BillingProfile")
model BillingProfile {
  ...Shared.Resource;

  /**
   * The name and contact information for the supplier this billing profile represents
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  supplier: Invoices.BillingParty;

  /**
   * The billing workflow settings for this profile
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  workflow: BillingWorkflow;

  /**
   * The applications used by this billing profile.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  apps: BillingProfileAppReferences;

  /**
   * Whether this is the default profile.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  default: boolean;
}

/**
 * Billing profile reference.
 */
@friendlyName("BillingProfileReference")
model BillingProfileReference {
  /**
   * The ID of the billing profile.
   */
  id: Shared.ULID;
}

/**
 * Billing workflow settings.
 */
@friendlyName("BillingWorkflow")
model BillingWorkflow {
  /**
   * The collection settings for this workflow
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  collection?: BillingWorkflowCollectionSettings;

  /**
   * The invoicing settings for this workflow
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  invoicing?: BillingWorkflowInvoicingSettings;

  /**
   * The payment settings for this workflow
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  payment?: BillingWorkflowPaymentSettings;

  /**
   * The tax settings for this workflow
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  tax?: BillingWorkflowTaxSettings;
}

/**
 * Workflow collection specifies how to collect the pending line items for an invoice.
 */
@summary("Workflow collection settings")
@friendlyName("BillingWorkflowCollectionSettings")
model BillingWorkflowCollectionSettings {
  /**
   * The alignment for collecting the pending line items into an invoice.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  alignment?: BillingWorkflowCollectionAlignment = DefaultBillingWorkflowCollectionAlignment;

  /**
   * This grace period can be used to delay the collection of the pending line items specified in
   * alignment.
   *
   * This is useful, in case of multiple subscriptions having slightly different billing periods.
   */
  @encode(DurationKnownEncoding.ISO8601)
  @example("P1D")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  interval?: string = "PT1H";
}

/**
 * BillingCollectionAlignment specifies when the pending line items should be collected into
 * an invoice.
 */
@friendlyName("BillingCollectionAlignment")
@summary("Collection alignment")
enum BillingCollectionAlignmentType {
  /**
   * Align the collection to the start of the subscription period.
   */
  Subscription: "subscription",

  /**
   * Align the collection to the anchor time and cadence.
   */
  Anchored: "anchored",
}

const DefaultBillingWorkflowCollectionAlignment: BillingWorkflowCollectionAlignmentSubscription = #{
  type: BillingCollectionAlignmentType.Subscription,
};

/**
 * The alignment for collecting the pending line items into an invoice.
 *
 * Defaults to subscription, which means that we are to create a new invoice every time the
 * a subscription period starts (for in advance items) or ends (for in arrears items).
 */
@friendlyName("BillingWorkflowCollectionAlignment")
@discriminated(#{ discriminatorPropertyName: "type", envelope: "none" })
union BillingWorkflowCollectionAlignment {
  subscription: BillingWorkflowCollectionAlignmentSubscription,
  anchored: BillingWorkflowCollectionAlignmentAnchored,
}

/**
 * BillingWorkflowCollectionAlignmentAnchored specifies the alignment for collecting the pending line items
 * into an invoice.
 */
@friendlyName("BillingWorkflowCollectionAlignmentAnchored")
model BillingWorkflowCollectionAlignmentAnchored {
  /**
   * The type of alignment.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: BillingCollectionAlignmentType.Anchored;

  /**
   * The recurring period for the alignment.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  recurring_period: Shared.RecurringPeriod;
}

/**
 * BillingWorkflowCollectionAlignmentSubscription specifies the alignment for collecting the pending line items
 * into an invoice.
 */
@friendlyName("BillingWorkflowCollectionAlignmentSubscription")
model BillingWorkflowCollectionAlignmentSubscription {
  /**
   * The type of alignment.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: BillingCollectionAlignmentType.Subscription;
}

/**
 * Invoice settings for a billing workflow.
 */
@summary("Workflow invoice settings")
@friendlyName("BillingWorkflowInvoicingSettings")
model BillingWorkflowInvoicingSettings {
  /**
   * Whether to automatically issue the invoice after the draftPeriod has passed.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  auto_advance?: boolean = true;

  /**
   * The period for the invoice to be kept in draft status for manual reviews.
   */
  @encode(DurationKnownEncoding.ISO8601)
  @example("P1D")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  draft_period?: string = "P0D";

  /**
   * Should progressive billing be allowed for this workflow?
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  progressive_billing?: boolean = false;
}

/**
 * Tax settings for a billing workflow.
 */
@summary("Workflow tax settings")
@friendlyName("BillingWorkflowTaxSettings")
model BillingWorkflowTaxSettings {
  /**
   * Enable automatic tax calculation when tax is supported by the app.
   * For example, with Stripe Invoicing when enabled, tax is calculated via Stripe Tax.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  enabled?: boolean = true;

  /**
   * Enforce tax calculation when tax is supported by the app.
   * When enabled, the billing system will not allow to create an invoice without tax calculation.
   * Enforcement is different per apps, for example, Stripe app requires customer
   * to have a tax location when starting a paid subscription.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  enforced?: boolean = false;

  /**
   * Default tax configuration to apply to the invoices for line items.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  default_tax_config?: TaxConfig;
}

/**
 * Applications used by a billing profile.
 */
@friendlyName("BillingProfileApps")
model BillingProfileApps {
  /**
   * The tax app used for this workflow.
   */
  @visibility(Lifecycle.Read)
  tax: Apps.App;

  /**
   * The invoicing app used for this workflow.
   */
  @visibility(Lifecycle.Read)
  invoicing: Apps.App;

  /**
   * The payment app used for this workflow.
   */
  @visibility(Lifecycle.Read)
  payment: Apps.App;
}

/**
 * References to the applications used by a billing profile.
 */
@friendlyName("BillingProfileAppReferences")
model BillingProfileAppReferences {
  /**
   * The tax app used for this workflow.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  tax: Apps.AppReference;

  /**
   * The invoicing app used for this workflow.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  invoicing: Apps.AppReference;

  /**
   * The payment app used for this workflow.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create)
  payment: Apps.AppReference;
}

/**
 * Collection method specifies how the invoice should be collected (automatic or manual).
 */
@friendlyName("BillingCollectionMethod")
@summary("Collection method")
enum CollectionMethod {
  @summary("Charge automatically")
  ChargeAutomatically: "charge_automatically",

  @summary("Send invoice")
  SendInvoice: "send_invoice",
}

/**
 * Payment settings for a billing workflow.
 */
@friendlyName("BillingWorkflowPaymentSettings")
@discriminated(#{
  discriminatorPropertyName: "collection_method",
  envelope: "none",
})
union BillingWorkflowPaymentSettings {
  charge_automatically: BillingWorkflowPaymentChargeAutomaticallySettings,
  send_invoice: BillingWorkflowPaymentSendInvoiceSettings,
}

/**
 * Payment settings for a billing workflow when the collection method is charge automatically.
 */
@friendlyName("BillingWorkflowPaymentChargeAutomaticallySettings")
model BillingWorkflowPaymentChargeAutomaticallySettings {
  /**
   * The collection method for the invoice.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  collection_method: CollectionMethod.ChargeAutomatically;
}

/**
 * Payment settings for a billing workflow when the collection method is send invoice.
 */
@friendlyName("BillingWorkflowPaymentSendInvoiceSettings")
model BillingWorkflowPaymentSendInvoiceSettings {
  /**
   * The collection method for the invoice.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  collection_method: CollectionMethod.SendInvoice;

  /**
   * The period after which the invoice is due.
   * With some payment solutions it's only applicable for manual collection method.
   */
  @encode(DurationKnownEncoding.ISO8601)
  @example("P30D")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  due_after?: string = "P30D";
}
