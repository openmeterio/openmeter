package creditpurchase

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
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

	CreditAmount alpacadecimal.Decimal `json:"amount"`
	// EffectiveAt is the time at which the credit purchase is effective.
	// Warning/TODO[later]: Currently this is not supported in credit purchase handler and the charge will be created
	// with booked_at set to CreatedAt.
	EffectiveAt *time.Time `json:"effectiveAt"`
	Priority    *int       `json:"priority"`

	// Settlement intent
	Settlement Settlement `json:"settlement"`
}

func (i Intent) Normalized() Intent {
	i.Intent = i.Intent.Normalized()
	i.EffectiveAt = meta.NormalizeOptionalTimestamp(i.EffectiveAt)

	calc, err := i.Currency.Calculator()
	if err == nil {
		i.CreditAmount = calc.RoundToPrecision(i.CreditAmount)
	}

	return i
}

func (i Intent) CalculateEffectiveAt() time.Time {
	return lo.FromPtrOr(i.EffectiveAt, clock.Now().UTC())
}

func (i Intent) Validate() error {
	var errs []error

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent meta: %w", err))
	}

	if !i.CreditAmount.IsPositive() {
		errs = append(errs, fmt.Errorf("credit amount must be positive"))
	}

	if err := i.Settlement.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement: %w", err))
	}

	switch i.Settlement.Type() {
	case SettlementTypeInvoice:
		settlement, err := i.Settlement.AsInvoiceSettlement()
		if err != nil {
			errs = append(errs, fmt.Errorf("settlement: %w", err))
		} else if settlement.Currency != i.Currency {
			errs = append(errs, fmt.Errorf("settlement currency %q must match credit currency %q", settlement.Currency, i.Currency))
		}
	case SettlementTypeExternal:
		settlement, err := i.Settlement.AsExternalSettlement()
		if err != nil {
			errs = append(errs, fmt.Errorf("settlement: %w", err))
		} else if settlement.Currency != i.Currency {
			errs = append(errs, fmt.Errorf("settlement currency %q must match credit currency %q", settlement.Currency, i.Currency))
		}
	}

	if i.EffectiveAt != nil {
		return errors.New("effective at is not yet supported")
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
