package billing

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	timeutil "github.com/openmeterio/openmeter/pkg/timeutil"
)

type LineID models.NamespacedID

func (i LineID) Validate() error {
	return models.NamespacedID(i).Validate()
}

type InvoiceLineManagedBy string

const (
	// SubscriptionManagedLine is a line that is managed by a subscription.
	SubscriptionManagedLine InvoiceLineManagedBy = "subscription"
	// SystemManagedLine is a line that is managed by the system (non editable, detailed lines)
	SystemManagedLine InvoiceLineManagedBy = "system"
	// ManuallyManagedLine is a line that is managed manually (e.g. overridden by our API users)
	ManuallyManagedLine InvoiceLineManagedBy = "manual"
)

func (InvoiceLineManagedBy) Values() []string {
	return []string{
		string(SubscriptionManagedLine),
		string(SystemManagedLine),
		string(ManuallyManagedLine),
	}
}

// Period represents a time period, in billing the time period is always interpreted as
// [from, to) (i.e. from is inclusive, to is exclusive).
// TODO: Lets merge this with recurrence.Period
type Period struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func (p Period) Validate() error {
	if p.Start.IsZero() {
		return errors.New("start is required")
	}

	if p.End.IsZero() {
		return errors.New("end is required")
	}

	if p.Start.After(p.End) {
		return errors.New("start must be before end")
	}

	return nil
}

func (p Period) Truncate(resolution time.Duration) Period {
	return Period{
		Start: p.Start.Truncate(resolution),
		End:   p.End.Truncate(resolution),
	}
}

func (p Period) Equal(other Period) bool {
	return p.Start.Equal(other.Start) && p.End.Equal(other.End)
}

func (p Period) IsEmpty() bool {
	return !p.End.After(p.Start)
}

func (p Period) Contains(t time.Time) bool {
	return t.After(p.Start) && t.Before(p.End)
}

func (p Period) Duration() time.Duration {
	return p.End.Sub(p.Start)
}

func (p Period) ToClosedPeriod() timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: p.Start,
		To:   p.End,
	}
}

type GetLinesForSubscriptionInput struct {
	Namespace      string
	SubscriptionID string
	CustomerID     string
}

func (i GetLinesForSubscriptionInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.SubscriptionID == "" {
		return errors.New("subscription id is required")
	}

	if i.CustomerID == "" {
		return errors.New("customer id is required")
	}

	return nil
}

type GenericInvoiceLine interface {
	GenericInvoiceLineReader

	Clone() (GenericInvoiceLine, error)
	CloneWithoutChildren() (GenericInvoiceLine, error)

	SetDeletedAt(at *time.Time)
	SetPrice(price productcatalog.Price)
	UpdateServicePeriod(func(p *timeutil.ClosedPeriod))
	SetChildUniqueReferenceID(id *string)
}

// GenericInvoiceLineReader is an interface that provides access to the generic invoice fields.
type GenericInvoiceLineReader interface {
	GetDeletedAt() *time.Time
	GetID() string
	GetLineID() LineID
	GetManagedBy() InvoiceLineManagedBy
	GetAnnotations() models.Annotations
	GetInvoiceID() string
	GetPrice() *productcatalog.Price
	GetServicePeriod() timeutil.ClosedPeriod
	GetChildUniqueReferenceID() *string
	GetFeatureKey() string

	Validate() error
	AsInvoiceLine() InvoiceLine
	GetRateCardDiscounts() Discounts
	GetSubscriptionReference() *SubscriptionReference
	GetSplitLineGroupID() *string
}

type InvoiceAtAccessor interface {
	GetInvoiceAt() time.Time
	SetInvoiceAt(at time.Time)
}

type QuantityAccessor interface {
	GetQuantity() *alpacadecimal.Decimal
}

type InvoiceLineType string

const (
	InvoiceLineTypeStandard  InvoiceLineType = "standard"
	InvoiceLineTypeGathering InvoiceLineType = "gathering"
)

var InvoiceLineTypes = []InvoiceLineType{
	InvoiceLineTypeStandard,
	InvoiceLineTypeGathering,
}

func (t InvoiceLineType) Validate() error {
	if !slices.Contains(InvoiceLineTypes, t) {
		return fmt.Errorf("invalid invoice line type: %s", t)
	}

	return nil
}

type InvoiceLine struct {
	t             InvoiceLineType
	standardLine  *StandardLine
	gatheringLine *GatheringLine
}

func (i InvoiceLine) Validate() error {
	switch i.t {
	case InvoiceLineTypeStandard:
		if i.standardLine == nil {
			return fmt.Errorf("standard line is nil")
		}

		return i.standardLine.Validate()
	case InvoiceLineTypeGathering:
		if i.gatheringLine == nil {
			return fmt.Errorf("gathering line is nil")
		}

		return i.gatheringLine.Validate()
	default:
		return fmt.Errorf("invalid invoice line type: %s", i.t)
	}
}

func (i InvoiceLine) Type() InvoiceLineType {
	return i.t
}

func (i InvoiceLine) AsStandardLine() (StandardLine, error) {
	if i.t != InvoiceLineTypeStandard {
		return StandardLine{}, fmt.Errorf("line is not a standard line")
	}

	if i.standardLine == nil {
		return StandardLine{}, fmt.Errorf("standard line is nil")
	}

	return *i.standardLine, nil
}

func (i InvoiceLine) AsGatheringLine() (GatheringLine, error) {
	if i.t != InvoiceLineTypeGathering {
		return GatheringLine{}, fmt.Errorf("line is not a gathering line")
	}

	if i.gatheringLine == nil {
		return GatheringLine{}, fmt.Errorf("gathering line is nil")
	}

	return *i.gatheringLine, nil
}

func (i InvoiceLine) AsGenericLine() (GenericInvoiceLine, error) {
	switch i.t {
	case InvoiceLineTypeStandard:
		if i.standardLine == nil {
			return nil, fmt.Errorf("standard line is nil")
		}

		return &standardInvoiceLineGenericWrapper{StandardLine: i.standardLine}, nil
	case InvoiceLineTypeGathering:
		if i.gatheringLine == nil {
			return nil, fmt.Errorf("gathering line is nil")
		}

		return &gatheringInvoiceLineGenericWrapper{GatheringLine: *i.gatheringLine}, nil
	}

	return nil, fmt.Errorf("invalid invoice line type: %s", i.t)
}
