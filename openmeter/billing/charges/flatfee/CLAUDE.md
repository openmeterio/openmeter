# flatfee

<!-- archie:ai-start -->

> Flat-fee charge type package: defines the domain model (Charge/ChargeBase, Intent, State, RealizationRun) and the segmented Adapter/Handler/Service interface contracts for fixed-amount charges, including proration math and the dotted per-settlement-mode Status hierarchy. The package root holds value types and interfaces only; service/ implements lifecycle state machines and adapter/ implements Ent persistence.

## Patterns

**Every input/domain type has a Validate() that collects errs** — All structs implement Validate() accumulating into var errs []error and returning models.NewNillableGenericValidationError(errors.Join(errs...)), wrapping nested errors with field context via fmt.Errorf("field: %w", err). (`func (i CreateChargesInput) Validate() error { var errs []error; ...; return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**Segmented adapter interfaces composed into one Adapter** — Adapter embeds ChargeAdapter, ChargeDetailedLineAdapter, ChargeCreditAllocationAdapter, ChargeRunAdapter, ChargeInvoicedUsageAdapter, ChargePaymentAdapter plus entutils.TxCreator. New persistence methods go on the narrow sub-interface, not a monolith. (`type Adapter interface { ChargeAdapter; ChargeDetailedLineAdapter; ...; entutils.TxCreator }`)
**Status strings hierarchically encode meta.ChargeStatus** — Status constants are dotted paths (e.g. "active.realization.started"); ToMetaChargeStatus() splits on the first '.' and validates the prefix against meta.ChargeStatus. Never persist a flat-fee Status without mapping through ToMetaChargeStatus(). (`metaStatus := meta.ChargeStatus(strings.SplitN(string(s), ".", 2)[0])`)
**Update inputs use mo.Option for partial patches** — UpdateRealizationRunInput fields are mo.Option[T]; only IsPresent() fields are written, and Validate() guards each present field individually (non-empty LineID, non-zero DeletedAt, etc.). (`Type mo.Option[RealizationRunType]; LineID mo.Option[*string]; if r.LineID.IsPresent() { ... }`)
**Decimal amounts rounded through the currency Calculator** — All monetary math (Intent.Normalized, CalculateAmountAfterProration) obtains i.Currency.Calculator() and calls RoundToPrecision before returning; negative amounts are a validation error everywhere. (`calc, err := i.Currency.Calculator(); return calc.RoundToPrecision(amount), nil`)
**Normalized() canonicalizes timestamps/periods before compare/persist** — Intent, State, RealizationRunBase, and UpdateRealizationRunInput expose Normalized() routing through meta.NormalizeTimestamp / NormalizeOptionalTimestamp / NormalizeClosedPeriod so equality and persistence are stable. (`s.AdvanceAfter = meta.NormalizeOptionalTimestamp(s.AdvanceAfter)`)
**Handler is an event-callback contract returning ledger/credit values** — Handler methods (OnAllocateCredits, OnInvoiceUsageAccrued, OnCorrectCreditAllocations, OnPayment*) take an *Input value and return creditrealization/ledgertransaction value types; they describe lifecycle events, not direct persistence. (`OnAllocateCredits(ctx, OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `charge.go` | Core domain model: ChargeBase (meta.ManagedResource + Intent + Status + State), Charge (adds Realizations, is the meta.ChargeAccessor), Intent with proration config, State, Realizations.GetByLineID. | Mutate Charge via WithStatus/WithBase, not piecemeal field assignment. Intent.Normalized rounds AmountBeforeProration — call it before persisting. |
| `charge.go (CalculateAmountAfterProration)` | Proration math: returns AmountBeforeProration unchanged unless ProRating.Enabled and Mode==ProRatePrices and ServicePeriod < FullServicePeriod. | Proration must never increase the amount; zero-length or ServicePeriod>=FullServicePeriod returns the full amount; result is currency-rounded. See charge_test.go for the 15/31-day and JPY zero-decimal cases. |
| `adapter.go` | Declares the composed Adapter interface, its narrow sub-interfaces, and create/get input types (CreateChargesInput, CreateCurrentRunInput, GetByIDsInput, IntentWithInitialStatus). | validateExpands enforces ExpandDetailedLines requires ExpandRealizations and ExpandDeletedRealizations requires ExpandRealizations. CreateCurrentRunInput requires LineID and InvoiceID together (both nil or both set). |
| `realizationrun.go` | RealizationRun model: RealizationRunBase, RealizationRun (adds CreditRealizations/AccruedUsage/Payment/DetailedLines), RealizationRunType enum, UpdateRealizationRunInput, IsVoidedBillingHistory. | InitialType must never be RealizationRunTypeInvalidDueToUnsupportedCreditNote (validation rejects it). Immutable runs cannot mutate the invoice line in place — deletion requires a credit note. IsVoidedBillingHistory is true for voided-type or DeletedAt!=nil runs. |
| `statemachine.go` | Defines the Status enum (dotted hierarchy) and ToMetaChargeStatus mapping; the actual state machines live in service/. | Adding a Status requires updating both the const block and Values(); the dotted prefix must be a valid meta.ChargeStatus or ToMetaChargeStatus fails. |
| `handler.go` | Handler interface plus its On*Input value types (OnAllocateCreditsInput, OnInvoiceUsageAccruedInput, CorrectCreditAllocationsInput, PaymentEventInput). | CorrectCreditAllocationsInput has both Validate() and ValidateWith(currencyCalculator) — currency-sensitive correction checks only run in ValidateWith. OnPaymentAuthorized/Settled share the PaymentEventInput alias. |
| `bookedat.go` | UsageBookedAt: ledger booking time = ServicePeriod.To for InArrears, ServicePeriod.From otherwise. | Drives credit/ledger booking timestamps — keep aligned with productcatalog.PaymentTermType semantics. |
| `detailedline.go / prorating.go` | detailedline.go aliases DetailedLine = stddetailedline.Base and adds DetailedLines Clone/Validate; prorating.go maps productcatalog ProRating modes to ProRatingModeAdapterEnum. | ProRatingModeAdapterEnum adds a no_prorate value absent from productcatalog — keep Values() in sync when persisting the adapter-side mode. |

## Anti-Patterns

- Returning on the first invalid field instead of collecting into var errs []error and joining via models.NewNillableGenericValidationError.
- Persisting a flat-fee Status or amount without normalizing (Normalized()) and mapping through ToMetaChargeStatus() / currency RoundToPrecision first.
- Adding a persistence method to a fat single interface rather than the correct narrow sub-interface (ChargeAdapter/ChargeRunAdapter/...).
- Allowing proration to increase the amount, or skipping the zero-length / ServicePeriod>=FullServicePeriod guards in CalculateAmountAfterProration.
- Mutating an invoice line in place for an Immutable RealizationRun instead of issuing a credit note, or setting InitialType to the unsupported-credit-note voided type.

## Decisions

- **Status is a dotted string hierarchy rather than a flat enum.** — Fine-grained lifecycle states (active.realization.*) collapse to coarse meta.ChargeStatus via a prefix split, so the broader billing system sees a small stable status set while flat-fee tracks detailed realization progress.
- **The Adapter is split into six narrow capability interfaces composed into one.** — Charge, run, detailed-line, credit-allocation, payment, and invoiced-usage persistence concerns evolve independently and can be mocked/implemented in isolation while still backed by one Ent adapter struct.
- **Domain types carry both Validate() and Normalized(), and amount math is currency-aware in the model layer.** — Proration and rounding correctness is a billing invariant; keeping it on Intent/RealizationRun (not the service) guarantees every caller — production wiring and tests — gets identical rounded, validated values.

## Example: Currency-aware proration on the Intent domain type

```
func (i Intent) CalculateAmountAfterProration() (alpacadecimal.Decimal, error) {
	if !i.ProRating.Enabled || i.ProRating.Mode != productcatalog.ProRatingModeProratePrices {
		return i.AmountBeforeProration, nil
	}
	sp := int64(i.ServicePeriod.Duration())
	full := int64(i.FullServicePeriod.Duration())
	if sp == 0 || full == 0 || sp >= full {
		return i.AmountBeforeProration, nil
	}
	pct := alpacadecimal.NewFromInt(sp).Div(alpacadecimal.NewFromInt(full))
	calc, err := i.Currency.Calculator()
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("creating currency calculator: %w", err)
	}
	return calc.RoundToPrecision(i.AmountBeforeProration.Mul(pct)), nil
// ...
```

<!-- archie:ai-end -->
