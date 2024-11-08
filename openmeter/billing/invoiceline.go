package billing

import (
	"errors"
	"fmt"
	"time"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
)

type CreateInvoiceLinesInput struct {
	CustomerID string
	Namespace  string
	Lines      []billingentity.Line
}

func (c CreateInvoiceLinesInput) Validate() error {
	if c.Namespace == "" {
		return errors.New("namespace is required")
	}

	if c.CustomerID == "" {
		return errors.New("customer key or ID is required")
	}

	for _, line := range c.Lines {
		if err := line.Validate(); err != nil {
			return fmt.Errorf("Line: %w", err)
		}
	}

	return nil
}

type CreateInvoiceLinesAdapterInput []billingentity.Line

func (c CreateInvoiceLinesAdapterInput) Validate() error {
	for i, line := range c {
		if err := line.Validate(); err != nil {
			return fmt.Errorf("Line[%d]: %w", i, err)
		}

		if line.Namespace == "" {
			return fmt.Errorf("Line[%d]: namespace is required", i)
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

type ListInvoiceLinesAdapterInput struct {
	Namespace string

	CustomerID                 string
	InvoiceStatuses            []billingentity.InvoiceStatus
	InvoiceAtBefore            *time.Time
	ParentLineIDs              []string
	ParentLineIDsIncludeParent bool
	Statuses                   []billingentity.InvoiceLineStatus

	LineIDs []string
}

func (g ListInvoiceLinesAdapterInput) Validate() error {
	if g.Namespace == "" {
		return errors.New("namespace is required")
	}

	return nil
}

type AssociateLinesToInvoiceAdapterInput struct {
	Invoice billingentity.InvoiceID

	LineIDs []string
}

func (i AssociateLinesToInvoiceAdapterInput) Validate() error {
	if err := i.Invoice.Validate(); err != nil {
		return fmt.Errorf("invoice: %w", err)
	}

	if len(i.LineIDs) == 0 {
		return errors.New("line ids are required")
	}

	return nil
}

type UpdateInvoiceLineAdapterInput billingentity.Line
