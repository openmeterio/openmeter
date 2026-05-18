# UnitConfig — Pricing pipeline & scenarios

> 10,000m view. How does a metered number become money? What pricing shapes exist, and where does UnitConfig fit?

## The pipeline (when each step runs)

Every billable event travels this path:

```
[Customer does a thing]
       ↓ event ("user X called API at T, used 1,204 tokens")
[[Meter]] aggregates events → raw number
       ↓
[[Entitlement]] checks allowance (precise, unrounded)
       ↓
UnitConfig (optional) → converts raw → billable quantity
       ↓
Price turns billable quantity → money
       ↓
[[Invoice]] line
```

Two handoffs do the real work:

- The **meter** picks what the raw number means (calls, bytes, tokens, dollars-of-cost…).
- **UnitConfig**, if present, sits between the meter and the price — same number, different unit. The price then acts on whatever it receives: raw if no UnitConfig, converted if one is set.

## Price shapes

| Shape | What it means | Math |
|---|---|---|
| **Flat** | A fixed fee, not usage-based | `amount` |
| **Unit** | $X per unit consumed | `quantity × amount` |
| **Tiered (graduated)** | Different price per band, like tax brackets | sum of (units in tier × tier price) |
| **Tiered (volume)** | Total picks one tier; that price applies to *all* units | `quantity × tier_price` |
| **Dynamic** *(v1 only)* | Multiplier on cost passed in via the event | `quantity × multiplier` |
| **Package** *(v1 only)* | $X per bundle of N units, round up | `⌈quantity ÷ N⌉ × amount` |
| **UnitConfig** | Not a price — a *conversion layer* before the price | `raw × factor` (multiply) or `raw ÷ factor` (divide), optionally rounded |

UnitConfig has three knobs: an **operation** (multiply or divide), a **conversion_factor** (number), and optional **rounding** + **display_unit**.

---

## Scenario A — Plain per-unit ("$0.01 per API call")

Customer makes **1,247 calls**.

| | Configuration | Math | Bill |
|---|---|---|---|
| v1 | `UnitPrice{$0.01}` | 1,247 × $0.01 | **$12.47** |
| v3 | `UnitPrice{$0.01}` (no UnitConfig) | 1,247 × $0.01 | **$12.47** |

Meter output is already the billable unit. Nothing to convert.

## Scenario B — Per-package ("$10 per 1,000 calls, round up")

Customer makes **1,247 calls**.

| | Configuration | Math | Bill |
|---|---|---|---|
| v1 | `PackagePrice{$10, per: 1000}` | ⌈1,247 ÷ 1,000⌉ × $10 = 2 × $10 | **$20** |
| v3 | `UnitPrice{$10}` + `UnitConfig{divide, 1000, ceiling}` | ⌈1,247 ÷ 1,000⌉ × $10 = 2 × $10 | **$20** |

Same answer. v3 makes the divide-and-round step **explicit** instead of baked into the price type.

## Scenario C — Cost-plus markup ("Resell tokens at 1.5×")

Meter sums a `provider_cost` field per event. Customer accumulates **$4.20** of cost.

| | Configuration | Math | Bill |
|---|---|---|---|
| v1 | `DynamicPrice{multiplier: 1.5}` | $4.20 × 1.5 | **$6.30** |
| v3 | `UnitPrice{$1}` + `UnitConfig{multiply, 1.5}` | $4.20 × 1.5 × $1 | **$6.30** |

Same answer. v3 separates "margin" (UnitConfig) from "per-unit price" (UnitPrice) — two knobs you can turn independently.

---

## What v3 unlocks (v1 cannot do this in one rate card)

### Scenario D — Tiered by GB

Meter sums bytes. Plan: **first 100 GB free, next 1 TB at $0.05/GB, above 1.1 TB at $0.03/GB.** Customer at **1.5 TB** raw.

| | Boundaries authored in… | Failure mode |
|---|---|---|
| v1 | bytes — `up_to_amount: 100_000_000_000`, `1_100_000_000_000` | If the meter ever changes unit (bytes → KB), the whole plan must be rewritten. |
| v3 | GB — `up_to_amount: 100, 1100` after `UnitConfig{divide, 1e9, ceiling, "GB"}` | Boundaries match the human unit; meter changes don't touch the plan. |

Math is the same: `0 + 1,000 × $0.05 + 400 × $0.03 = $62`. The unlock is **readability + stability**, and the invoice can display "GB" alongside the bill.

### Scenario E — Markup composed with tiers

"Resell tokens at 1.5× **and** give a volume discount: $0.001/token for the first 1M, $0.0005 after." Customer at **5M provider tokens**.

- **v1: not expressible.** `DynamicPrice` is flat (no tiers). `TieredPrice` has no multiplier.
- **v3:** `UnitConfig{multiply, 1.5}` (margin first) → `TieredPrice{graduated, [1M @ $0.001, rest @ $0.0005]}` on the converted 7.5M.
- Math: 1M × $0.001 + 6.5M × $0.0005 = **$4,250**.

Margin and tier shape become **separable decisions** on the same rate card.

---

## One quirk worth flagging

**Entitlement and invoicing use different precisions when rounding is set.** Invoice sees the rounded value (5 packages, 1,500 GB); the [[Entitlement]] check sees the unrounded one (1.247 packages, 1,499.8 GB). A customer can hit their cap on a fractional unit while the invoice still bills in whole units. By design — gating wants precision, invoices want clean lines.
