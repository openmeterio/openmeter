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

type UpsertInvoiceLinesAdapterInput struct {
	Namespace string
	Lines     []*billingentity.Line
}

func (c UpsertInvoiceLinesAdapterInput) Validate() error {
	if c.Namespace == "" {
		return errors.New("namespace is required")
	}

	for i, line := range c.Lines {
		if err := line.Validate(); err != nil {
			return fmt.Errorf("line[%d]: %w", i, err)
		}

		if line.Namespace == "" {
			return fmt.Errorf("line[%d]: namespace is required", i)
		}

		if line.InvoiceID == "" {
			return fmt.Errorf("line[%d]: invoice id is required", i)
		}
	}

	return nil
}

type ListInvoiceLinesAdapterInput struct {
	Namespace string

	CustomerID                 string
	InvoiceStatuses            []billingentity.InvoiceStatus
	InvoiceAtBefore            *time.Time
	IncludeDeleted             bool
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

type GetInvoiceLineInput struct {
	Namespace string
	LineID    string
	InvoiceID string
}

func (g GetInvoiceLineInput) Validate() error {
	if g.Namespace == "" {
		return errors.New("namespace is required")
	}

	if g.LineID == "" {
		return errors.New("line id is required")
	}

	if g.InvoiceID == "" {
		return errors.New("invoice id is required")
	}

	return nil
}

type GetInvoiceLineOwnershipAdapterInput = billingentity.LineID

type ValidateLineOwnershipInput struct {
	Namespace  string
	LineID     string
	InvoiceID  string
	CustomerID string
}

func (v ValidateLineOwnershipInput) Validate() error {
	if v.Namespace == "" {
		return errors.New("namespace is required")
	}

	if v.LineID == "" {
		return errors.New("line id is required")
	}

	if v.InvoiceID == "" {
		return errors.New("invoice id is required")
	}

	if v.CustomerID == "" {
		return errors.New("customer id is required")
	}

	return nil
}
