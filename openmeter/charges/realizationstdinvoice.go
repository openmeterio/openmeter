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

func (r StandardInvoiceRealization) Validate() error {
	var errs []error

	if err := r.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	return errors.Join(errs...)
}

type StandardInvoiceRealizationWithLine struct {
	StandardInvoiceRealization
	billing.StandardLineWithInvoiceHeader
}

func (r StandardInvoiceRealizationWithLine) Validate() error {
	var errs []error

	if err := r.StandardInvoiceRealization.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("standard invoice realization: %w", err))
	}

	return errors.Join(errs...)
}

type StandardInvoiceRealizations []StandardInvoiceRealization

func (r StandardInvoiceRealizations) Validate() error {
	var errs []error

	for idx, realization := range r {
		if err := realization.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("standard invoice realization[%d]: %w", idx, err))
		}
	}

	return errors.Join(errs...)
}

func (r StandardInvoiceRealizations) GetByLineID(lineID string) (StandardInvoiceRealization, bool) {
	for _, realization := range r {
		if realization.LineID == lineID {
			return realization, true
		}
	}
	return StandardInvoiceRealization{}, false
}
