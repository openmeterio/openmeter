package billing

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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

type InvoiceExpand struct {
	Preceding bool

	Lines        bool
	DeletedLines bool

	// RecalculateGatheringInvoice is used to calculate the totals and status details of the invoice when gathering,
	// this is temporary until we implement the full progressive billing stack, including gathering invoice recalculations.
	RecalculateGatheringInvoice bool
}

var InvoiceExpandAll = InvoiceExpand{
	Preceding:    true,
	Lines:        true,
	DeletedLines: false,
}

func (e InvoiceExpand) Validate() error {
	return nil
}

func (e InvoiceExpand) SetLines(v bool) InvoiceExpand {
	e.Lines = v
	return e
}

func (e InvoiceExpand) SetDeletedLines(v bool) InvoiceExpand {
	e.DeletedLines = v
	return e
}

func (e InvoiceExpand) SetRecalculateGatheringInvoice(v bool) InvoiceExpand {
	e.RecalculateGatheringInvoice = v
	return e
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
	IDs        []string
	Customers  []string

	// StandardInvoiceStatuses searches by short StandardInvoiceStatus (e.g. draft, issued)
	StandardInvoiceStatuses []string
	// StandardInvoiceExtendedStatuses searches by exact StandardInvoiceStatus
	StandardInvoiceExtendedStatuses []StandardInvoiceStatus
	InvoiceTypes                    []InvoiceType

	HasAvailableAction []InvoiceAvailableActionsFilter

	Currencies []currencyx.Code

	IssuedAfter  *time.Time
	IssuedBefore *time.Time

	PeriodStartAfter  *time.Time
	PeriodStartBefore *time.Time

	// Filter by invoice creation time
	CreatedAfter  *time.Time
	CreatedBefore *time.Time

	IncludeDeleted bool

	// DraftUtil allows to filter invoices which have their draft state expired based on the provided time.
	// Invoice is expired if the time defined by Invoice.DraftUntil is in the past compared to ListInvoicesInput.DraftUntil.
	DraftUntil *time.Time

	// CollectionAt allows to filter invoices which have their collection_at attribute is in the past compared
	// to the time provided in CollectionAt parameter.
	CollectionAt *time.Time

	Expand InvoiceExpand

	ExternalIDs *ListInvoicesExternalIDFilter

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

	if i.ExternalIDs != nil {
		if err := i.ExternalIDs.Validate(); err != nil {
			outErr = append(outErr, fmt.Errorf("external IDs: %w", err))
		}
	}

	if len(i.InvoiceTypes) > 0 {
		errs := errors.Join(
			lo.Map(i.InvoiceTypes, func(invoiceType InvoiceType, _ int) error {
				return invoiceType.Validate()
			})...,
		)
		if errs != nil {
			outErr = append(outErr, errs)
		}
	}

	willListStandardInvoices := len(i.InvoiceTypes) == 0 || slices.Contains(i.InvoiceTypes, InvoiceTypeStandard)
	if !willListStandardInvoices {
		if len(i.StandardInvoiceStatuses) > 0 {
			outErr = append(outErr, errors.New("standard invoice statuses are not supported when listing non-standard invoices"))
		}
		if len(i.StandardInvoiceExtendedStatuses) > 0 {
			outErr = append(outErr, errors.New("standard invoice extended statuses are not supported when listing non-standard invoices"))
		}
	}

	if len(i.HasAvailableAction) > 0 {
		errs := errors.Join(
			lo.Map(i.HasAvailableAction, func(action InvoiceAvailableActionsFilter, _ int) error {
				return action.Validate()
			})...,
		)
		if errs != nil {
			outErr = append(outErr, errs)
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
