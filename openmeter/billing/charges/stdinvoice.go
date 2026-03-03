package charges

import (
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type StandardInvoicePaymentSettlementStatus string

const (
	StandardInvoicePaymentSettlementStatusAuthorized StandardInvoicePaymentSettlementStatus = "authorized"
	StandardInvoicePaymentSettlementStatusSettled    StandardInvoicePaymentSettlementStatus = "settled"
)

func (o StandardInvoicePaymentSettlementStatus) Values() []string {
	return []string{
		string(StandardInvoicePaymentSettlementStatusAuthorized),
		string(StandardInvoicePaymentSettlementStatusSettled),
	}
}

func (o StandardInvoicePaymentSettlementStatus) Validate() error {
	if !slices.Contains(o.Values(), string(o)) {
		return fmt.Errorf("invalid standard invoice settlement status: %s", o)
	}
	return nil
}

// StandardInvoicePaymentSettlement represents a payment settlement using a standard invoice managed
// by the OpenMeter platform.
type StandardInvoicePaymentSettlement struct {
	models.NamespacedID
	models.ManagedModel

	Annotations   models.Annotations    `json:"annotations"`
	LineID        string                `json:"lineID"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`

	Status StandardInvoicePaymentSettlementStatus `json:"status"`
	Amount alpacadecimal.Decimal                  `json:"amount"`

	Authorized *TimedLedgerTransactionGroupReference `json:"authorized"`
	Settled    *TimedLedgerTransactionGroupReference `json:"settled"`
}

func (r StandardInvoicePaymentSettlement) Validate() error {
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
		if r.Settled != nil {
			if err := r.Settled.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("settled: %w", err))
			}
		}
	}

	if r.LineID == "" {
		errs = append(errs, fmt.Errorf("line ID is required"))
	}

	if r.Amount.IsNegative() {
		errs = append(errs, fmt.Errorf("amount must be positive"))
	}

	switch r.Status {
	case StandardInvoicePaymentSettlementStatusAuthorized:
		if r.Authorized == nil {
			errs = append(errs, fmt.Errorf("authorized is required"))
		}
	case StandardInvoicePaymentSettlementStatusSettled:
		if r.Settled == nil {
			errs = append(errs, fmt.Errorf("settled is required"))
		}

		if r.Authorized == nil {
			errs = append(errs, fmt.Errorf("authorized is required"))
		}
	}

	return errors.Join(errs...)
}

type StandardInvoiceAccruedUsage struct {
	models.NamespacedID
	models.ManagedModel

	Annotations       models.Annotations               `json:"annotations"`
	LineID            *string                          `json:"lineID"`
	ServicePeriod     timeutil.ClosedPeriod            `json:"servicePeriod"`
	Mutable           bool                             `json:"mutable"`
	LedgerTransaction *LedgerTransactionGroupReference `json:"ledgerTransaction"`

	Totals billing.Totals `json:"totals"`
}

func (r StandardInvoiceAccruedUsage) Validate() error {
	var errs []error

	if !r.Mutable {
		if r.LineID == nil {
			errs = append(errs, fmt.Errorf("line ID is required when mutable is false"))
		}
	}

	if err := r.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if err := r.Totals.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("totals: %w", err))
	}

	if r.LineID != nil {
		if *r.LineID == "" {
			errs = append(errs, fmt.Errorf("line ID is required"))
		}
	}

	if r.LedgerTransaction != nil {
		if err := r.LedgerTransaction.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("ledger transaction: %w", err))
		}
	}

	return errors.Join(errs...)
}
