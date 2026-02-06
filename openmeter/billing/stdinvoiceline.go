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

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// StandardLineBase represents the common fields for an invoice item.
type StandardLineBase struct {
	models.ManagedResource

	Metadata    models.Metadata      `json:"metadata,omitempty"`
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

func (i StandardLineBase) Equal(other StandardLineBase) bool {
	return deriveEqualLineBase(&i, &other)
}

func (i StandardLineBase) GetParentID() (string, bool) {
	if i.ParentLineID == nil {
		return "", false
	}
	return *i.ParentLineID, true
}

func (i StandardLineBase) Validate() error {
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

	if i.RateCardDiscounts.Percentage != nil {
		if err := i.RateCardDiscounts.Percentage.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("percentage discounts: %w", err))
		}
	}

	if i.RateCardDiscounts.Usage != nil {
		if err := i.RateCardDiscounts.Usage.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("usage discounts: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (i StandardLineBase) Clone() StandardLineBase {
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

func (i SubscriptionReference) Clone() *SubscriptionReference {
	return &SubscriptionReference{
		SubscriptionID: i.SubscriptionID,
		PhaseID:        i.PhaseID,
		ItemID:         i.ItemID,
		BillingPeriod:  i.BillingPeriod,
	}
}

type LineExternalIDs struct {
	Invoicing string `json:"invoicing,omitempty"`
}

func (i LineExternalIDs) Equal(other LineExternalIDs) bool {
	return i.Invoicing == other.Invoicing
}

var _ GenericInvoiceLine = (*standardInvoiceLineGenericWrapper)(nil)

// standardInvoiceLineGenericWrapper is a wrapper around a standard line that implements the GenericInvoiceLine interface.
// for methods that are present for the specific line type too.
type standardInvoiceLineGenericWrapper struct {
	*StandardLine
}

func (i standardInvoiceLineGenericWrapper) Clone() (GenericInvoiceLine, error) {
	cloned, err := i.StandardLine.Clone()
	if err != nil {
		return nil, err
	}

	return standardInvoiceLineGenericWrapper{StandardLine: cloned}, nil
}

func (i standardInvoiceLineGenericWrapper) CloneWithoutChildren() (GenericInvoiceLine, error) {
	cloned, err := i.StandardLine.CloneWithoutChildren()
	if err != nil {
		return nil, err
	}

	return standardInvoiceLineGenericWrapper{StandardLine: cloned}, nil
}

type StandardLine struct {
	StandardLineBase `json:",inline"`

	UsageBased *UsageBasedLine `json:"usageBased,omitempty"`

	DetailedLines      DetailedLines       `json:"detailedLines,omitempty"`
	SplitLineHierarchy *SplitLineHierarchy `json:"progressiveLineHierarchy,omitempty"`

	Discounts LineDiscounts `json:"discounts,omitempty"`

	DBState *StandardLine `json:"-"`
}

func (i StandardLine) GetLineID() LineID {
	return LineID{
		Namespace: i.Namespace,
		ID:        i.ID,
	}
}

func (i StandardLine) GetID() string {
	return i.ID
}

func (i StandardLine) GetManagedBy() InvoiceLineManagedBy {
	return i.ManagedBy
}

func (i StandardLine) GetAnnotations() models.Annotations {
	return i.Annotations
}

func (i *StandardLine) SetDeletedAt(at *time.Time) {
	i.DeletedAt = at
}

func (i *StandardLine) UpdateServicePeriod(fn func(p *timeutil.ClosedPeriod)) {
	period := i.Period.ToClosedPeriod()
	fn(&period)
	i.Period = Period{
		Start: period.From,
		End:   period.To,
	}
}

func (i StandardLine) GetInvoiceID() string {
	return i.InvoiceID
}

func (i StandardLine) GetChildUniqueReferenceID() *string {
	return i.ChildUniqueReferenceID
}

func (i StandardLine) AsInvoiceLine() InvoiceLine {
	return InvoiceLine{
		t:            InvoiceLineTypeStandard,
		standardLine: &i,
	}
}

// ToGatheringLineBase converts the standard line to a gathering line base.
// This is temporary until the full gathering invoice functionality is split.
func (i StandardLine) ToGatheringLineBase() (GatheringLineBase, error) {
	if i.UsageBased == nil {
		return GatheringLineBase{}, errors.New("usage based line is required")
	}

	if i.UsageBased.Price == nil {
		return GatheringLineBase{}, errors.New("usage based line price is required")
	}

	clonedMetadata := i.Metadata.Clone()

	clonedAnnotations, err := i.Annotations.Clone()
	if err != nil {
		return GatheringLineBase{}, fmt.Errorf("cloning annotations: %w", err)
	}

	return GatheringLineBase{
		ManagedResource: i.ManagedResource,
		Metadata:        clonedMetadata,
		Annotations:     clonedAnnotations,
		ManagedBy:       i.ManagedBy,
		InvoiceID:       i.InvoiceID,
		Currency:        i.Currency,
		ServicePeriod: timeutil.ClosedPeriod{
			From: i.Period.Start,
			To:   i.Period.End,
		},
		InvoiceAt:              i.InvoiceAt,
		Price:                  lo.FromPtr(i.UsageBased.Price),
		FeatureKey:             i.UsageBased.FeatureKey,
		TaxConfig:              i.TaxConfig,
		RateCardDiscounts:      i.RateCardDiscounts,
		ChildUniqueReferenceID: i.ChildUniqueReferenceID,
		Subscription:           i.Subscription,
		SplitLineGroupID:       i.SplitLineGroupID,
		UBPConfigID:            i.UsageBased.ConfigID,
	}, nil
}

type StandardLineEditFunction func(*StandardLine)

// CloneWithoutDependencies returns a clone of the line without any external dependencies. Could be used
// for creating a new line without any references to the parent or children (or config IDs).
func (i StandardLine) CloneWithoutDependencies(edits ...StandardLineEditFunction) (*StandardLine, error) {
	clone, err := i.clone(cloneOptions{
		skipDBState:   true,
		skipChildren:  true,
		skipDiscounts: true,
	})
	if err != nil {
		return nil, fmt.Errorf("cloning line: %w", err)
	}

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

	return clone, nil
}

func (i StandardLine) WithoutDBState() *StandardLine {
	i.DBState = nil
	return &i
}

func (i StandardLine) WithoutSplitLineHierarchy() *StandardLine {
	i.SplitLineHierarchy = nil
	return &i
}

func (i StandardLine) RemoveCircularReferences() (*StandardLine, error) {
	clone, err := i.Clone()
	if err != nil {
		return nil, err
	}

	clone.DBState = nil

	return clone, nil
}

// RemoveMetaForCompare returns a copy of the invoice without the fields that are not relevant for higher level
// tests that compare invoices. What gets removed:
// - Line's DB state
// - Line's dependencies are marked as resolved
// - Parent pointers are removed
func (i StandardLine) RemoveMetaForCompare() (*StandardLine, error) {
	out, err := i.Clone()
	if err != nil {
		return nil, err
	}

	out.DetailedLines = nil
	out.DBState = nil
	return out, nil
}

func (i StandardLine) Clone() (*StandardLine, error) {
	return i.clone(cloneOptions{})
}

func (i StandardLine) GetFeatureKey() string {
	if i.UsageBased == nil {
		return ""
	}

	return i.UsageBased.FeatureKey
}

func (i StandardLine) GetPrice() *productcatalog.Price {
	if i.UsageBased == nil {
		return nil
	}

	return i.UsageBased.Price
}

func (i *StandardLine) SetPrice(price productcatalog.Price) {
	if i.UsageBased == nil {
		return
	}

	i.UsageBased.Price = price.Clone()
}

func (i StandardLine) GetRateCardDiscounts() Discounts {
	return i.RateCardDiscounts
}

func (i StandardLine) GetServicePeriod() timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: i.Period.Start,
		To:   i.Period.End,
	}
}

func (i StandardLine) GetSplitLineGroupID() *string {
	return i.SplitLineGroupID
}

func (i StandardLine) GetInvoiceAt() time.Time {
	return i.InvoiceAt
}

type cloneOptions struct {
	skipDBState   bool
	skipChildren  bool
	skipDiscounts bool
}

func (i StandardLine) clone(opts cloneOptions) (*StandardLine, error) {
	res := &StandardLine{}
	if !opts.skipDBState {
		// DBStates are considered immutable, so it's safe to clone
		res.DBState = i.DBState
	}

	res.UsageBased = i.UsageBased.Clone()
	res.StandardLineBase = i.StandardLineBase.Clone()

	if !opts.skipChildren {
		res.DetailedLines = i.DetailedLines.Clone()
	}

	if !opts.skipDiscounts {
		res.Discounts = i.Discounts.Clone()
	}

	if i.SplitLineHierarchy != nil {
		cloned, err := i.SplitLineHierarchy.Clone()
		if err != nil {
			return nil, fmt.Errorf("cloning split line hierarchy: %w", err)
		}

		res.SplitLineHierarchy = lo.ToPtr(cloned)
	}

	return res, nil
}

func (i StandardLine) CloneWithoutChildren() (*StandardLine, error) {
	return i.clone(cloneOptions{
		skipChildren: true,
	})
}

func (i *StandardLine) SaveDBSnapshot() error {
	cloned, err := i.Clone()
	if err != nil {
		return err
	}

	i.DBState = cloned
	return nil
}

func (i StandardLine) Validate() error {
	var errs []error

	// Fail fast cases (most of the validation logic uses these)
	if i.UsageBased == nil {
		return errors.New("usage based line is required")
	}

	if i.UsageBased.Price == nil {
		return errors.New("usage based line price is required")
	}

	if err := i.StandardLineBase.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.Discounts.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("discounts: %w", err))
	}

	if err := i.DetailedLines.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("detailed lines: %w", err))
	}

	for _, detailedLine := range i.DetailedLines {
		if detailedLine.Currency != i.Currency {
			errs = append(errs, fmt.Errorf("detailed line[%s]: currency[%s] is not equal to line currency[%s]", detailedLine.ID, detailedLine.Currency, i.Currency))
		}
	}

	if err := i.UsageBased.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.UsageBased.Price.Type() != productcatalog.FlatPriceType {
		if i.InvoiceAt.
			Truncate(streaming.MinimumWindowSizeDuration).
			Before(i.Period.Truncate(streaming.MinimumWindowSizeDuration).End) {
			errs = append(errs, fmt.Errorf("invoice at (%s) must be after period end (%s) for usage based line", i.InvoiceAt, i.Period.Truncate(streaming.MinimumWindowSizeDuration).End))
		}

		if i.Period.Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
			errs = append(errs, ValidationError{
				Err: ErrInvoiceCreateUBPLinePeriodIsEmpty,
			})
		}
	} else {
		if i.RateCardDiscounts.Usage != nil {
			errs = append(errs, fmt.Errorf("usage discounts are not allowed for flat price lines"))
		}
	}

	if err := i.RateCardDiscounts.ValidateForPrice(i.UsageBased.Price); err != nil {
		errs = append(errs, fmt.Errorf("rateCardDiscounts: %w", err))
	}

	return errors.Join(errs...)
}

// NormalizeValues normalizes the values of the line to ensure they are matching the expected invariants:
// - Period is truncated to the minimum window size duration
// - InvoiceAt is truncated to the minimum window size duration
// - UsageBased.Price is normalized to have the default inAdvance payment term for flat prices
func (i StandardLine) WithNormalizedValues() (*StandardLine, error) {
	out, err := i.Clone()
	if err != nil {
		return nil, err
	}

	if out.UsageBased == nil {
		return nil, fmt.Errorf("usage based line is nil")
	}

	if out.UsageBased.Price == nil {
		return nil, fmt.Errorf("usage based line price is nil")
	}

	out.Period = out.Period.Truncate(streaming.MinimumWindowSizeDuration)
	out.InvoiceAt = out.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration)

	if err := setDefaultPaymentTermForFlatPrice(out.UsageBased.Price); err != nil {
		return nil, fmt.Errorf("setting default payment term for flat price: %w", err)
	}

	return out, nil
}

func setDefaultPaymentTermForFlatPrice(price *productcatalog.Price) error {
	if price.Type() != productcatalog.FlatPriceType {
		return nil
	}

	flatPrice, err := price.AsFlat()
	if err != nil {
		return err
	}

	if flatPrice.PaymentTerm == "" {
		flatPrice.PaymentTerm = productcatalog.InAdvancePaymentTerm
		*price = lo.FromPtr(productcatalog.NewPriceFrom(flatPrice))
	}

	return nil
}

// DissacociateChildren removes the Children both from the DBState and the current line, so that the
// line can be safely persisted/managed without the children.
//
// The childrens receive DBState objects, so that they can be safely persisted/managed without the parent.
func (i *StandardLine) DisassociateChildren() {
	i.DetailedLines = nil
	if i.DBState != nil {
		i.DBState.DetailedLines = nil
	}
}

func (i StandardLine) DependsOnMeteredQuantity() bool {
	return i.UsageBased.Price.Type() != productcatalog.FlatPriceType
}

func (i *StandardLine) SortDetailedLines() {
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
func NewFlatFeeLine(input NewFlatFeeLineInput, opts ...usageBasedLineOption) *StandardLine {
	ubpOptions := usageBasedLineOptions{}

	for _, opt := range opts {
		opt(&ubpOptions)
	}

	return &StandardLine{
		StandardLineBase: StandardLineBase{
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
func (c StandardLine) DetailedLinesWithIDReuse(l DetailedLines) DetailedLines {
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

type StandardLines []*StandardLine

func NewStandardLines(children []*StandardLine) StandardLines {
	// Note: this helps with test equality checks
	if len(children) == 0 {
		children = nil
	}

	return StandardLines(children)
}

func (c StandardLines) Validate() error {
	return errors.Join(lo.Map(c, func(line *StandardLine, idx int) error {
		return ValidationWithFieldPrefix(fmt.Sprintf("%d", idx), line.Validate())
	})...)
}

func (c StandardLines) GetByChildUniqueReferenceID(id string) *StandardLine {
	return lo.FindOrElse(c, nil, func(line *StandardLine) bool {
		return lo.FromPtr(line.ChildUniqueReferenceID) == id
	})
}

func (c StandardLines) Map(fn func(*StandardLine) *StandardLine) StandardLines {
	return StandardLines(
		lo.Map(c, func(l *StandardLine, _ int) *StandardLine {
			return fn(l)
		}),
	)
}

func (c *StandardLines) Sort() {
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

func (c StandardLines) GetReferencedFeatureKeys() ([]string, error) {
	out := make([]string, 0, len(c))

	for _, line := range c {
		if line.UsageBased == nil {
			return nil, fmt.Errorf("usage based line is required")
		}

		if line.UsageBased.FeatureKey == "" {
			continue
		}

		out = append(out, line.UsageBased.FeatureKey)
	}

	return lo.Uniq(out), nil
}

func (i StandardLine) SetDiscountExternalIDs(externalIDs map[string]string) []string {
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

func (i StandardLines) Clone() (StandardLines, error) {
	return slicesx.MapWithErr(i, func(line *StandardLine) (*StandardLine, error) {
		return line.Clone()
	})
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

type UpsertInvoiceLinesAdapterInput struct {
	Namespace   string
	Lines       StandardLines
	SchemaLevel int
	InvoiceID   string
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

	if c.SchemaLevel < 1 {
		return fmt.Errorf("schema level must be at least 1")
	}

	if c.InvoiceID == "" {
		return errors.New("invoice id is required")
	}

	return nil
}

type ListInvoiceLinesAdapterInput struct {
	Namespace string

	CustomerID      string
	InvoiceIDs      []string
	InvoiceStatuses []StandardInvoiceStatus
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

type GetInvoiceLineAdapterInput = LineID

type GetInvoiceLineInput = LineID

type GetInvoiceLineOwnershipAdapterInput = LineID

type DeleteInvoiceLineInput = LineID

type SnapshotLineQuantityInput struct {
	Invoice *StandardInvoice
	Line    *StandardLine
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
