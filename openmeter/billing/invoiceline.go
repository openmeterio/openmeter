package billing

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/equal"
	"github.com/openmeterio/openmeter/pkg/models"
)

type LineID models.NamespacedID

func (i LineID) Validate() error {
	return models.NamespacedID(i).Validate()
}

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
	// InvoiceLineStatusDetailed is a detailed invoice line.
	InvoiceLineStatusDetailed InvoiceLineStatus = "detailed"
)

func (InvoiceLineStatus) Values() []string {
	return []string{
		string(InvoiceLineStatusValid),
		string(InvoiceLineStatusDetailed),
	}
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

// LineBase represents the common fields for an invoice item.
type LineBase struct {
	models.ManagedResource

	Metadata    map[string]string    `json:"metadata,omitempty"`
	Annotations models.Annotations   `json:"annotations,omitempty"`
	Type        InvoiceLineType      `json:"type"`
	ManagedBy   InvoiceLineManagedBy `json:"managedBy"`

	InvoiceID string         `json:"invoiceID,omitempty"`
	Currency  currencyx.Code `json:"currency"`

	// Lifecycle
	Period    Period    `json:"period"`
	InvoiceAt time.Time `json:"invoiceAt"`

	// Relationships
	ParentLineID     *string `json:"parentLine,omitempty"`
	SplitLineGroupID *string `json:"splitLineGroupId,omitempty"`

	Status                 InvoiceLineStatus `json:"status"`
	ChildUniqueReferenceID *string           `json:"childUniqueReferenceID,omitempty"`

	TaxConfig         *productcatalog.TaxConfig `json:"taxOverrides,omitempty"`
	RateCardDiscounts Discounts                 `json:"rateCardDiscounts,omitempty"`

	ExternalIDs  LineExternalIDs        `json:"externalIDs,omitempty"`
	Subscription *SubscriptionReference `json:"subscription,omitempty"`

	Totals Totals `json:"totals,omitempty"`
}

func (i LineBase) Equal(other LineBase) bool {
	return reflect.DeepEqual(i, other)
}

func (i LineBase) GetParentID() (string, bool) {
	if i.ParentLineID == nil {
		return "", false
	}
	return *i.ParentLineID, true
}

func (i LineBase) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Period.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("period: %w", err))
	}

	if i.InvoiceAt.IsZero() {
		errs = append(errs, errors.New("invoice at is required"))
	}

	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if i.Type == "" {
		errs = append(errs, errors.New("type is required"))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if !slices.Contains(InvoiceLineManagedBy("").Values(), string(i.ManagedBy)) {
		errs = append(errs, fmt.Errorf("invalid managed by %s", i.ManagedBy))
	}

	if i.Status == InvoiceLineStatusDetailed && i.ManagedBy != SystemManagedLine {
		errs = append(errs, errors.New("detailed lines must be system managed"))
	}

	return errors.Join(errs...)
}

func (i LineBase) Clone() LineBase {
	out := i

	// Clone pointer fields (where they are mutable)
	if i.Metadata != nil {
		out.Metadata = make(map[string]string, len(i.Metadata))
		for k, v := range i.Metadata {
			out.Metadata[k] = v
		}
	}

	if i.Annotations != nil {
		out.Annotations = make(models.Annotations, len(i.Annotations))
		for k, v := range i.Annotations {
			out.Annotations[k] = v
		}
	}

	if i.TaxConfig != nil {
		tc := *i.TaxConfig
		out.TaxConfig = &tc
	}

	out.RateCardDiscounts = i.RateCardDiscounts.Clone()

	return out
}

type SubscriptionReference struct {
	SubscriptionID string `json:"subscriptionID"`
	PhaseID        string `json:"phaseID"`
	ItemID         string `json:"itemID"`
}

func (i SubscriptionReference) Validate() error {
	var errs []error

	if i.SubscriptionID == "" {
		errs = append(errs, errors.New("subscriptionID is required"))
	}

	if i.PhaseID == "" {
		errs = append(errs, errors.New("phaseID is required"))
	}

	if i.ItemID == "" {
		errs = append(errs, errors.New("itemID is required"))
	}

	return errors.Join(errs...)
}

type LineExternalIDs struct {
	Invoicing string `json:"invoicing,omitempty"`
}

func (i LineExternalIDs) Equal(other LineExternalIDs) bool {
	return i.Invoicing == other.Invoicing
}

type FlatFeeCategory string

const (
	// FlatFeeCategoryRegular is a regular flat fee, that is based on the usage or a subscription.
	FlatFeeCategoryRegular FlatFeeCategory = "regular"
	// FlatFeeCategoryCommitment is a flat fee that is based on a commitment such as min spend.
	FlatFeeCategoryCommitment FlatFeeCategory = "commitment"
)

func (FlatFeeCategory) Values() []string {
	return []string{
		string(FlatFeeCategoryRegular),
		string(FlatFeeCategoryCommitment),
	}
}

type FlatFeeLine struct {
	ConfigID      string                         `json:"configId"`
	PerUnitAmount alpacadecimal.Decimal          `json:"perUnitAmount"`
	PaymentTerm   productcatalog.PaymentTermType `json:"paymentTerm"`
	Category      FlatFeeCategory                `json:"category"`

	Quantity alpacadecimal.Decimal `json:"quantity"`
	// Index is the index of the line in the invoice. Only valid for detailed lines. Used for sorting the lines in the invoice.
	Index *int `json:"index,omitempty"`
}

func (i FlatFeeLine) Clone() *FlatFeeLine {
	return &i
}

func (i FlatFeeLine) Equal(other *FlatFeeLine) bool {
	if other == nil {
		return false
	}
	return reflect.DeepEqual(i, *other)
}

type Line struct {
	LineBase `json:",inline"`

	// TODO[OM-1060]: Make it a proper union type instead of having both fields as public
	FlatFee    *FlatFeeLine    `json:"flatFee,omitempty"`
	UsageBased *UsageBasedLine `json:"usageBased,omitempty"`

	Children           LineChildren        `json:"children,omitempty"`
	ParentLine         *Line               `json:"parent,omitempty"`
	SplitLineHierarchy *SplitLineHierarchy `json:"progressiveLineHierarchy,omitempty"`

	Discounts LineDiscounts `json:"discounts,omitempty"`

	DBState *Line `json:"-"`
}

func (i Line) LineID() LineID {
	return LineID{
		Namespace: i.Namespace,
		ID:        i.ID,
	}
}

// CloneWithoutDependencies returns a clone of the line without any external dependencies. Could be used
// for creating a new line without any references to the parent or children (or config IDs).
func (i Line) CloneWithoutDependencies() *Line {
	clone := i.clone(cloneOptions{
		skipDBState:   true,
		skipChildren:  true,
		skipDiscounts: true,
	})

	clone.ID = ""
	clone.ParentLineID = nil
	clone.ParentLine = nil
	clone.SplitLineHierarchy = nil
	clone.SplitLineGroupID = nil

	if clone.FlatFee != nil {
		clone.FlatFee.ConfigID = ""
	}

	if clone.UsageBased != nil {
		clone.UsageBased.ConfigID = ""
	}

	return clone
}

func (i Line) WithoutDBState() *Line {
	i.DBState = nil
	return &i
}

func (i Line) WithoutSplitLineHierarchy() *Line {
	i.SplitLineHierarchy = nil
	return &i
}

func (i Line) RemoveCircularReferences() *Line {
	clone := i.Clone()

	clone.ParentLine = nil
	clone.DBState = nil

	clone.Children = lo.Map(clone.Children, func(l *Line, _ int) *Line {
		return l.RemoveCircularReferences()
	})

	return clone
}

// RemoveMetaForCompare returns a copy of the invoice without the fields that are not relevant for higher level
// tests that compare invoices. What gets removed:
// - Line's DB state
// - Line's dependencies are marked as resolved
// - Parent pointers are removed
func (i Line) RemoveMetaForCompare() *Line {
	out := i.Clone()

	if len(out.Children) == 0 {
		out.Children = NewLineChildren(nil)
	}

	for _, child := range out.Children {
		child.ParentLine = out

		if len(child.Children) == 0 {
			child.Children = NewLineChildren(nil)
		}
	}

	out.ParentLine = nil
	out.DBState = nil
	return out
}

func (i Line) Clone() *Line {
	return i.clone(cloneOptions{})
}

type cloneOptions struct {
	skipDBState   bool
	skipChildren  bool
	skipDiscounts bool
}

func (i Line) clone(opts cloneOptions) *Line {
	res := &Line{}
	if !opts.skipDBState {
		// DBStates are considered immutable, so it's safe to clone
		res.DBState = i.DBState
	}

	switch i.Type {
	case InvoiceLineTypeFee:
		res.FlatFee = i.FlatFee.Clone()
	case InvoiceLineTypeUsageBased:
		res.UsageBased = i.UsageBased.Clone()
	}

	res.LineBase = i.LineBase.Clone()

	if !opts.skipChildren {
		res.Children = lo.Map(i.Children, func(line *Line, _ int) *Line {
			cloned := line.Clone()
			cloned.ParentLine = line
			return cloned
		})
	}

	if !opts.skipDiscounts {
		res.Discounts = i.Discounts.Clone()
	}

	if i.SplitLineHierarchy != nil {
		res.SplitLineHierarchy = lo.ToPtr(i.SplitLineHierarchy.Clone())
	}

	return res
}

func (i Line) CloneWithoutChildren() *Line {
	return i.clone(cloneOptions{
		skipChildren: true,
	})
}

func (i *Line) SaveDBSnapshot() {
	i.DBState = i.Clone()
}

func (i Line) Validate() error {
	var errs []error
	if err := i.LineBase.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.Discounts.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("discounts: %w", err))
	}

	if len(i.Children) > 0 {
		if i.Status == InvoiceLineStatusDetailed {
			errs = append(errs, errors.New("detailed lines are not allowed for detailed lines (e.g. no nesting is allowed)"))
		} else {
			for j, detailedLine := range i.Children {
				if err := detailedLine.Validate(); err != nil {
					errs = append(errs, fmt.Errorf("detailedLines[%d]: %w", j, err))
				}

				switch i.Status {
				case InvoiceLineStatusValid:
					if detailedLine.Status != InvoiceLineStatusDetailed {
						errs = append(errs, fmt.Errorf("detailedLines[%d]: valid line's detailed lines must have detailed status", j))
						continue
					}

					if detailedLine.Type != InvoiceLineTypeFee {
						errs = append(errs, fmt.Errorf("detailedLines[%d]: valid line's detailed lines must be fee typed", j))
						continue
					}
				}
			}
		}
	}

	switch i.Type {
	case InvoiceLineTypeFee:
		if err := i.ValidateFee(); err != nil {
			errs = append(errs, err)
		}

		price := productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      i.FlatFee.PerUnitAmount,
			PaymentTerm: i.FlatFee.PaymentTerm,
		})

		if err := i.RateCardDiscounts.ValidateForPrice(price); err != nil {
			errs = append(errs, fmt.Errorf("rateCardDiscounts: %w", err))
		}
	case InvoiceLineTypeUsageBased:
		if err := i.ValidateUsageBased(); err != nil {
			errs = append(errs, err)
		}

		if err := i.RateCardDiscounts.ValidateForPrice(i.UsageBased.Price); err != nil {
			errs = append(errs, fmt.Errorf("rateCardDiscounts: %w", err))
		}

	default:
		errs = append(errs, fmt.Errorf("unsupported type: %s", i.Type))
	}

	return errors.Join(errs...)
}

func (i Line) ValidateFee() error {
	var errs []error

	if i.FlatFee == nil {
		return errors.New("flat fee is required")
	}

	if i.FlatFee.PerUnitAmount.IsNegative() {
		errs = append(errs, errors.New("price should be positive or zero"))
	}

	if i.FlatFee.Quantity.IsNegative() {
		errs = append(errs, errors.New("quantity should be positive or zero"))
	}

	if !slices.Contains(FlatFeeCategory("").Values(), string(i.FlatFee.Category)) {
		errs = append(errs, fmt.Errorf("invalid category %s", i.FlatFee.Category))
	}

	if !slices.Contains(productcatalog.PaymentTermType("").Values(), string(i.FlatFee.PaymentTerm)) {
		errs = append(errs, fmt.Errorf("invalid payment term %s", i.FlatFee.PaymentTerm))
	}

	return errors.Join(errs...)
}

func (i Line) ValidateUsageBased() error {
	var errs []error

	if err := i.UsageBased.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.DependsOnMeteredQuantity() && i.InvoiceAt.Before(i.Period.Truncate(streaming.MinimumWindowSizeDuration).End) {
		errs = append(errs, fmt.Errorf("invoice at (%s) must be after period end (%s) for usage based line", i.InvoiceAt, i.Period.Truncate(streaming.MinimumWindowSizeDuration).End))
	}

	return errors.Join(errs...)
}

// DissacociateChildren removes the Children both from the DBState and the current line, so that the
// line can be safely persisted/managed without the children.
//
// The childrens receive DBState objects, so that they can be safely persisted/managed without the parent.
func (i *Line) DisassociateChildren() {
	i.Children = LineChildren{}
	if i.DBState != nil {
		i.DBState.Children = LineChildren{}
	}
}

func (i Line) DependsOnMeteredQuantity() bool {
	if i.Type != InvoiceLineTypeUsageBased {
		return false
	}

	if i.UsageBased.Price.Type() == productcatalog.FlatPriceType {
		return false
	}

	return true
}

// helper functions for generating new lines
type NewFlatFeeLineInput struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time

	Namespace string
	Period    Period
	InvoiceAt time.Time

	InvoiceID string

	Name        string
	Metadata    map[string]string
	Annotations models.Annotations
	Description *string

	Currency currencyx.Code

	ManagedBy InvoiceLineManagedBy

	PerUnitAmount alpacadecimal.Decimal
	PaymentTerm   productcatalog.PaymentTermType

	RateCardDiscounts Discounts
}

func NewFlatFeeLine(input NewFlatFeeLineInput) *Line {
	return &Line{
		LineBase: LineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   input.Namespace,
				ID:          input.ID,
				CreatedAt:   input.CreatedAt,
				UpdatedAt:   input.UpdatedAt,
				Name:        input.Name,
				Description: input.Description,
			}),

			Period:    input.Period,
			InvoiceAt: input.InvoiceAt,
			InvoiceID: input.InvoiceID,

			Metadata:    input.Metadata,
			Annotations: input.Annotations,

			Status: InvoiceLineStatusValid,

			Type: InvoiceLineTypeFee,

			ManagedBy: lo.CoalesceOrEmpty(input.ManagedBy, SystemManagedLine),

			Currency:          input.Currency,
			RateCardDiscounts: input.RateCardDiscounts,
		},
		FlatFee: &FlatFeeLine{
			PerUnitAmount: input.PerUnitAmount,
			PaymentTerm:   input.PaymentTerm,
			Category:      FlatFeeCategoryRegular,
			Quantity:      alpacadecimal.NewFromInt(1),
		},
	}
}

type usageBasedLineOptions struct {
	featureKey string
}

type usageBasedLineOption func(*usageBasedLineOptions)

func WithFeatureKey(fk string) usageBasedLineOption {
	return func(ublo *usageBasedLineOptions) {
		ublo.featureKey = fk
	}
}

// NewUsageBasedFlatFeeLine creates a new usage based flat fee line (which is semantically equivalent to the line returned by
// NewFlatFeeLine, but based on the usage based line semantic).
//
// Note: this is temporary in it's current form until we validate the usage based flat fee schema
func NewUsageBasedFlatFeeLine(input NewFlatFeeLineInput, opts ...usageBasedLineOption) *Line {
	ubpOptions := usageBasedLineOptions{}

	for _, opt := range opts {
		opt(&ubpOptions)
	}

	return &Line{
		LineBase: LineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   input.Namespace,
				ID:          input.ID,
				CreatedAt:   input.CreatedAt,
				UpdatedAt:   input.UpdatedAt,
				Name:        input.Name,
				Description: input.Description,
			}),
			Period:    input.Period,
			InvoiceAt: input.InvoiceAt,
			InvoiceID: input.InvoiceID,

			Metadata:    input.Metadata,
			Annotations: input.Annotations,

			Status: InvoiceLineStatusValid,

			Type: InvoiceLineTypeUsageBased,

			ManagedBy: lo.CoalesceOrEmpty(input.ManagedBy, SystemManagedLine),

			Currency:          input.Currency,
			RateCardDiscounts: input.RateCardDiscounts,
		},
		UsageBased: &UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      input.PerUnitAmount,
				PaymentTerm: input.PaymentTerm,
			}),

			FeatureKey: ubpOptions.featureKey,
		},
	}
}

// TODO[OM-1016]: For events we need a json marshaler
type LineChildren []*Line

func NewLineChildren(children []*Line) LineChildren {
	// Note: this helps with test equality checks
	if len(children) == 0 {
		children = nil
	}

	return LineChildren(children)
}

func (c LineChildren) Validate() error {
	return errors.Join(lo.Map(c, func(line *Line, idx int) error {
		return ValidationWithFieldPrefix(fmt.Sprintf("%d", idx), line.Validate())
	})...)
}

func (c LineChildren) GetByID(id string) *Line {
	return lo.FindOrElse(c, nil, func(line *Line) bool {
		return line.ID == id
	})
}

func (c *LineChildren) RemoveByID(id string) bool {
	toBeRemoved := c.GetByID(id)
	if toBeRemoved == nil {
		return false
	}

	*c = lo.Filter(*c, func(l *Line, _ int) bool {
		return l.ID != id
	})

	return true
}

func (c LineChildren) GetByChildUniqueReferenceID(id string) *Line {
	return lo.FindOrElse(c, nil, func(line *Line) bool {
		return lo.FromPtr(line.ChildUniqueReferenceID) == id
	})
}

// ChildrenWithIDReuse returns a new LineChildren instance with the given lines. If the line has a child
// with a unique reference ID, it will try to retain the database ID of the existing child to avoid a delete/create.
func (c Line) ChildrenWithIDReuse(l []*Line) (LineChildren, error) {
	clonedNewLines := lo.Map(l, func(line *Line, _ int) *Line {
		return line.Clone()
	})

	existingItems := c.Children
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

			discountsWithIDReuse, err := newChild.Discounts.ReuseIDsFrom(existing.Discounts)
			if err != nil {
				return LineChildren{}, fmt.Errorf("failed to reuse discount ids: %w", err)
			}

			newChild.Discounts = discountsWithIDReuse
		}
	}

	return NewLineChildren(clonedNewLines), nil
}

type UsageBasedLine struct {
	ConfigID string `json:"configId,omitempty"`

	// Price is the price of the usage based line. Note: this should be a pointer or marshaling will fail for
	// empty prices.
	Price      *productcatalog.Price `json:"price"`
	FeatureKey string                `json:"featureKey"`

	Quantity        *alpacadecimal.Decimal `json:"quantity,omitempty"`
	MeteredQuantity *alpacadecimal.Decimal `json:"meteredQuantity,omitempty"`

	PreLinePeriodQuantity        *alpacadecimal.Decimal `json:"preLinePeriodQuantity,omitempty"`
	MeteredPreLinePeriodQuantity *alpacadecimal.Decimal `json:"meteredPreLinePeriodQuantity,omitempty"`
}

func (i UsageBasedLine) Equal(other *UsageBasedLine) bool {
	if other == nil {
		return false
	}

	if !i.Price.Equal(other.Price) {
		return false
	}

	if i.FeatureKey != other.FeatureKey {
		return false
	}

	if !equal.PtrEqual(i.Quantity, other.Quantity) {
		return false
	}

	if !equal.PtrEqual(i.MeteredQuantity, other.MeteredQuantity) {
		return false
	}

	if !equal.PtrEqual(i.PreLinePeriodQuantity, other.PreLinePeriodQuantity) {
		return false
	}

	if !equal.PtrEqual(i.MeteredPreLinePeriodQuantity, other.MeteredPreLinePeriodQuantity) {
		return false
	}

	return true
}

func (i UsageBasedLine) Clone() *UsageBasedLine {
	return &i
}

func (i UsageBasedLine) Validate() error {
	var errs []error

	if err := i.Price.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("price: %w", err))
	}

	if i.Price.Type() != productcatalog.FlatPriceType {
		if i.FeatureKey == "" {
			errs = append(errs, errors.New("featureKey is required"))
		}
	}

	return errors.Join(errs...)
}

type CreatePendingInvoiceLinesInput struct {
	Customer customer.CustomerID `json:"customer"`
	Currency currencyx.Code      `json:"currency"`

	Lines []*Line `json:"lines"`
}

func (c CreatePendingInvoiceLinesInput) Validate() error {
	var errs []error

	if err := c.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if err := c.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	for id, line := range c.Lines {
		// Note: this is for validation purposes, as Line is copied, we are not altering the struct itself
		line.Currency = c.Currency

		if err := line.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("line.%d: %w", id, err))
		}

		if line.InvoiceID != "" {
			errs = append(errs, fmt.Errorf("line.%d: invoice ID is not allowed for pending lines", id))
		}

		if len(line.Children) > 0 {
			errs = append(errs, fmt.Errorf("line.%d: children are not allowed for pending lines", id))
		}

		if line.ParentLineID != nil {
			errs = append(errs, fmt.Errorf("line.%d: parent line ID is not allowed for pending lines", id))
		}

		if line.SplitLineGroupID != nil {
			errs = append(errs, fmt.Errorf("line.%d: split line group ID is not allowed for pending lines", id))
		}
	}

	return errors.Join(errs...)
}

type CreatePendingInvoiceLinesResult struct {
	Lines        []*Line
	Invoice      Invoice
	IsInvoiceNew bool
}

type UpsertInvoiceLinesAdapterInput struct {
	Namespace string
	Lines     []*Line
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
	InvoiceIDs                 []string
	InvoiceStatuses            []InvoiceStatus
	InvoiceAtBefore            *time.Time
	IncludeDeleted             bool
	ParentLineIDs              []string
	ParentLineIDsIncludeParent bool
	Statuses                   []InvoiceLineStatus

	LineIDs []string
}

func (g ListInvoiceLinesAdapterInput) Validate() error {
	if g.Namespace == "" {
		return errors.New("namespace is required")
	}

	return nil
}

type AssociateLinesToInvoiceAdapterInput struct {
	Invoice InvoiceID

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

type UpdateInvoiceLineAdapterInput Line

type UpdateInvoiceLineInput struct {
	// Mandatory fields for update
	Line LineID
	Type InvoiceLineType

	LineBase   UpdateInvoiceLineBaseInput
	UsageBased UpdateInvoiceLineUsageBasedInput
	FlatFee    UpdateInvoiceLineFlatFeeInput
}

func (u UpdateInvoiceLineInput) Validate() error {
	var outErr error
	if err := u.LineBase.Validate(); err != nil {
		outErr = errors.Join(outErr, err)
	}

	if err := u.Line.Validate(); err != nil {
		outErr = errors.Join(outErr, fmt.Errorf("validating LineID: %w", err))
	}

	if !slices.Contains(u.Type.Values(), string(u.Type)) {
		outErr = errors.Join(outErr, ValidationWithFieldPrefix(
			"type", fmt.Errorf("line base: invalid type %s", u.Type),
		))
		return outErr
	}

	switch u.Type {
	case InvoiceLineTypeUsageBased:
		if err := u.UsageBased.Validate(); err != nil {
			outErr = errors.Join(outErr, err)
		}
	case InvoiceLineTypeFee:
		if err := u.FlatFee.Validate(); err != nil {
			outErr = errors.Join(outErr, err)
		}
	}

	return outErr
}

func (u UpdateInvoiceLineInput) Apply(l *Line) (*Line, error) {
	oldParentLine := l.ParentLine

	l = l.Clone()

	// Clone doesn't carry over parent line, so that the cloned hierarchy and the new one are disjunct,
	// however in this specific case we don't care about that, so we just copy it over
	l.ParentLine = oldParentLine

	if u.Type != l.Type {
		return l, fmt.Errorf("line type cannot be changed")
	}

	if err := u.LineBase.Apply(l); err != nil {
		return l, err
	}

	switch l.Type {
	case InvoiceLineTypeUsageBased:
		if err := u.UsageBased.Apply(l.UsageBased); err != nil {
			return l, err
		}
	case InvoiceLineTypeFee:
		if err := u.FlatFee.Apply(l.FlatFee); err != nil {
			return l, err
		}
	}

	return l, nil
}

type UpdateInvoiceLineBaseInput struct {
	InvoiceAt mo.Option[time.Time]

	Metadata    mo.Option[map[string]string]
	Annotations mo.Option[models.Annotations]
	Name        mo.Option[string]
	ManagedBy   mo.Option[InvoiceLineManagedBy]
	Period      mo.Option[Period]
	TaxConfig   mo.Option[*productcatalog.TaxConfig]
}

func (u UpdateInvoiceLineBaseInput) Validate() error {
	var outErr error

	if u.InvoiceAt.IsPresent() {
		invoiceAt := u.InvoiceAt.OrEmpty()

		if invoiceAt.IsZero() {
			outErr = errors.Join(outErr, ValidationWithFieldPrefix("invoice_at", ErrFieldRequired))
		}
	}

	if u.Name.IsPresent() && u.Name.OrEmpty() == "" {
		outErr = errors.Join(outErr, ValidationWithFieldPrefix("name", ErrFieldRequired))
	}

	if u.Period.IsPresent() {
		if err := u.Period.OrEmpty().Validate(); err != nil {
			outErr = errors.Join(outErr, ValidationWithFieldPrefix("period", err))
		}
	}

	if u.TaxConfig.IsPresent() {
		if err := u.TaxConfig.OrEmpty().Validate(); err != nil {
			outErr = errors.Join(outErr, ValidationWithFieldPrefix("tax_config", err))
		}
	}

	if u.ManagedBy.IsPresent() {
		if !slices.Contains(InvoiceLineManagedBy("").Values(), string(u.ManagedBy.OrEmpty())) {
			outErr = errors.Join(outErr, ValidationWithFieldPrefix("managed_by", fmt.Errorf("invalid managed by %s", u.ManagedBy.OrEmpty())))
		}
	}

	return outErr
}

func (u UpdateInvoiceLineBaseInput) Apply(l *Line) error {
	if u.InvoiceAt.IsPresent() {
		l.InvoiceAt = u.InvoiceAt.OrEmpty().In(time.UTC)
	}

	if u.Metadata.IsPresent() {
		l.Metadata = u.Metadata.OrEmpty()
	}

	if u.Annotations.IsPresent() {
		l.Annotations = u.Annotations.OrEmpty()
	}

	if u.Name.IsPresent() {
		l.Name = u.Name.OrEmpty()
	}

	if u.Period.IsPresent() {
		l.Period = u.Period.OrEmpty()
	}

	if u.TaxConfig.IsPresent() {
		l.TaxConfig = u.TaxConfig.OrEmpty()
	}

	if u.ManagedBy.IsPresent() {
		newManagedBy := u.ManagedBy.OrEmpty()
		switch newManagedBy {
		case SystemManagedLine:
			return ValidationError{
				Err: fmt.Errorf("managed by cannot be changed to system managed via the API"),
			}
		case SubscriptionManagedLine:
			if l.Subscription == nil || l.Subscription.SubscriptionID == "" {
				return ValidationError{
					Err: fmt.Errorf("subscription managed line must have a subscription"),
				}
			}
		}

		l.ManagedBy = newManagedBy
	}

	return nil
}

type UpdateInvoiceLineUsageBasedInput struct {
	Price *productcatalog.Price
}

func (u UpdateInvoiceLineUsageBasedInput) Validate() error {
	var outErr error

	if u.Price != nil {
		if err := u.Price.Validate(); err != nil {
			outErr = errors.Join(outErr, ValidationWithFieldPrefix("price", err))
		}
	}

	return outErr
}

func (u UpdateInvoiceLineUsageBasedInput) Apply(l *UsageBasedLine) error {
	if u.Price != nil {
		l.Price = u.Price
	}

	return nil
}

type UpdateInvoiceLineFlatFeeInput struct {
	PerUnitAmount mo.Option[alpacadecimal.Decimal]
	Quantity      mo.Option[alpacadecimal.Decimal]
	PaymentTerm   mo.Option[productcatalog.PaymentTermType]
}

func (u UpdateInvoiceLineFlatFeeInput) Validate() error {
	var outErr error

	if u.PerUnitAmount.IsPresent() && !u.PerUnitAmount.OrEmpty().IsPositive() {
		outErr = errors.Join(outErr, ValidationWithFieldPrefix("per_unit_amount", ErrFieldMustBePositive))
	}

	if u.Quantity.IsPresent() && u.Quantity.OrEmpty().IsNegative() {
		outErr = errors.Join(outErr, ValidationWithFieldPrefix("quantity", ErrFieldMustBePositiveOrZero))
	}

	if u.PaymentTerm.IsPresent() && !slices.Contains(productcatalog.PaymentTermType("").Values(), string(u.PaymentTerm.OrEmpty())) {
		outErr = errors.Join(outErr, ValidationWithFieldPrefix("payment_term", fmt.Errorf("invalid payment term %s", u.PaymentTerm.OrEmpty())))
	}

	return outErr
}

func (u UpdateInvoiceLineFlatFeeInput) Apply(l *FlatFeeLine) error {
	if u.PerUnitAmount.IsPresent() {
		l.PerUnitAmount = u.PerUnitAmount.OrEmpty()
	}

	if u.Quantity.IsPresent() {
		l.Quantity = u.Quantity.OrEmpty()
	}

	if u.PaymentTerm.IsPresent() {
		l.PaymentTerm = u.PaymentTerm.OrEmpty()
	}

	return nil
}

type GetInvoiceLineAdapterInput = LineID

type GetInvoiceLineInput = LineID

type GetInvoiceLineOwnershipAdapterInput = LineID

type DeleteInvoiceLineInput = LineID

type GetLinesForSubscriptionInput struct {
	Namespace      string
	SubscriptionID string
}

func (i GetLinesForSubscriptionInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.SubscriptionID == "" {
		return errors.New("subscription id is required")
	}

	return nil
}

type SnapshotLineQuantityInput struct {
	Invoice *Invoice
	Line    *Line
}

func (i SnapshotLineQuantityInput) Validate() error {
	if i.Invoice == nil {
		return errors.New("invoice is required")
	}

	if i.Line == nil {
		return errors.New("line is required")
	}

	return nil
}
