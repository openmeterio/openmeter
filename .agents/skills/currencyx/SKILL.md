---
name: currencyx
description: Work on OpenMeter currency primitives in pkg/currencyx for fiat and custom currency codes, shared currency interfaces, rounding modes, calculators, allocation precision, fiat/custom boundaries, and callers in billing, charges, ledger, product catalog, subscriptions, API, or currency registry code.
---

# Currencyx

Use this skill when changes touch `pkg/currencyx` or any caller that depends on currency code shape, fiat/custom classification, calculator behavior, rounding, allocation, or invoice/ledger currency boundaries.

Also load the domain skill for each touched caller area: `billing`, `charges`, `ledger`, `subscription`, `api`, `ent`, `db-migration`, and `test`.

## Source Of Truth

- Source code: `pkg/currencyx/*.go`.
- Primary tests: `pkg/currencyx/*_test.go`.
- Package usage examples: `pkg/currencyx/README.md`.
- This skill is a how-to/reference for agents. Update it whenever `Currency`, `Code`, `CustomCurrency`, `Calculator`, rounding, allocation, or validation behavior changes.
- One canonical repo skill lives at `.agents/skills/currencyx`; do not create duplicate currencyx guidance elsewhere.

## Package Layout

- `currency.go`: currency type constants, the shared `Currency` interface, `Code`, `CustomCurrency`, and fiat/custom constructors.
- `validation.go`: code format validation, fiat collision checks, precision validation, `PostgresCodeSchemaType`, and `CustomCurrency.Validate`.
- `fiat.go`: rounding modes, `Calculator`, and precision helpers.
- `allocation.go`: deterministic largest-remainder allocation using calculator precision.
- `README.md`: short examples for fiat, custom, and allocation usage.

## Boundary Model

- **Currency code**: durable identifier. Fiat and custom codes use `currencyx.Code`, but validation differs by boundary.
- **Currency interface**: shared behavior contract. Callers that know a configured currency should expose `CurrencyCode()`, `CurrencyType()`, `CurrencyPrecision()`, and `CurrencyRoundingMode()`.
- **Fiat currency**: `currencyx.Code` implements `currencyx.Currency` as fiat. `Code.Calculator()` preserves existing fiat behavior and derives precision from GOBL/ISO definitions.
- **Custom currency**: `currencyx.CustomCurrency` implements `currencyx.Currency`. Custom currencies carry configured precision and rounding mode; missing rounding mode defaults to bankers rounding.
- **Calculator**: construct with `currencyx.NewCalculator(currencyx.Currency)`. The calculator must branch from `CurrencyType()` and use `CurrencyPrecision()` / `CurrencyRoundingMode()` from the interface for custom currencies.
- **Allocation**: use calculator precision for units and largest-remainder distribution. Do not reach into fiat-only `Def.Subunits` for allocation logic.
- **Validation**: `Code.Validate()` remains fiat semantic for existing callers. Use `ValidateFormat()` for structural code checks and `ValidateCustom()` for custom currency codes. Custom codes must not contain the `|` route delimiter.
- **Registry boundary**: owns custom currency definition, fiat-code collision checks, archive/activation rules, cost-basis history, and future persisted rounding configuration.
- **Cost basis history**: entries are effective-dated with `effective_from` and optional `effective_to`; API responses expose the cost-basis `id`. Use `currency_id` terminology in domain/schema models, even when the current route is under custom currencies. If renaming old `custom_currency_id` storage, use data-preserving migrations.
- **Finance boundary**: snapshots fiat basis and applies fiat rounding when custom units become fiat amounts.
- **Ledger boundary**: records durable currency codes and balanced single-currency legs. Round before posting only when the upstream domain owns normalization.
- **Invoice boundary**: invoice currency stays fiat. Custom units must be materialized to fiat before invoice artifacts.

## Rounding Rules

- Preserve fiat rounding unless the task explicitly changes fiat money behavior.
- Custom currency default rounding is `RoundingModeBankers` (`RoundBank`, half-even).
- Custom currencies can opt into `RoundingModeHalfAwayFromZero` through `NewCustomCurrencyWithRounding`.
- `Calculator.RoundToPrecision` is the single place that applies the effective rounding mode.
- `Calculator.RoundDown` and `Calculator.Unit` are precision helpers; they should not apply banker/half-away rounding.
- `Calculator.IsRoundedToPrecision` must use `RoundToPrecision`, so it follows the configured rounding mode.

## Process

1. Name the surface before editing: code validation, type/interface, rounding, calculator, allocation, registry, ledger fact, fiat materialization, or invoice boundary.
2. Keep `pkg/currencyx` free of imports from `openmeter/...`; callers can implement `currencyx.Currency` to supply registry-backed custom settings.
3. Prefer `CurrencyType()` at the boundary that truly requires fiat or custom. Do not add broad split helpers unless the caller boundary needs a named domain rule.
4. Preserve `currencyx.Code(...).Calculator()` for existing fiat callers.
5. For custom currencies, validate structural code, route delimiter exclusion, fiat-code collisions, precision, and rounding mode.
6. Keep allocation deterministic: precision defines units, largest remainder distributes residual units, and tie-breakers remain stable.
7. After editing, run focused `pkg/currencyx` tests, `go vet`, and caller tests or compile checks for every touched boundary.

## Test Checklist

Cover the named risk introduced by the change:

- Fiat regression behavior: code validation, precision from ISO definition, and existing rounding.
- Validation boundaries: `Code.Validate()` stays fiat-only while `ValidateFormat()` accepts structurally valid custom codes.
- Custom interface behavior: code, type, precision, and rounding mode all flow through `currencyx.Currency`.
- Banker ties: positive, negative, and zero-precision custom rounding tie to even.
- Alternate custom rounding: half-away-from-zero remains selectable and tested.
- Invalid config: bad precision or rounding mode fails validation.
- Allocation precision: custom precision affects units and largest-remainder allocation.
- Boundary tests: billing/invoice rejects custom invoice currency explicitly; ledger accepts structurally valid custom codes only when that domain supports them.

Focused commands:

```bash
env GOCACHE=/private/tmp/openmeter-go-build go test ./pkg/currencyx
env GOCACHE=/private/tmp/openmeter-go-build go vet ./pkg/currencyx
```

For caller compile checks, keep the package list scoped to touched boundaries and include `-tags=dynamic` when billing/ledger paths require it.

## Review Checks

- Fiat and custom currencies share the `currencyx.Currency` interface.
- `Calculator.RoundToPrecision` applies the effective rounding rule.
- `Calculator` does not require fiat definitions for custom currencies.
- Allocation code uses calculator methods, not fiat-only definition fields.
- Invalid rounding precision or mode fails validation.
- Tests cover banker ties, configured custom precision, fiat regression behavior, invalid rounding config, and allocation precision.
