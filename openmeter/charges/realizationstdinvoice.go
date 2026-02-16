package charges

import (
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type StandardInvoiceRealizationStatus string

const (
	StandardInvoiceRealizationStatusDraft      StandardInvoiceRealizationStatus = "draft"
	StandardInvoiceRealizationStatusAuthorized StandardInvoiceRealizationStatus = "authorized"
	StandardInvoiceRealizationStatusSettled    StandardInvoiceRealizationStatus = "settled"
)

func (o StandardInvoiceRealizationStatus) Values() []string {
	return []string{
		string(StandardInvoiceRealizationStatusDraft),
		string(StandardInvoiceRealizationStatusAuthorized),
		string(StandardInvoiceRealizationStatusSettled),
	}
}

func (o StandardInvoiceRealizationStatus) Validate() error {
	if !slices.Contains(o.Values(), string(o)) {
		return fmt.Errorf("invalid invoice outcome status: %s", o)
	}
	return nil
}

type StandardInvoiceRealization struct {
	models.NamespacedID
	models.ManagedModel

	Annotations   models.Annotations    `json:"annotations"`
	LineID        string                `json:"lineID"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`

	Status StandardInvoiceRealizationStatus `json:"status"`

	// Usage snapshots
	// Before any usage discounts are applied
	MeteredServicePeriodQuantity    alpacadecimal.Decimal `json:"servicePeriodQuantity"`
	MeteredPreServicePeriodQuantity alpacadecimal.Decimal `json:"preServicePeriodQuantity"`

	// The realized totals of the line after any usage discounts are applied
	Totals billing.Totals `json:"totals"`
}
