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
  @operationId("billingListProfiles")
  @summary("List billing profiles")
  @extension("x-internal", true)
  list(
    @query
    includeArchived?: boolean = false,

    @query
    expand?: ProfileExpand[],

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<ProfileOrderBy>,
  ): OpenMeter.PaginatedResponse<Profile> | OpenMeter.CommonErrors;

  /**
   * Create a new billing profile
   *
   * Billing profiles are representations of a customer's billing information. Customer overrides
   * can be applied to a billing profile to customize the billing behavior for a specific customer.
   */
  @post
  @summary("Create a new billing profile")
  @operationId("billingCreateProfile")
  @extension("x-internal", true)
  create(@body profile: ProfileCreate): {
    @statusCode _: 201;
    @body profile: Profile;
  } | OpenMeter.CommonErrors;

  /**
   * Delete a billing profile
   */
  @route("/{id}")
  @delete
  @summary("Delete a billing profile")
  @operationId("billingDeleteProfile")
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
  @operationId("billingGetProfile")
  @extension("x-internal", true)
  get(
    @path
    id: ULID,

    @query expand?: ProfileExpand[],
  ): Profile | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Update a billing profile
   */
  @route("/{id}")
  @put
  @summary("Update a billing profile")
  @operationId("billingUpdateProfile")
  @extension("x-internal", true)
  update(
    @path
    id: ULID,

    @body profile: Rest.Resource.ResourceReplaceModel<Profile>,
  ): Profile | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

/**
 * ProfileOrderBy specifies the ordering options for profiles
 */
@friendlyName("BillingProfileOrderBy")
enum ProfileOrderBy {
  createdAt: "createdAt",
  updatedAt: "updatedAt",
  default: "default",
  name: "name",
}

/**
 * ProfileExpand details what profile fields to expand
 */
@friendlyName("BillingProfileExpand")
@extension("x-enum-varnames", ["apps", "all"])
enum ProfileExpand {
  apps: "apps",
  all: "*",
}

/**
 * Profile represents a billing profile
 */
@friendlyName("BillingProfile")
model Profile {
  ...global.Resource;

  /**
   * The name and contact information for the supplier this billing profile represents
   */
  @visibility("read", "create", "update")
  supplier: Invoices.Party;

  /**
   * The billing workflow settings for this profile
   */
  @visibility("read")
  workflow: Workflow;

  /**
   * The applications used by this billing profile.
   *
   * Expand settings govern if this includes the whole app object or just the ID references.
   */
  @visibility("read")
  apps: ProfileAppsOrReference;

  /**
   * Is this the default profile?
   */
  @visibility("read", "create", "update")
  default: boolean;
}

/**
 * ProfileCreateInput represents the input for creating a billing profile
 */
@friendlyName("BillingProfileCreate")
model ProfileCreate is Rest.Resource.ResourceCreateModel<Profile> {
  /**
   * The billing workflow settings for this profile.
   */
  @visibility("create")
  workflow: Rest.Resource.ResourceCreateModel<Workflow>;

  /**
   * The apps used by this billing profile.
   */
  @visibility("create")
  apps: ProfileAppsCreate;
}

/**
 * ProfileReplaceUpdate represents the input for updating a billing profile
 *
 * The apps field cannot be updated directly, if an app change is desired a new
 * profile should be created.
 */
@friendlyName("BillingProfileReplaceUpdate")
model ProfileReplaceUpdate is Rest.Resource.ResourceReplaceModel<Profile> {
  /**
   * The billing workflow settings for this profile.
   */
  @visibility("update")
  workflow: Workflow;
}

/**
 * ProfileApps represents the applications used by a billing profile
 */
@friendlyName("BillingProfileApps")
model ProfileApps {
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
 * ProfileAppsReference represents the references (id, type) to the apps used by a billing profile
 */
@friendlyName("BillingProfileAppReferences")
model ProfileAppReferences {
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
union ProfileAppsOrReference {
  profileApps: ProfileApps,
  profileAppReferences: ProfileAppReferences,
}

/**
 * App reference type specifies the type of reference inside an app reference
 */
@friendlyName("BillingWorkflowAppReferenceType")
enum AppReferenceType {
  appId: "app_id",
  appType: "app_type",
}

/**
 * ProfileAppsCreate represents the input for creating a billing profile's apps
 */
@friendlyName("BillingProfileAppsCreate")
model ProfileAppsCreate {
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
 * Workflow represents a billing workflow
 */
@friendlyName("BillingWorkflow")
model Workflow {
  ...ResourceTimestamps;
  ...WorkflowSettings;

  /**
   * A unique identifier for the resource.
   */
  @visibility("read")
  @example("01G65Z755AFWAKHE12NY0CQ9FH")
  id: ULID;
}

/**
 * WorkflowSettings represents the settings for a billing workflow.
 */
@friendlyName("BillingWorkflowSettings")
model WorkflowSettings {
  /**
   * The collection settings for this workflow
   */
  @visibility("read", "create", "update")
  collection?: WorkflowCollectionSettings;

  /**
   * The invoicing settings for this workflow
   */
  @visibility("read", "create", "update")
  invoicing?: WorkflowInvoicingSettings;

  /**
   * The payment settings for this workflow
   */
  @visibility("read", "create", "update")
  payment?: WorkflowPaymentSettings;
}

/**
 * Workflow collection specifies how to collect the pending line items for an invoice
 */
@friendlyName("BillingWorkflowCollectionSettings")
model WorkflowCollectionSettings {
  alignment?: WorkflowCollectionAlignment = #{
    type: CollectionAlignment.subscription,
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
 * CollectionAlignment specifies when the pending line items should be collected into
 * an invoice.
 */
@friendlyName("BillingCollectionAlignment")
@summary("Collection alignment")
enum CollectionAlignment {
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
union WorkflowCollectionAlignment {
  subscription: WorkflowCollectionAlignmentSubscription,
}

/**
 * WorkflowCollectionAlignmentSubscription specifies the alignment for collecting the pending line items
 * into an invoice.
 */
@friendlyName("BillingWorkflowCollectionAlignmentSubscription")
model WorkflowCollectionAlignmentSubscription {
  @visibility("read", "create", "update")
  type: CollectionAlignment.subscription;
}

/**
 * WorkflowPaymentSettings represents the payment settings for a billing workflow
 */
@summary("Workflow payment settings")
@friendlyName("BillingWorkflowPaymentSettings")
model WorkflowPaymentSettings {
  /**
   * The payment method for the invoice.
   */
  @visibility("read", "create", "update")
  collectionMethod?: CollectionMethod = CollectionMethod.chargeAutomatically;
}

/**
 * WorkflowInvoice represents the invoice settings for a billing workflow
 */
@summary("Workflow invoice settings")
@friendlyName("BillingWorkflowInvoicingSettings")
model WorkflowInvoicingSettings {
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
@friendlyName("BillingWorkflowCollectionMethod")
@summary("Collection method")
enum CollectionMethod {
  chargeAutomatically: "charge_automatically",
  sendInvoice: "send_invoice",
}

/**
 * LineResolution specifies how the line items should be resolved in the invoice
 */
@friendlyName("BillingWorkflowLineResolution")
@summary("Item resolution")
enum LineResolution {
  /**
   * Create line items for each day in the billing period.
   */
  day: "day",

  /**
   * Create one line item for the entire billing period.
   */
  period: "period",
}
