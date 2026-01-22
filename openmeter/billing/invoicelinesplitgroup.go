package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SplitLineGroupMutableFields struct {
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	Metadata    models.Metadata `json:"metadata,omitempty"`

	ServicePeriod Period `json:"period"`

	RatecardDiscounts Discounts                 `json:"ratecardDiscounts"`
	TaxConfig         *productcatalog.TaxConfig `json:"taxConfig,omitempty"`
}

func (i SplitLineGroupMutableFields) ValidateForPrice(price *productcatalog.Price) error {
	var errs []error

	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.RatecardDiscounts.ValidateForPrice(price); err != nil {
		errs = append(errs, err)
	}

	if i.TaxConfig != nil {
		if err := i.TaxConfig.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (i SplitLineGroupMutableFields) Clone() SplitLineGroupMutableFields {
	clone := i
	clone.RatecardDiscounts = i.RatecardDiscounts.Clone()

	if i.TaxConfig != nil {
		clone.TaxConfig = lo.ToPtr(i.TaxConfig.Clone())
	}

	return clone
}

type SplitLineGroupCreate struct {
	Namespace string `json:"namespace"`

	SplitLineGroupMutableFields `json:",inline"`

	Price             *productcatalog.Price  `json:"price"`
	FeatureKey        *string                `json:"featureKey,omitempty"`
	Subscription      *SubscriptionReference `json:"subscription,omitempty"`
	Currency          currencyx.Code         `json:"currency"`
	UniqueReferenceID *string                `json:"childUniqueReferenceId,omitempty"`
}

func (i SplitLineGroupCreate) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.SplitLineGroupMutableFields.ValidateForPrice(i.Price); err != nil {
		errs = append(errs, err)
	}

	if i.Price == nil {
		errs = append(errs, errors.New("price is required"))
	} else {
		if err := i.Price.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if i.Subscription != nil {
		if err := i.Subscription.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if i.Currency == "" {
		errs = append(errs, errors.New("currency is required"))
	}

	if i.UniqueReferenceID != nil && *i.UniqueReferenceID == "" {
		errs = append(errs, errors.New("unique reference id is required"))
	}

	return errors.Join(errs...)
}

type SplitLineGroupUpdate struct {
	models.NamespacedID `json:",inline"`

	SplitLineGroupMutableFields `json:",inline"`
}

func (i SplitLineGroupUpdate) ValidateWithPrice(price *productcatalog.Price) error {
	var errs []error

	if err := i.SplitLineGroupMutableFields.ValidateForPrice(price); err != nil {
		errs = append(errs, err)
	}

	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

type SplitLineGroup struct {
	models.ManagedModel         `json:",inline"`
	models.NamespacedID         `json:",inline"`
	SplitLineGroupMutableFields `json:",inline"`

	Price             *productcatalog.Price  `json:"price"`
	FeatureKey        *string                `json:"featureKey,omitempty"`
	Subscription      *SubscriptionReference `json:"subscription,omitempty"`
	Currency          currencyx.Code         `json:"currency"`
	UniqueReferenceID *string                `json:"childUniqueReferenceId,omitempty"`
}

func (i SplitLineGroup) Validate() error {
	var errs []error

	if err := i.SplitLineGroupMutableFields.ValidateForPrice(i.Price); err != nil {
		errs = append(errs, err)
	}

	if i.Price != nil {
		if err := i.Price.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if i.Currency == "" {
		errs = append(errs, errors.New("currency is required"))
	}

	return errors.Join(errs...)
}

func (i SplitLineGroup) ToUpdate() SplitLineGroupUpdate {
	return SplitLineGroupUpdate{
		NamespacedID:                i.NamespacedID,
		SplitLineGroupMutableFields: i.SplitLineGroupMutableFields.Clone(),
	}
}

func (i SplitLineGroup) Clone() SplitLineGroup {
	return SplitLineGroup{
		ManagedModel:                i.ManagedModel,
		SplitLineGroupMutableFields: i.SplitLineGroupMutableFields.Clone(),
		Price:                       i.Price,
		FeatureKey:                  i.FeatureKey,
		Subscription:                i.Subscription,
		Currency:                    i.Currency,
		UniqueReferenceID:           i.UniqueReferenceID,
	}
}

type LineWithInvoiceHeader struct {
	Line    *StandardLine
	Invoice StandardInvoiceBase
}

func (i LineWithInvoiceHeader) Clone() LineWithInvoiceHeader {
	return LineWithInvoiceHeader{
		Line:    i.Line.Clone(),
		Invoice: i.Invoice,
	}
}

type SplitLineHierarchy struct {
	Group SplitLineGroup
	Lines []LineWithInvoiceHeader
}

func (h *SplitLineHierarchy) Clone() SplitLineHierarchy {
	return SplitLineHierarchy{
		Group: h.Group.Clone(),
		Lines: lo.Map(h.Lines, func(line LineWithInvoiceHeader, _ int) LineWithInvoiceHeader {
			return line.Clone()
		}),
	}
}

type SumNetAmountInput struct {
	PeriodEndLTE   time.Time
	IncludeCharges bool
}

// SumNetAmount returns the sum of the net amount (pre-tax) of the progressive billed line and its children
// containing the values for all lines whose period's end is <= in.UpTo and are not deleted or not part of
// an invoice that has been deleted.
func (h *SplitLineHierarchy) SumNetAmount(in SumNetAmountInput) alpacadecimal.Decimal {
	netAmount := alpacadecimal.Zero

	_ = h.ForEachChild(ForEachChildInput{
		PeriodEndLTE: in.PeriodEndLTE,
		Callback: func(child LineWithInvoiceHeader) error {
			netAmount = netAmount.Add(child.Line.Totals.Amount)

			if in.IncludeCharges {
				netAmount = netAmount.Add(child.Line.Totals.ChargesTotal)
			}

			return nil
		},
	})

	return netAmount
}

type ForEachChildInput struct {
	PeriodEndLTE time.Time
	Callback     func(child LineWithInvoiceHeader) error
}

func (h *SplitLineHierarchy) ForEachChild(in ForEachChildInput) error {
	for _, child := range h.Lines {
		// The line is not in scope
		if !in.PeriodEndLTE.IsZero() && child.Line.Period.End.After(in.PeriodEndLTE) {
			continue
		}

		// The line is deleted
		if child.Line.DeletedAt != nil {
			continue
		}

		// The invoice is deleted
		if child.Invoice.DeletedAt != nil || child.Invoice.Status == StandardInvoiceStatusDeleted {
			continue
		}

		if err := in.Callback(child); err != nil {
			return err
		}
	}

	return nil
}

// Adapter
type (
	CreateSplitLineGroupAdapterInput = SplitLineGroupCreate
	UpdateSplitLineGroupInput        = SplitLineGroupUpdate
	DeleteSplitLineGroupInput        = models.NamespacedID
	GetSplitLineGroupInput           = models.NamespacedID
)

// LineOrHierarchy is a wrapper around a line or a split line hierarchy

type LineOrHierarchyType string

const (
	LineOrHierarchyTypeLine      LineOrHierarchyType = "line"
	LineOrHierarchyTypeHierarchy LineOrHierarchyType = "hierarchy"
)

type LineOrHierarchy struct {
	t                  LineOrHierarchyType
	line               *StandardLine
	splitLineHierarchy *SplitLineHierarchy
}

func NewLineOrHierarchy[T StandardLine | SplitLineHierarchy](line *T) LineOrHierarchy {
	switch v := any(line).(type) {
	case *StandardLine:
		return LineOrHierarchy{t: LineOrHierarchyTypeLine, line: v}
	case *SplitLineHierarchy:
		return LineOrHierarchy{t: LineOrHierarchyTypeHierarchy, splitLineHierarchy: v}
	}

	return LineOrHierarchy{}
}

func (i LineOrHierarchy) Type() LineOrHierarchyType {
	return i.t
}

func (i LineOrHierarchy) AsStandardLine() (*StandardLine, error) {
	if i.t != LineOrHierarchyTypeLine {
		return nil, fmt.Errorf("line or hierarchy is not a line")
	}

	if i.line == nil {
		return nil, fmt.Errorf("line is nil")
	}

	return i.line, nil
}

func (i LineOrHierarchy) AsHierarchy() (*SplitLineHierarchy, error) {
	if i.t != LineOrHierarchyTypeHierarchy {
		return nil, fmt.Errorf("line or hierarchy is not a hierarchy")
	}

	if i.splitLineHierarchy == nil {
		return nil, fmt.Errorf("split line hierarchy is nil")
	}

	return i.splitLineHierarchy, nil
}

func (i LineOrHierarchy) ChildUniqueReferenceID() *string {
	switch i.t {
	case LineOrHierarchyTypeLine:
		return i.line.ChildUniqueReferenceID
	case LineOrHierarchyTypeHierarchy:
		return i.splitLineHierarchy.Group.UniqueReferenceID
	}

	return nil
}

func (i LineOrHierarchy) ServicePeriod() Period {
	switch i.t {
	case LineOrHierarchyTypeLine:
		return i.line.Period
	case LineOrHierarchyTypeHierarchy:
		return i.splitLineHierarchy.Group.ServicePeriod
	}

	return Period{}
}
