package billing

import (
	"errors"
	"fmt"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
)

type CreateInvoiceLinesInput struct {
	CustomerKeyOrID string
	Namespace       string
	Lines           []billingentity.Line
}

func (c CreateInvoiceLinesInput) Validate() error {
	if c.Namespace == "" {
		return errors.New("namespace is required")
	}

	if c.CustomerKeyOrID == "" {
		return errors.New("customer key or ID is required")
	}

	for _, line := range c.Lines {
		if err := line.Validate(); err != nil {
			return fmt.Errorf("Line: %w", err)
		}
	}

	return nil
}

type CreateInvoiceLinesAdapterInput struct {
	Namespace string
	Lines     []billingentity.Line
}

func (c CreateInvoiceLinesAdapterInput) Validate() error {
	if c.Namespace == "" {
		return errors.New("namespace is required")
	}

	for i, line := range c.Lines {
		if err := line.Validate(); err != nil {
			return fmt.Errorf("Line[%d]: %w", i, err)
		}

		if line.InvoiceID == "" {
			return fmt.Errorf("Line[%d]: invoice id is required", i)
		}
	}

	return nil
}

type CreateInvoiceLinesResponse struct {
	Lines []billingentity.Line
}
