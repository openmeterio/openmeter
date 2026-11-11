# Reusing usage-based rating totals

A caller that needs totals can attach a bounded `TotalsReuse` to its context with
`WithTotalsReuse`. The RTE credit calculator shares one such cache across its calls. Ordinary
callers continue through the same rating path with their own context.

`GetTotalsForUsage` queries the cumulative meter quantity for the effective service period and
storage cutoff on every call. Once it has the quantity, it can skip pure rating work for a charge
if the last successful result was calculated from identical inputs. A ledger posting, for example,
can require a new credit balance while the open charge’s quantity and price are unchanged. Late
usage, a service-period override, or changed pricing can invalidate that comparison.

Each entry belongs to one rating service instance and namespaced charge. The comparison includes
the exact decimal quantity, effective price and period, name, discounts, unit configuration,
currency type and details (including precision), and minimum-commitment policy. When an input to
generating totals changes, update the reuse comparison and its
[regression cases](totalsreuse_test.go) together.

The cache holds only successful results and retains one input/result pair per entry. An unknown
currency implementation, input encoding failure or encoded inputs larger than 64 KiB use ordinary
rating. A miss, eviction or restart costs another rating calculation. A reported hit covers the
rating step alone, even though usage and charge inputs were read afresh.
