# Known issues

## Legacy billing: missing minimum spend after a subscription period change

Legacy invoice-backed billing can omit the minimum-spend adjustment when a
subscription cancellation or another subscription change shortens a service
period that has already been partially invoiced through progressive billing.
The last invoice can retain its usage amount without the adjustment needed to
meet the period minimum, resulting in underbilling.

This is an existing limitation of legacy billing. The recommended path for
workloads requiring these combinations is to enable credits and use
charge-backed billing, as described below.

### Example: a cancellation processed after partial invoices were created

All amounts in this example are synthetic. Consider a subscription with a
January 1–February 1 service period, a price of 1 per unit, and a minimum spend
of 100. Usage is 40 units on January 5 and 20 units on January 18.

Progressive billing creates these draft invoices:

| Invoice | Service period | Amount |
| --- | --- | ---: |
| First | January 1–11 | 40 |
| Second | January 11–25 | 20 |

The remaining January 25–February 1 period is still pending on a gathering
invoice. Neither draft includes a minimum adjustment because neither originally
ended the full service period.

A cancellation effective January 20 is processed after those drafts have been
created. Subscription sync shortens the second invoice's service period to
January 11–20 and deletes the pending remainder. The first invoice stays at 40.
The second should now include its usage of 20 plus a minimum adjustment of 40.

| Result after cancellation | Expected | Actual legacy behavior |
| --- | ---: | ---: |
| First invoice | 40 | 40 |
| Second invoice's usage | 20 | 20 |
| Second invoice's minimum adjustment | 40 | 0 |
| Combined invoice total | **100** | **60** |

The second invoice is calculated while the split-line group still has its old
February 1 end date. It is therefore treated as an intermediate slice rather
than the final slice. The group end is updated afterward, without recalculating
the invoice. Changing the order of invoice updates does not resolve this case.

### Example: repeating subscription sync does not repair the amount

Running subscription sync again with the same January 20 cancellation leaves
the combined invoice total at 60. The invoice periods and group period already
match the subscription, so the unchanged sync does not recalculate the missing
minimum adjustment. Retrying sync is not a workaround for this issue.

### Recommended approach: enable credits and use charges

For new billing work that combines progressive invoicing, minimum commitments,
and subscription period changes, enable credits to use the charges billing
backend. For subscriptions using `credit_then_invoice` settlement, also enable
charge-backed invoice settlement:

```yaml
credits:
  enabled: true
  enableCreditThenInvoice: true
```

If deployment-specific feature gates control credits, enable credits for the
affected namespace as well. See the [example configuration](config.example.yaml)
and [charges documentation](openmeter/billing/charges/README.md) for setup and
supported behavior.

Existing legacy invoice lines and split-line groups retain their billing
backend. Enabling credits does not automatically migrate them or repair their
amounts. Review affected legacy invoices before finalization and plan their
transition separately. See
[subscription sync backend ownership](openmeter/billing/worker/subscriptionsync/README.md#backend-ownership).
