import "../apps/index.tsp";
import "../billing/index.tsp";

namespace Customers;

/**
 * Billing customer data.
 */
@friendlyName("BillingCustomerData")
model CustomerBillingData {
  /**
   * The billing profile for the customer.
   *
   * If not provided, the default billing profile will be used.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Billing profile")
  billing_profile?: Common.BillingRoot.BillingProfileReference;

  /**
   * App customer data.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("App customer data")
  app_data?: Apps.AppCustomerData;
}

/**
 * Billing customer data upsert request.
 *
 * The request replaces the customer's billing data: omitting an optional field
 * removes its current value, and nullable fields treat an explicit `null` the same
 * as omission.
 */
@friendlyName("UpsertCustomerBillingDataRequest")
model CustomerBillingDataUpsertRequest {
  /**
   * The billing profile override to pin for the customer.
   *
   * Omit or set to `null` to remove the override so the default billing profile
   * applies.
   */
  @summary("Billing profile")
  billing_profile?: Common.BillingRoot.BillingProfileReference | null;

  /**
   * App customer data for the app of the effective billing profile.
   *
   * Omitted app data is deleted.
   */
  @summary("App customer data")
  app_data?: Shared.UpsertNullableRequest<Apps.AppCustomerData>;
}
