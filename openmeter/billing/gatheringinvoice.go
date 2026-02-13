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
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/expand"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
	timeutil "github.com/openmeterio/openmeter/pkg/timeutil"
)

type GatheringInvoiceBase struct {
	models.ManagedResource

	Metadata models.Metadata `json:"metadata"`

	Number        string                `json:"number"`
	CustomerID    string                `json:"customerID"`
	Currency      currencyx.Code        `json:"currency"`
	ServicePeriod timeutil.ClosedPeriod `json:"servicePeriod"`

	NextCollectionAt time.Time `json:"nextCollectionAt"`

	SchemaLevel int `json:"schemaLevel"`
}

func (g GatheringInvoiceBase) Validate() error {
	var errs []error

	if g.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if g.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := g.Currency.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := g.ServicePeriod.Validate(); err != nil {
		errs = append(errs, err)
	}

	if g.SchemaLevel == 0 {
		errs = append(errs, errors.New("schema level is required"))
	}

	return errors.Join(errs...)
}

var _ GenericInvoice = (*GatheringInvoice)(nil)

type GatheringInvoice struct {
	GatheringInvoiceBase `json:",inline"`

	// Entities external to the invoice entity
	Lines GatheringInvoiceLines `json:"lines,omitempty"`

	// TODO[later]: implement this once we have a lineservice capable of operating on
	// these lines too.
	AvailableActions *GatheringInvoiceAvailableActions `json:"availableActions,omitempty"`

	SplitLineHierarchy *SplitLineHierarchy `json:"splitLineHierarchy,omitempty"`
}

func (g GatheringInvoice) WithoutDBState() (GatheringInvoice, error) {
	clone, err := g.Clone()
	if err != nil {
		return GatheringInvoice{}, fmt.Errorf("cloning invoice: %w", err)
	}

	clone.Lines, err = clone.Lines.MapWithErr(func(l GatheringLine) (GatheringLine, error) {
		return l.WithoutDBState()
	})
	if err != nil {
		return GatheringInvoice{}, fmt.Errorf("cloning lines: %w", err)
	}

	return clone, nil
}

func (g GatheringInvoice) GetID() string {
	return g.ID
}

func (g GatheringInvoice) GetInvoiceID() InvoiceID {
	return InvoiceID{
		Namespace: g.Namespace,
		ID:        g.ID,
	}
}

func (g GatheringInvoice) GetDeletedAt() *time.Time {
	return g.DeletedAt
}

func (g GatheringInvoice) AsInvoice() Invoice {
	return Invoice{
		t:                InvoiceTypeGathering,
		gatheringInvoice: &g,
	}
}

func (g GatheringInvoice) Validate() error {
	var errs []error

	if err := g.GatheringInvoiceBase.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := g.Lines.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (g GatheringInvoice) GetCustomerID() customer.CustomerID {
	return customer.CustomerID{
		Namespace: g.Namespace,
		ID:        g.CustomerID,
	}
}

func (g *GatheringInvoice) SortLines() {
	if !g.Lines.IsPresent() {
		return
	}

	g.Lines.Sort()
}

func (g GatheringInvoice) Clone() (GatheringInvoice, error) {
	clone := g

	clone.Metadata = g.Metadata.Clone()

	clonedLines, err := clone.Lines.MapWithErr(func(l GatheringLine) (GatheringLine, error) {
		return l.Clone()
	})
	if err != nil {
		return GatheringInvoice{}, fmt.Errorf("cloning lines: %w", err)
	}

	clone.Lines = clonedLines

	return clone, nil
}

func (g GatheringInvoice) GetGenericLines() mo.Option[[]GenericInvoiceLine] {
	if !g.Lines.IsPresent() {
		return mo.None[[]GenericInvoiceLine]()
	}

	return mo.Some(lo.Map(g.Lines.OrEmpty(), func(l GatheringLine, _ int) GenericInvoiceLine {
		return &gatheringInvoiceLineGenericWrapper{GatheringLine: l}
	}))
}

func (g *GatheringInvoice) SetLines(lines []GenericInvoiceLine) error {
	mappedLines, err := slicesx.MapWithErr(lines, func(l GenericInvoiceLine) (GatheringLine, error) {
		return l.AsInvoiceLine().AsGatheringLine()
	})
	if err != nil {
		return fmt.Errorf("mapping lines: %w", err)
	}

	g.Lines = NewGatheringInvoiceLines(mappedLines)
	return nil
}

type GatheringInvoiceExpand string

const (
	GatheringInvoiceExpandLines              GatheringInvoiceExpand = "lines"
	GatheringInvoiceExpandDeletedLines       GatheringInvoiceExpand = "deletedLines"
	GatheringInvoiceExpandAvailableActions   GatheringInvoiceExpand = "availableActions"
	GatheringInvoiceExpandSplitLineHierarchy GatheringInvoiceExpand = "splitLineHierarchy"
)

func (e GatheringInvoiceExpand) Values() []GatheringInvoiceExpand {
	return []GatheringInvoiceExpand{
		GatheringInvoiceExpandLines,
		GatheringInvoiceExpandDeletedLines,
		GatheringInvoiceExpandAvailableActions,
		GatheringInvoiceExpandSplitLineHierarchy,
	}
}

type GatheringInvoiceExpands = expand.Expand[GatheringInvoiceExpand]

var GatheringInvoiceExpandAll = GatheringInvoiceExpands(GatheringInvoiceExpand("").Values())

func NewGatheringInvoiceExpands(values ...GatheringInvoiceExpand) GatheringInvoiceExpands {
	return GatheringInvoiceExpands(values)
}

type GatheringInvoiceAvailableActions struct {
	CanBeInvoiced bool `json:"canBeInvoiced"`
}

type GatheringLines []GatheringLine

func (l GatheringLines) Validate() error {
	return errors.Join(
		lo.Map(l, func(l GatheringLine, _ int) error {
			err := l.Validate()
			if err != nil {
				return fmt.Errorf("line[%s]: %w", l.ID, err)
			}
			return nil
		})...,
	)
}

func (l GatheringLines) AsGenericLines() []GenericInvoiceLine {
	return lo.Map(l, func(l GatheringLine, _ int) GenericInvoiceLine {
		return &gatheringInvoiceLineGenericWrapper{GatheringLine: l}
	})
}

type GatheringInvoiceLines struct {
	mo.Option[GatheringLines]
}

func (l GatheringInvoiceLines) Validate() error {
	if l.IsAbsent() {
		return nil
	}

	return l.OrEmpty().Validate()
}

func (l *GatheringInvoiceLines) Sort() {
	if l.IsAbsent() {
		return
	}

	lines := l.OrEmpty()
	slices.SortFunc(lines, func(a, b GatheringLine) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	l.Option = mo.Some(lines)
}

func (l GatheringInvoiceLines) NonDeletedLineCount() int {
	return lo.CountBy(l.OrEmpty(), func(l GatheringLine) bool {
		return l.DeletedAt == nil
	})
}

func (l GatheringInvoiceLines) Map(fn func(GatheringLine) GatheringLine) GatheringInvoiceLines {
	res, _ := l.MapWithErr(func(gl GatheringLine) (GatheringLine, error) {
		return fn(gl), nil
	})

	return res
}

func (l GatheringInvoiceLines) MapWithErr(fn func(GatheringLine) (GatheringLine, error)) (GatheringInvoiceLines, error) {
	if l.IsAbsent() {
		return l, nil
	}

	out, err := slicesx.MapWithErr(l.OrEmpty(), fn)
	if err != nil {
		return l, err
	}

	return GatheringInvoiceLines{
		Option: mo.Some(GatheringLines(out)),
	}, nil
}

func (l GatheringInvoiceLines) WithNormalizedValues() (GatheringInvoiceLines, error) {
	return l.MapWithErr(func(gl GatheringLine) (GatheringLine, error) {
		return gl.WithNormalizedValues()
	})
}

func (l *GatheringInvoiceLines) Append(lines ...GatheringLine) {
	l.Option = mo.Some(append(l.OrEmpty(), lines...))
}

func (l GatheringInvoiceLines) GetReferencedFeatureKeys() ([]string, error) {
	if l.IsAbsent() {
		return nil, nil
	}

	keys := make([]string, 0, len(l.OrEmpty()))
	for _, line := range l.OrEmpty() {
		if line.FeatureKey == "" {
			continue
		}

		keys = append(keys, line.FeatureKey)
	}

	return lo.Uniq(keys), nil
}

func (l GatheringInvoiceLines) GetByID(id string) (GatheringLine, bool) {
	if l.IsAbsent() {
		return GatheringLine{}, false
	}

	lines := l.OrEmpty()
	for _, line := range lines {
		if line.ID == id {
			return line, true
		}
	}

	return GatheringLine{}, false
}

func (l *GatheringInvoiceLines) ReplaceByID(line GatheringLine) error {
	if l.IsAbsent() {
		return fmt.Errorf("lines are absent")
	}

	lines := l.OrEmpty()
	for i := range lines {
		if lines[i].ID == line.ID {
			lines[i] = line
			return nil
		}
	}

	return fmt.Errorf("line[%s]: line not found", line.ID)
}

func NewGatheringInvoiceLines(children []GatheringLine) GatheringInvoiceLines {
	return GatheringInvoiceLines{
		Option: mo.Some(GatheringLines(children)),
	}
}

type GatheringLineBase struct {
	models.ManagedResource

	Metadata    models.Metadata      `json:"metadata"`
	Annotations models.Annotations   `json:"annotations"`
	ManagedBy   InvoiceLineManagedBy `json:"managedBy"`
	InvoiceID   string               `json:"invoiceID"`

	Currency      currencyx.Code        `json:"currency"`
	ServicePeriod timeutil.ClosedPeriod `json:"period"`
	InvoiceAt     time.Time             `json:"invoiceAt"`
	Price         productcatalog.Price  `json:"price"`
	FeatureKey    string                `json:"featureKey"`

	TaxConfig         *productcatalog.TaxConfig `json:"taxOverrides,omitempty"`
	RateCardDiscounts Discounts                 `json:"rateCardDiscounts,omitempty"`

	ChildUniqueReferenceID *string                `json:"childUniqueReferenceID,omitempty"`
	Subscription           *SubscriptionReference `json:"subscription,omitempty"`
	SplitLineGroupID       *string                `json:"splitLineGroupID,omitempty"`

	// TODO: Remove once we have dedicated db field for gathering invoice lines
	UBPConfigID string `json:"ubpConfigID"`
}

func (i GatheringLineBase) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if i.InvoiceAt.IsZero() {
		errs = append(errs, errors.New("invoice at is required"))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if !slices.Contains(InvoiceLineManagedBy("").Values(), string(i.ManagedBy)) {
		errs = append(errs, fmt.Errorf("invalid managed by %s", i.ManagedBy))
	}

	if i.Subscription != nil {
		if err := i.Subscription.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("subscription: %w", err))
		}
	}

	if i.TaxConfig != nil {
		if err := i.TaxConfig.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("tax config: %w", err))
		}
	}

	if err := i.Price.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("price: %w", err))
	}

	if err := i.RateCardDiscounts.ValidateForPrice(i.Price); err != nil {
		errs = append(errs, fmt.Errorf("rate card discounts: %w", err))
	}

	if i.ChildUniqueReferenceID != nil && *i.ChildUniqueReferenceID == "" {
		errs = append(errs, errors.New("child unique reference id is required"))
	}

	if i.Price.Type() != productcatalog.FlatPriceType && i.FeatureKey == "" {
		errs = append(errs, errors.New("feature key is required for non-flat prices"))
	}

	return errors.Join(errs...)
}

func (i *GatheringLineBase) NormalizeValues() error {
	i.ServicePeriod = i.ServicePeriod.Truncate(streaming.MinimumWindowSizeDuration)
	i.InvoiceAt = i.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration)

	if err := setDefaultPaymentTermForFlatPrice(&i.Price); err != nil {
		return fmt.Errorf("setting default payment term for flat price: %w", err)
	}

	return nil
}

func (i GatheringLineBase) Clone() (GatheringLineBase, error) {
	var err error

	out := i

	out.Annotations, err = i.Annotations.Clone()
	if err != nil {
		return GatheringLineBase{}, fmt.Errorf("cloning annotations: %w", err)
	}

	out.Metadata = i.Metadata.Clone()

	if i.TaxConfig != nil {
		out.TaxConfig = &productcatalog.TaxConfig{}
		*out.TaxConfig = *i.TaxConfig
	}

	if i.Subscription != nil {
		out.Subscription = &SubscriptionReference{}
		*out.Subscription = *i.Subscription
	}

	return out, nil
}

func (i GatheringLineBase) GetFeatureKey() string {
	return i.FeatureKey
}

func (i GatheringLineBase) GetServicePeriod() timeutil.ClosedPeriod {
	return i.ServicePeriod
}

func (i GatheringLineBase) GetPrice() productcatalog.Price {
	return i.Price
}

func (i *GatheringLineBase) SetPrice(price productcatalog.Price) {
	i.Price = price
}

func (i GatheringLineBase) GetID() string {
	return i.ID
}

func (i GatheringLineBase) GetInvoiceAt() time.Time {
	return i.InvoiceAt
}

func (g *GatheringLineBase) SetInvoiceAt(at time.Time) {
	g.InvoiceAt = at
}

func (i GatheringLineBase) GetChildUniqueReferenceID() *string {
	return i.ChildUniqueReferenceID
}

func (i *GatheringLineBase) SetChildUniqueReferenceID(id *string) {
	i.ChildUniqueReferenceID = id
}

func (i GatheringLineBase) GetSplitLineGroupID() *string {
	return i.SplitLineGroupID
}

func (g GatheringLineBase) GetLineID() LineID {
	return LineID{
		Namespace: g.Namespace,
		ID:        g.ID,
	}
}

func (g GatheringLineBase) GetManagedBy() InvoiceLineManagedBy {
	return g.ManagedBy
}

func (g GatheringLineBase) GetAnnotations() models.Annotations {
	return g.Annotations
}

func (g *GatheringLineBase) SetDeletedAt(at *time.Time) {
	g.DeletedAt = at
}

func (g *GatheringLineBase) UpdateServicePeriod(fn func(p *timeutil.ClosedPeriod)) {
	fn(&g.ServicePeriod)
}

func (g GatheringLineBase) GetInvoiceID() string {
	return g.InvoiceID
}

func (g GatheringLineBase) GetRateCardDiscounts() Discounts {
	return g.RateCardDiscounts
}

func (g GatheringLineBase) Equal(other GatheringLineBase) bool {
	return deriveEqualGatheringLineBase(&g, &other)
}

func (g GatheringLineBase) GetSubscriptionReference() *SubscriptionReference {
	if g.Subscription == nil {
		return nil
	}

	return g.Subscription.Clone()
}

var (
	_ GenericInvoiceLine = (*gatheringInvoiceLineGenericWrapper)(nil)
	_ InvoiceAtAccessor  = (*gatheringInvoiceLineGenericWrapper)(nil)
)

// gatheringInvoiceLineGenericWrapper is a wrapper around a gathering line that implements the GenericInvoiceLine interface.
// for methods that are present for the specific line type too.
type gatheringInvoiceLineGenericWrapper struct {
	GatheringLine
}

func (i gatheringInvoiceLineGenericWrapper) Clone() (GenericInvoiceLine, error) {
	cloned, err := i.GatheringLine.Clone()
	if err != nil {
		return nil, err
	}

	return &gatheringInvoiceLineGenericWrapper{GatheringLine: cloned}, nil
}

func (i gatheringInvoiceLineGenericWrapper) CloneWithoutChildren() (GenericInvoiceLine, error) {
	// Gathering lines don't have children, so we can just clone the line (db state is preserved as with the standard lines)
	return i.Clone()
}

type GatheringLine struct {
	GatheringLineBase `json:",inline"`

	DBState            *GatheringLine      `json:"-"`
	SplitLineHierarchy *SplitLineHierarchy `json:"splitLineHierarchy,omitempty"`
}

func (g GatheringLine) Clone() (GatheringLine, error) {
	base, err := g.GatheringLineBase.Clone()
	if err != nil {
		return GatheringLine{}, fmt.Errorf("cloning line base: %w", err)
	}

	return GatheringLine{
		GatheringLineBase: base,
		DBState:           g.DBState,
	}, nil
}

func (i GatheringLine) CloneForCreate(edits ...func(*GatheringLine)) (GatheringLine, error) {
	clone, err := i.Clone()
	if err != nil {
		return GatheringLine{}, fmt.Errorf("cloning line: %w", err)
	}

	clone.ID = ""
	clone.UBPConfigID = ""
	clone.CreatedAt = time.Time{}
	clone.UpdatedAt = time.Time{}
	clone.DeletedAt = nil
	clone.DBState = nil

	for _, edit := range edits {
		edit(&clone)
	}

	return clone, nil
}

func (g GatheringLine) WithoutDBState() (GatheringLine, error) {
	clone, err := g.Clone()
	if err != nil {
		return GatheringLine{}, fmt.Errorf("cloning line: %w", err)
	}

	clone.DBState = nil
	return clone, nil
}

func (g GatheringLine) WithNormalizedValues() (GatheringLine, error) {
	clone, err := g.Clone()
	if err != nil {
		return GatheringLine{}, fmt.Errorf("cloning line: %w", err)
	}

	if err := clone.GatheringLineBase.NormalizeValues(); err != nil {
		return GatheringLine{}, fmt.Errorf("normalizing line values: %w", err)
	}

	return clone, nil
}

func (g GatheringLine) AsInvoiceLine() InvoiceLine {
	return InvoiceLine{
		t:             InvoiceLineTypeGathering,
		gatheringLine: &g,
	}
}

func (g GatheringLine) Equal(other GatheringLine) bool {
	return g.GatheringLineBase.Equal(other.GatheringLineBase)
}

func (g GatheringLine) RemoveMetaForCompare() (GatheringLine, error) {
	return g.WithoutDBState()
}

func (g *GatheringLine) SetSplitLineHierarchy(hierarchy *SplitLineHierarchy) {
	g.SplitLineHierarchy = hierarchy
}

type CreatePendingInvoiceLinesInput struct {
	Customer customer.CustomerID `json:"customer"`
	Currency currencyx.Code      `json:"currency"`

	Lines []GatheringLine `json:"lines"`
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

		if line.SplitLineGroupID != nil {
			errs = append(errs, fmt.Errorf("line.%d: split line group ID is not allowed for pending lines", id))
		}
	}

	return errors.Join(errs...)
}

type CreatePendingInvoiceLinesResult struct {
	Lines        []GatheringLine
	Invoice      GatheringInvoice
	IsInvoiceNew bool
}

type CreateGatheringInvoiceAdapterInput struct {
	Namespace string
	Number    string
	Currency  currencyx.Code
	Metadata  map[string]string

	Description      *string
	NextCollectionAt *time.Time

	// TODO[later]: This should be just a CustomerID once we have split the invoices table
	Customer      customer.Customer
	MergedProfile Profile
}

func (c CreateGatheringInvoiceAdapterInput) Validate() error {
	var errs []error

	if c.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := c.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if c.Number == "" {
		errs = append(errs, errors.New("number is required"))
	}

	if err := c.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if err := c.MergedProfile.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("merged profile: %w", err))
	}

	return errors.Join(errs...)
}

type DeleteGatheringInvoiceAdapterInput = InvoiceID

type UpdateGatheringInvoiceAdapterInput = GatheringInvoice

type ListGatheringInvoicesInput struct {
	pagination.Page

	Namespaces      []string
	IDs             []string
	Customers       []string
	Currencies      []currencyx.Code
	OrderBy         api.InvoiceOrderBy
	Order           sortx.Order
	IncludeDeleted  bool
	Expand          GatheringInvoiceExpands
	CollectionAtLTE *time.Time
}

func (i ListGatheringInvoicesInput) Validate() error {
	var errs []error

	if !lo.IsEmpty(i.Page) {
		if err := i.Page.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("page: %w", err))
		}
	}

	if err := i.Expand.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expand: %w", err))
	}

	return errors.Join(errs...)
}

func NewFlatFeeGatheringLine(input NewFlatFeeLineInput, opts ...usageBasedLineOption) GatheringLine {
	ubpOptions := usageBasedLineOptions{}

	for _, opt := range opts {
		opt(&ubpOptions)
	}

	return GatheringLine{
		GatheringLineBase: GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   input.Namespace,
				ID:          input.ID,
				CreatedAt:   input.CreatedAt,
				UpdatedAt:   input.UpdatedAt,
				Name:        input.Name,
				Description: input.Description,
			}),
			ServicePeriod: timeutil.ClosedPeriod{
				From: input.Period.Start,
				To:   input.Period.End,
			},
			InvoiceAt: input.InvoiceAt,
			InvoiceID: input.InvoiceID,

			Metadata:    input.Metadata,
			Annotations: input.Annotations,

			ManagedBy: lo.CoalesceOrEmpty(input.ManagedBy, SystemManagedLine),

			Currency:          input.Currency,
			RateCardDiscounts: input.RateCardDiscounts,
			Price: lo.FromPtr(
				productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      input.PerUnitAmount,
					PaymentTerm: input.PaymentTerm,
				}),
			),
			FeatureKey: ubpOptions.featureKey,
		},
	}
}

type GetGatheringInvoiceByIdInput struct {
	Invoice InvoiceID
	Expand  GatheringInvoiceExpands
}

func (i GetGatheringInvoiceByIdInput) Validate() error {
	var errs []error

	if err := i.Invoice.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invoice: %w", err))
	}

	if err := i.Expand.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expand: %w", err))
	}

	return errors.Join(errs...)
}

type UpdateGatheringInvoiceInput struct {
	Invoice InvoiceID
	EditFn  func(*GatheringInvoice) error
	// IncludeDeletedLines signals the update to populate the deleted lines into the lines field, for the edit function
	IncludeDeletedLines bool
}

func (i UpdateGatheringInvoiceInput) Validate() error {
	if err := i.Invoice.Validate(); err != nil {
		return fmt.Errorf("id: %w", err)
	}

	if i.EditFn == nil {
		return errors.New("edit function is required")
	}

	return nil
}
