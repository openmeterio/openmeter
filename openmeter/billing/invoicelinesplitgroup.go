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

	ServicePeriod timeutil.ClosedPeriod `json:"period"`

	RatecardDiscounts Discounts `json:"ratecardDiscounts"`
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

	return errors.Join(errs...)
}

func (i SplitLineGroupMutableFields) Clone() SplitLineGroupMutableFields {
	clone := i
	clone.Metadata = i.Metadata.Clone()
	if i.Description != nil {
		clone.Description = lo.ToPtr(*i.Description)
	}

	clone.RatecardDiscounts = i.RatecardDiscounts.Clone()

	return clone
}

type SplitLineGroupCreate struct {
	Namespace string `json:"namespace"`

	SplitLineGroupMutableFields `json:",inline"`

	Price             *productcatalog.Price  `json:"price"`
	FeatureKey        *string                `json:"featureKey,omitempty"`
	Subscription      *SubscriptionReference `json:"subscription,omitempty"`
	Currency          currencyx.FiatCode     `json:"currency"`
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
	Currency          currencyx.FiatCode     `json:"currency"`
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
	clone := i
	clone.SplitLineGroupMutableFields = i.SplitLineGroupMutableFields.Clone()
	if i.DeletedAt != nil {
		clone.DeletedAt = lo.ToPtr(*i.DeletedAt)
	}

	if i.Price != nil {
		clone.Price = i.Price.Clone()
	}

	if i.FeatureKey != nil {
		clone.FeatureKey = lo.ToPtr(*i.FeatureKey)
	}

	if i.Subscription != nil {
		clone.Subscription = i.Subscription.Clone()
	}

	if i.UniqueReferenceID != nil {
		clone.UniqueReferenceID = lo.ToPtr(*i.UniqueReferenceID)
	}

	return clone
}

type GatheringLineWithInvoiceHeader struct {
	Line    GatheringLine
	Invoice GatheringInvoice
}

func (l GatheringLineWithInvoiceHeader) Validate() error {
	if err := l.Line.Validate(); err != nil {
		return fmt.Errorf("line: %w", err)
	}

	// This transport only carries the parent invoice entity as a header object,
	// so we validate invoice identity here instead of requiring a fully fetched invoice aggregate.
	if l.Invoice.ID == "" {
		return fmt.Errorf("invoice id is required")
	}

	return nil
}

type StandardLineWithInvoiceHeader struct {
	Line    *StandardLine
	Invoice StandardInvoice
}

func (l StandardLineWithInvoiceHeader) Validate() error {
	if l.Line == nil {
		return fmt.Errorf("line is required")
	}

	if err := l.Line.Validate(); err != nil {
		return fmt.Errorf("line: %w", err)
	}

	// This transport only carries the parent invoice entity as a header object,
	// so we validate invoice identity here instead of requiring a fully fetched invoice aggregate.
	if l.Invoice.ID == "" {
		return fmt.Errorf("invoice id is required")
	}

	return nil
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

// SplitLineHierarchyInvoice carries the identity and deletion facts needed to
// interpret a child's billing history without loading its invoice aggregate.
type SplitLineHierarchyInvoice struct {
	InvoiceID `json:",inline"`
	Type      InvoiceType           `json:"type"`
	DeletedAt *time.Time            `json:"deletedAt,omitempty"`
	Status    StandardInvoiceStatus `json:"status"`
}

// SplitLineHierarchyLine is the persisted history used by legacy rating and
// subscription reconciliation. Updates describe intent; the invoice updater
// loads the canonical line before applying them.
type SplitLineHierarchyLine struct {
	LineID        `json:",inline"`
	Invoice       SplitLineHierarchyInvoice `json:"invoice"`
	DeletedAt     *time.Time                `json:"deletedAt,omitempty"`
	ServicePeriod timeutil.ClosedPeriod     `json:"period"`
	ManagedBy     InvoiceLineManagedBy      `json:"managedBy"`
	Annotations   models.Annotations        `json:"annotations,omitempty"`
	Subscription  *SubscriptionReference    `json:"subscription,omitempty"`
	InvoiceAt     time.Time                 `json:"invoiceAt"`
	Amount        alpacadecimal.Decimal     `json:"amount"`
	ChargesTotal  alpacadecimal.Decimal     `json:"chargesTotal"`
}

func (l SplitLineHierarchyLine) Clone() (SplitLineHierarchyLine, error) {
	clone := l
	annotations, err := l.Annotations.Clone()
	if err != nil {
		return SplitLineHierarchyLine{}, fmt.Errorf("cloning annotations: %w", err)
	}

	clone.Annotations = annotations
	if l.Subscription != nil {
		clone.Subscription = l.Subscription.Clone()
	}

	if l.DeletedAt != nil {
		clone.DeletedAt = lo.ToPtr(*l.DeletedAt)
	}

	if l.Invoice.DeletedAt != nil {
		clone.Invoice.DeletedAt = lo.ToPtr(*l.Invoice.DeletedAt)
	}

	return clone, nil
}

type SplitLineHierarchy struct {
	Group SplitLineGroup
	// Preserve database traversal order, including historical deleted children.
	Lines []SplitLineHierarchyLine
}

func (h *SplitLineHierarchy) Clone() (SplitLineHierarchy, error) {
	lines, err := slicesx.MapWithErr(h.Lines, SplitLineHierarchyLine.Clone)
	if err != nil {
		return SplitLineHierarchy{}, fmt.Errorf("cloning hierarchy lines: %w", err)
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

// SumNetAmount sums prior standard-line amounts through PeriodEndLTE, excluding
// deleted lines and invoices. A zero cutoff includes all periods. Gathering
// children do not represent billed amounts and never contribute to the sum.
func (h *SplitLineHierarchy) SumNetAmount(in SumNetAmountInput) (alpacadecimal.Decimal, error) {
	netAmount := alpacadecimal.Zero

	err := h.ForEachChild(ForEachChildInput{
		PeriodEndLTE: in.PeriodEndLTE,
		Callback: func(child SplitLineHierarchyLine) error {
			if child.Invoice.Type != InvoiceTypeStandard {
				return nil
			}

			netAmount = netAmount.Add(child.Amount)

			if in.IncludeCharges {
				netAmount = netAmount.Add(child.ChargesTotal)
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
	Callback     func(child SplitLineHierarchyLine) error
}

func (h *SplitLineHierarchy) ForEachChild(in ForEachChildInput) error {
	for _, child := range h.Lines {
		// The line is not in scope
		if !in.PeriodEndLTE.IsZero() && child.ServicePeriod.To.After(in.PeriodEndLTE) {
			continue
		}

		// The line is deleted
		if child.DeletedAt != nil {
			continue
		}

		// The invoice is deleted
		if child.Invoice.DeletedAt != nil {
			continue
		}

		if child.Invoice.Type == InvoiceTypeStandard && child.Invoice.Status == StandardInvoiceStatusDeleted {
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
