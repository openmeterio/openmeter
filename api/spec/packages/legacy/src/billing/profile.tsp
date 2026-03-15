import "../rest.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

@route("/api/v1/billing/profiles")
@tag("Billing")
@friendlyName("BillingProfiles")
interface BillingProfilesEndpoints {
  /**
   * List all billing profiles matching the specified filters.
   *
   * The expand option can be used to include additional information (besides the billing profile)
   * in the response. For example by adding the expand=apps option the apps used by the billing profile
   * will be included in the response.
   */
  @get
  @operationId("listBillingProfiles")
  @summary("List billing profiles")
  list(
    @query
    includeArchived?: boolean = false,

    @query(#{ explode: true })
    expand?: BillingProfileExpand[],

    ...QueryPagination,
    ...QueryOrdering<BillingProfileOrderBy>,
  ): PaginatedResponse<BillingProfile> | CommonErrors;

  /**
   * Create a new billing profile
   *
   * Billing profiles are representations of a customer's billing information. Customer overrides
   * can be applied to a billing profile to customize the billing behavior for a specific customer.
   */
  @post
  @summary("Create a new billing profile")
  @operationId("createBillingProfile")
  create(@body profile: BillingProfileCreate): {
    @statusCode _: 201;
    @body profile: BillingProfile;
  } | CommonErrors;

  /**
   * Delete a billing profile by id.
   *
   * Only such billing profiles can be deleted that are:
   * - not the default one
   * - not pinned to any customer using customer overrides
   * - only have finalized invoices
   */
  @route("/{id}")
  @delete
  @summary("Delete a billing profile")
  @operationId("deleteBillingProfile")
  delete(
    @path
    id: ULID,
  ): {
    @statusCode _: 204;
  } | NotFoundError | CommonErrors;

  /**
   * Get a billing profile by id.
   *
   * The expand option can be used to include additional information (besides the billing profile)
   * in the response. For example by adding the expand=apps option the apps used by the billing profile
   * will be included in the response.
   */
  @route("/{id}")
  @get
  @summary("Get a billing profile")
  @operationId("getBillingProfile")
  get(
    @path
    id: ULID,

    @query(#{ explode: true })
    expand?: BillingProfileExpand[],
  ): BillingProfile | NotFoundError | CommonErrors;

  /**
   * Update a billing profile by id.
   *
   * The apps field cannot be updated directly, if an app change is desired a new
   * profile should be created.
   */
  @route("/{id}")
  @put
  @summary("Update a billing profile")
  @operationId("updateBillingProfile")
  update(
    @path
    id: ULID,

    @body profile: BillingProfileReplaceUpdate,
  ): BillingProfile | NotFoundError | CommonErrors;
}

/**
 * BillingProfileOrderBy specifies the ordering options for profiles
 */
@friendlyName("BillingProfileOrderBy")
enum BillingProfileOrderBy {
  createdAt: "createdAt",
  updatedAt: "updatedAt",
  default: "default",
  name: "name",
}

/**
 * BillingProfileExpand details what profile fields to expand
 */
@friendlyName("BillingProfileExpand")
enum BillingProfileExpand {
  apps: "apps",
}

/**
 * BillingProfile represents a billing profile
 */
@friendlyName("BillingProfile")
model BillingProfile {
  ...global.Resource;

  /**
   * The name and contact information for the supplier this billing profile represents
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  supplier: BillingParty;

  /**
   * The billing workflow settings for this profile
   */
  @visibility(Lifecycle.Read)
  workflow: BillingWorkflow;

  /**
   * The applications used by this billing profile.
   *
   * Expand settings govern if this includes the whole app object or just the ID references.
   */
  @visibility(Lifecycle.Read)
  apps: BillingProfileAppsOrReference;

  /**
   * Is this the default profile?
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  default: boolean;
}

/**
 * BillingProfileCreate represents the input for creating a billing profile
 */
@friendlyName("BillingProfileCreate")
model BillingProfileCreate
  is TypeSpec.Rest.Resource.ResourceCreateModel<BillingProfile> {
  /**
   * The billing workflow settings for this profile.
   */
  @visibility(Lifecycle.Create)
  workflow: TypeSpec.Rest.Resource.ResourceCreateModel<BillingWorkflow>;

  /**
   * The apps used by this billing profile.
   */
  @visibility(Lifecycle.Create)
  apps: BillingProfileAppsCreate;
}

/**
 * BillingProfileReplaceUpdate represents the input for updating a billing profile
 *
 * The apps field cannot be updated directly, if an app change is desired a new
 * profile should be created.
 */
// Note: Rest.Resource.ResourceReplaceModel<BillingProfile> yields a BillingProfileReplaceUpdate that's why we
// need a different friendly name here.
@friendlyName("BillingProfileReplaceUpdateWithWorkflow")
model BillingProfileReplaceUpdate
  is TypeSpec.Rest.Resource.ResourceReplaceModel<BillingProfile> {
  /**
   * The billing workflow settings for this profile.
   */
  @visibility(Lifecycle.Update)
  workflow: BillingWorkflow;
}

/**
 * BillingProfileApps represents the applications used by a billing profile
 */
@friendlyName("BillingProfileApps")
model BillingProfileApps {
  /**
   * The tax app used for this workflow
   */
  @visibility(Lifecycle.Read)
  tax: App;

  /**
   * The invoicing app used for this workflow
   */
  @visibility(Lifecycle.Read)
  invoicing: App;

  /**
   * The payment app used for this workflow
   */
  @visibility(Lifecycle.Read)
  payment: App;
}

/**
 * BillingProfileAppReferences represents the references (id, type) to the apps used by a billing profile
 */
@friendlyName("BillingProfileAppReferences")
model BillingProfileAppReferences {
  /**
   * The tax app used for this workflow
   */
  @visibility(Lifecycle.Read)
  tax: AppReference;

  /**
   * The invoicing app used for this workflow
   */
  @visibility(Lifecycle.Read)
  invoicing: AppReference;

  /**
   * The payment app used for this workflow
   */
  @visibility(Lifecycle.Read)
  payment: AppReference;
}

/**
 * ProfileAppsOrReference represents the union of ProfileApps and ProfileAppReferences
 * for a billing profile.
 */
@friendlyName("BillingProfileAppsOrReference")
union BillingProfileAppsOrReference {
  profileApps: BillingProfileApps,
  profileAppReferences: BillingProfileAppReferences,
}

/**
 * App reference type specifies the type of reference inside an app reference
 */
@friendlyName("BillingWorkflowAppReferenceType")
enum BillingWorkflowAppReferenceType {
  appId: "app_id",
  appType: "app_type",
}

/**
 * BillingProfileAppsCreate represents the input for creating a billing profile's apps
 */
@friendlyName("BillingProfileAppsCreate")
model BillingProfileAppsCreate {
  /**
   * The tax app used for this workflow
   */
  @visibility(Lifecycle.Create)
  @extension("x-go-type", "string")
  tax: ULID;

  /**
   * The invoicing app used for this workflow
   */
  @extension("x-go-type", "string")
  @visibility(Lifecycle.Create)
  invoicing: ULID;

  /**
   * The payment app used for this workflow
   */
  @extension("x-go-type", "string")
  @visibility(Lifecycle.Create)
  payment: ULID;
}

/**
 * BillingWorkflow represents the settings for a billing workflow.
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
 * Workflow collection specifies how to collect the pending line items for an invoice
 */
@friendlyName("BillingWorkflowCollectionSettings")
model BillingWorkflowCollectionSettings {
  /**
   * The alignment for collecting the pending line items into an invoice.
   */
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
  subscription: "subscription",

  /**
   * Align the collection to the anchor time and cadence.
   */
  anchored: "anchored",
}

const DefaultBillingWorkflowCollectionAlignment: BillingWorkflowCollectionAlignmentSubscription = #{
  type: BillingCollectionAlignmentType.subscription,
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
  type: BillingCollectionAlignmentType.anchored;

  /**
   * The recurring period for the alignment.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  recurringPeriod: RecurringPeriodV2;
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
  type: BillingCollectionAlignmentType.subscription;
}

/**
 * BillingWorkflowPaymentSettings represents the payment settings for a billing workflow
 */
@summary("Workflow payment settings")
@friendlyName("BillingWorkflowPaymentSettings")
model BillingWorkflowPaymentSettings {
  /**
   * The payment method for the invoice.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  collectionMethod?: CollectionMethod = CollectionMethod.chargeAutomatically;
}

/**
 * BillingWorkflowInvoicingSettings represents the invoice settings for a billing workflow
 */
@summary("Workflow invoice settings")
@friendlyName("BillingWorkflowInvoicingSettings")
model BillingWorkflowInvoicingSettings {
  /**
   * Whether to automatically issue the invoice after the draftPeriod has passed.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  autoAdvance?: boolean = true;

  /**
   * The period for the invoice to be kept in draft status for manual reviews.
   */
  @encode(DurationKnownEncoding.ISO8601)
  @example("P1D")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  draftPeriod?: string = "P0D";

  /**
   * The period after which the invoice is due.
   * With some payment solutions it's only applicable for manual collection method.
   */
  @encode(DurationKnownEncoding.ISO8601)
  @example("P30D")
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  dueAfter?: string = "P30D";

  /**
   * Should progressive billing be allowed for this workflow?
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  progressiveBilling?: boolean = true;

  /**
   * Default tax configuration to apply to the invoices.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  defaultTaxConfig?: TaxConfig;
}

/**
 * BillingWorkflowTaxSettings represents the tax settings for a billing workflow
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
   * When enabled, OpenMeter will not allow to create an invoice without tax calculation.
   * Enforcement is different per apps, for example, Stripe app requires customer
   * to have a tax location when starting a paid subscription.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  enforced?: boolean = false;
}

/**
 * CollectionMethod specifies how the invoice should be collected (automatic vs manual)
 */
@friendlyName("CollectionMethod")
@summary("Collection method")
enum CollectionMethod {
  chargeAutomatically: "charge_automatically",
  sendInvoice: "send_invoice",
}

/**
 * BillingWorkflowLineResolution specifies how the line items should be resolved in the invoice
 */
@friendlyName("BillingWorkflowLineResolution")
@summary("Item resolution")
enum BillingWorkflowLineResolution {
  /**
   * Create line items for each day in the billing period.
   */
  day: "day",

  /**
   * Create one line item for the entire billing period.
   */
  period: "period",
}
