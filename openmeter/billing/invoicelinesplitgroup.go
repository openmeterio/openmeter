package billing

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	timeutil "github.com/openmeterio/openmeter/pkg/timeutil"
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

type GatheringLineWithInvoiceHeader struct {
	Line    GatheringLine
	Invoice GatheringInvoice
}

type StandardLineWithInvoiceHeader struct {
	Line    *StandardLine
	Invoice StandardInvoice
}

type LineWithInvoiceHeader struct {
	Line    GenericInvoiceLine
	Invoice GenericInvoiceReader
}

type lineWithInvoiceHeaderSerde[IT StandardInvoice | GatheringInvoice, LT StandardLine | GatheringLine] struct {
	Type InvoiceType `json:"type"`

	Line    LT `json:"line"`
	Invoice IT `json:"invoice"`
}

func (l *LineWithInvoiceHeader) UnmarshalJSON(data []byte) error {
	var genericSerde struct {
		Type InvoiceType `json:"type"`
	}

	if err := json.Unmarshal(data, &genericSerde); err != nil {
		return err
	}

	switch genericSerde.Type {
	case InvoiceTypeStandard:
		unmarshalled := lineWithInvoiceHeaderSerde[StandardInvoice, StandardLine]{}
		if err := json.Unmarshal(data, &unmarshalled); err != nil {
			return err
		}

		l.Line = &standardInvoiceLineGenericWrapper{StandardLine: &unmarshalled.Line}
		l.Invoice = &unmarshalled.Invoice

		return nil
	case InvoiceTypeGathering:
		unmarshalled := lineWithInvoiceHeaderSerde[GatheringInvoice, GatheringLine]{}
		if err := json.Unmarshal(data, &unmarshalled); err != nil {
			return err
		}

		l.Line = &gatheringInvoiceLineGenericWrapper{GatheringLine: unmarshalled.Line}
		l.Invoice = &unmarshalled.Invoice

		return nil
	default:
		return fmt.Errorf("unknown invoice type: %s", genericSerde.Type)
	}
}

func (l LineWithInvoiceHeader) MarshalJSON() ([]byte, error) {
	invoice := l.Invoice.AsInvoice()

	invoiceType := invoice.Type()

	switch invoiceType {
	case InvoiceTypeStandard:
		stdInvoice, err := invoice.AsStandardInvoice()
		if err != nil {
			return nil, err
		}

		stdLine, err := l.Line.AsInvoiceLine().AsStandardLine()
		if err != nil {
			return nil, err
		}

		return json.Marshal(lineWithInvoiceHeaderSerde[StandardInvoice, StandardLine]{
			Type:    InvoiceTypeStandard,
			Line:    stdLine,
			Invoice: stdInvoice,
		})
	case InvoiceTypeGathering:
		gatheringInvoice, err := invoice.AsGatheringInvoice()
		if err != nil {
			return nil, err
		}

		gatheringLine, err := l.Line.AsInvoiceLine().AsGatheringLine()
		if err != nil {
			return nil, err
		}

		return json.Marshal(lineWithInvoiceHeaderSerde[GatheringInvoice, GatheringLine]{
			Type:    InvoiceTypeGathering,
			Line:    gatheringLine,
			Invoice: gatheringInvoice,
		})
	}

	return nil, fmt.Errorf("unknown invoice type: %s", invoiceType)
}

func NewLineWithInvoiceHeader[T StandardLineWithInvoiceHeader | GatheringLineWithInvoiceHeader](line T) LineWithInvoiceHeader {
	switch v := any(line).(type) {
	case StandardLineWithInvoiceHeader:
		return LineWithInvoiceHeader{Line: &standardInvoiceLineGenericWrapper{StandardLine: v.Line}, Invoice: v.Invoice}
	case GatheringLineWithInvoiceHeader:
		return LineWithInvoiceHeader{Line: &gatheringInvoiceLineGenericWrapper{GatheringLine: v.Line}, Invoice: v.Invoice}
	}

	return LineWithInvoiceHeader{}
}

type LinesWithInvoiceHeaders []LineWithInvoiceHeader

func (i LinesWithInvoiceHeaders) Lines() []GenericInvoiceLine {
	return lo.Map(i, func(line LineWithInvoiceHeader, _ int) GenericInvoiceLine {
		return line.Line
	})
}

type SplitLineHierarchy struct {
	Group SplitLineGroup
	Lines LinesWithInvoiceHeaders
}

func (h *SplitLineHierarchy) Clone() (SplitLineHierarchy, error) {
	lines, err := slicesx.MapWithErr(h.Lines, func(line LineWithInvoiceHeader) (LineWithInvoiceHeader, error) {
		clonedLine, err := line.Line.Clone()
		if err != nil {
			return LineWithInvoiceHeader{}, err
		}

		// TODO: We might want to clone the invoice too, but that data is mostly read-only, so it's fine for now.
		return LineWithInvoiceHeader{Line: clonedLine, Invoice: line.Invoice}, nil
	})
	if err != nil {
		return SplitLineHierarchy{}, err
	}

	return SplitLineHierarchy{
		Group: h.Group.Clone(),
		Lines: lines,
	}, nil
}

type SumNetAmountInput struct {
	PeriodEndLTE   time.Time
	IncludeCharges bool
}

// SumNetAmount returns the sum of the net amount (pre-tax) of the progressive billed line and its children
// containing the values for all lines whose period's end is <= in.UpTo and are not deleted or not part of
// an invoice that has been deleted.
// As gathering lines do not represent any kind of actual charge, they are not included in the sum.
func (h *SplitLineHierarchy) SumNetAmount(in SumNetAmountInput) (alpacadecimal.Decimal, error) {
	netAmount := alpacadecimal.Zero

	err := h.ForEachChild(ForEachChildInput{
		PeriodEndLTE: in.PeriodEndLTE,
		Callback: func(child LineWithInvoiceHeader) error {
			if child.Invoice.AsInvoice().Type() != InvoiceTypeStandard {
				return nil
			}

			stdLine, err := child.Line.AsInvoiceLine().AsStandardLine()
			if err != nil {
				return err
			}

			netAmount = netAmount.Add(stdLine.Totals.Amount)

			if in.IncludeCharges {
				netAmount = netAmount.Add(stdLine.Totals.ChargesTotal)
			}

			return nil
		},
	})
	if err != nil {
		return alpacadecimal.Zero, err
	}

	return netAmount, nil
}

type ForEachChildInput struct {
	PeriodEndLTE time.Time
	Callback     func(child LineWithInvoiceHeader) error
}

func (h *SplitLineHierarchy) ForEachChild(in ForEachChildInput) error {
	for _, child := range h.Lines {
		line := child.Line

		// The line is not in scope
		if !in.PeriodEndLTE.IsZero() && line.GetServicePeriod().To.After(in.PeriodEndLTE) {
			continue
		}

		// The line is deleted
		if line.GetDeletedAt() != nil {
			continue
		}

		// The invoice is deleted
		if child.Invoice.GetDeletedAt() != nil {
			continue
		}

		invoice := child.Invoice.AsInvoice()
		if invoice.Type() == InvoiceTypeStandard {
			stdInvoice, err := invoice.AsStandardInvoice()
			if err != nil {
				return err
			}

			if stdInvoice.Status == StandardInvoiceStatusDeleted {
				continue
			}
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
	line               GenericInvoiceLine
	splitLineHierarchy *SplitLineHierarchy
}

func NewLineOrHierarchy[T *StandardLine | GatheringLine | *SplitLineHierarchy](line T) LineOrHierarchy {
	switch v := any(line).(type) {
	case *StandardLine:
		return LineOrHierarchy{t: LineOrHierarchyTypeLine, line: standardInvoiceLineGenericWrapper{StandardLine: v}}
	case GatheringLine:
		return LineOrHierarchy{t: LineOrHierarchyTypeLine, line: &gatheringInvoiceLineGenericWrapper{GatheringLine: v}}
	case *SplitLineHierarchy:
		return LineOrHierarchy{t: LineOrHierarchyTypeHierarchy, splitLineHierarchy: v}
	}

	return LineOrHierarchy{}
}

func (i LineOrHierarchy) Type() LineOrHierarchyType {
	return i.t
}

func (i LineOrHierarchy) AsGenericLine() (GenericInvoiceLine, error) {
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
		return i.line.GetChildUniqueReferenceID()
	case LineOrHierarchyTypeHierarchy:
		return i.splitLineHierarchy.Group.UniqueReferenceID
	}

	return nil
}

func (i LineOrHierarchy) ServicePeriod() timeutil.ClosedPeriod {
	switch i.t {
	case LineOrHierarchyTypeLine:
		return i.line.GetServicePeriod()
	case LineOrHierarchyTypeHierarchy:
		return i.splitLineHierarchy.Group.ServicePeriod.ToClosedPeriod()
	}

	return timeutil.ClosedPeriod{}
}

type GetSplitLineGroupHeadersInput struct {
	Namespace         string
	SplitLineGroupIDs []string
}

type SplitLineGroupHeaders = []SplitLineGroup

func (i GetSplitLineGroupHeadersInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	return errors.Join(errs...)
}
