import "../apps/index.tsp";
import "../billing/index.tsp";

namespace Customers;

/**
 * Billing customer data.
 */
@friendlyName("BillingCustomerData")
model CustomerBillingData {
  ...CustomerBillingProfile;

  /**
   * App customer data.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("App customer data")
  app_data?: Apps.AppCustomerData;
}

/**
 * Customer billing profile selection.
 */
@friendlyName("BillingCustomerProfile")
model CustomerBillingProfile {
  /**
   * The billing profile explicitly selected for the customer.
   *
   * Unpinned customers use the current namespace default. Selecting the current
   * default explicitly pins the customer to that profile.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Billing profile")
  billing_profile?: Common.BillingRoot.BillingProfileReference | null;

  /**
   * The effective billing profile for the customer. Uses the explicitly selected
   * billing profile when pinned, otherwise the current default billing profile.
   */
  @visibility(Lifecycle.Read)
  @summary("Effective billing profile")
  effective_billing_profile: Common.BillingRoot.BillingProfileReference;
}
