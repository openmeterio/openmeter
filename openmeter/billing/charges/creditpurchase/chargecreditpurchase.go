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
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ meta.ChargeAccessor = (*Charge)(nil)

type Charge struct {
	meta.ManagedResource

	Status meta.ChargeStatus `json:"status"`

	Intent Intent `json:"intent"`
	State  State  `json:"state"`
}

func (c Charge) Validate() error {
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

	return errors.Join(errs...)
}

func (c Charge) GetChargeID() meta.ChargeID {
	return meta.ChargeID{
		Namespace: c.Namespace,
		ID:        c.ID,
	}
}

func (c Charge) ErrorAttributes() models.Attributes {
	return models.Attributes{
		"charge_id":   c.ID,
		"namespace":   c.Namespace,
		"charge_type": string(meta.ChargeTypeCreditPurchase),
	}
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

	if i.EffectiveAt != nil {
		return errors.New("effective at is not yet supported")
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type State struct {
	CreditGrantRealization    *ledgertransaction.TimedGroupReference `json:"creditGrantRealization"`
	ExternalPaymentSettlement *payment.External                      `json:"externalPaymentSettlement"`
	InvoiceSettlement         *payment.Invoiced                      `json:"invoiceSettlement"`
}

func (s State) Validate() error {
	var errs []error

	if s.CreditGrantRealization != nil {
		if err := s.CreditGrantRealization.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit grant realization: %w", err))
		}
	}

	if s.ExternalPaymentSettlement != nil {
		if err := s.ExternalPaymentSettlement.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("external payment settlement: %w", err))
		}
	}

	if s.InvoiceSettlement != nil {
		if err := s.InvoiceSettlement.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invoice settlement: %w", err))
		}
	}

	return errors.Join(errs...)
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

	return errors.Join(errs...)
}
