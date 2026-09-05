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
	"github.com/openmeterio/openmeter/pkg/currencyx"
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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

// GetFiatSettlementAmount returns the purchased credit's value in its
// settlement currency using the resolved cost basis.
func (c Charge) GetFiatSettlementAmount() (alpacadecimal.Decimal, error) {
	fiatCurrency, err := c.Intent.GetSettlementFiatCurrency()
	if err != nil {
		return alpacadecimal.Zero, fmt.Errorf("getting settlement fiat currency: %w", err)
	}

	if c.State.ResolvedCostBasis == nil {
		return alpacadecimal.Zero, errors.New("cost basis is unresolved")
	}

	fiatAmount, err := meta.CalculateFiatAmount(
		c.Intent.CreditAmount,
		c.State.ResolvedCostBasis.CostBasis,
		fiatCurrency,
	)
	if err != nil {
		return alpacadecimal.Zero, fmt.Errorf("calculating fiat settlement amount: %w", err)
	}

	return fiatAmount, nil
}

// ValidateSettlementAmount prevents issuing paid credits whose rounded purchase
// value cannot enter the payment lifecycle. Dynamic purchases call this after
// resolving their cost basis; promotional credits have no settlement amount.
func (c Charge) ValidateSettlementAmount() error {
	amount, err := c.GetFiatSettlementAmount()
	if err != nil {
		return err
	}
	if !amount.IsPositive() {
		return models.NewGenericValidationError(errors.New("purchase amount must be positive after rounding to settlement currency precision"))
	}
	return nil
}

type Intent struct {
	meta.Intent
	IntentMutableFields

	// CostBasis describes the purchase price independently from the payment
	// settlement mechanism. Promotional purchases do not have one.
	CostBasis CostBasis `json:"costBasis,omitzero"`

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

	if f.EffectiveAt != nil && f.ExpiresAt != nil && !f.ExpiresAt.After(*f.EffectiveAt) {
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
		if !i.CostBasis.IsEmpty() {
			return errors.New("promotional credit purchase cannot have a cost basis")
		}

		return nil
	case SettlementTypeInvoice, SettlementTypeExternal:
		if i.CostBasis.IsEmpty() {
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

// GetSettlementFiatCurrency returns the fiat currency in which a
// payment-backed credit purchase is settled.
func (i Intent) GetSettlementFiatCurrency() (*currencyx.FiatCurrency, error) {
	switch i.Settlement.Type() {
	case SettlementTypeInvoice, SettlementTypeExternal:
	case SettlementTypePromotional:
		return nil, errors.New("promotional credit purchase does not have a settlement fiat currency")
	default:
		return nil, fmt.Errorf("unsupported credit purchase settlement type: %s", i.Settlement.Type())
	}

	if i.CostBasis.IsEmpty() {
		return nil, errors.New("credit purchase cost basis is required")
	}

	switch i.CostBasis.Type() {
	case CostBasisTypeFiat:
		if i.Currency.IsCustom() {
			return nil, errors.New("fiat cost basis requires a fiat purchase currency")
		}

		fiatCurrency, err := currencyx.NewFiatCurrency(i.Currency.GetCode())
		if err != nil {
			return nil, fmt.Errorf("mapping purchase currency to fiat currency: %w", err)
		}

		return fiatCurrency, nil
	case CostBasisTypeCustomCurrency:
		costBasisIntent, err := i.CostBasis.AsCustomCurrency()
		if err != nil {
			return nil, err
		}

		fiatCurrency, err := costBasisIntent.GetFiatCurrency()
		if err != nil {
			return nil, fmt.Errorf("getting custom-currency fiat currency: %w", err)
		}

		return fiatCurrency, nil
	default:
		return nil, fmt.Errorf("unsupported credit purchase cost basis type: %s", i.CostBasis.Type())
	}
}

// State holds durable base-row scheduling fields for the credit purchase charge.
type State struct {
	// VoidedAt is set when the remaining value was forfeited through the
	// ledger void flow; the breakage records stay the accounting source of truth.
	VoidedAt *time.Time `json:"voidedAt,omitempty"`
	// ChargeCostBasisID references the persisted custom-currency cost-basis
	// intent and state owned by this charge. Fiat purchases do not have one.
	ChargeCostBasisID *string `json:"chargeCostBasisID,omitempty"`

	// ResolvedCostBasis is populated before the first monetary realization.
	// Fiat state is reconstructed from immutable charge-row fields, while custom
	// currency state is persisted on the shared cost-basis child. Dynamic custom
	// currency intent remains unresolved until that realization boundary.
	ResolvedCostBasis *chargecostbasis.State `json:"resolvedCostBasis,omitempty"`
}

func (s State) Validate() error {
	var errs []error
	if s.ChargeCostBasisID != nil && *s.ChargeCostBasisID == "" {
		errs = append(errs, errors.New("charge cost basis ID cannot be empty"))
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
