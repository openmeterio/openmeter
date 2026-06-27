package billing

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ExistingLineOverride struct {
	Name        mo.Option[string]
	Description mo.Option[*string]
	Metadata    mo.Option[models.Metadata]

	Period    mo.Option[timeutil.ClosedPeriod]
	InvoiceAt mo.Option[time.Time]
	TaxConfig mo.Option[*TaxConfig]

	Price      mo.Option[*productcatalog.Price]
	FeatureKey mo.Option[string]
	Discounts  mo.Option[Discounts]
}

func (o ExistingLineOverride) Validate() error {
	var errs []error

	if o.Name.IsPresent() {
		if o.Name.OrEmpty() == "" {
			errs = append(errs, errors.New("name is required"))
		}
	}

	if o.Description.IsPresent() {
		description := o.Description.OrEmpty()
		if description != nil && *description == "" {
			errs = append(errs, errors.New("description cannot be empty"))
		}
	}

	if o.Period.IsPresent() {
		period := o.Period.OrEmpty()
		if err := period.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("period: %w", err))
		}
	}

	if o.InvoiceAt.IsPresent() {
		invoiceAt := o.InvoiceAt.OrEmpty()
		if invoiceAt.IsZero() {
			errs = append(errs, errors.New("invoice at is required"))
		}
	}

	if o.TaxConfig.IsPresent() {
		taxConfig := o.TaxConfig.OrEmpty()
		if taxConfig != nil {
			if err := taxConfig.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("tax config: %w", err))
			}
		}
	}

	if o.Price.IsPresent() {
		price := o.Price.OrEmpty()
		if price == nil {
			errs = append(errs, errors.New("price is required, when overridden on an existing line"))
		} else {
			if err := price.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("price: %w", err))
			}
		}
	}

	return errors.Join(errs...)
}

func (o ExistingLineOverride) IsPresent() bool {
	return o.Name.IsPresent() ||
		o.Description.IsPresent() ||
		o.Metadata.IsPresent() ||
		o.Period.IsPresent() ||
		o.InvoiceAt.IsPresent() ||
		o.TaxConfig.IsPresent() ||
		o.Price.IsPresent() ||
		o.FeatureKey.IsPresent() ||
		o.Discounts.IsPresent()
}

type InvoiceLineOverride struct {
	ExistingLine   GenericInvoiceLine
	ChangesToApply ExistingLineOverride
}

func (o InvoiceLineOverride) Validate() error {
	var errs []error

	if o.ExistingLine == nil {
		return errors.New("existing line is required")
	}

	if err := o.ExistingLine.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("existing line: %w", err))
	}

	if err := o.ChangesToApply.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("changes to apply: %w", err))
	}

	if !o.ChangesToApply.IsPresent() {
		errs = append(errs, errors.New("changes to apply is empty"))
	}

	return errors.Join(errs...)
}

type InvoiceLineOverrides []InvoiceLineOverride

func (o InvoiceLineOverrides) Validate() error {
	var errs []error

	if len(o) == 0 {
		errs = append(errs, errors.New("line overrides are required"))
	}

	errs = append(errs, lo.Map(o, func(override InvoiceLineOverride, idx int) error {
		if err := override.Validate(); err != nil {
			return fmt.Errorf("lineOverrides[%d]: %w", idx, err)
		}

		return nil
	})...)

	return errors.Join(errs...)
}

func (o InvoiceLineOverrides) Lines() []GenericInvoiceLine {
	lines := make([]GenericInvoiceLine, 0, len(o))
	for _, override := range o {
		lines = append(lines, override.ExistingLine)
	}

	return lines
}

type DefaultTaxCodeResolver func(context.Context) (string, error)

type DefaultTaxCodeResolvers struct {
	Invoicing   DefaultTaxCodeResolver
	CreditGrant DefaultTaxCodeResolver
}

func (r DefaultTaxCodeResolvers) Validate() error {
	var errs []error

	if r.Invoicing == nil {
		errs = append(errs, errors.New("invoicing default tax code resolver is required"))
	}

	if r.CreditGrant == nil {
		errs = append(errs, errors.New("credit grant default tax code resolver is required"))
	}

	return errors.Join(errs...)
}

type OnMutableInvoiceUpdateInput struct {
	Invoice GenericInvoiceReader

	// DefaultTaxCodeResolvers lazily resolve invoice-context tax-code defaults
	// for API invoice-line edits. They are not part of edited line state; line
	// engines use them only when their downstream charge intent requires a tax
	// code ID and the edited line did not provide one.
	DefaultTaxCodeResolvers DefaultTaxCodeResolvers

	Created []GenericInvoiceLine
	Updated InvoiceLineOverrides
	Deleted []GenericInvoiceLine
}

func (i OnMutableInvoiceUpdateInput) IsEmpty() bool {
	return len(i.Created) == 0 && len(i.Updated) == 0 && len(i.Deleted) == 0
}

func (i OnMutableInvoiceUpdateInput) Validate() error {
	var errs []error

	if i.IsEmpty() {
		errs = append(errs, fmt.Errorf("line changes are required"))
	}

	if err := i.DefaultTaxCodeResolvers.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("default tax code resolvers: %w", err))
	}

	errs = append(errs, lo.Map(i.Created, func(line GenericInvoiceLine, idx int) error {
		if line == nil {
			return fmt.Errorf("created[%d]: line is nil", idx)
		}

		if err := line.Validate(); err != nil {
			return fmt.Errorf("created[%d]: %w", idx, err)
		}

		return nil
	})...)

	if len(i.Updated) > 0 {
		if err := i.Updated.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("updated: %w", err))
		}
	}

	if len(i.Deleted) > 0 {
		errs = append(errs, lo.Map(i.Deleted, func(line GenericInvoiceLine, idx int) error {
			if line == nil {
				return fmt.Errorf("deleted[%d]: line is nil", idx)
			}

			if err := line.Validate(); err != nil {
				return fmt.Errorf("deleted[%d]: %w", idx, err)
			}

			return nil
		})...)
	}

	return errors.Join(errs...)
}

// GroupByLineEngine groups the input's changes by line engine.
func (i OnMutableInvoiceUpdateInput) GroupByLineEngine() (map[LineEngineType]OnMutableInvoiceUpdateInput, error) {
	out := make(map[LineEngineType]OnMutableInvoiceUpdateInput)

	for _, line := range i.Created {
		if line.GetEngine() == "" {
			return nil, fmt.Errorf("line[%s]: line engine is required for created line", line.GetID())
		}
	}

	for _, override := range i.Updated {
		if override.ExistingLine.GetEngine() == "" {
			return nil, fmt.Errorf("line[%s]: line engine is required for updated line", override.ExistingLine.GetID())
		}
	}

	for _, line := range i.Deleted {
		if line.GetEngine() == "" {
			return nil, fmt.Errorf("line[%s]: line engine is required for deleted line", line.GetID())
		}
	}

	createsByLineEngine := lo.GroupBy(i.Created, func(line GenericInvoiceLine) LineEngineType {
		return line.GetEngine()
	})

	updatesByLineEngine := lo.GroupBy(i.Updated, func(override InvoiceLineOverride) LineEngineType {
		return override.ExistingLine.GetEngine()
	})

	deletesByLineEngine := lo.GroupBy(i.Deleted, func(line GenericInvoiceLine) LineEngineType {
		return line.GetEngine()
	})

	presentLineEngines := lo.Uniq(
		slices.Concat(
			lo.Keys(createsByLineEngine),
			lo.Keys(updatesByLineEngine),
			lo.Keys(deletesByLineEngine),
		),
	)

	for _, engine := range presentLineEngines {
		out[engine] = OnMutableInvoiceUpdateInput{
			Invoice:                 i.Invoice,
			DefaultTaxCodeResolvers: i.DefaultTaxCodeResolvers,
			Created:                 createsByLineEngine[engine],
			Updated:                 updatesByLineEngine[engine],
			Deleted:                 deletesByLineEngine[engine],
		}

		if err := out[engine].Validate(); err != nil {
			return nil, fmt.Errorf("validating mutable invoice update input for engine %s: %w", engine, err)
		}
	}

	return out, nil
}

type OnMutableInvoiceUpdateResult struct {
	// CreatedLines must contain exactly one line for each input Created line.
	CreatedLines []GenericInvoiceLine
	// UpdatedLines must contain exactly one line for each input Updated override.
	UpdatedLines []GenericInvoiceLine
}

func (o ExistingLineOverride) Apply(line GenericInvoiceLine) (GenericInvoiceLine, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}

	if line == nil {
		return nil, errors.New("line is required")
	}

	clonedLine, err := line.Clone()
	if err != nil {
		return nil, fmt.Errorf("cloning line: %w", err)
	}

	switch clonedLine.AsInvoiceLine().Type() {
	case InvoiceLineTypeStandard:
		standardLine, err := clonedLine.AsInvoiceLine().AsStandardLine()
		if err != nil {
			return nil, err
		}

		if err := applyExistingLineOverrideToStandardLine(o, &standardLine); err != nil {
			return nil, err
		}

		return standardLine.AsGenericLine(), nil
	case InvoiceLineTypeGathering:
		gatheringLine, err := clonedLine.AsInvoiceLine().AsGatheringLine()
		if err != nil {
			return nil, err
		}

		applyExistingLineOverrideToGatheringLine(o, &gatheringLine)

		return gatheringLine.AsGenericLine(), nil
	default:
		return nil, fmt.Errorf("unsupported invoice line type: %s", clonedLine.AsInvoiceLine().Type())
	}
}

func applyExistingLineOverrideToStandardLine(o ExistingLineOverride, line *StandardLine) error {
	if val, ok := o.Name.Get(); ok {
		line.Name = val
	}

	if val, ok := o.Description.Get(); ok {
		line.Description = val
	}

	if val, ok := o.Metadata.Get(); ok {
		line.Metadata = maps.Clone(val)
	}

	if val, ok := o.Period.Get(); ok {
		line.Period = val
	}

	// InvoiceAt is not meaningful for a standard line thus ignored

	if val, ok := o.TaxConfig.Get(); ok {
		line.TaxConfig = val
	}

	if (o.Price.IsPresent() || o.FeatureKey.IsPresent()) && line.UsageBased == nil {
		return errors.New("usage based line is required for price or feature key override")
	}

	if val, ok := o.Price.Get(); ok {
		line.UsageBased.Price = val.Clone()
	}

	if val, ok := o.FeatureKey.Get(); ok {
		line.UsageBased.FeatureKey = val
	}

	if val, ok := o.Discounts.Get(); ok {
		line.RateCardDiscounts = val.Clone()
	}

	return nil
}

func applyExistingLineOverrideToGatheringLine(o ExistingLineOverride, line *GatheringLine) {
	if val, ok := o.Name.Get(); ok {
		line.Name = val
	}

	if val, ok := o.Description.Get(); ok {
		line.Description = val
	}

	if val, ok := o.Metadata.Get(); ok {
		line.Metadata = maps.Clone(val)
	}

	if val, ok := o.Period.Get(); ok {
		line.ServicePeriod = val
	}

	if val, ok := o.InvoiceAt.Get(); ok {
		line.InvoiceAt = val
	}

	if val, ok := o.TaxConfig.Get(); ok {
		line.TaxConfig = val.ToProductCatalog()
	}

	if val, ok := o.Price.Get(); ok {
		line.Price = *val
	}

	if val, ok := o.FeatureKey.Get(); ok {
		line.FeatureKey = val
	}

	if val, ok := o.Discounts.Get(); ok {
		line.RateCardDiscounts = val.Clone()
	}
}

func (l LineWithInvoiceHeader) Validate() error {
	if l.Line == nil {
		return errors.New("line is required")
	}

	if err := l.Line.Validate(); err != nil {
		return fmt.Errorf("line: %w", err)
	}

	if l.Invoice == nil {
		return errors.New("invoice is required")
	}

	if l.Invoice.GetID() == "" {
		return errors.New("invoice id is required")
	}

	return nil
}
