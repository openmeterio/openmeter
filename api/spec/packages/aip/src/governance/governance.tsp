import "../shared/index.tsp";
import "../common/pagination.tsp";
import "../customers/customer.tsp";

namespace Governance;

/**
 * List of customer identifiers to evaluate access for.
 */
@friendlyName("GovernanceQueryRequestCustomers")
model GovernanceQueryRequestCustomers {
  /**
   * Each entry can be a customer `key` or a usage-attribution subject `key`.
   * Identifiers that cannot be resolved to a customer are reported in the response
   * `errors` array.
   */
  @summary("Customer keys and usage-attribution subjects")
  @minItems(1)
  @maxItems(100)
  keys: string[];
}

/**
 * Optional list of feature keys to evaluate access for. If omitted, all features
 * available in the organization are returned. Providing this list is recommended
 * to reduce the response size and the load on the backend services.
 */
@friendlyName("GovernanceQueryRequestFeatures")
model GovernanceQueryRequestFeatures {
  /**
   * List of feature keys to evaluate access for.
   */
  @summary("Feature Keys")
  @minItems(1)
  @maxItems(100)
  keys: string[];
}

/**
 * Query to evaluate feature access for a list of customers.
 */
@friendlyName("GovernanceQueryRequest")
model GovernanceQueryRequest {
  /**
   * Whether to include credit balance availability for each resolved customer. When
   * true, each result entry includes `has_available_credits`.
   *
   * Defaults to `false`.
   */
  @summary("Include credits")
  include_credits?: boolean = false;

  #suppress "@openmeter/api-spec-aip/doc-decorator" "-"
  @summary("Customer")
  customer: GovernanceQueryRequestCustomers;

  #suppress "@openmeter/api-spec-aip/doc-decorator" "-"
  @summary("Feature")
  feature?: GovernanceQueryRequestFeatures;
}

/**
 * Response of the governance query.
 */
@friendlyName("GovernanceQueryResponse")
model GovernanceQueryResponse {
  /**
   * Access evaluation results, one entry per resolved customer.
   */
  @visibility(Lifecycle.Read)
  @summary("Data")
  data: GovernanceQueryResult[];

  /**
   * Partial errors encountered while processing the request.
   */
  @visibility(Lifecycle.Read)
  @summary("Errors")
  errors: GovernanceQueryError[];

  /**
   * Pagination metadata. The endpoint may return a partial response if the full
   * response would exceed server-side limits.
   */
  @visibility(Lifecycle.Read)
  @summary("Meta")
  meta: Common.CursorMeta;
}

/**
 * Access evaluation result for a single resolved customer.
 */
@friendlyName("GovernanceQueryResult")
model GovernanceQueryResult {
  /**
   * The list of identifiers from the request that resolved to this customer. Each
   * entry is either the customer `key` or one of its usage-attribution subject
   * `key`s.
   *
   * Duplicate or aliased identifiers that resolve to the same customer collapse to a
   * single result entry, with every requested identifier listed here.
   */
  @visibility(Lifecycle.Read)
  @summary("Matched identifiers")
  matched: string[];

  /**
   * The customer the matched identifiers resolved to.
   */
  @visibility(Lifecycle.Read)
  @summary("Customer")
  customer: Customers.Customer;

  /**
   * Map of features with their access status.
   *
   * Map keys are the feature keys requested in `filters.features.keys`, or every
   * feature `key` available in the organization when the feature filter was omitted.
   */
  @visibility(Lifecycle.Read)
  @summary("Features")
  features: Record<GovernanceFeatureAccess>;

  /**
   * Timestamp of the most recent change to the customer's access state reflected in
   * this result.
   */
  @visibility(Lifecycle.Read)
  @summary("Updated at")
  updated_at: Shared.DateTime;
}

/**
 * Access status for a single feature.
 */
@friendlyName("GovernanceFeatureAccess")
model GovernanceFeatureAccess {
  /**
   * Whether the customer currently has access to the feature.
   *
   * `true` for boolean and static entitlements that are available, and for metered
   * entitlements with remaining balance. `false` when the feature is unavailable,
   * the usage limit has been reached, or (when applicable) credits have been
   * exhausted.
   */
  @visibility(Lifecycle.Read)
  @summary("Has access")
  has_access: boolean;

  /**
   * Optional reason when the customer does not have access to the feature. Populated
   * when `has_access` is `false`.
   */
  @visibility(Lifecycle.Read)
  @summary("Reason")
  reason?: GovernanceFeatureAccessReason;
}

/**
 * Reason a feature is not accessible to a customer.
 */
@friendlyName("GovernanceFeatureAccessReason")
model GovernanceFeatureAccessReason
  is Shared.BaseError<GovernanceFeatureAccessReasonCode>;

/**
 * Machine-readable reason code for denied feature access.
 */
@friendlyName("GovernanceFeatureAccessReasonCode")
enum GovernanceFeatureAccessReasonCode {
  /**
   * Default zero value. Reserved for forward compatibility.
   */
  Unknown: "unknown",

  /**
   * The customer has reached the metered usage limit for the feature
   * within the current usage period.
   */
  UsageLimitReached: "usage_limit_reached",

  /**
   * The feature is not available to the customer.
   */
  FeatureUnavailable: "feature_unavailable",

  /**
   * The feature `key` referenced by the request is not configured in the
   * organization.
   */
  FeatureNotFound: "feature_not_found",

  /**
   * The customer has no available prepaid credit balance.
   */
  NoCreditAvailable: "no_credit_available",
}

/**
 * Query error within a partially successful governance query response.
 */
@friendlyName("GovernanceQueryError")
model GovernanceQueryError is Shared.BaseError<GovernanceQueryErrorCode> {
  /**
   * The customer identifier from the request that produced this error.
   */
  @visibility(Lifecycle.Read)
  @summary("Customer identifier")
  customer?: string;
}

/**
 * Error code for a governance query failure.
 */
@friendlyName("GovernanceQueryErrorCode")
enum GovernanceQueryErrorCode {
  /**
   * Default zero value. Reserved for forward compatibility.
   */
  Unknown: "unknown",

  /**
   * The provided identifier could not be resolved to any customer
   * (neither by customer `key` nor by a usage-attribution subject `key`).
   */
  CustomerNotFound: "customer_not_found",
}
