# Tiered pricing — graduated vs volume

Totally fair — "tiered" without a modifier is genuinely ambiguous and the L4 rating-engine note even calls it out as a frequent source of confusion. The cleanest way to internalize it is one example, run through both modes:

**Setup:** tiers are `[0–100 @ $10/unit, 101+ @ $15/unit]`. Customer at **150 units**.

| Mode | Mental model | Math | Bill |
|---|---|---|---|
| **Graduated** | Tax brackets. Each tier's price applies *only to units in that band*. | (100 × $10) + (50 × $15) = $1,000 + $750 | **$1,750** |
| **Volume** | "Final quantity picks the rate." The total falls in one tier; that price applies to *all* units. | 150 × $15 (because 150 is in tier 2) | **$2,250** |

Same input, $500 apart. Different rule for which units pay what.

**Why both exist (the intent behind each):**

- **Graduated** = customer benefits *as they grow*. Used for "the more you use, the cheaper the marginal unit." Common for usage-discount plans where the customer's effective rate naturally improves with scale.
- **Volume** = customer benefits *all-or-nothing once they cross*. Used for "buy more than X and we'll re-rate the whole thing at the better price." Common for commitment-style pricing where crossing a threshold reprices everything retroactively.

**One operational gotcha** (from the L4 doc) that makes the distinction real beyond pricing semantics:

- **Volume can't be progressive-billed.** Its price depends on the *final* quantity at period close. Mid-period invoicing doesn't know which tier you'll land in, so the code explicitly errors on a `SplitLineGroup` for a volume-tiered line.
- **Graduated can be progressive-billed** (with `PreLinePeriodQuantity` tracking) because each band's price is settled the moment that band's units are consumed.

So if you ever see "tiered" in a conversation or code, the first question to ask is *which mode* — the math, the split behavior, and the customer-facing intent are all different.
