import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace GOBL.Bill;

@summary("An invoice (provided by gobl)")
// TODO: we need to convert the json schema to typespec from here: https://github.com/invopop/gobl/tree/main/data/schemas
@extension("x-go-type", "bill.Invoice")
@extension(
  "x-go-type-import",
  {
    path: "github.com/invopop/gobl/bill",
  }
)
model Invoice {
  ...Record<unknown>;
}

@extension("x-go-type", "bill.Line")
@extension(
  "x-go-type-import",
  {
    path: "github.com/invopop/gobl/bill",
  }
)
model InvoiceItem {
  @summary("Line number inside the parent (calculated)")
  @visibility("read")
  @encodedName("application/json", "i")
  index: int32;

  quantity: Numeric;

  @visibility("read")
  sum: Numeric;

  discounts: LineDiscount[];
  charges: LineCharge[];
  taxes: GOBL.Tax.Combo[];
  total: Numeric;
  notes: GOBL.CBC.Note[];
}

model GenericCharge {
  /**
   * Percentage if fixed amount not applied
   */
  @summary("Percent")
  percent?: Percentage;

  /**
   * Fixed discount amount to apply (calculated if percent present).
   */
  @summary("Amount")
  amount: Numeric;

  @summary("Reason code.")
  code?: string;

  @summary("Text description as to why the discount was applied.")
  reason?: string;
}

@extension("x-go-type", "bill.LineDiscount")
@extension(
  "x-go-type-import",
  {
    path: "github.com/invopop/gobl/bill",
  }
)
@summary("LineDiscount represents an amount deducted from the line, and will be applied before taxes.")
model LineDiscount {
  ...GenericCharge;
}

@extension("x-go-type", "bill.LineCharge")
@extension(
  "x-go-type-import",
  {
    path: "github.com/invopop/gobl/bill",
  }
)
@summary("LineCharge represents an amount added to the line, and will be applied before taxes.")
model LineCharge {
  ...GenericCharge;
}
