import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter.Billing;

@route("/api/v1/billing/customer")
interface Customers {
  /**
   * List customer overrides
   */
  @get
  @summary("List customer overrides")
  // TODO: tags
  list(
    @query
    billingProfile?: Array<ULID>,

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryLimitOffset,
    ...OpenMeter.QueryOrdering<CustomerOverrideOrderBy>,
  ): OpenMeter.PaginatedResponse<CustomerOverride> | OpenMeter.CommonErrors;

  /**
   * Create/update a new customer override.
   */
  @post
  @route("/:id")
  @summary("Create/update a customer override")
  create(
    @path
    customerID: ULID,

    @body
    request: CustomerOverrideCreateRequest,
  ): {
    @statusCode _: 200;
    @body body: CustomerOverride;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Get a customer override by id.
   */
  @get
  @route("/:id")
  @summary("Get a customer override")
  get(
    @path
    customerID: ULID,
  ): CustomerOverride | OpenMeter.NotFoundError | OpenMeter.CommonErrors;

  /**
   * Delete a customer override by id.
   */
  @delete
  @route("/:id")
  @summary("Delete a customer override")
  delete(
    @path
    customerID: ULID,
  ): {
    @statusCode _: 204;
  } | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}

/**
 * Order by options for customers.
 */
@friendlyName("CustomerOrderBy")
enum CustomerOverrideOrderBy {
  id: "id",
  created: "created",
  updated: "updated",
}

model CustomerOverrideCreateRequest {
  billingProfileID: ULID;
}

model CustomerOverride {
  ...ResourceTimestamps;

  /**
   * A unique identifier for the resource.
   */
  @visibility("read")
  @example("01G65Z755AFWAKHE12NY0CQ9FH")
  @summary("ID")
  id: ULID;

  workflow: WorkflowOverride;
}

/**
 * Workflow represents a billing workflow
 */
@friendlyName("WorkflowOverride")
model WorkflowOverride {
  collectionAlignment: CollectionAlignment;
  itemCollectionPeriod: Period; // TODO: type
  invoice: WorkflowInvoice;
}

/**
 * WorkflowInvoice represents the invoice settings for a billing workflow
 */
@summary("Workflow invoice settings")
@friendlyName("WorkflowOverrideInvoice")
model WorkflowOverrideInvoice {
  autoAdvance: boolean;
  draftPeriod: Period;
  dueAfter: Period;
  collectionMethod: CollectionMethod;
  itemResolution: ItemResolution;
  itemPerSubject: boolean;
}
