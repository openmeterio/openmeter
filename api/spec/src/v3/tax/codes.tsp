import "../shared/resource.tsp";
import "../shared/index.tsp";
import "../apps/app.tsp";

namespace Tax;

/**
 * Mapping of app types to tax codes.
 */
@friendlyName("BillingTaxCodeAppMapping")
model TaxCodeAppMapping {
  /**
   * The app type that the tax code is associated with.
   */
  @summary("App type")
  app_type: Apps.AppType;

  /**
   * Tax code.
   */
  @summary("Tax code")
  tax_code: string;
}

/**
 * Tax codes by provider.
 */
@friendlyName("BillingTaxCode")
model TaxCode {
  ...Shared.ResourceWithKey;

  /**
   * Mapping of app types to tax codes.
   */
  @visibility(Lifecycle.Read, Lifecycle.Update, Lifecycle.Create)
  @summary("App type to tax code mappings")
  app_mappings: TaxCodeAppMapping[];
}
