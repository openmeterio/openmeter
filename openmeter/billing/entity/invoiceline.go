package billingentity

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type InvoiceLineType string

const (
	// InvoiceLineTypeFee is an item that represents a single charge without meter backing.
	InvoiceLineTypeFee InvoiceLineType = "flat_fee"
	// InvoiceLineTypeUsageBased is an item that is added to the invoice and is usage based.
	InvoiceLineTypeUsageBased InvoiceLineType = "usage_based"
)

func (InvoiceLineType) Values() []string {
	return []string{
		string(InvoiceLineTypeFee),
		string(InvoiceLineTypeUsageBased),
	}
}

type InvoiceLineStatus string

const (
	// InvoiceLineStatusValid is a valid invoice line.
	InvoiceLineStatusValid InvoiceLineStatus = "valid"
	// InvoiceLineStatusSplit is a split invoice line (the child lines will have this set as parent).
	InvoiceLineStatusSplit InvoiceLineStatus = "split"
	// InvoiceLineStatusDetailed is a detailed invoice line.
	InvoiceLineStatusDetailed InvoiceLineStatus = "detailed"
)

func (InvoiceLineStatus) Values() []string {
	return []string{
		string(InvoiceLineStatusValid),
		string(InvoiceLineStatusSplit),
		string(InvoiceLineStatusDetailed),
	}
}

// Period represents a time period, in billing the time period is always interpreted as
// [start, end) (i.e. start is inclusive, end is exclusive).
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

// LineBase represents the common fields for an invoice item.
type LineBase struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	Metadata    map[string]string `json:"metadata"`
	Name        string            `json:"name"`
	Type        InvoiceLineType   `json:"type"`
	Description *string           `json:"description,omitempty"`

	InvoiceID string         `json:"invoiceID,omitempty"`
	Currency  currencyx.Code `json:"currency"`

	// Lifecycle
	Period    Period    `json:"period"`
	InvoiceAt time.Time `json:"invoiceAt"`

	// Relationships
	ParentLineID *string `json:"parentLine,omitempty"`

	Status                 InvoiceLineStatus `json:"status"`
	ChildUniqueReferenceID *string           `json:"childUniqueReferenceID,omitempty"`

	TaxConfig *TaxConfig `json:"taxOverrides,omitempty"`

	Total alpacadecimal.Decimal `json:"total"`
}

func (i LineBase) Equal(other LineBase) bool {
	return reflect.DeepEqual(i, other)
}

func (i LineBase) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if err := i.Period.Validate(); err != nil {
		return fmt.Errorf("period: %w", err)
	}

	if i.InvoiceAt.IsZero() {
		return errors.New("invoice at is required")
	}

	if i.InvoiceAt.Before(i.Period.Start) {
		return errors.New("invoice at must be after period start")
	}

	if i.Name == "" {
		return errors.New("name is required")
	}

	if i.Type == "" {
		return errors.New("type is required")
	}

	if err := i.Currency.Validate(); err != nil {
		return errors.New("currency is required")
	}

	return nil
}

func (i LineBase) Clone(line *Line) LineBase {
	out := i

	// Clone pointer fields (where they are mutable)
	if i.Metadata != nil {
		out.Metadata = make(map[string]string, len(i.Metadata))
		for k, v := range i.Metadata {
			out.Metadata[k] = v
		}
	}

	if i.TaxConfig != nil {
		tc := *i.TaxConfig
		out.TaxConfig = &tc
	}

	return out
}

type FlatFeeLine struct {
	ConfigID      string                `json:"configId"`
	PerUnitAmount alpacadecimal.Decimal `json:"perUnitAmount"`
	PaymentTerm   plan.PaymentTermType  `json:"paymentTerm"`

	Quantity alpacadecimal.Decimal `json:"quantity"`
}

func (i FlatFeeLine) Clone() FlatFeeLine {
	return i
}

func (i FlatFeeLine) Equal(other FlatFeeLine) bool {
	return reflect.DeepEqual(i, other)
}

type Line struct {
	LineBase

	FlatFee    FlatFeeLine    `json:"flatFee,omitempty"`
	UsageBased UsageBasedLine `json:"usageBased,omitempty"`

	Children   LineChildren `json:"children,omitempty"`
	ParentLine *Line        `json:"parent,omitempty"`

	Discounts LineDiscounts `json:"discounts,omitempty"`

	DBState *Line
}

// CloneWithoutDependencies returns a clone of the line without any external dependencies. Could be used
// for creating a new line without any references to the parent or children (or config IDs).
func (i Line) CloneWithoutDependencies() *Line {
	clone := i.Clone()
	clone.ID = ""
	clone.ParentLineID = nil
	clone.ParentLine = nil
	clone.Children = LineChildren{}
	clone.Discounts = LineDiscounts{}
	clone.FlatFee.ConfigID = ""
	clone.UsageBased.ConfigID = ""
	clone.DBState = nil

	return clone
}

func (i Line) WithoutDBState() *Line {
	i.DBState = nil
	return &i
}

// RemoveMetaForCompare returns a copy of the invoice without the fields that are not relevant for higher level
// tests that compare invoices. What gets removed:
// - Line's DB state
// - Line's dependencies are marked as resolved
// - Parent pointers are removed
func (i Line) RemoveMetaForCompare() *Line {
	out := i.Clone()

	if !out.Discounts.IsPresent() || len(out.Discounts.Get()) == 0 {
		out.Discounts = NewLineDiscounts(nil)
	}

	if !out.Children.IsPresent() || len(out.Children.Get()) == 0 {
		out.Children = NewLineChildren(nil)
	}

	for _, child := range out.Children.Get() {
		child.ParentLine = out
		if !child.Discounts.IsPresent() || len(child.Discounts.Get()) == 0 {
			child.Discounts = NewLineDiscounts(nil)
		}

		if !child.Children.IsPresent() || len(child.Children.Get()) == 0 {
			child.Children = NewLineChildren(nil)
		}
	}

	out.ParentLine = nil
	out.DBState = nil
	return out
}

func (i Line) Clone() *Line {
	res := &Line{
		FlatFee:    i.FlatFee.Clone(),
		UsageBased: i.UsageBased.Clone(),

		// DBStates are considered immutable, so it's safe to clone
		DBState: i.DBState,
	}

	res.LineBase = i.LineBase.Clone(res)

	res.Children = i.Children.Map(func(line *Line) *Line {
		cloned := line.Clone()
		cloned.ParentLine = line
		return cloned
	})

	res.Discounts = i.Discounts.Map(func(ld LineDiscount) LineDiscount {
		return ld
	})

	return res
}

func (i *Line) SaveDBSnapshot() {
	i.DBState = i.Clone()
}

func (i Line) Validate() error {
	if err := i.LineBase.Validate(); err != nil {
		return fmt.Errorf("base: %w", err)
	}

	if i.InvoiceAt.Before(i.Period.Truncate(DefaultMeterResolution).Start) {
		return errors.New("invoice at must be after period start")
	}

	if i.Children.IsPresent() {
		if i.Status == InvoiceLineStatusDetailed {
			return errors.New("detailed lines are not allowed for detailed lines (e.g. no nesting is allowed)")
		}

		for j, detailedLine := range i.Children.Get() {
			if err := detailedLine.Validate(); err != nil {
				return fmt.Errorf("detailedLines[%d]: %w", j, err)
			}

			switch i.Status {
			case InvoiceLineStatusValid:
				if detailedLine.Status != InvoiceLineStatusDetailed {
					return fmt.Errorf("detailedLines[%d]: valid line's detailed lines must have detailed status", j)
				}

				if detailedLine.Type != InvoiceLineTypeFee {
					return fmt.Errorf("detailedLines[%d]: valid line's detailed lines must be fee typed", j)
				}
			case InvoiceLineStatusSplit:
				if detailedLine.Status != InvoiceLineStatusValid {
					return fmt.Errorf("detailedLines[%d]: split line's detailed lines must have valid status", j)
				}
			}
		}
	}

	switch i.Type {
	case InvoiceLineTypeFee:
		return i.ValidateFee()
	case InvoiceLineTypeUsageBased:
		return i.ValidateUsageBased()
	default:
		return fmt.Errorf("unsupported type: %s", i.Type)
	}
}

func (i Line) ValidateFee() error {
	if !i.FlatFee.PerUnitAmount.IsPositive() {
		return errors.New("price should be greater than zero")
	}

	if !i.FlatFee.Quantity.IsPositive() {
		return errors.New("quantity should be positive required")
	}

	// TODO[OM-947]: Validate currency specifics
	return nil
}

func (i Line) ValidateUsageBased() error {
	if err := i.UsageBased.Validate(); err != nil {
		return fmt.Errorf("usage based price: %w", err)
	}

	if i.InvoiceAt.Before(i.Period.Truncate(DefaultMeterResolution).End) {
		return errors.New("invoice at must be after period end for usage based line")
	}

	if err := i.UsageBased.Price.Validate(); err != nil {
		return fmt.Errorf("price: %w", err)
	}

	return nil
}

// TODO[OM-1016]: For events we need a json marshaler
type LineChildren struct {
	slicesx.OptionalSlice[*Line]
}

func NewLineChildren(children []*Line) LineChildren {
	return LineChildren{slicesx.NewOptionalSlice(children)}
}

func (c LineChildren) Map(fn func(*Line) *Line) LineChildren {
	return LineChildren{
		c.OptionalSlice.Map(fn),
	}
}

// ChildrenRetainingRecords returns a new LineChildren instance with the given lines. If the line has a child
// with a unique reference ID, it will try to retain the database ID of the existing child to avoid a delete/create.
func (c Line) ChildrenRetainingRecords(l []*Line) LineChildren {
	if !c.Children.IsPresent() {
		return NewLineChildren(l)
	}

	clonedNewLines := lo.Map(l, func(line *Line, _ int) *Line {
		return line.Clone()
	})

	existingItems := c.Children.Get()
	childrenRefToLine := make(map[string]*Line, len(existingItems))

	for _, child := range existingItems {
		if child.ChildUniqueReferenceID == nil {
			continue
		}

		childrenRefToLine[*child.ChildUniqueReferenceID] = child
	}

	for _, newChild := range clonedNewLines {
		newChild.ParentLineID = lo.ToPtr(c.ID)

		if newChild.ChildUniqueReferenceID == nil {
			continue
		}

		if existing, ok := childrenRefToLine[*newChild.ChildUniqueReferenceID]; ok {
			// Let's retain the database ID to achieve an update instead of a delete/create
			newChild.ID = existing.ID

			// Let's make sure we retain the created and updated at timestamps so that we
			// don't trigger an update in vain
			newChild.CreatedAt = existing.CreatedAt
			newChild.UpdatedAt = existing.UpdatedAt

			newChild.Discounts = existing.Discounts.ChildrenRetainingRecords(newChild.Discounts)
		}
	}

	return NewLineChildren(clonedNewLines)
}

type Price = plan.Price

type UsageBasedLine struct {
	ConfigID string `json:"configId"`

	Price      Price                  `json:"price"`
	FeatureKey string                 `json:"featureKey"`
	Quantity   *alpacadecimal.Decimal `json:"quantity"`
}

func (i UsageBasedLine) Equal(other UsageBasedLine) bool {
	return reflect.DeepEqual(i, other)
}

func (i UsageBasedLine) Clone() UsageBasedLine {
	return i
}

func (i UsageBasedLine) Validate() error {
	if err := i.Price.Validate(); err != nil {
		return fmt.Errorf("price: %w", err)
	}

	if i.FeatureKey == "" {
		return errors.New("featureKey is required")
	}

	return nil
}

type LineDiscountType string

const (
	// LineMaximumSpendDiscountType is a discount applied due to maximum spend.
	LineMaximumSpendDiscountType LineDiscountType = "line_maximum_spend"
	// MaximumSpendDiscountType is a discount applied due to multi-line maximum spend.
	MaximumSpendDiscountType LineDiscountType = "maximum_spend"
)

func (LineDiscountType) Values() []string {
	return []string{
		string(LineMaximumSpendDiscountType),
		string(MaximumSpendDiscountType),
	}
}

type LineDiscount struct {
	ID        string     `json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	Amount      alpacadecimal.Decimal `json:"amount"`
	Description *string               `json:"description,omitempty"`
	Type        *LineDiscountType     `json:"type,omitempty"`
}

func (i LineDiscount) Equal(other LineDiscount) bool {
	return reflect.DeepEqual(i, other)
}

// TODO[OM-1016]: For events we need a json marshaler
type LineDiscounts struct {
	slicesx.OptionalSlice[LineDiscount]
}

func NewLineDiscounts(discounts []LineDiscount) LineDiscounts {
	return LineDiscounts{slicesx.NewOptionalSlice(discounts)}
}

func (c LineDiscounts) Map(fn func(LineDiscount) LineDiscount) LineDiscounts {
	return LineDiscounts{
		c.OptionalSlice.Map(fn),
	}
}

func (c LineDiscounts) ChildrenRetainingRecords(l LineDiscounts) LineDiscounts {
	if !c.IsPresent() {
		return l
	}

	clonedNewItems := lo.Map(l.Get(), func(item LineDiscount, _ int) LineDiscount {
		return item
	})

	existingItemsByType := lo.GroupBy(
		lo.Filter(c.Get(), func(item LineDiscount, _ int) bool {
			return item.Type != nil
		}),
		func(item LineDiscount) LineDiscountType {
			return *item.Type
		},
	)

	clonedNewItems = lo.Map(clonedNewItems, func(newItem LineDiscount, _ int) LineDiscount {
		if newItem.Type != nil {
			if existingItems, ok := existingItemsByType[*newItem.Type]; ok {
				existing := existingItems[0]

				// Let's retain the created and updated at timestamps so that we
				// don't trigger an update in vain
				newItem.CreatedAt = existing.CreatedAt
				newItem.UpdatedAt = existing.UpdatedAt
				newItem.ID = existing.ID
			}
		}

		return newItem
	})

	return NewLineDiscounts(clonedNewItems)
}
