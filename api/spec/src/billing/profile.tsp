import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;

@route("/api/v1/billing/profile")
interface Profiles {
  /**
   * List all billing profiles
   */
  @get
  @operationId("listBillingProfiles")
  list(
    @query
    includeArchived?: boolean = false,

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
  @operationId("createBillingProfile")
  create(@body profile: Profile): {
    @statusCode _: 201;
    @body profile: Profile;
  } | OpenMeter.CommonErrors;

  /**
   * Archive a billing profile
   */
  @route("/:id")
  @delete
  @operationId("archiveBillingProfile")
  archive(@path id: string): {
    @statusCode _: 204;
  } | OpenMeter.CommonErrors | OpenMeter.NotFoundError;

  /**
   * Get a billing profile by ID
   */
  @route("/:id")
  @get
  @operationId("getBillingProfile")
  get(@path id: string): Profile | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Update a billing profile
   */
  @route("/:id")
  @post // TODO: align it with notifications
  @operationId("updateBillingProfile")
  update(@path id: string, @body profile: Profile): {
    @statusCode _: 200;
    @body profile: Profile;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  // Customer overrides
}

/**
 * ProfileOrderBy specifies the ordering options for profiles
 */
@friendlyName("ProfileOrderBy")
enum ProfileOrderBy {
  created: "created",
  updated: "updated",
}

/**
 * Profile represents a billing profile
 */
@friendlyName("Profile")
model Profile {
  ...global.Resource;
  supplier: Supplier;
  workflow: Workflow;
  default: boolean;
}

/**
 * Supplier represents a supplier
 */
@friendlyName("Supplier")
model Supplier {
  ...BillingAddress;
  name: string;
}

/**
 * Workflow represents a billing workflow
 */
@friendlyName("Workflow")
model Workflow {
  ...ResourceTimestamps;

  /**
   * A unique identifier for the resource.
   */
  @visibility("read")
  @example("01G65Z755AFWAKHE12NY0CQ9FH")
  @summary("ID")
  id: ULID;

  collectionAlignment: CollectionAlignment;
  itemCollectionPeriod: Period; // TODO: type
  invoice: WorkflowInvoice;

  // provider settings for creation (update is not supported)
  @visibility("create")
  taxAppID?: ULID;

  @visibility("create")
  invoicingAppID?: ULID;

  @visibility("create")
  paymentAppID?: ULID;

  // TODO: let's add the APP references here for get etc.
}

/**
 * CollectionAlignment specifies when the pending line items should be collected into
 * an invoice.
 */
@friendlyName("CollectionAlignment")
@summary("Collection alignment")
enum CollectionAlignment {
  /**
   * Align the collection to the start of the subscription period.
   */
  subscription: "subscription",
}

/**
 * WorkflowInvoice represents the invoice settings for a billing workflow
 */
@summary("Workflow invoice settings")
@friendlyName("WorkflowInvoice")
model WorkflowInvoice {
  autoAdvance: boolean;
  draftPeriod: Period;
  dueAfter: Period;
  collectionMethod: CollectionMethod;
  itemResolution: ItemResolution;
  itemPerSubject: boolean;
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
 * ItemResolution specifies how the line items should be resolved in the invoice
 */
@friendlyName("ItemResolution")
@summary("Item resolution")
enum ItemResolution {
  /**
   * Create line items for each day in the billing period.
   */
  day: "day",

  /**
   * Create one line item for the entire billing period.
   */
  period: "period",
}
