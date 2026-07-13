package creditpurchase

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
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

func (c ChargeBase) GetCurrency() currencyx.Code {
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

func (f IntentMutableFields) Normalized(currency currencyx.Code) IntentMutableFields {
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

	calc, err := currency.Calculator()
	if err == nil {
		f.CreditAmount = calc.RoundToPrecision(f.CreditAmount)
	}

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

	settlementErr := f.Settlement.Validate()
	if settlementErr != nil {
		errs = append(errs, fmt.Errorf("settlement: %w", settlementErr))
	}

	if err := f.FeatureFilters.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("feature filters: %w", err))
	}

	switch f.Settlement.Type() {
	case SettlementTypeInvoice:
		if _, err := f.Settlement.AsInvoiceSettlement(); err != nil {
			errs = append(errs, fmt.Errorf("settlement: %w", err))
		}
	case SettlementTypeExternal:
		if _, err := f.Settlement.AsExternalSettlement(); err != nil {
			errs = append(errs, fmt.Errorf("settlement: %w", err))
		}
	}

	if settlementErr == nil && f.CreditAmount.IsPositive() && f.Settlement.Type() != SettlementTypePromotional {
		if _, _, err := SettlementAmount(f.Settlement, f.CreditAmount); err != nil {
			errs = append(errs, fmt.Errorf("settlement amount: %w", err))
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

	if !slices.Contains(billing.InvoiceLineManagedBy("").Values(), string(i.ManagedBy)) {
		errs = append(errs, fmt.Errorf("intent meta: invalid managed by %s", i.ManagedBy))
	}

	if i.CustomerID == "" {
		errs = append(errs, fmt.Errorf("intent meta: customer ID is required"))
	}

	if err := i.Currency.ValidateFormat(); err != nil {
		errs = append(errs, fmt.Errorf("intent meta: currency: %w", err))
	}

	if err := i.TaxConfig.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent meta: tax config: %w", err))
	}

	if i.Subscription != nil {
		if err := i.Subscription.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("intent meta: subscription: %w", err))
		}
	}

	if i.UniqueReferenceID != nil && *i.UniqueReferenceID == "" {
		errs = append(errs, fmt.Errorf("intent meta: unique reference ID cannot be empty"))
	}

	if err := i.IntentMutableFields.Validate(); err != nil {
		errs = append(errs, err)
	}

	switch i.Settlement.Type() {
	case SettlementTypeInvoice:
		settlement, err := i.Settlement.AsInvoiceSettlement()
		if err == nil && i.Currency.IsKnownFiat() && settlement.Currency != i.Currency {
			errs = append(errs, fmt.Errorf("settlement currency %q must match credit currency %q", settlement.Currency, i.Currency))
		}
	case SettlementTypeExternal:
		settlement, err := i.Settlement.AsExternalSettlement()
		if err == nil && i.Currency.IsKnownFiat() && settlement.Currency != i.Currency {
			errs = append(errs, fmt.Errorf("settlement currency %q must match credit currency %q", settlement.Currency, i.Currency))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// State holds durable base-row scheduling fields for the credit purchase charge.
// Currently empty — all lifecycle outcomes live in Realizations.
type State struct{}

func (s State) Validate() error {
	return nil
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
