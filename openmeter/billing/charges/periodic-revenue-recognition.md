# Periodic Revenue Recognition: usage-based runs

Design notes. The tables, the domain type, and an adapter that can create a run
and read one by ID exist. Nothing creates runs yet. The
[charges README](README.md) remains the source for current behavior.

## Goal

Persist charge spend during the service period instead of only at invoicing or
period end, so that the history of postings and their corrections survives and
can be exported later.

- Once a day, each active usage-based charge records what changed since its
  previous revenue recognition run.
- Late or corrected usage is never written back into an old run. The next run
  carries the difference.
- A run does not allocate anything. It only collects and stores point-in-time
  data.
- Daily runs mean about 30 runs per charge per month.

Treated as settled or left open elsewhere:

| Topic | Status |
| --- | --- |
| Scheduling | TBD |
| Organization-wide ledger export | Separate work; reads the ledger only |
| Flat-fee charges | Out of scope for now |
| Ledger postings for a run | TBD |

## Model

A revenue recognition run is its own record. It is not a realization run and
does not share the realization run table, chain, or lifecycle. Realization runs
keep serving invoicing and final realization exactly as they do today.

### Storage

Table `charge_usage_based_revenue_recognitions`, named after
`charge_usage_based_runs`.

| Column | Meaning |
| --- | --- |
| `id`, `namespace`, `created_at`, `updated_at`, `deleted_at` | Standard |
| `charge_id` | Owning usage-based charge |
| `feature_id` | Feature the run was metered on |
| `stored_at_lt` | Only usage stored before this instant is included |
| `service_period_to` | End of the period the run covers; the day boundary |
| `period` | Cadence the run belongs to, as an ISO 8601 duration. Defaults to daily (`P1D`) |
| `prior_run_id` | The charge's previous revenue recognition run, the baseline for the deltas. Empty for the first run |
| `metered_quantity` | Cumulative quantity from the charge start to `service_period_to` |
| `delta_metered_quantity` | Change in quantity since the previous run |
| Totals columns (`amount`, `credits_total`, `total`, ...) | Change in amounts since the previous run. May be renamed to amounts |
| `detailed_lines_include_credit_allocations` | Same meaning as on realization runs |
| `invoice_usage_total` | Part of the run that credits do not cover, as one amount. Empty when there is none |

There is no run type, invoice line, invoice, or payment. The prior-run pointer
has the same shape as the one on realization runs, but it only ever points at
another revenue recognition run.

A run owns three kinds of child rows, each pointing at it through `run_id`:

| Table | Holds | Columns |
| --- | --- | --- |
| `charge_usage_based_revenue_recognition_credits` | Credits allocated in the charge currency | `id`, `annotations`, service period, `amount`, `type`, optional `ledger_transaction_group_id` |
| `charge_usage_based_revenue_recognition_overage_credits` | Credits allocated against the fiat overage | Same as above |
| `charge_usage_based_revenue_recognition_detailed_line` | Detailed lines | `id`, `name`, `description`, `annotations`, `metadata`, `currency`, service period, `child_unique_reference_id`, `index`, `quantity`, `per_unit_amount`, `amount`, `amount_discounts`, `credits_applied` |

These tables are narrower than the realization run ones on purpose:

- The credit tables have no invoice line, sort hint, or pointer to the
  realization a correction corrects.
- A detailed line has one `amount` (before discounts and credits) instead of
  the totals columns, and no category, payment term, pricer reference, or
  invoicing app reference.

### Domain

- `usagebased.RevenueRecognitionRun` is the run. `Period` is a
  `datetime.ISODuration` and `PriorRunID` points at the previous run.
  `CreditsAllocated` and
  `FiatOverageCreditRealizations` use the same `creditrealization.Realizations`
  type as realization runs, with the fields that are not stored left empty.
- `usagebased.RevenueRecognitionDetailedLine` is a run's detailed line. It is
  its own type, not the realization run detailed line.
- `Charge.RevenueRecognitionRuns` holds a charge's runs, a slice like
  `Realizations`. Reading a charge does not fill it yet.
- Adapter: `CreateRevenueRecognitionRun` writes a run and its child rows
  together, and `GetRevenueRecognitionRunByID` reads them back. A run is never
  updated.

## What the separate table avoids

An earlier draft stored these as realization runs with a new type. These
problems came only from sharing that table and no longer apply:

- A new run type that invoicing code must learn to skip.
- The single current-run slot, which a daily run would have had to bypass.
- Helpers that treat every non-voided run as billing history (latest run, sums,
  and voiding every run on shrink, extend, and override changes).
- A daily run and an invoice run that end at the same instant both booking the
  same usage, because rating only subtracts strictly earlier runs.
- Invoice lines having to cover several runs.

## Still required

- **Downward corrections.** A run's totals follow the `totals.DeltaTotals`
  rules: `Amount` and `DiscountsTotal` may be negative, `Total` and
  `CreditsTotal` may not. A correction that would take the net amount or the
  credits below zero cannot be stored yet.
- **Credit corrections.** A correction row can be stored, but it cannot name the
  allocation it corrects, and that allocation would live in an earlier run.
- **Fiat amount for custom-currency charges.** The totals are in the charge
  currency. `invoice_usage_total` carries no currency of its own, and
  converting per run rounds per run, so daily conversions can drift from a
  single invoice conversion.
- **Ledger overlap.** If runs post to the ledger, invoice issuance must not
  book the same amount as accrued again.
- **Uniqueness.** A charge has one run per `service_period_to`. That only
  stops holding if runs of different periods (hourly, daily, monthly) exist
  side by side. No unique index enforces it yet.
- **Choosing the prior run.** The adapter stores the pointer it is given. It
  does not pick the prior run or check that it belongs to the same charge.

## Open questions

- Is the ledger reference on a credit row required once runs post to the
  ledger? It is optional today.
- Should the credits get a domain type of their own? `CreditsAllocated` and
  `FiatOverageCreditRealizations` reuse `creditrealization.Realizations`, so
  each one carries `LineID`, `CorrectsRealizationID`, and `SortHint`, which are
  never stored and always empty. A type specific to revenue recognition would
  hold only the stored fields, as the detailed line type already does.
- What happens to a charge's runs when the charge is shrunk, extended,
  overridden, or deleted?
  - Note (borosr): the change should be noted and booked at the next revenue
    recognition run, so runs should be handled as an immutable, append-only
    slice.
- Do percentage discounts and maximum spend work across daily runs?
  - Note (borosr): I guess it should be booked at the period end.
- Scheduling: which day boundary applies, UTC midnight or the billing anchor
  time?
- Does a day with no change write a zero run or no run?
  - Note (borosr): this should be configurable in the billing profile.
- Does the first increment need `credit_then_invoice`, or is `credit_only`
  enough?
