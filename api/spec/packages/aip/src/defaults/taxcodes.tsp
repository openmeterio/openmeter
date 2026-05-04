import "../shared/index.tsp";
import "../tax/codes.tsp";

namespace Defaults;

/**
 * Organization-level default tax code references.
 *
 * Stores the default tax codes applied to specific billing contexts for this
 * organization. Provisioned automatically when the organization is created.
 */
@friendlyName("OrganizationDefaultTaxCodes")
model OrgDefaultTaxCodes {
  /**
   * Default tax code for invoicing.
   */
  @summary("Invoicing tax code")
  invoicing_tax_code: Shared.ResourceReference<Tax.TaxCode>;

  /**
   * Default tax code for credit grants.
   */
  @summary("Credit grant tax code")
  credit_grant_tax_code: Shared.ResourceReference<Tax.TaxCode>;

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
