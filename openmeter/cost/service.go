package cost

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	GetInvoiceLineCost(ctx context.Context, params GetInvoiceLineCostParams) (InvoiceLineCost, error)
}

type GetInvoiceLineCostParams struct {
	InvoiceID     billing.InvoiceID
	InvoiceLineID string

	WindowSize     *meter.WindowSize
	WindowTimeZone *time.Location
	GroupBy        *[]string
}

func (p GetInvoiceLineCostParams) Validate() error {
	var errs []error

	if err := p.InvoiceID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if p.InvoiceLineID == "" {
		errs = append(errs, models.NewNillableGenericValidationError(
			errors.New("invoice line id is required"),
		))
	}

	return errors.Join(errs...)
}
