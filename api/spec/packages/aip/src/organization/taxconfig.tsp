import "../shared/index.tsp";
import "../tax/codes.tsp";

namespace Organization;

/**
 * Organization-level tax configuration.
 *
 * Stores default tax codes for the organization applied to specific billing
 * contexts. Provisioned automatically when the organization is created.
 */
@friendlyName("OrganizationTaxConfig")
model OrgTaxConfig {
  /**
   * Default tax code for credit purchases.
   *
   * Applied when creating credit-purchase charge lines if no line-level tax
   * code is set.
   */
  @visibility(Lifecycle.Read)
  @summary("Credit purchase tax code")
  credit_purchase_tax_code: Tax.TaxCode;

  /**
   * Default tax code for invoices.
   *
   * Applied as the default tax code for invoice lines if no line-level or
   * billing-profile-level tax code is set.
   */
  @visibility(Lifecycle.Read)
  @summary("Invoice tax code")
  invoice_tax_code: Tax.TaxCode;

  /**
   * Timestamp of creation.
   */
  @visibility(Lifecycle.Read)
  created_at: Shared.DateTime;

  /**
   * Timestamp of last update.
   */
  @visibility(Lifecycle.Read)
  updated_at: Shared.DateTime;
}

/**
 * Update request for organization tax configuration.
 */
@friendlyName("OrganizationTaxConfigUpdate")
model OrgTaxConfigUpdate {
  /**
   * Tax code ID for credit purchases.
   */
  @summary("Credit purchase tax code ID")
  credit_purchase_tax_code_id: Shared.ULID;

  /**
   * Tax code ID for invoices.
   */
  @summary("Invoice tax code ID")
  invoice_tax_code_id: Shared.ULID;
}
