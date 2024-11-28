import "../rest.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;

@route("/api/v1/billing/profile")
@tag("Billing (Experimental)")
interface Profiles {
  /**
   * List all billing profiles
   */
  @get
  @operationId("listBillingProfiles")
  @summary("List billing profiles")
  @extension("x-internal", true)
  list(
    @query
    includeArchived?: boolean = false,

    @query
    expand?: BillingProfileExpand[],

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<BillingProfileOrderBy>,
  ): OpenMeter.PaginatedResponse<BillingProfile> | OpenMeter.CommonErrors;

  /**
   * Create a new billing profile
   *
   * Billing profiles are representations of a customer's billing information. Customer overrides
   * can be applied to a billing profile to customize the billing behavior for a specific customer.
   */
  @post
  @summary("Create a new billing profile")
  @operationId("createBillingProfile")
  @extension("x-internal", true)
  create(@body profile: BillingProfileCreate): {
    @statusCode _: 201;
    @body profile: BillingProfile;
  } | OpenMeter.CommonErrors;

  /**
   * Delete a billing profile
   */
  @route("/{id}")
  @delete
  @summary("Delete a billing profile")
  @operationId("deleteBillingProfile")
  @extension("x-internal", true)
  delete(
    @path
    id: ULID,
  ): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get a billing profile by ID
   */
  @route("/{id}")
  @get
  @summary("Get a billing profile")
  @operationId("getBillingProfile")
  @extension("x-internal", true)
  get(
    @path
    id: ULID,

    @query expand?: BillingProfileExpand[],
  ): BillingProfile | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Update a billing profile
   */
  @route("/{id}")
  @put
  @summary("Update a billing profile")
  @operationId("updateBillingProfile")
  @extension("x-internal", true)
  update(
    @path
    id: ULID,

    @body profile: Rest.Resource.ResourceReplaceModel<BillingProfile>,
  ): BillingProfile | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
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
@extension("x-enum-varnames", ["apps", "all"])
enum BillingProfileExpand {
  apps: "apps",
  all: "*",
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
  @visibility("read", "create", "update")
  supplier: Invoices.BillingParty;

  /**
   * The billing workflow settings for this profile
   */
  @visibility("read")
  workflow: BillingWorkflow;

  /**
   * The applications used by this billing profile.
   *
   * Expand settings govern if this includes the whole app object or just the ID references.
   */
  @visibility("read")
  apps: BillingProfileAppsOrReference;

  /**
   * Is this the default profile?
   */
  @visibility("read", "create", "update")
  default: boolean;
}

/**
 * BillingProfileCreate represents the input for creating a billing profile
 */
@friendlyName("BillingProfileCreate")
model BillingProfileCreate
  is Rest.Resource.ResourceCreateModel<BillingProfile> {
  /**
   * The billing workflow settings for this profile.
   */
  @visibility("create")
  workflow: Rest.Resource.ResourceCreateModel<BillingWorkflow>;

  /**
   * The apps used by this billing profile.
   */
  @visibility("create")
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
  is Rest.Resource.ResourceReplaceModel<BillingProfile> {
  /**
   * The billing workflow settings for this profile.
   */
  @visibility("update")
  workflow: BillingWorkflow;
}

/**
 * BillingProfileApps represents the applications used by a billing profile
 */
@friendlyName("BillingProfileApps")
model BillingProfileApps {
  /*
   * The tax app used for this workflow
   */
  @visibility("read")
  tax: OpenMeter.App.App;

  /*
   * The invoicing app used for this workflow
   */
  @visibility("read")
  invoicing: OpenMeter.App.App;

  /*
   * The payment app used for this workflow
   */
  @visibility("read")
  payment: OpenMeter.App.App;
}

/**
 * BillingProfileAppReferences represents the references (id, type) to the apps used by a billing profile
 */
@friendlyName("BillingProfileAppReferences")
model BillingProfileAppReferences {
  /**
   * The tax app used for this workflow
   */
  @visibility("read")
  tax: OpenMeter.App.AppReference;

  /**
   * The invoicing app used for this workflow
   */
  @visibility("read")
  invoicing: OpenMeter.App.AppReference;

  /**
   * The payment app used for this workflow
   */
  @visibility("read")
  payment: OpenMeter.App.AppReference;
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
  @visibility("create")
  @extension("x-go-type", "string")
  tax: ULID | string;

  /**
   * The invoicing app used for this workflow
   */
  @extension("x-go-type", "string")
  @visibility("create")
  invoicing: ULID | string;

  /**
   * The payment app used for this workflow
   */
  @extension("x-go-type", "string")
  @visibility("create")
  payment: ULID | string;
}

/**
 * BillingWorkflow represents a billing workflow
 */
@friendlyName("BillingWorkflow")
model BillingWorkflow {
  ...ResourceTimestamps;
  ...BillingWorkflowSettings;

  /**
   * A unique identifier for the resource.
   */
  @visibility("read")
  @example("01G65Z755AFWAKHE12NY0CQ9FH")
  id: ULID;
}

/**
 * BillingWorkflowSettings represents the settings for a billing workflow.
 */
@friendlyName("BillingWorkflowSettings")
model BillingWorkflowSettings {
  /**
   * The collection settings for this workflow
   */
  @visibility("read", "create", "update")
  collection?: BillingWorkflowCollectionSettings;

  /**
   * The invoicing settings for this workflow
   */
  @visibility("read", "create", "update")
  invoicing?: BillingWorkflowInvoicingSettings;

  /**
   * The payment settings for this workflow
   */
  @visibility("read", "create", "update")
  payment?: BillingWorkflowPaymentSettings;
}

/**
 * Workflow collection specifies how to collect the pending line items for an invoice
 */
@friendlyName("BillingWorkflowCollectionSettings")
model BillingWorkflowCollectionSettings {
  alignment?: BillingWorkflowCollectionAlignment = #{
    type: BillingCollectionAlignment.subscription,
  };

  /**
   * This grace period can be used to delay the collection of the pending line items specified in
   * alignment.
   *
   * This is useful, in case of multiple subscriptions having slightly different billing periods.
   */
  @encode(DurationKnownEncoding.ISO8601)
  @example("P1D")
  @visibility("read", "create", "update")
  interval?: string = "PT1H";
}

/**
 * BillingCollectionAlignment specifies when the pending line items should be collected into
 * an invoice.
 */
@friendlyName("BillingCollectionAlignment")
@summary("Collection alignment")
enum BillingCollectionAlignment {
  /**
   * Align the collection to the start of the subscription period.
   */
  subscription: "subscription",
}

/**
 * The alignment for collecting the pending line items into an invoice.
 *
 * Defaults to subscription, which means that we are to create a new invoice every time the
 * a subscription period starts (for in advance items) or ends (for in arrears items).
 */
@friendlyName("BillingWorkflowCollectionAlignment")
@discriminator("type")
union BillingWorkflowCollectionAlignment {
  subscription: BillingWorkflowCollectionAlignmentSubscription,
}

/**
 * BillingWorkflowCollectionAlignmentSubscription specifies the alignment for collecting the pending line items
 * into an invoice.
 */
@friendlyName("BillingWorkflowCollectionAlignmentSubscription")
model BillingWorkflowCollectionAlignmentSubscription {
  @visibility("read", "create", "update")
  type: BillingCollectionAlignment.subscription;
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
  @visibility("read", "create", "update")
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
  @visibility("read", "create", "update")
  autoAdvance?: boolean = true;

  /**
   * The period for the invoice to be kept in draft status for manual reviews.
   */
  @encode(DurationKnownEncoding.ISO8601)
  @example("P1D")
  @visibility("read", "create", "update")
  draftPeriod?: string = "P1D";

  /**
   * The period after which the invoice is due.
   */
  @encode(DurationKnownEncoding.ISO8601)
  @example("P1D")
  @visibility("read", "create", "update")
  dueAfter?: string = "P7D";
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
