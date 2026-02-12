package billing

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/expand"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type InvoiceType string

const (
	InvoiceTypeStandard  InvoiceType = InvoiceType("standard")
	InvoiceTypeGathering InvoiceType = InvoiceType("gathering")
)

func (t InvoiceType) Values() []string {
	return []string{
		string(InvoiceTypeStandard),
		string(InvoiceTypeGathering),
	}
}

func (t InvoiceType) Validate() error {
	for _, status := range t.Values() {
		if string(t) == status {
			return nil
		}
	}

	return fmt.Errorf("invalid invoice type: %s", t)
}

type InvoiceID models.NamespacedID

func (i InvoiceID) Validate() error {
	return models.NamespacedID(i).Validate()
}

type GenericInvoice interface {
	GenericInvoiceReader

	SetLines(lines []GenericInvoiceLine) error
}

type GenericInvoiceReader interface {
	GetDeletedAt() *time.Time
	GetID() string
	GetInvoiceID() InvoiceID
	GetCustomerID() customer.CustomerID

	// GetGenericLines returns the lines of the invoice as generic lines.
	GetGenericLines() mo.Option[[]GenericInvoiceLine]

	AsInvoice() Invoice
}

type InvoiceExpand string

const (
	InvoiceExpandLines                                 InvoiceExpand = "lines"
	InvoiceExpandDeletedLines                          InvoiceExpand = "deletedLines"
	InvoiceExpandCalculateGatheringInvoiceWithLiveData InvoiceExpand = "calculateGatheringInvoiceWithLiveData"
)

func (e InvoiceExpand) Values() []InvoiceExpand {
	return []InvoiceExpand{
		InvoiceExpandLines,
		InvoiceExpandDeletedLines,
		InvoiceExpandCalculateGatheringInvoiceWithLiveData,
	}
}

type InvoiceExpands = expand.Expand[InvoiceExpand]

var InvoiceExpandAll = InvoiceExpands{
	InvoiceExpandLines,
}

type InvoiceExternalIDs struct {
	Invoicing string `json:"invoicing,omitempty"`
	Payment   string `json:"payment,omitempty"`
}

func (i *InvoiceExternalIDs) GetInvoicingOrEmpty() string {
	if i == nil {
		return ""
	}
	return i.Invoicing
}

type Invoice struct {
	t                InvoiceType
	standardInvoice  *StandardInvoice
	gatheringInvoice *GatheringInvoice
}

func NewInvoice[T StandardInvoice | GatheringInvoice](invoice T) Invoice {
	switch v := any(invoice).(type) {
	case StandardInvoice:
		return Invoice{
			t:               InvoiceTypeStandard,
			standardInvoice: &v,
		}
	case GatheringInvoice:
		return Invoice{
			t:                InvoiceTypeGathering,
			gatheringInvoice: &v,
		}
	}

	return Invoice{}
}

func (i Invoice) Type() InvoiceType {
	return i.t
}

func (i Invoice) AsStandardInvoice() (StandardInvoice, error) {
	if i.t != InvoiceTypeStandard {
		return StandardInvoice{}, fmt.Errorf("invoice is not a standard invoice")
	}

	if i.standardInvoice == nil {
		return StandardInvoice{}, fmt.Errorf("standard invoice is nil")
	}

	return *i.standardInvoice, nil
}

func (i Invoice) AsGatheringInvoice() (GatheringInvoice, error) {
	if i.t != InvoiceTypeGathering {
		return GatheringInvoice{}, fmt.Errorf("invoice is not a gathering invoice")
	}

	if i.gatheringInvoice == nil {
		return GatheringInvoice{}, fmt.Errorf("gathering invoice is nil")
	}

	return *i.gatheringInvoice, nil
}

func (i Invoice) AsGenericInvoice() (GenericInvoice, error) {
	switch i.t {
	case InvoiceTypeStandard:
		if i.standardInvoice == nil {
			return nil, fmt.Errorf("standard invoice is nil")
		}

		cloned, err := i.standardInvoice.Clone()
		if err != nil {
			return nil, err
		}

		return &cloned, nil
	case InvoiceTypeGathering:
		if i.gatheringInvoice == nil {
			return nil, fmt.Errorf("gathering invoice is nil")
		}

		cloned, err := i.gatheringInvoice.Clone()
		if err != nil {
			return nil, err
		}

		return &cloned, nil
	default:
		return nil, fmt.Errorf("invalid invoice type: %s", i.t)
	}
}

func (i Invoice) Validate() error {
	switch i.t {
	case InvoiceTypeStandard:
		if i.standardInvoice == nil {
			return fmt.Errorf("standard invoice is nil")
		}

		return i.standardInvoice.Validate()
	case InvoiceTypeGathering:
		if i.gatheringInvoice == nil {
			return fmt.Errorf("gathering invoice is nil")
		}

		return i.gatheringInvoice.Validate()
	default:
		return fmt.Errorf("invalid invoice type: %s", i.t)
	}
}

type GetInvoiceByIdInput struct {
	Invoice InvoiceID
	Expand  InvoiceExpands
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
	DeleteGatheringInvoicesInput     = genericMultiInvoiceInput
	LockInvoicesForUpdateInput       = genericMultiInvoiceInput
	AssociatedLineCountsAdapterInput = genericMultiInvoiceInput
)

type ExternalIDType string

const (
	InvoicingExternalIDType ExternalIDType = "invoicing"
	PaymentExternalIDType   ExternalIDType = "payment"
	TaxExternalIDType       ExternalIDType = "tax"
)

func (t ExternalIDType) Validate() error {
	if !slices.Contains([]ExternalIDType{
		InvoicingExternalIDType,
		PaymentExternalIDType,
		TaxExternalIDType,
	}, t) {
		return fmt.Errorf("invalid external ID type: %s", t)
	}

	return nil
}

type ListInvoicesExternalIDFilter struct {
	Type ExternalIDType
	IDs  []string
}

func (f ListInvoicesExternalIDFilter) Validate() error {
	if err := f.Type.Validate(); err != nil {
		return err
	}

	if len(f.IDs) == 0 {
		return errors.New("IDs are required")
	}

	return nil
}

type InvoiceAvailableActionsFilter string

const (
	InvoiceAvailableActionsFilterAdvance InvoiceAvailableActionsFilter = "advance"
	InvoiceAvailableActionsFilterApprove InvoiceAvailableActionsFilter = "approve"
)

func (f InvoiceAvailableActionsFilter) Values() []InvoiceAvailableActionsFilter {
	return []InvoiceAvailableActionsFilter{
		InvoiceAvailableActionsFilterAdvance,
		InvoiceAvailableActionsFilterApprove,
	}
}

func (f InvoiceAvailableActionsFilter) Validate() error {
	if !slices.Contains(f.Values(), f) {
		return fmt.Errorf("invalid available action filter: %s", f)
	}

	return nil
}

type ListInvoicesInput struct {
	pagination.Page

	Namespaces []string
	Customers  []string
	// Statuses searches by short InvoiceStatus (e.g. draft, issued)
	Statuses []string

	// ExtendedStatuses searches by exact InvoiceStatus
	ExtendedStatuses []StandardInvoiceStatus

	IssuedAfter  *time.Time
	IssuedBefore *time.Time

	PeriodStartAfter  *time.Time
	PeriodStartBefore *time.Time

	// Filter by invoice creation time
	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	IncludeDeleted bool

	Expand InvoiceExpands

	OrderBy api.InvoiceOrderBy
	Order   sortx.Order
}

func (i ListInvoicesInput) Validate() error {
	var outErr []error

	if i.IssuedAfter != nil && i.IssuedBefore != nil && i.IssuedAfter.After(*i.IssuedBefore) {
		outErr = append(outErr, errors.New("issuedAfter must be before issuedBefore"))
	}

	if i.CreatedAfter != nil && i.CreatedBefore != nil && i.CreatedAfter.After(*i.CreatedBefore) {
		outErr = append(outErr, errors.New("createdAfter must be before createdBefore"))
	}

	if i.PeriodStartAfter != nil && i.PeriodStartBefore != nil && i.PeriodStartAfter.After(*i.PeriodStartBefore) {
		outErr = append(outErr, errors.New("periodStartAfter must be before periodStartBefore"))
	}

	if err := i.Expand.Validate(); err != nil {
		outErr = append(outErr, fmt.Errorf("expand: %w", err))
	}

	return errors.Join(outErr...)
}

type ListInvoicesAdapterInput struct {
	pagination.Page

	Namespaces []string
	IDs        []string
	Customers  []string
	// Statuses searches by short InvoiceStatus (e.g. draft, issued)
	Statuses []string

	// ExtendedStatuses searches by exact InvoiceStatus
	ExtendedStatuses []StandardInvoiceStatus

	HasAvailableAction []InvoiceAvailableActionsFilter

	ExternalIDs *ListInvoicesExternalIDFilter

	DraftUntilLTE   *time.Time
	CollectionAtLTE *time.Time

	IssuedAfter  *time.Time
	IssuedBefore *time.Time

	PeriodStartAfter  *time.Time
	PeriodStartBefore *time.Time

	// Filter by invoice creation time
	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	IncludeDeleted bool

	Expand InvoiceExpands

	OrderBy api.InvoiceOrderBy
	Order   sortx.Order

	// OnlyStandard is used to filter for only standard invoices
	OnlyStandard bool
	// OnlyGathering is used to filter for only gathering invoices
	OnlyGathering bool
}

func (i ListInvoicesAdapterInput) Validate() error {
	var outErr []error

	if i.IssuedAfter != nil && i.IssuedBefore != nil && i.IssuedAfter.After(*i.IssuedBefore) {
		outErr = append(outErr, errors.New("issuedAfter must be before issuedBefore"))
	}

	if i.CreatedAfter != nil && i.CreatedBefore != nil && i.CreatedAfter.After(*i.CreatedBefore) {
		outErr = append(outErr, errors.New("createdAfter must be before createdBefore"))
	}

	if i.PeriodStartAfter != nil && i.PeriodStartBefore != nil && i.PeriodStartAfter.After(*i.PeriodStartBefore) {
		outErr = append(outErr, errors.New("periodStartAfter must be before periodStartBefore"))
	}

	if err := i.Expand.Validate(); err != nil {
		outErr = append(outErr, fmt.Errorf("expand: %w", err))
	}

	if i.OnlyStandard && i.OnlyGathering {
		outErr = append(outErr, errors.New("onlyStandard and onlyGathering cannot be true at the same time"))
	}

	if i.OnlyGathering {
		if len(i.Statuses) > 0 {
			outErr = append(outErr, errors.New("statuses cannot be set for standard invoices"))
		}

		if len(i.ExtendedStatuses) > 0 {
			outErr = append(outErr, errors.New("extendedStatuses cannot be set for standard invoices"))
		}

		if i.ExternalIDs != nil {
			outErr = append(outErr, errors.New("externalIDs cannot be set for standard invoices"))
		}

		if i.DraftUntilLTE != nil {
			outErr = append(outErr, errors.New("draftUntilLTE cannot be set for standard invoices"))
		}
	}

	return errors.Join(outErr...)
}

type ListInvoicesResponse = pagination.Result[Invoice]

type InvoicePendingLinesInput struct {
	Customer customer.CustomerID

	IncludePendingLines mo.Option[[]string]
	AsOf                *time.Time

	// ProgressiveBillingOverride allows to override the progressive billing setting of the customer.
	// This is used to make sure that system collection does not use progressive billing.
	ProgressiveBillingOverride *bool
}

func (i InvoicePendingLinesInput) Validate() error {
	if err := i.Customer.Validate(); err != nil {
		return fmt.Errorf("customer: %w", err)
	}

	if i.AsOf != nil && i.AsOf.After(clock.Now()) {
		return errors.New("asOf must be in the past")
	}

	if i.IncludePendingLines.IsPresent() {
		if len(i.IncludePendingLines.OrEmpty()) == 0 {
			return errors.New("includePendingLines must contain at least one line ID")
		}
	}

	return nil
}

type UpdateInvoiceInput struct {
	Invoice InvoiceID
	EditFn  func(Invoice) (Invoice, error)
	// IncludeDeletedLines signals the update to populate the deleted lines into the lines field, for the edit function
	IncludeDeletedLines bool
}

func (i UpdateInvoiceInput) Validate() error {
	var outErr []error

	if err := i.Invoice.Validate(); err != nil {
		outErr = append(outErr, fmt.Errorf("id: %w", err))
	}

	if i.EditFn == nil {
		outErr = append(outErr, errors.New("edit function is required"))
	}

	return errors.Join(outErr...)
}

type GetInvoiceTypeAdapterInput = InvoiceID
