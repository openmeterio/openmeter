import "../apps/index.tsp";
import "../billing/index.tsp";

namespace Customers;

/**
 * Billing customer data.
 */
#suppress "@openmeter/api-spec/repeated-prefix-grouping" "Read and write properties"
@friendlyName("BillingCustomerData")
model CustomerBillingData {
  // /**
  //  * Customer specific billing workflow overrides.
  //  */
  // @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  // @summary("Billing workflow override")
  // override?: Billing.BillingWorkflow;

  /**
   * The billing profile for the customer.
   *
   * If not provided, the default billing profile will be used.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("Billing profile")
  billing_profile?: Billing.BillingProfileReference;

  /**
   * App customer data.
   */
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  @summary("App customer data")
  app_data?: Apps.AppCustomerData;
}
