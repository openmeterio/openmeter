package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/api"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type InvoiceExpand struct {
	Lines        bool
	Preceding    bool
	Workflow     bool
	WorkflowApps bool
}

var InvoiceExpandAll = InvoiceExpand{
	Lines:        true,
	Preceding:    true,
	Workflow:     true,
	WorkflowApps: true,
}

func (e InvoiceExpand) Validate() error {
	if !e.Workflow && e.WorkflowApps {
		return errors.New("workflow.apps can only be expanded when workflow is expanded")
	}

	return nil
}

type GetInvoiceByIdInput struct {
	Invoice models.NamespacedID
	Expand  InvoiceExpand
}

func (i GetInvoiceByIdInput) Validate() error {
	if err := i.Invoice.Validate(); err != nil {
		return fmt.Errorf("id: %w", err)
	}

	if err := i.Expand.Validate(); err != nil {
		return fmt.Errorf("expand: %w", err)
	}

	return nil
}

type ListInvoicesInput struct {
	pagination.Page

	Namespace  string
	Customers  []string
	Statuses   []billingentity.InvoiceStatus
	Currencies []currencyx.Code

	IssuedAfter  *time.Time
	IssuedBefore *time.Time

	Expand InvoiceExpand

	OrderBy api.BillingInvoiceOrderBy
	Order   sortx.Order
}

func (i ListInvoicesInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.IssuedAfter != nil && i.IssuedBefore != nil && i.IssuedAfter.After(*i.IssuedBefore) {
		return errors.New("issuedAfter must be before issuedBefore")
	}

	if err := i.Expand.Validate(); err != nil {
		return fmt.Errorf("expand: %w", err)
	}

	return nil
}

type ListInvoicesResponse = pagination.PagedResponse[billingentity.Invoice]

type CreateInvoiceAdapterInput struct {
	Namespace string
	Customer  customerentity.Customer
	Profile   billingentity.Profile
	Currency  currencyx.Code
	Status    billingentity.InvoiceStatus
	Metadata  map[string]string
	IssuedAt  time.Time

	Type        billingentity.InvoiceType
	Description *string
	DueAt       *time.Time
}

func (c CreateInvoiceAdapterInput) Validate() error {
	if c.Namespace == "" {
		return errors.New("namespace is required")
	}

	if err := c.Customer.Validate(); err != nil {
		return fmt.Errorf("customer: %w", err)
	}

	if err := c.Profile.Validate(); err != nil {
		return fmt.Errorf("profile: %w", err)
	}

	if err := c.Currency.Validate(); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	if err := c.Status.Validate(); err != nil {
		return fmt.Errorf("status: %w", err)
	}

	if err := c.Type.Validate(); err != nil {
		return fmt.Errorf("type: %w", err)
	}

	return nil
}

type CreateInvoiceAdapterRespone = billingentity.Invoice
