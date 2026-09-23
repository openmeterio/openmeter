import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "../../common/error.tsp";
import "../../shared/index.tsp";
import "./entitlement.tsp";
import "./history.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Customers;

namespace Entitlements {
  interface CustomerEntitlementsOperations {
    /**
     * Create an entitlement for the customer.
     *
     * A customer can have only one active entitlement per feature. The feature must be
     * compatible with the entitlement type. Entitlements cannot be modified after
     * creation, only deleted.
     */
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.PrivateExtension, true)
    @extension(Shared.InternalExtension, true)
    @post
    @operationId("create-customer-entitlement")
    @summary("Create customer entitlement")
    create(
      @path customerId: Shared.ULID,
      @body entitlement: CreateEntitlement,
    ):
      | Shared.CreateResponse<Entitlement>
      | Common.NotFound
      | Common.Conflict
      | Common.ErrorResponses;

    /**
     * Get the balance and usage history of a metered entitlement. The queried range
     * may span multiple usage periods.
     *
     * `windowed_history` groups usage into windows of the requested size and reports
     * the balance at the start of each window. `burndown_history` lists the periods in
     * which grants were consumed in a fixed order, together with the usage taken from
     * each grant.
     */
    @extension(Shared.UnstableExtension, true)
    @extension(Shared.InternalExtension, true)
    @extension(Shared.PrivateExtension, true)
    @get
    @route("/{entitlementId}/history")
    @operationId("get-customer-entitlement-history")
    @summary("Get customer entitlement history")
    getHistory(
      @path customerId: Shared.ULID,
      @path entitlementId: Shared.ULID,

      /**
       * The start of the queried range. Defaults to the last reset. Truncated to the
       * minute. The range may span at most 1000 windows.
       */
      @query
      from?: Shared.DateTime,

      /**
       * The end of the queried range. Defaults to the current time. Truncated to the
       * minute unless it is the current time.
       */
      @query
      to?: Shared.DateTime,

      /**
       * The size of the windows the usage is grouped into.
       */
      @query
      window_size: EntitlementHistoryWindowSize,

      /**
       * The IANA time zone the windows are aligned to.
       */
      @query
      time_zone?: string = "UTC",
    ):
      | Shared.GetResponse<EntitlementHistory>
      | Common.NotFound
      | Common.Conflict
      | Common.ErrorResponses;
  }
}
