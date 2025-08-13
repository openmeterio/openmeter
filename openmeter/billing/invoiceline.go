package billing

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
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

// InvoiceLineType is deprecated, to be removed, once the line types are consolidated into seperate tables
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
	Namespace string `json:"namespace,omitempty"`
	ID        string `json:"id,omitempty"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	Metadata    map[string]string    `json:"metadata,omitempty"`
	Annotations models.Annotations   `json:"annotations,omitempty"`
	Name        string               `json:"name"`
	Type        InvoiceLineType      `json:"type"`
	ManagedBy   InvoiceLineManagedBy `json:"managedBy"`
	Description *string              `json:"description,omitempty"`

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

type Line struct {
	LineBase `json:",inline"`

	// TODO[OM-1060]: Make it a proper union type instead of having both fields as public
	// TODO: Merge with line base and have a simple object for the data contents
	UsageBased UsageBasedLine `json:"usageBased,omitempty"`

	DetailedLines      DetailedLines       `json:"detailedLines,omitempty"`
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
	clone.SplitLineHierarchy = nil
	clone.SplitLineGroupID = nil

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

	clone.DBState = nil

	return clone
}

// RemoveMetaForCompare returns a copy of the invoice without the fields that are not relevant for higher level
// tests that compare invoices. What gets removed:
// - Line's DB state
// - Line's dependencies are marked as resolved
// - Parent pointers are removed
func (i Line) RemoveMetaForCompare() *Line {
	out := i.Clone()

	if len(out.DetailedLines) == 0 {
		out.DetailedLines = DetailedLines{}
	}
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

	res.LineBase = i.LineBase.Clone()

	if !opts.skipChildren {
		res.DetailedLines = i.DetailedLines.Map(func(line DetailedLine) DetailedLine {
			return line.Clone()
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

	for _, detailedLine := range i.DetailedLines {
		if err := detailedLine.Validate(); err != nil {
			// ID might not be present at this point, so we can't use it for the error message
			errs = append(errs, fmt.Errorf("detailedLines[id=%s,child_unique_reference_id=%s]: %w", detailedLine.ID, lo.FromPtrOr(detailedLine.ChildUniqueReferenceID, "<nil>"), err))
		}
	}

	if err := i.UsageBased.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.DependsOnMeteredQuantity() && i.InvoiceAt.Before(i.Period.Truncate(streaming.MinimumWindowSizeDuration).End) {
		errs = append(errs, fmt.Errorf("invoice at (%s) must be after period end (%s) for usage based line", i.InvoiceAt, i.Period.Truncate(streaming.MinimumWindowSizeDuration).End))
	}

	if err := i.RateCardDiscounts.ValidateForPrice(i.UsageBased.Price); err != nil {
		errs = append(errs, fmt.Errorf("rateCardDiscounts: %w", err))
	}

	return errors.Join(errs...)
}

// DissacociateChildren removes the Children both from the DBState and the current line, so that the
// line can be safely persisted/managed without the children.
//
// The childrens receive DBState objects, so that they can be safely persisted/managed without the parent.
func (i *Line) DisassociateChildren() {
	i.DetailedLines = DetailedLines{}

	if i.DBState != nil {
		i.DBState.DetailedLines = DetailedLines{}
	}
}

func (i Line) DependsOnMeteredQuantity() bool {
	if i.UsageBased.Price.Type() == productcatalog.FlatPriceType {
		return false
	}

	return true
}

func (i *Line) SortDetailedLines() {
	sort.Slice(i.DetailedLines, func(a, b int) bool {
		lineA := i.DetailedLines[a]
		lineB := i.DetailedLines[b]

		if lineA.Index != nil && lineB.Index != nil {
			return *lineA.Index < *lineB.Index
		}

		if lineA.Index != nil {
			return true
		}

		if lineB.Index != nil {
			return false
		}

		if nameOrder := strings.Compare(lineA.Name, lineB.Name); nameOrder != 0 {
			return nameOrder < 0
		}

		if !lineA.ServicePeriod.Start.Equal(lineB.ServicePeriod.Start) {
			return lineA.ServicePeriod.Start.Before(lineB.ServicePeriod.Start)
		}

		return strings.Compare(lineA.ID, lineB.ID) < 0
	})
}

func (i Line) SetDiscountExternalIDs(externalIDs map[string]string) []string {
	foundIDs := []string{}

	for idx := range i.Discounts.Amount {
		discount := &i.Discounts.Amount[idx]
		if externalID, ok := externalIDs[discount.ID]; ok {
			discount.ExternalIDs.Invoicing = externalID
			foundIDs = append(foundIDs, discount.ID)
		}
	}

	for idx := range i.Discounts.Usage {
		discount := &i.Discounts.Usage[idx]

		if externalID, ok := externalIDs[discount.ID]; ok {
			discount.ExternalIDs.Invoicing = externalID
			foundIDs = append(foundIDs, discount.ID)
		}
	}

	return foundIDs
}

// helper functions for generating new lines
// TODO: Refactor this to UBP lines, we might not need this at all if the Line type gets simplified
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
			Namespace: input.Namespace,
			ID:        input.ID,
			CreatedAt: input.CreatedAt,
			UpdatedAt: input.UpdatedAt,

			Period:    input.Period,
			InvoiceAt: input.InvoiceAt,
			InvoiceID: input.InvoiceID,

			Name:        input.Name,
			Metadata:    input.Metadata,
			Annotations: input.Annotations,
			Description: input.Description,

			Status: InvoiceLineStatusValid,

			ManagedBy: lo.CoalesceOrEmpty(input.ManagedBy, SystemManagedLine),

			Currency:          input.Currency,
			RateCardDiscounts: input.RateCardDiscounts,
		},
		UsageBased: UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      input.PerUnitAmount,
				PaymentTerm: input.PaymentTerm,
			}),

			FeatureKey: ubpOptions.featureKey,
		},
	}
}

// TODO[OM-1016]: For events we need a json marshaler
type LineChildren struct {
	mo.Option[[]*Line]
}

func NewLineChildren(children []*Line) LineChildren {
	// Note: this helps with test equality checks
	if len(children) == 0 {
		children = nil
	}

	return LineChildren{mo.Some(children)}
}

func (c LineChildren) Map(fn func(*Line) *Line) LineChildren {
	if !c.IsPresent() {
		return c
	}

	return LineChildren{
		mo.Some(
			lo.Map(c.OrEmpty(), func(item *Line, _ int) *Line {
				return fn(item)
			}),
		),
	}
}

func (c LineChildren) Validate() error {
	return errors.Join(lo.Map(c.OrEmpty(), func(line *Line, idx int) error {
		return ValidationWithFieldPrefix(fmt.Sprintf("%d", idx), line.Validate())
	})...)
}

func (c *LineChildren) Append(l ...*Line) {
	c.Option = mo.Some(append(c.OrEmpty(), l...))
}

func (c LineChildren) GetByID(id string) *Line {
	return lo.FindOrElse(c.Option.OrEmpty(), nil, func(line *Line) bool {
		return line.ID == id
	})
}

func (c *LineChildren) RemoveByID(id string) bool {
	toBeRemoved := c.GetByID(id)
	if toBeRemoved == nil {
		return false
	}

	c.Option = mo.Some(
		lo.Filter(c.Option.OrEmpty(), func(l *Line, _ int) bool {
			return l.ID != id
		}),
	)

	return true
}

func (c *LineChildren) ReplaceByID(id string, newLine *Line) bool {
	if c.IsAbsent() {
		return false
	}

	lines := c.OrEmpty()

	for i, line := range lines {
		if line.ID == id {
			// Let's preserve the DB state of the original line (as we are only replacing the current state)
			originalDBState := line.DBState

			lines[i] = newLine
			lines[i].DBState = originalDBState
			return true
		}
	}

	return false
}

func (c *LineChildren) GetByChildUniqueReferenceID(id string) *Line {
	return lo.FindOrElse(c.Option.OrEmpty(), nil, func(line *Line) bool {
		return lo.FromPtr(line.ChildUniqueReferenceID) == id
	})
}

// DetailedLinesWithIDReuse returns a new LineChildren instance with the given lines. If the line has a child
// with a unique reference ID, it will try to retain the database ID of the existing child to avoid a delete/create.
func (c Line) DetailedLinesWithIDReuse(mergeWith DetailedLines) DetailedLines {
	clonedMergeWith := lo.Map(mergeWith, func(line DetailedLine, _ int) DetailedLine {
		return line.Clone()
	})

	existingItems := c.DetailedLines
	childrenRefToLine := make(map[string]DetailedLine, len(existingItems))

	for _, child := range existingItems {
		if child.ChildUniqueReferenceID == nil {
			continue
		}

		childrenRefToLine[*child.ChildUniqueReferenceID] = child
	}

	for idx := range clonedMergeWith {
		mergedLine := &clonedMergeWith[idx]

		mergedLine.ParentLineID = c.ID

		if mergedLine.ChildUniqueReferenceID == nil {
			continue
		}

		if existing, ok := childrenRefToLine[*mergedLine.ChildUniqueReferenceID]; ok {
			// Let's retain the database ID to achieve an update instead of a delete/create
			mergedLine.ID = existing.ID

			// Let's make sure we retain the created and updated at timestamps so that we
			// don't trigger an update in vain
			mergedLine.CreatedAt = existing.CreatedAt
			mergedLine.UpdatedAt = existing.UpdatedAt
			mergedLine.AmountDiscounts = mergedLine.AmountDiscounts.ReuseIDsFrom(existing.AmountDiscounts)
		}
	}

	return clonedMergeWith
}

func (c LineChildren) Clone() LineChildren {
	return c.Map(func(l *Line) *Line {
		return l.Clone()
	})
}

// NonDeletedLineCount returns the number of lines that are not deleted and have a valid status (e.g. we are ignoring split lines)
func (c LineChildren) NonDeletedLineCount() int {
	return lo.CountBy(c.OrEmpty(), func(l *Line) bool {
		return l.DeletedAt == nil && l.Status == InvoiceLineStatusValid
	})
}

func (c LineChildren) Sorted() LineChildren {
	if !c.IsPresent() {
		return c
	}

	lines := c.OrEmpty()

	sort.Slice(lines, func(a, b int) bool {
		lineA := lines[a]
		lineB := lines[b]

		if nameOrder := strings.Compare(lineA.Name, lineB.Name); nameOrder != 0 {
			return nameOrder < 0
		}

		if !lineA.Period.Start.Equal(lineB.Period.Start) {
			return lineA.Period.Start.Before(lineB.Period.Start)
		}

		return strings.Compare(lineA.ID, lineB.ID) < 0
	})

	for _, line := range lines {
		line.SortDetailedLines()
	}

	return NewLineChildren(lines)
}

type UsageBasedLine struct {
	ConfigID string `json:"configId,omitempty"`

	// Price is the price of the usage based line. Note: this should be a pointer or marshaling will fail for
	// empty prices.
	// TODO[later]: This must not be a pointer, as it's mandatory
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

	// TODO[later]: Let's have a proper type for Line creates
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

		if len(line.DetailedLines) > 0 {
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
