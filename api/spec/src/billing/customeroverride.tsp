using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;

@route("/api/v1/billing/customers")
@tag("Billing")
@friendlyName("CustomerOverrides")
interface CustomerOverridesEndpoints {
  /**
   * List customer overrides using the specified filters.
   *
   * The response will include the customer override values and the merged billing profile values.
   *
   * If the includeAllCustomers is set to true, the list contains all customers. This mode is
   * useful for getting the current effective billing workflow settings for all users regardless
   * if they have customer orverrides or not.
   */
  @get
  @summary("List customer overrides")
  @operationId("listBillingProfileCustomerOverrides")
  list(
    ...ListCustomerOverridesParams,
    ...QueryPagination,
  ): PaginatedResponse<BillingProfileCustomerOverrideWithDetails> | CommonErrors;

  /**
   * The customer override can be used to pin a given customer to a billing profile
   * different from the default one.
   *
   * This can be used to test the effect of different billing profiles before making them
   * the default ones or have different workflow settings for example for enterprise customers.
   */
  @put
  @route("/{customerId}")
  @operationId("upsertBillingProfileCustomerOverride")
  @summary("Create a new or update a customer override")
  upsert(
    @path
    customerId: ULID,

    @body
    request: BillingProfileCustomerOverrideCreate,
  ): BillingProfileCustomerOverrideWithDetails | NotFoundError | CommonErrors;

  /**
   * Get a customer override by customer id.
   *
   * The response will include the customer override values and the merged billing profile values.
   *
   * If the customer override is not found, the default billing profile's values are returned. This behavior
   * allows for getting a merged profile regardless of the customer override existence.
   */
  @get
  @route("/{customerId}")
  @summary("Get a customer override")
  @operationId("getBillingProfileCustomerOverride")
  get(
    @path
    customerId: ULID,

    @query(#{ explode: true })
    expand?: BillingProfileCustomerOverrideExpand[],
  ): BillingProfileCustomerOverrideWithDetails | NotFoundError | CommonErrors;

  /**
   * Delete a customer override by customer id.
   *
   * This will remove the customer override and the customer will be subject to the default
   * billing profile's settings again.
   */
  @delete
  @route("/{customerId}")
  @summary("Delete a customer override")
  @operationId("deleteBillingProfileCustomerOverride")
  delete(
    @path
    customerId: ULID,
  ): {
    @statusCode _: 204;
  } | NotFoundError | CommonErrors;
}

/**
 * Order by options for customers.
 */
@friendlyName("BillingProfileCustomerOverrideOrderBy")
enum BillingProfileCustomerOverrideOrderBy {
  customerId: "customerId",
  customerName: "customerName",
  customerKey: "customerKey",
  customerPrimaryEmail: "customerPrimaryEmail",
  customerCreatedAt: "customerCreatedAt",
}

/**
 * CustomerOverrideExpand specifies the parts of the profile to expand.
 */
@friendlyName("BillingProfileCustomerOverrideExpand")
enum BillingProfileCustomerOverrideExpand {
  apps: "apps",
  customer: "customer",
}

/**
 * Customer override values.
 */
@friendlyName("BillingProfileCustomerOverride")
model BillingProfileCustomerOverride {
  ...OmitProperties<
    ResourceTimestamps,
    "deletedAt" // This is not interesting as we are not allowing deleted listing
  >;

  // TODO: this is not supported yet, let's enable it once we choose to add wrokflow overrides here.
  // workflow?: BillingProfileCustomerWorkflowOverride;

  /**
   * The billing profile this override is associated with.
   *
   * If empty the default profile is looked up dynamically.
   */
  billingProfileId?: ULID;

  /**
   * The customer id this override is associated with.
   */
  customerId: ULID;
}

/**
 * Customer specific merged profile.
 *
 * This profile is calculated from the customer override and the billing profile it references or the default.
 *
 * Thus this does not have any kind of resource fields, only the calculated values.
 */
@friendlyName("BillingCustomerProfile")
model BillingCustomerProfile {
  /**
   * The name and contact information for the supplier this billing profile represents
   */
  @visibility(Lifecycle.Read)
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
}

/**
 * Customer specific workflow overrides.
 */
@friendlyName("BillingProfileCustomerOverrideWithDetails")
model BillingProfileCustomerOverrideWithDetails {
  /**
   * The customer override values.
   *
   * If empty the merged values are calculated based on the default profile.
   */
  customerOverride?: BillingProfileCustomerOverride;

  /**
   * The billing profile the customerProfile is associated with at the time of query.
   *
   * customerOverride contains the explicit mapping set in the customer override object. If that is
   * empty, then the baseBillingProfileId is the default profile.
   */
  baseBillingProfileId: ULID;

  /**
   * Merged billing profile with the customer specific overrides.
   */
  customerProfile?: BillingCustomerProfile;

  /**
   * The customer this override belongs to.
   */
  customer?: Customer;
}

/**
 * List customer overrides query parameters.
 */
@friendlyName("BillingProfileListCustomerOverridesParams")
model ListCustomerOverridesParams {
  /**
   * Filter by billing profile.
   */
  @query(#{ explode: true })
  billingProfile?: ULID[];

  /**
   * Only return customers without pinned billing profiles. This implicitly sets includeAllCustomers to true.
   */
  @query(#{ explode: true })
  customersWithoutPinnedProfile?: boolean;

  /**
   * Include customers without customer overrides.
   *
   * If set to false only the customers specifically associated with a billing profile will be returned.
   *
   * If set to true, in case of the default billing profile, all customers will be returned.
   */
  @query(#{ explode: true })
  includeAllCustomers?: boolean = true;

  /**
   * Filter by customer id.
   */
  @query(#{ explode: true })
  customerId?: ULID[];

  /**
   * Filter by customer name.
   */
  @query
  customerName?: string;

  /**
   * Filter by customer key
   */
  @query
  customerKey?: string;

  /**
   * Filter by customer primary email
   */
  @query
  customerPrimaryEmail?: string;

  /**
   * Expand the response with additional details.
   */
  @query(#{ explode: true })
  expand?: BillingProfileCustomerOverrideExpand[];

  /**
   * Order the response by.
   */
  ...QueryOrdering<BillingProfileCustomerOverrideOrderBy>;
}
/**
 * Payload for creating a new or updating an existing customer override.
 */
@friendlyName("BillingProfileCustomerOverrideCreate")
model BillingProfileCustomerOverrideCreate {
  // TODO: this is not supported yet, let's enable it once we choose to add workflow overrides here.
  // workflow?: Rest.Resource.ResourceCreateModel<BillingProfileCustomerWorkflowOverride>;

  /**
   * The billing profile this override is associated with.
   *
   * If not provided, the default billing profile is chosen if available.
   */
  billingProfileId?: ULID;
}

/**
 * Customer specific workflow overrides.
 */
@friendlyName("BillingProfileCustomerWorkflowOverride")
model BillingProfileCustomerWorkflowOverride {
  ...BillingWorkflow;

  // Note: these are only available for read, as provider override is not supported via the customer override.
  // to do that the customer should be assocatied with a new billing profile instead.

  /**
   * The tax app used for this workflow
   */
  @visibility(Lifecycle.Read)
  taxApp: App;

  /**
   * The invoicing app used for this workflow
   */
  @visibility(Lifecycle.Read)
  invoicingApp: App;

  /**
   * The payment app used for this workflow
   */
  @visibility(Lifecycle.Read)
  paymentApp: App;
}
