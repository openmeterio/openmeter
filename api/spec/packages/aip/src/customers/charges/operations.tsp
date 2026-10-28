import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../common/error.tsp";
import "../../common/pagination.tsp";
import "../../shared/index.tsp";
import "../../common/parameters.tsp";
import "./charges.tsp";
import "../../invoices/index.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Customers;

/**
 * Filter options for listing charges.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "feature_id and feature_key should not be grouped"
@friendlyName("ListCustomerChargesParamsFilter")
model ListCustomerChargesParamsFilter {
  /**
   * Filter charges by status.
   *
   * Supported statuses are:
   *
   * - `created`
   * - `active`
   * - `final`
   * - `deleted`
   *
   * If omitted, all statuses are returned except for `deleted`.
   */
  status?: Common.StringFieldFilterExact;

  /**
   * Filter charges by the ID of their associated feature.
   */
  feature_id?: Common.ULIDFieldFilter;

  /**
   * Filter charges by the key of their associated feature.
   */
  feature_key?: Common.StringFieldFilterExact;

  /**
   * Filter charges by the start of their service period.
   */
  service_period_from?: Common.DateTimeFieldFilter;

  /**
   * Filter charges by the end of their service period.
   */
  service_period_to?: Common.DateTimeFieldFilter;

  /**
   * Filter by charge last update time. `updated_at` changes whenever the charge
   * itself is saved. A change saved only to its realizations may not update it, and
   * live usage moving on an open charge does not. Deleted charges are listed only
   * when `filter[status]` selects `deleted`. For incremental syncs, filter with
   * `gte` on the last seen value and de-duplicate by `id`.
   */
  updated_at?: Common.DateTimeFieldFilter;
}

/**
 * Options for manually collecting a customer's pending lines into invoices.
 */
@friendlyName("InvoiceCustomerChargesInput")
model InvoiceCustomerChargesInput {
  /**
   * Collection cutoff. Defaults to the current time and cannot be in the future.
   */
  as_of?: Shared.DateTime;

  /**
   * Override the customer's progressive billing setting for this collection. When
   * enabled, eligible usage-based lines can be collected for a partial period.
   */
  progressive_billing_override?: boolean;
}

/**
 * Result of manually collecting a customer's pending lines into invoices.
 */
@friendlyName("InvoiceCustomerChargesResult")
model InvoiceCustomerChargesResult {
  /**
   * Created standard invoices, one per collected currency.
   */
  invoices: Invoices.Invoice[];
}

namespace Charges {
  interface CustomerChargesOperations {
    /**
     * List customer charges.
     *
     * Returns the customer's charges that are represented as either flat fee or
     * usage-based charges.
     */
    @extension(Shared.UnstableExtension, true)
    @get
    @operationId("list-customer-charges")
    @summary("List customer charges")
    list(
      ...Common.PagePaginationQuery,
      @path customerId: Shared.ULID,

      /**
       * Sort charges returned in the response.
       *
       * Supported sort attributes are:
       *
       * - `id`
       * - `created_at`
       * - `service_period.from`
       * - `billing_period.from`
       */
      @query(#{ name: "sort" })
      sort?: Common.SortQuery,

      /**
       * Filter charges.
       *
       * To filter charges by status add the following query param:
       * `filter[status][oeq]=created,active`
       *
       * To filter charges by feature, use `filter[feature_id][oeq]=<id>,<id>` or
       * `filter[feature_key][oeq]=<key>,<key>`.
       *
       * See the `service_period_from` filter field for expressing a service-period
       * window query.
       */
      @query(#{ style: "deepObject", explode: true })
      filter?: ListCustomerChargesParamsFilter,

      /**
       * Expand full objects for referenced entities.
       *
       * Supported values are:
       *
       * - `real_time_usage`: Expand the charge's real-time usage; it sets the `usage`
       *   and the `totals.realtime` fields.
       * - `customer`: Expand the charge's customer to the complete entity.
       * - `feature`: Expand the charge's feature to the complete entity.
       * - `subscription`: Expand the charge's subscription to the complete entity.
       * - `realization.invoice`: Expand each realization's invoice to the complete
       *   entity.
       * - `realization.totals`: Expand each realization run's `totals`.
       * - `realization.detailed_lines`: Expand each realization run's `detailed_lines`.
       */
      @query(#{ explode: false })
      expand?: ChargesExpand[],
    ):
      | Shared.PagePaginatedResponse<Charge>
      | Common.NotFound
      | Common.ErrorResponses;

    /**
     * Create customer charge.
     */
    @extension(Shared.UnstableExtension, true)
    @post
    @operationId("create-customer-charges")
    @summary("Create customer charge")
    create(@path customerId: Shared.ULID, @body charge: CreateCharge):
      | Shared.CreateResponse<Charge>
      | Common.ErrorResponses;

    /**
     * Invoice a customer's charges outside the normal billing cycle. Collection
     * alignment is bypassed. The customer's progressive billing setting applies unless
     * explicitly overridden. All eligible pending lines are considered.
     *
     * Returns the created standard invoices, one per collected currency. Lines blocked
     * by their owning engine remain pending. If all eligible lines are blocked or no
     * lines are eligible, returns a validation error.
     */
    @post
    @route("/invoice")
    @operationId("invoice-customer-charges")
    @summary("Invoice a customer's charges")
    @extension(Shared.RPCEndpointExtension, true)
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.PrivateExtension, true)
    @extension(Shared.InternalExtension, true)
    invoiceCharges(
      @path customerId: Shared.ULID,
      @body request?: InvoiceCustomerChargesInput,
    ):
      | {
          @statusCode _: 201;
          @body body: InvoiceCustomerChargesResult;
        }
      | Common.NotFound
      | Common.ErrorResponses;
  }
}
