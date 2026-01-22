package billing

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type CreatePendingInvoiceLinesInput struct {
	Customer customer.CustomerID `json:"customer"`
	Currency currencyx.Code      `json:"currency"`

	Lines []*StandardLine `json:"lines"`
}

func (c CreatePendingInvoiceLinesInput) Validate() error {
	var errs []error

	if err := c.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if err := c.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	for id, line := range c.Lines {
		// Note: this is for validation purposes, as Line is copied, we are not altering the struct itself
		line.Currency = c.Currency

		if err := line.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("line.%d: %w", id, err))
		}

		if line.InvoiceID != "" {
			errs = append(errs, fmt.Errorf("line.%d: invoice ID is not allowed for pending lines", id))
		}

		if len(line.DetailedLines) > 0 {
			errs = append(errs, fmt.Errorf("line.%d: detailed lines are not allowed for pending lines", id))
		}

		if line.ParentLineID != nil {
			errs = append(errs, fmt.Errorf("line.%d: parent line ID is not allowed for pending lines", id))
		}

		if line.SplitLineGroupID != nil {
			errs = append(errs, fmt.Errorf("line.%d: split line group ID is not allowed for pending lines", id))
		}
	}

	return errors.Join(errs...)
}

type CreatePendingInvoiceLinesResult struct {
	Lines        []*StandardLine
	Invoice      StandardInvoice
	IsInvoiceNew bool
}
