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

type StandardInvoiceSettlementStatus string

const (
	StandardInvoiceSettlementStatusDraft      StandardInvoiceSettlementStatus = "draft"
	StandardInvoiceSettlementStatusAuthorized StandardInvoiceSettlementStatus = "authorized"
	StandardInvoiceSettlementStatusSettled    StandardInvoiceSettlementStatus = "settled"
)

func (o StandardInvoiceSettlementStatus) Values() []string {
	return []string{
		string(StandardInvoiceSettlementStatusDraft),
		string(StandardInvoiceSettlementStatusAuthorized),
		string(StandardInvoiceSettlementStatusSettled),
	}
}

func (o StandardInvoiceSettlementStatus) Validate() error {
	if !slices.Contains(o.Values(), string(o)) {
		return fmt.Errorf("invalid standard invoice settlement status: %s", o)
	}
	return nil
}

type StandardInvoiceSettlement struct {
	models.NamespacedID
	models.ManagedModel

	Annotations   models.Annotations    `json:"annotations"`
	LineID        string                `json:"lineID"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`

	Status StandardInvoiceSettlementStatus `json:"status"`

	// Usage snapshots
	// Before any usage discounts are applied
	MeteredServicePeriodQuantity    alpacadecimal.Decimal `json:"servicePeriodQuantity"`
	MeteredPreServicePeriodQuantity alpacadecimal.Decimal `json:"preServicePeriodQuantity"`

	// The realized totals of the line after any usage discounts are applied
	Totals billing.Totals `json:"totals"` // TODO: is this neededed?
}

func (r StandardInvoiceSettlement) Validate() error {
	var errs []error

	if err := r.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	return errors.Join(errs...)
}
