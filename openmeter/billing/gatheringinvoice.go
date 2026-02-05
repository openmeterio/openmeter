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

type GatheringInvoice struct {
	GatheringInvoiceBase `json:",inline"`

	// Entities external to the invoice entity
	Lines GatheringInvoiceLines `json:"lines,omitempty"`

	// TODO[later]: implement this once we have a lineservice capable of operating on
	// these lines too.
	AvailableActions *GatheringInvoiceAvailableActions `json:"availableActions,omitempty"`
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

func (g GatheringInvoice) InvoiceID() InvoiceID {
	return InvoiceID{
		Namespace: g.Namespace,
		ID:        g.ID,
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

type GatheringInvoiceExpand string

func (e GatheringInvoiceExpand) Validate() error {
	if slices.Contains(GatheringInvoiceExpandValues, e) {
		return nil
	}

	return fmt.Errorf("invalid gathering invoice expand: %s", e)
}

const (
	GatheringInvoiceExpandLines            GatheringInvoiceExpand = "lines"
	GatheringInvoiceExpandAvailableActions GatheringInvoiceExpand = "availableActions"
)

var GatheringInvoiceExpandValues = []GatheringInvoiceExpand{
	GatheringInvoiceExpandLines,
	GatheringInvoiceExpandAvailableActions,
}

type GatheringInvoiceExpands []GatheringInvoiceExpand

func (e GatheringInvoiceExpands) Validate() error {
	for _, expand := range e {
		if err := expand.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (e GatheringInvoiceExpands) Has(expand GatheringInvoiceExpand) bool {
	return slices.Contains(e, expand)
}

func (e GatheringInvoiceExpands) With(expand GatheringInvoiceExpand) GatheringInvoiceExpands {
	return append(e, expand)
}

type GatheringInvoiceAvailableActions struct {
	CanBeInvoiced bool `json:"canBeInvoiced"`
}

type GatheringLines []GatheringLine

func (l GatheringLines) GetReferencedFeatureKeys() ([]string, error) {
	out := make([]string, 0, len(l))
	for _, line := range l {

		if line.FeatureKey == "" {
			if line.Price.Type() != productcatalog.FlatPriceType {
				return nil, fmt.Errorf("line[%s]: feature key is required for non-flat prices", line.ID)
			}

			continue
		}

		out = append(out, line.FeatureKey)
	}

	return lo.Uniq(out), nil
}

type GatheringInvoiceLines struct {
	mo.Option[GatheringLines]
}

func (l GatheringInvoiceLines) Validate() error {
	if l.IsAbsent() {
		return nil
	}

	return errors.Join(
		lo.Map(l.OrEmpty(), func(l GatheringLine, _ int) error {
			err := l.Validate()
			if err != nil {
				return fmt.Errorf("line[%s]: %w", l.ID, err)
			}
			return nil
		})...,
	)
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

func (l *GatheringInvoiceLines) Append(lines ...GatheringLine) {
	l.Option = mo.Some(append(l.OrEmpty(), lines...))
}

func (l GatheringInvoiceLines) GetByID(id string) *GatheringLine {
	lines := l.OrEmpty()

	for idx := range lines {
		if lines[idx].ID == id {
			return &lines[idx]
		}
	}

	return nil
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

	if err := i.RateCardDiscounts.ValidateForPrice(&i.Price); err != nil {
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

// TODO: rename to GatheringLine
type GatheringLine struct {
	GatheringLineBase `json:",inline"`

	DBState *GatheringLine `json:"-"`
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

func (g GatheringLine) CloneAsNew(edits ...func(*GatheringLine)) (GatheringLine, error) {
	cloned, err := g.Clone()
	if err != nil {
		return GatheringLine{}, fmt.Errorf("cloning line: %w", err)
	}

	cloned.ID = ""
	cloned.CreatedAt = time.Time{}
	cloned.UpdatedAt = time.Time{}
	cloned.DeletedAt = nil
	cloned.DBState = nil

	for _, edit := range edits {
		edit(&cloned)
	}

	return cloned, nil
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

func (g GatheringLine) GetPrice() *productcatalog.Price {
	return g.Price.Clone()
}

func (g GatheringLine) GetServicePeriod() timeutil.ClosedPeriod {
	return g.ServicePeriod
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

	Namespaces     []string
	Customers      []string
	Currencies     []currencyx.Code
	OrderBy        api.InvoiceOrderBy
	Order          sortx.Order
	IncludeDeleted bool
	Expand         GatheringInvoiceExpands
}

func (i ListGatheringInvoicesInput) Validate() error {
	var errs []error

	if err := i.Page.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("page: %w", err))
	}

	if len(i.Namespaces) == 0 {
		errs = append(errs, errors.New("namespaces is required"))
	}

	for _, expand := range i.Expand {
		if err := expand.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("expand: %w", err))
		}
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

	for _, expand := range i.Expand {
		if err := expand.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("expand: %w", err))
		}
	}
	return errors.Join(errs...)
}
