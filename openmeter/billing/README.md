# Billing

This package contains the implementation for the billing stack (invoicing, tax and payments).

The package has the following main entities:

## BillingProfile

Captures all the billing details, two main information is stored inside:
- The [billing workflow](./entity/customeroverride.go) (when to invoice, due periods etc)
- References to the apps responsible for tax, invoicing and payments (Sandbox or Stripe for now)

Only one default billing profile can exist per namespace.

## CustomerOverride

Contains customer specific overrides for billing pruposes. It can reference a billing profile other than the default (e.g. when different apps or lifecycle should be used) and allows to override the billing workflow.

## Invoice

Invoices are used to store the data required by tax, invoicing and payment app's master copy at OpenMeter side.

Upon creation all the data required to generate invoices are snapshotted into the invoice entity, so that no updates to entities like Customer, BillingProfile, CustomerOverride change an invoice retrospectively.

### Gathering invoices

There are two general kinds of invoices (Invoice.Status) `gathering` invoices are used to collect upcoming lines that are to be added to future invoices. `gathering` invocie's state never changes: when upcoming line items become due, they are just assigned to a new invoice, so that we clone the data required afresh.

Each customer can have one `gathering` issue per currency.
> For example, if the customer has upcoming charges in USD and HUF, then there will be one `gathering` invoice for HUF and one for USD.

If there are no upcoming items, the `gathering` invoices are (soft) deleted.

### Collection

TODO: document when implemented

### Invoices

The invoices are goverened by the [invoice state machine](./service/invoicestate.go).

Invoices are composed of [lines](./entity/invoiceline.go). Each invoice can only have lines from the same currency.

The lines can be of different types:
- ManualFee: one time manually added charge
- ManualUsageBased: manually added usage-based charge (can be used to charge addition usage-based prices without the product catalog features)

Each line has a `period` (`start`, `end`) and an `invoiceAt` property. The period specifies which period of time the line is referring to (in case of usage-based pricing, the underlying meter will be queried for this time-period). `invoiceAt` specifies the time when it is expected to create an invoice that contains this line. The invoice's collection settings can defer this.

Invoices are always created by collecting one or more line from the `gathering` invoices. The `/v1/api/billing/invoices/lines` endpoint can be used to create new future line items. A new invoice can be created any time. In such case the `gathering` items that are to be invoiced (`invoiceAt`) already are added to the invoice. Any usage-based line, that we can bill early is also added to the invoice for the period between the `period.start` of the line and the time of invoice creation.

### Line splitting

To achieve the behavior described above, we are using line splitting. By default we would have one line per billing period that would eventually be part of an invoice:

```
                period.start                                              period.end
Line1 [status=valid] |--------------------------------------------------------|
```

When the usage-based line can be billed mid-period, we `split` the line into two:

```
                     period.start              asOf                              period.end
Line1 [status=split]         |--------------------------------------------------------|
SplitLine1 [status=valid]    |------------------|
SplitLine2 [status=valid]                       |-------------------------------------|
```

As visible:
- Line1's status changes from `valid` to `split`: it will be ignored in any calculation, it becomes a grouping line between invoices
- SplitLine1 is created with a period between `period.start` and `asof` (time of invoicing): it will be addedd to the freshly created invoice
- SplitLine2 is created with a period between `asof` and `period.end`: it will be pushed to the gathering invoice

When creating a new invoice between `asof` and `period.end` the same logic continues, but without marking SplitLine2 `split`, instead the new line is added to the original line's parent line:

```
                     period.start              asOf1          asof2                period.end
Line1 [status=split]         |--------------------------------------------------------|
SplitLine1 [status=valid]    |------------------|
SplitLine2 [status=valid]                       |---------------|
SplitLine3 [status=valid]                                       |---------------------|
```

This flattening approach allows us to not to have to recusively traverse lines in the database.

### Usage-based quantity

When a line is created for an invoice, the quantity of the underlying merter is captured into the line's qty field. This information is never updated, so late events will have to create new invoice lines when needed.

### Detailed Lines

Each (`valid`) line can have one or more detailed lines (children). These lines represent the actual sub-charges that are caused by the parent line.

Example:
> If a line has:
> - Usage of 200 units
> - Tiered pricing:
> - Tier1: 1 - 50 units cost flat $300
> - Tier2: 51 - 100 units cost flat $400
> - Tier3: 100 - 150 units cost flat $400 + $1/unit
> - Tier4: more than 150 units cost $15/unit

This would yield the following lines:

- Line with quantity=200
  - Line quantity=1 per_unit_amount=300 total=300 (Tier1)
  - Line quantity=1 per_unit_amount=400 total=400 (Tier2)
  - Line quantity=1 per_unit_amount=400 total=400 (Tier3, flat component)
  - Line quantity=50 per_unit_amount=1 total=50 (Tier3, per unit price)
  - Line quantity=50 per_unit_amount=15 total=759 (Tier4)

Apps can choose to syncronize the original line (if the upstream system understands our pricing model) or can use the sublines to syncronize individual lines without having to understand billing details.

### Detailed Lines vs Splitting

TODO: this is TBD as not implemented.

When we are dealing with a split line, the calculation is by taking the meter's quantity for the whole line period ([`parent.period.start`, `splitline.period.end`]) and the splitline's period (`splitline.period.start`, `splitline.period.end`).

When substracting the two we get two values:
- line qty: splitline's period
- before line usage: whole line period - splitline's period

We execute the pricing logic (e.g. tiered pricing) for the line qty, while considering the before usage, as it reflects the already billed for items.

Corner cases:
- Graduating tiered prices cannot be billed mid billing period (always arrears)
- Min, Avg meters are always billed arrears as they are not composable


