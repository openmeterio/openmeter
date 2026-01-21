package billing

import (
	"errors"
	"fmt"
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
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

// LineBase represents the common fields for an invoice item.
type LineBase struct {
	models.ManagedResource

	Metadata    map[string]string    `json:"metadata,omitempty"`
	Annotations models.Annotations   `json:"annotations,omitempty"`
	ManagedBy   InvoiceLineManagedBy `json:"managedBy"`

	InvoiceID string         `json:"invoiceID,omitempty"`
	Currency  currencyx.Code `json:"currency"`

	// Lifecycle
	Period    Period    `json:"period"`
	InvoiceAt time.Time `json:"invoiceAt"`

	// Relationships
	ParentLineID     *string `json:"parentLine,omitempty"`
	SplitLineGroupID *string `json:"splitLineGroupId,omitempty"`

	ChildUniqueReferenceID *string `json:"childUniqueReferenceID,omitempty"`

	TaxConfig         *productcatalog.TaxConfig `json:"taxOverrides,omitempty"`
	RateCardDiscounts Discounts                 `json:"rateCardDiscounts,omitempty"`

	ExternalIDs  LineExternalIDs        `json:"externalIDs,omitempty"`
	Subscription *SubscriptionReference `json:"subscription,omitempty"`

	Totals Totals `json:"totals,omitempty"`
}

func (i LineBase) Equal(other LineBase) bool {
	return deriveEqualLineBase(&i, &other)
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

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if !slices.Contains(InvoiceLineManagedBy("").Values(), string(i.ManagedBy)) {
		errs = append(errs, fmt.Errorf("invalid managed by %s", i.ManagedBy))
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
	SubscriptionID string                `json:"subscriptionID"`
	PhaseID        string                `json:"phaseID"`
	ItemID         string                `json:"itemID"`
	BillingPeriod  timeutil.ClosedPeriod `json:"billingPeriod"`
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

	if err := i.BillingPeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("billingPeriod: %w", err))
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

	UsageBased *UsageBasedLine `json:"usageBased,omitempty"`

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

type LineEditFunction func(*Line)

// CloneWithoutDependencies returns a clone of the line without any external dependencies. Could be used
// for creating a new line without any references to the parent or children (or config IDs).
func (i Line) CloneWithoutDependencies(edits ...LineEditFunction) *Line {
	clone := i.clone(cloneOptions{
		skipDBState:   true,
		skipChildren:  true,
		skipDiscounts: true,
	})

	clone.ID = ""
	clone.CreatedAt = time.Time{}
	clone.UpdatedAt = time.Time{}
	clone.DeletedAt = nil

	clone.ParentLineID = nil
	clone.SplitLineHierarchy = nil
	clone.SplitLineGroupID = nil

	if clone.UsageBased != nil {
		clone.UsageBased.ConfigID = ""
	}

	for _, edit := range edits {
		if edit != nil {
			edit(clone)
		}
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

	out.DetailedLines = nil
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

	res.UsageBased = i.UsageBased.Clone()
	res.LineBase = i.LineBase.Clone()

	if !opts.skipChildren {
		res.DetailedLines = i.DetailedLines.Clone()
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

	if err := i.DetailedLines.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("detailed lines: %w", err))
	}

	if err := i.ValidateUsageBased(); err != nil {
		errs = append(errs, err)
	}

	if err := i.RateCardDiscounts.ValidateForPrice(i.UsageBased.Price); err != nil {
		errs = append(errs, fmt.Errorf("rateCardDiscounts: %w", err))
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
	i.DetailedLines = nil
	if i.DBState != nil {
		i.DBState.DetailedLines = nil
	}
}

func (i Line) DependsOnMeteredQuantity() bool {
	return i.UsageBased.Price.Type() != productcatalog.FlatPriceType
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

type usageBasedLineOptions struct {
	featureKey string
}

type usageBasedLineOption func(*usageBasedLineOptions)

func WithFeatureKey(fk string) usageBasedLineOption {
	return func(ublo *usageBasedLineOptions) {
		ublo.featureKey = fk
	}
}

// NewFlatFeeLine creates a new invoice-level flat fee line.
func NewFlatFeeLine(input NewFlatFeeLineInput, opts ...usageBasedLineOption) *Line {
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

// DetailedLinesWithIDReuse returns a new DetailedLines instance with the given lines. If the line has a child
// with a unique reference ID, it will try to retain the database ID of the existing child to avoid a delete/create.
func (c Line) DetailedLinesWithIDReuse(l DetailedLines) DetailedLines {
	clonedNewLines := l.Clone()

	existingItems := c.DetailedLines
	childrenRefToLine := make(map[string]DetailedLine, len(existingItems))

	for _, child := range existingItems {
		if child.ChildUniqueReferenceID == nil {
			continue
		}

		// Let's only reuse lines that were not deleted before
		if child.DeletedAt != nil {
			continue
		}

		childrenRefToLine[*child.ChildUniqueReferenceID] = child
	}

	for idx := range clonedNewLines {
		newChild := &clonedNewLines[idx]

		if newChild.ChildUniqueReferenceID == nil {
			continue
		}

		if existing, ok := childrenRefToLine[*newChild.ChildUniqueReferenceID]; ok {
			// Let's retain the database ID to achieve an update instead of a delete/create
			newChild.ID = existing.ID
			newChild.FeeLineConfigID = existing.FeeLineConfigID

			// Let's make sure we retain the created and updated at timestamps so that we
			// don't trigger an update in vain
			newChild.CreatedAt = existing.CreatedAt
			newChild.UpdatedAt = existing.UpdatedAt

			discountsWithIDReuse := newChild.AmountDiscounts.ReuseIDsFrom(existing.AmountDiscounts)
			newChild.AmountDiscounts = discountsWithIDReuse
		}
	}

	return clonedNewLines
}

type Lines []*Line

func NewLines(children []*Line) Lines {
	// Note: this helps with test equality checks
	if len(children) == 0 {
		children = nil
	}

	return Lines(children)
}

func (c Lines) Validate() error {
	return errors.Join(lo.Map(c, func(line *Line, idx int) error {
		return ValidationWithFieldPrefix(fmt.Sprintf("%d", idx), line.Validate())
	})...)
}

func (c Lines) GetByChildUniqueReferenceID(id string) *Line {
	return lo.FindOrElse(c, nil, func(line *Line) bool {
		return lo.FromPtr(line.ChildUniqueReferenceID) == id
	})
}

func (c Lines) Map(fn func(*Line) *Line) Lines {
	return Lines(
		lo.Map(c, func(l *Line, _ int) *Line {
			return fn(l)
		}),
	)
}

func (c *Lines) Sort() {
	sort.Slice(*c, func(a, b int) bool {
		lineA := (*c)[a]
		lineB := (*c)[b]

		if nameOrder := strings.Compare(lineA.Name, lineB.Name); nameOrder != 0 {
			return nameOrder < 0
		}

		if !lineA.Period.Start.Equal(lineB.Period.Start) {
			return lineA.Period.Start.Before(lineB.Period.Start)
		}

		return strings.Compare(lineA.ID, lineB.ID) < 0
	})

	for idx := range *c {
		(*c)[idx].SortDetailedLines()
	}
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
	return deriveEqualUsageBasedLine(&i, other)
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

		if len(line.DetailedLines) > 0 {
			errs = append(errs, fmt.Errorf("line.%d: detailed lines are not allowed for pending lines", id))
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

	CustomerID      string
	InvoiceIDs      []string
	InvoiceStatuses []InvoiceStatus
	IncludeDeleted  bool
	Statuses        []InvoiceLineStatus

	LineIDs []string
}

func (g ListInvoiceLinesAdapterInput) Validate() error {
	if g.Namespace == "" {
		return errors.New("namespace is required")
	}

	return nil
}

type UpdateInvoiceLineAdapterInput Line

type UpdateInvoiceLineInput struct {
	// Mandatory fields for update
	Line LineID

	LineBase   UpdateInvoiceLineBaseInput
	UsageBased UpdateInvoiceLineUsageBasedInput
}

func (u UpdateInvoiceLineInput) Validate() error {
	var outErr error
	if err := u.LineBase.Validate(); err != nil {
		outErr = errors.Join(outErr, err)
	}

	if err := u.Line.Validate(); err != nil {
		outErr = errors.Join(outErr, fmt.Errorf("validating LineID: %w", err))
	}

	if err := u.UsageBased.Validate(); err != nil {
		outErr = errors.Join(outErr, err)
	}

	return outErr
}

func (u UpdateInvoiceLineInput) Apply(l *Line) (*Line, error) {
	l = l.Clone()

	if err := u.LineBase.Apply(l); err != nil {
		return l, err
	}

	if err := u.UsageBased.Apply(l.UsageBased); err != nil {
		return l, err
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
