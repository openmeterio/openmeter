# creditpurchase

<!-- archie:ai-start -->

> Domain package for the credit-purchase charge type: defines the Charge/ChargeBase/Intent value types, the three-variant Settlement (invoice/external/promotional), the persistence Adapter and lifecycle Service interfaces, and the ledger Handler contract. Constraint: every public input has a Validate() and all settlement variants must agree on currency with the credit Intent.

## Patterns

**Validate() collects errors then NewNillableGenericValidationError** — Every input/value type implements Validate() by accumulating into `var errs []error` with `fmt.Errorf("field: %w", err)` wrapping, returning models.NewNillableGenericValidationError(errors.Join(errs...)) (`CreateChargeInput.Validate, Charge.Validate, Intent.Validate`)
**Tagged-union Settlement with private fields + constructor** — Settlement holds a `t SettlementType` discriminant and private `*InvoiceSettlement/*ExternalSettlement/*PromotionalSettlement`; build only via NewSettlement[T], read via AsInvoiceSettlement/AsExternalSettlement/AsPromotionalSettlement (error on type mismatch), with custom MarshalJSON/UnmarshalJSON keyed on `type` (`NewSettlement(InvoiceSettlement{...}); s.AsExternalSettlement()`)
**Status enum aliased from meta.ChargeStatus** — Status constants (StatusCreated/Active/Final/Deleted) are defined as Status(meta.ChargeStatus...) and convert back via ToMetaChargeStatus(); never invent a separate status space (`StatusActive = Status(meta.ChargeStatusActive)`)
**Intent.Normalized() before persist** — Intent.Normalized() normalizes embedded meta.Intent, optional timestamps via meta.NormalizeOptionalTimestamp, feature filters, and rounds CreditAmount via Currency.Calculator() (`i.CreditAmount = calc.RoundToPrecision(i.CreditAmount)`)
**Realizations are expand-only edge data** — Charge embeds ChargeBase + Realizations (CreditGrantRealization/ExternalPaymentSettlement/InvoiceSettlement pointers); State struct is intentionally empty — all lifecycle outcomes live in Realizations, loaded only under meta.ExpandRealizations (`type State struct{}`)
**Handler interface mediates all ledger side-effects** — Ledger interaction is an injected Handler (OnPromotionalCreditPurchase, OnCreditPurchaseInitiated, OnCreditPurchasePaymentAuthorized/Settled) returning ledgertransaction.GroupReference; the package's value types never call ledger directly (`Handler.OnCreditPurchaseInitiated(ctx, charge)`)
**ValidationIssue errors with HTTP attributes** — Errors are models.NewValidationIssue with an ErrorCode const, severity and commonhttp.WithHTTPStatusCodeAttribute — not plain fmt.Errorf (`ErrCreditPurchaseChargeNotActive (StatusBadRequest)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `charge.go` | ChargeBase/Charge/Intent/Realizations/State value types and their Validate/normalize/getters | EffectiveAt is validated as not-yet-supported (returns error if non-nil); CreditAmount must be positive; settlement currency must equal Intent.Currency |
| `settlement.go` | SettlementType enum + tagged-union Settlement with JSON round-trip and AsX accessors | Construct only via NewSettlement; MarshalJSON errors if the variant pointer for the discriminant is nil; promotional carries no payload |
| `adapter.go` | Adapter interface (Charge/ExternalPayment/CreditGrant/InvoicedPayment + entutils.TxCreator) and its input structs | All inputs Validate namespace + Expands; GetByID/GetByIDs require ChargeID/Expands validation before DB access |
| `service.go` | Service interface composed of CreditPurchaseService + ExternalPaymentLifecycle + InvoicePaymentLifecycle; ChargeWithGatheringLine result | Create handles a single Intent only; invoice-settled lifecycle is driven by PostInvoice* hooks, not Create |
| `handler.go` | Ledger Handler contract documenting the cost-basis>0 happy path (initiated -> authorized -> settled) vs promotional (single call) | Promotional purchases call ONLY OnPromotionalCreditPurchase; failed payments can occur after initiated or after authorized |
| `statemachine.go` | Status enum aliased to meta.ChargeStatus + ToMetaChargeStatus | Keep the four statuses in lockstep with meta.ChargeStatus values |
| `featurefilters.go` | FeatureFilters []string with Validate (no empty/dup) and Normalize (sorted+uniq via slicesx.Normalize) | Normalize length mismatch is how duplicate detection works |
| `funded_credit_activity.go` | FundedCreditActivity read model + keyset cursor (FundedAt/ChargeCreatedAt/ChargeID) for ListFundedCreditActivities | After and Before cursors are mutually exclusive; Limit must be >= 1 |

## Anti-Patterns

- Constructing a Settlement by setting struct fields directly instead of NewSettlement[T] — private fields make this impossible and JSON marshal fails on nil variant
- Returning plain fmt.Errorf for lifecycle conflicts instead of the predefined ValidationIssue in errors.go
- Putting ledger/credit-grant side effects in this package's value types — they belong behind the injected Handler in the service sub-package
- Setting Intent.EffectiveAt — validation rejects it as not-yet-supported
- Reading Realizations without expanding meta.ExpandRealizations (they come back nil/unmapped)

## Decisions

- **Settlement is a JSON-tagged union with private variant pointers** — Forces construction through NewSettlement and a single `type` discriminant so invalid combinations are unrepresentable and persistence is stable
- **State struct is empty; all lifecycle outcomes live in Realizations edges** — Lifecycle results (credit grant, external/invoice settlement) are append-only child rows loaded on demand rather than mutable base-row fields
- **Status values alias meta.ChargeStatus** — Keeps the per-type charge status compatible with the shared charge-meta status space used by the generic charge engine

## Example: Validating a credit-purchase Intent with settlement-currency cross-check

```
func (i Intent) Validate() error {
  var errs []error
  if err := i.Intent.Validate(); err != nil { errs = append(errs, fmt.Errorf("intent meta: %w", err)) }
  if !i.CreditAmount.IsPositive() { errs = append(errs, fmt.Errorf("credit amount must be positive")) }
  if err := i.Settlement.Validate(); err != nil { errs = append(errs, fmt.Errorf("settlement: %w", err)) }
  if s, err := i.Settlement.AsInvoiceSettlement(); err == nil && s.Currency != i.Currency {
    errs = append(errs, fmt.Errorf("settlement currency %q must match credit currency %q", s.Currency, i.Currency))
  }
  return models.NewNillableGenericValidationError(errors.Join(errs...))
}
```

<!-- archie:ai-end -->
