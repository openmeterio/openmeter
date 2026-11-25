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
| `prior_rev_recog_run_id` | The charge's previous revenue recognition run, the baseline for the deltas. Empty for the first run |
| `metered_quantity` | Cumulative quantity from the charge start to `service_period_to` |
| Totals columns (`amount`, `credits_total`, `total`, ...) | Change in amounts since the previous run. May be renamed to amounts |
| `detailed_lines_include_credit_allocations` | Same meaning as on realization runs. It has no default, so every run sets it |

There is no run type, invoice line, invoice, or payment.
`prior_rev_recog_run_id` only ever points at another revenue recognition run.

A run owns three kinds of child rows, each pointing at it through `run_id`:

| Table | Holds | Columns |
| --- | --- | --- |
| `charge_usage_based_revenue_recognition_credits` | Credits allocated in the charge currency | `creditrealization.Mixin`: `id`, `annotations`, `line_id`, `amount`, service period, `ledger_transaction_group_id`, `sort_hint`, `type` |
| `charge_usage_based_revenue_recognition_invoiced_usages` | The part of the run that credits do not cover, as many rows | Same as above |
| `charge_usage_based_revenue_recognition_detailed_line` | Detailed lines | `id`, `name`, `description`, `annotations`, `metadata`, `currency`, service period, `child_unique_reference_id`, `index`, `quantity`, `per_unit_amount`, `amount`, `amount_discounts`, `credits_applied` |

How these compare with the realization run tables:

- The credit and invoiced usage tables use the same mixin as the realization
  run credit tables. They only lack `corrects_realization_id`, which lives in
  `creditrealization.CorrectionMixin` and is added by the tables that need it.
- Invoiced usage is stored as rows, not as one amount on the run, so part of it
  can be reverted by a later row. The run's invoiced total is the sum of the
  row amounts.
- A detailed line has one `amount` (before discounts and credits) instead of
  the totals columns, and no category, payment term, pricer reference, or
  invoicing app reference.
- Credits allocated against the fiat overage are not stored. That only happens
  once an invoice is finalized, and is left for when it is implemented.

### Domain

- `usagebased.RevenueRecognitionRun` is the run. `PriorRevRecogRunID` points at
  the previous run. `CreditsAllocated` and `InvoicedUsages` use the same
  `creditrealization.Realizations` type as realization runs, with
  `CorrectsRealizationID` always empty.
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
- **Corrections.** A credit or invoiced usage correction row can be stored, but
  it cannot name the row it corrects, and that row would live in an earlier
  run.
- **Fiat amount for custom-currency charges.** The totals are in the charge
  currency. Invoiced usage rows carry no currency of their own, and converting
  per run rounds per run, so daily conversions can drift from a single invoice
  conversion.
- **Ledger overlap.** If runs post to the ledger, invoice issuance must not
  book the same amount as accrued again.
- **Uniqueness.** A charge has one run per `service_period_to`. That only
  stops holding if runs of different periods (hourly, daily, monthly) exist
  side by side. No unique index enforces it yet.
- **Choosing the prior run.** The adapter stores the pointer it is given. It
  does not pick the prior run or check that it belongs to the same charge.

## Open questions

- Credit and invoiced usage rows require a ledger transaction, because the
  shared mixin does. A run cannot store these rows until it posts to the
  ledger. Is that the intended order of work?
- Should the credits and invoiced usages get a domain type of their own? They
  reuse `creditrealization.Realizations`, so each one carries
  `CorrectsRealizationID`, which is never stored and always empty.
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
