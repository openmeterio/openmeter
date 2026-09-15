import "../shared/resource.tsp";
import "../shared/index.tsp";
import "../apps/app.tsp";
import "../billing/tax.tsp";

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

/**
 * Tax configuration for a billable resource.
 *
 * Applies a tax code and tax behavior to the resulting invoice line items. When
 * not set, the applicable default is used: the billing profile default tax
 * configuration, then the organization default tax code.
 */
@friendlyName("TaxCodeConfig")
@summary("Tax code configuration")
model TaxCodeConfig {
  /**
   * Tax behavior.
   *
   * This enum is used to specify whether tax is included in the price or excluded
   * from the price. If not specified, the billing profile is used to determine the
   * tax behavior. If not specified in the billing profile, the provider's default
   * behavior is used.
   */
  @summary("Tax behavior")
  behavior?: Billing.TaxBehavior;

  /**
   * Tax code applied to the invoice line item.
   */
  @summary("Tax code")
  code?: Shared.ResourceReference<TaxCode>;
}
