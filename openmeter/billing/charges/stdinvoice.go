package charges

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type StandardInvoicePaymentSettlementCreateInput struct {
	PaymentSettlementBase

	Namespace string `json:"namespace"`
	LineID    string `json:"lineID"`
}

func (i StandardInvoicePaymentSettlementCreateInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if err := i.PaymentSettlementBase.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("payment settlement base: %w", err))
	}

	if i.LineID == "" {
		errs = append(errs, fmt.Errorf("line ID is required"))
	}

	return errors.Join(errs...)
}

// StandardInvoicePaymentSettlement represents a payment settlement using a standard invoice managed
// by the OpenMeter platform.
type StandardInvoicePaymentSettlement struct {
	models.NamespacedID
	models.ManagedModel

	PaymentSettlementBase

	LineID string `json:"lineID"`
}

var _ models.Validator = (*StandardInvoicePaymentSettlement)(nil)

func (r StandardInvoicePaymentSettlement) Validate() error {
	var errs []error

	if r.LineID == "" {
		errs = append(errs, fmt.Errorf("line ID is required"))
	}

	if err := r.PaymentSettlementBase.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := r.NamespacedID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("namespaced ID: %w", err))
	}

	if err := r.ManagedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("managed model: %w", err))
	}

	return errors.Join(errs...)
}

func (r StandardInvoicePaymentSettlement) ErrorAttributes() models.Attributes {
	return models.Attributes{
		PaymentSettlementStatusAttributeKey: string(r.Status),
		PaymentSettlementTypeAttributeKey:   string(PaymentSettlementTypeStandardInvoice),
		paymentSettlementIDAttributeKey:     r.ID,
	}
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
