import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoices;

/**
 * UnitPriceItem represents a product or service that is sold to the customer based on a unit price.
 */
/* Omitted fields:
 * - alt_prices: 	AltPrices defines a list of prices with their currencies that may be used as an alternative to the item's base price.
 * - origin: Country code of where this item was from originally.
 * - ext: 	Extension code map for any additional regime specific codes that may be required.
 * - identities: List of additional codes, IDs, or SKUs which can be used to identify the item. They should be agreed upon between supplier and customer.
 */
@friendlyName("BillingUnitPriceItem")
model UnitPriceItem {
  @summary("Primary reference code that identifies this item.")
  ref?: string;

  @summary("Special key used to classify the item sometimes required by some regimes.")
  key?: Key;

  @summary("Human readable name of the item.")
  name: string;

  @summary("Description of the item.")
  description?: string;

  @summary("Currency used for the item's price. Currency is mandatory for all items, so that we can support multi-currency setups.")
  currency: CurrencyCode;

  @summary("Base price of a single unit to be sold.")
  price: Numeric;

  @summary("Unit of measure")
  unit?: Unit;

  // TODO: We need to have proper references to the subscription/plan on a per item basis.
}
