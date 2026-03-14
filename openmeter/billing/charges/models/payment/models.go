package payment

import (
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const (
	PaymentSettlementStatusAttributeKey = "payment_settlement_status"
	PaymentSettlementTypeAttributeKey   = "payment_settlement_type"
	paymentSettlementIDAttributeKey     = "payment_settlement_id"

	// TODO: make sure we have a single constant for each payment settlement type
	PaymentSettlementTypeExternal        = "type_external"
	PaymentSettlementTypeStandardInvoice = "type_standard_invoice"
)

type Status string

const (
	StatusAuthorized Status = "authorized"
	StatusSettled    Status = "settled"
)

func (o Status) Values() []string {
	return []string{
		string(StatusAuthorized),
		string(StatusSettled),
	}
}

func (o Status) Validate() error {
	if !slices.Contains(o.Values(), string(o)) {
		return fmt.Errorf("invalid payment settlement status: %s", o)
	}
	return nil
}

// Base represents the generic payment settlement properties that are common to all payment settlements.
type Base struct {
	Annotations   models.Annotations    `json:"annotations"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`

	Status Status                `json:"status"`
	Amount alpacadecimal.Decimal `json:"amount"`

	Authorized *ledgertransaction.TimedGroupReference `json:"authorized"`
	Settled    *ledgertransaction.TimedGroupReference `json:"settled"`
}

func (r Base) Validate() error {
	var errs []error

	if err := r.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	if err := r.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if r.Authorized != nil {
		if err := r.Authorized.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("authorized: %w", err))
		}
	}

	if r.Settled != nil {
		if err := r.Settled.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("settled: %w", err))
		}
	}

	if !r.Amount.IsPositive() {
		errs = append(errs, fmt.Errorf("amount must be positive"))
	}

	switch r.Status {
	case StatusAuthorized:
		if r.Authorized == nil {
			errs = append(errs, fmt.Errorf("authorization transaction data is missing for authorized status"))
		}
	case StatusSettled:
		if r.Settled == nil {
			errs = append(errs, fmt.Errorf("settlement transaction data is missing for settled status"))
		}

		if r.Authorized == nil {
			errs = append(errs, fmt.Errorf("authorization transaction data is missing for settled status"))
		}
	}

	return errors.Join(errs...)
}

type Payment struct {
	models.NamespacedID
	models.ManagedModel

	Base
}

var _ models.Validator = (*Payment)(nil)

func (r Payment) Validate() error {
	var errs []error

	if err := r.Base.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("base: %w", err))
	}

	if err := r.NamespacedID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("namespaced ID: %w", err))
	}

	if err := r.ManagedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("managed model: %w", err))
	}

	return errors.Join(errs...)
}
