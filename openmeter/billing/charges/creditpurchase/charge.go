package creditpurchase

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargecostbasis "github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ChargeBase struct {
	meta.ManagedResource

	Intent Intent `json:"intent"`
	Status Status `json:"status"`

	State State `json:"state"`
}

func (c ChargeBase) Validate() error {
	var errs []error

	if err := c.ManagedResource.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("managed resource: %w", err))
	}

	if err := c.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if err := c.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	if err := c.State.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("state: %w", err))
	}

	if err := c.validateCostBasis(); err != nil {
		errs = append(errs, fmt.Errorf("cost basis: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c ChargeBase) validateCostBasis() error {
	if c.State.SchemaLevel == 0 {
		return nil
	}

	if c.Intent.Settlement.Type() == SettlementTypePromotional {
		if c.State.CostBasisID != nil || c.State.ResolvedCostBasis != nil {
			return errors.New("promotional credit purchase cannot have cost-basis state")
		}

		return nil
	}

	if c.Intent.CostBasis == nil {
		return errors.New("persisted payment-backed credit purchase requires a cost basis")
	}

	if !c.Intent.Currency.IsCustom() {
		if c.State.CostBasisID != nil || c.State.ResolvedCostBasis != nil {
			return errors.New("fiat credit purchase cannot have custom-currency cost-basis state")
		}

		return nil
	}

	if c.State.ResolvedCostBasis == nil {
		return errors.New("custom-currency credit purchase requires resolved cost-basis state")
	}

	switch c.State.SchemaLevel {
	case SchemaLevelLegacy:
		if c.State.CostBasisID != nil {
			return errors.New("legacy custom-currency credit purchase cannot reference persisted cost-basis state")
		}
	case SchemaLevelCostBasis:
		if c.State.CostBasisID == nil {
			return errors.New("cost-basis schema custom-currency credit purchase requires a cost-basis ID")
		}
	}

	return nil
}

func (c ChargeBase) GetChargeID() meta.ChargeID {
	return meta.ChargeID{
		Namespace: c.Namespace,
		ID:        c.ID,
	}
}

func (c ChargeBase) GetCustomerID() customer.CustomerID {
	return customer.CustomerID{
		Namespace: c.Namespace,
		ID:        c.Intent.CustomerID,
	}
}

func (c ChargeBase) GetCurrency() currencies.Currency {
	return c.Intent.Currency
}

func (c ChargeBase) ErrorAttributes() models.Attributes {
	return models.Attributes{
		"charge_id":   c.ID,
		"namespace":   c.Namespace,
		"charge_type": string(meta.ChargeTypeCreditPurchase),
	}
}

var _ meta.ChargeAccessor = (*Charge)(nil)

type Charge struct {
	ChargeBase

	Realizations Realizations `json:"realizations"`
}

func (c Charge) GetStatus() Status {
	return c.Status
}

func (c Charge) WithStatus(status Status) Charge {
	c.Status = status
	return c
}

func (c Charge) GetBase() ChargeBase {
	return c.ChargeBase
}

func (c Charge) WithBase(base ChargeBase) Charge {
	c.ChargeBase = base
	return c
}

func (c Charge) Validate() error {
	var errs []error

	if err := c.ChargeBase.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge base: %w", err))
	}

	if err := c.Realizations.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("realizations: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Intent struct {
	meta.Intent
	IntentMutableFields

	// CostBasis describes the purchase price independently from the payment
	// settlement mechanism. Promotional purchases do not have one.
	CostBasis *CostBasis `json:"costBasis,omitempty"`

	// Key is the optional idempotency key: a retried create with the same key returns a conflict.
	Key *string `json:"key,omitempty"`
}

type IntentMutableFields struct {
	meta.IntentMutableFields

	CreditAmount alpacadecimal.Decimal `json:"amount"`
	// EffectiveAt is the time at which the credit purchase is effective.
	// When set, the credit purchase service period is pinned to this instant.
	EffectiveAt *time.Time `json:"effectiveAt"`
	ExpiresAt   *time.Time `json:"expiresAt"`
	Priority    *int       `json:"priority"`

	FeatureFilters FeatureFilters `json:"featureFilters,omitempty"`

	// Settlement intent
	Settlement Settlement `json:"settlement"`
}

func (i Intent) Normalized() Intent {
	i.IntentMutableFields = i.IntentMutableFields.Normalized(i.Currency)

	return i
}

func (f IntentMutableFields) Normalized(currency currencies.Currency) IntentMutableFields {
	f.IntentMutableFields = f.IntentMutableFields.Normalized()
	f.EffectiveAt = meta.NormalizeOptionalTimestamp(f.EffectiveAt)
	f.ExpiresAt = meta.NormalizeOptionalTimestamp(f.ExpiresAt)
	f.FeatureFilters = f.FeatureFilters.Normalize()

	if f.EffectiveAt != nil {
		period := timeutil.ClosedPeriod{
			From: lo.FromPtr(f.EffectiveAt),
			To:   lo.FromPtr(f.EffectiveAt),
		}
		f.ServicePeriod = period
		f.FullServicePeriod = period
		f.BillingPeriod = period
	}

	f.CreditAmount = currency.RoundToPrecision(f.CreditAmount)

	return f
}

func (f IntentMutableFields) CalculateEffectiveAt() time.Time {
	return lo.FromPtrOr(f.EffectiveAt, clock.Now().UTC())
}

func (f IntentMutableFields) Validate() error {
	var errs []error

	if err := f.IntentMutableFields.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent mutable fields: %w", err))
	}

	if !f.CreditAmount.IsPositive() {
		errs = append(errs, fmt.Errorf("credit amount must be positive"))
	}

	if err := f.Settlement.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement: %w", err))
	}

	if err := f.FeatureFilters.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("feature filters: %w", err))
	}

	switch f.Settlement.Type() {
	case SettlementTypeExternal:
		if _, err := f.Settlement.AsExternalSettlement(); err != nil {
			errs = append(errs, fmt.Errorf("settlement: %w", err))
		}
	}

	if f.ExpiresAt != nil && !f.ExpiresAt.After(f.CalculateEffectiveAt()) {
		errs = append(errs, fmt.Errorf("expires at must be after effective at"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i Intent) CalculateEffectiveAt() time.Time {
	return i.IntentMutableFields.CalculateEffectiveAt()
}

func (i Intent) Validate() error {
	var errs []error

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent meta: %w", err))
	}

	if err := i.IntentMutableFields.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.validateCostBasis(); err != nil {
		errs = append(errs, err)
	}

	if i.Key != nil && *i.Key == "" {
		errs = append(errs, errors.New("key cannot be empty"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i Intent) validateCostBasis() error {
	switch i.Settlement.Type() {
	case SettlementTypePromotional:
		if i.CostBasis != nil {
			return errors.New("promotional credit purchase cannot have a cost basis")
		}

		return nil
	case SettlementTypeInvoice, SettlementTypeExternal:
		if i.CostBasis == nil {
			return errors.New("payment-backed credit purchase requires a cost basis")
		}
	default:
		// Settlement validation owns unsupported variants.
		return nil
	}

	if err := i.CostBasis.Validate(); err != nil {
		return fmt.Errorf("cost basis: %w", err)
	}

	if i.Currency.IsCustom() {
		if i.CostBasis.Type() != CostBasisTypeCustomCurrency {
			return errors.New("custom currency credit purchase requires a custom currency cost basis")
		}

		return nil
	}

	if i.CostBasis.Type() != CostBasisTypeFiat {
		return errors.New("fiat credit purchase requires a fiat cost basis")
	}

	return nil
}

// State holds durable base-row scheduling fields for the credit purchase charge.
type State struct {
	// SchemaLevel is the persisted cost-basis representation. Zero is reserved
	// for aggregates constructed outside persistence, including service inputs.
	SchemaLevel SchemaLevel `json:"-"`

	// VoidedAt is set when the remaining value was forfeited through the
	// ledger void flow; the breakage records stay the accounting source of truth.
	VoidedAt *time.Time `json:"voidedAt,omitempty"`

	// CostBasisID and ResolvedCostBasis apply only to custom-currency credit
	// purchases. Fiat cost basis is immutable intent stored on the charge row.
	CostBasisID       *string                `json:"-"`
	ResolvedCostBasis *chargecostbasis.State `json:"resolvedCostBasis,omitempty"`
}

func (s State) Validate() error {
	var errs []error

	if s.SchemaLevel != 0 {
		if err := s.SchemaLevel.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if s.CostBasisID != nil && *s.CostBasisID == "" {
		errs = append(errs, errors.New("cost basis ID cannot be empty"))
	}

	if s.ResolvedCostBasis != nil {
		if err := s.ResolvedCostBasis.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("resolved cost basis: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Realizations holds expand-only data loaded from child tables (edges).
type Realizations struct {
	CreditGrantRealization    *ledgertransaction.TimedGroupReference `json:"creditGrantRealization"`
	ExternalPaymentSettlement *payment.External                      `json:"externalPaymentSettlement"`
	InvoiceSettlement         *payment.Invoiced                      `json:"invoiceSettlement"`
}

func (r Realizations) Validate() error {
	var errs []error

	if r.CreditGrantRealization != nil {
		if err := r.CreditGrantRealization.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit grant realization: %w", err))
		}
	}

	if r.ExternalPaymentSettlement != nil {
		if err := r.ExternalPaymentSettlement.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("external payment settlement: %w", err))
		}
	}

	if r.InvoiceSettlement != nil {
		if err := r.InvoiceSettlement.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invoice settlement: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type UpdateExternalPaymentStateInput struct {
	ChargeID           meta.ChargeID
	TargetPaymentState payment.Status
}

func (i UpdateExternalPaymentStateInput) Validate() error {
	var errs []error

	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if err := i.TargetPaymentState.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("target payment state: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
