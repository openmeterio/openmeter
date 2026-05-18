# Prorating vs progressive billing

Close, but no — they're different mechanisms that both deal with "partial periods" in some sense.

| | Prorating | Progressive billing |
|---|---|---|
| **What it does** | Scales a fee by a *time fraction* of the period | Splits a usage line into multiple invoice lines *within* the same period |
| **Typical target** | Flat fees / subscription fees | Usage-based prices |
| **Trigger** | Customer joins, upgrades, or cancels mid-period | Billing window inside a longer service period (e.g. weekly billing on a monthly plan) |
| **Math** | `fee × days_active / period_days` | Same rating math, but the rater is told "X units of this period were already billed in a prior split" |

**Prorating example.** $100/month subscription, customer joins on day 16 of a 30-day month → first invoice is `$100 × 15/30 = $50`. The fee is computed once, scaled by time.

**Progressive billing example.** Customer is on a monthly metered plan but you invoice them weekly. The month is one service period, but it's split into ~4 billing windows. At week 1, you bill the usage observed so far; at week 2, you bill the *new* usage on top, and the graduated tiers need to know "we already consumed 80 units in the prior window so tier 1 has 20 left." That tracker is `PreLinePeriodQuantity` in the L4 doc.

**Why volume tiered can't progressive-bill.** It's not about time scaling — it's about the splitting mechanism. Volume's price depends on the *final* quantity at period close to decide which tier the entire usage lands in. Mid-window, you don't know yet. If a customer is at 80 units at week 1, they might end the month at 90 (tier 1) or 200 (tier 2) — and tier 2 reprices *all* 200 units, including the 80 from week 1. So you'd have to either (a) guess and reconcile later (messy), or (b) refuse to split, which is what the code does — it errors on `SplitLineGroup` for volume-tiered lines.

Graduated has no such problem because each band's price is fixed the moment those units are consumed. Tier 1's price applies to units 0–100 regardless of where you end up; once unit 101 happens, it's priced at tier 2's rate forever, no retroactive change.

**Prorating doesn't have this issue** because it only scales flat fees, which don't have usage-dependent tiers.
