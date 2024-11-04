package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/api"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type GetInvoiceByIdInput struct {
	Invoice billingentity.InvoiceID
	Expand  billingentity.InvoiceExpand
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

type genericMultiInvoiceInput struct {
	Namespace  string
	InvoiceIDs []string
}

func (i genericMultiInvoiceInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if len(i.InvoiceIDs) == 0 {
		return errors.New("invoice IDs are required")
	}

	return nil
}

type (
	DeleteInvoicesAdapterInput       = genericMultiInvoiceInput
	LockInvoicesForUpdateInput       = genericMultiInvoiceInput
	AssociatedLineCountsAdapterInput = genericMultiInvoiceInput
)

type ListInvoicesInput struct {
	pagination.Page

	Namespace string
	Customers []string
	// Statuses searches by short InvoiceStatus (e.g. draft, issued)
	Statuses []string
	// ExtendedStatuses searches by exact InvoiceStatus
	ExtendedStatuses []billingentity.InvoiceStatus
	Currencies       []currencyx.Code

	IssuedAfter  *time.Time
	IssuedBefore *time.Time

	Expand billingentity.InvoiceExpand

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

type CreateInvoiceInput struct {
	Customer customerentity.CustomerID

	IncludePendingLines []string
	AsOf                *time.Time
}

func (i CreateInvoiceInput) Validate() error {
	if err := i.Customer.Validate(); err != nil {
		return fmt.Errorf("customer: %w", err)
	}

	if i.AsOf != nil && i.AsOf.After(clock.Now()) {
		return errors.New("asOf must be in the past")
	}

	return nil
}

type AssociatedLineCountsAdapterResponse struct {
	Counts map[billingentity.InvoiceID]int64
}

type (
	AdvanceInvoiceInput = billingentity.InvoiceID
	ApproveInvoiceInput = billingentity.InvoiceID
	RetryInvoiceInput   = billingentity.InvoiceID
)

type UpdateInvoiceAdapterInput = billingentity.Invoice
