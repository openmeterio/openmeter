package lineservice

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type featureCacheItem struct {
	feature feature.Feature
	meter   meter.Meter
}

type Service struct {
	Config
}

type Config struct {
	BillingAdapter     billing.Adapter
	FeatureService     feature.FeatureConnector
	MeterService       meter.Service
	StreamingConnector streaming.Connector
}

func (c Config) Validate() error {
	if c.BillingAdapter == nil {
		return fmt.Errorf("adapter is required")
	}

	if c.FeatureService == nil {
		return fmt.Errorf("feature service is required")
	}

	if c.MeterService == nil {
		return fmt.Errorf("meter repo is required")
	}

	if c.StreamingConnector == nil {
		return fmt.Errorf("streaming connector is required")
	}

	return nil
}

func New(in Config) (*Service, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		Config: in,
	}, nil
}

func (s *Service) FromEntity(line *billing.Line) (Line, error) {
	currencyCalc, err := line.Currency.Calculator()
	if err != nil {
		return nil, fmt.Errorf("creating currency calculator: %w", err)
	}

	base := lineBase{
		service:  s,
		line:     line,
		currency: currencyCalc,
	}

	switch line.Type {
	case billing.InvoiceLineTypeFee:
		return &feeLine{
			lineBase: base,
		}, nil
	case billing.InvoiceLineTypeUsageBased:
		if line.UsageBased.Price.Type() == productcatalog.FlatPriceType {
			return &ubpFlatFeeLine{
				lineBase: base,
			}, nil
		}

		return &usageBasedLine{
			lineBase: base,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported line type: %s", line.Type)
	}
}

func (s *Service) FromEntities(line []*billing.Line) (Lines, error) {
	return slicesx.MapWithErr(line, func(l *billing.Line) (Line, error) {
		return s.FromEntity(l)
	})
}

func (s *Service) resolveFeatureMeter(ctx context.Context, ns string, featureKey string) (*featureCacheItem, error) {
	feat, err := s.FeatureService.GetFeature(
		ctx,
		ns,
		featureKey,
		feature.IncludeArchivedFeatureTrue,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching feature[%s]: %w", featureKey, err)
	}

	if feat.MeterSlug == nil {
		return nil, billing.ValidationError{
			Err: billing.ErrInvoiceLineFeatureHasNoMeters,
		}
	}

	// let's resolve the underlying meter
	meter, err := s.MeterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		Namespace: ns,
		IDOrSlug:  *feat.MeterSlug,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching meter[%s]: %w", *feat.MeterSlug, err)
	}

	return &featureCacheItem{
		feature: *feat,
		meter:   meter,
	}, nil
}

func (s *Service) AssociateLinesToInvoice(ctx context.Context, invoice *billing.Invoice, lines Lines) (Lines, error) {
	lineEntities, err := s.BillingAdapter.AssociateLinesToInvoice(ctx, billing.AssociateLinesToInvoiceAdapterInput{
		Invoice: billing.InvoiceID{
			ID:        invoice.ID,
			Namespace: invoice.Namespace,
		},

		LineIDs: lo.Map(lines, func(l Line, _ int) string {
			return l.ID()
		}),
	})
	if err != nil {
		return nil, err
	}

	invoice.Lines = billing.NewLineChildren(append(invoice.Lines.OrEmpty(), lineEntities...))

	return s.FromEntities(lineEntities)
}

// UpdateTotalsFromDetailedLines is a helper method to update the totals of a line from its detailed lines.
func (s *Service) UpdateTotalsFromDetailedLines(line *billing.Line) error {
	// Calculate the line totals
	for _, line := range line.Children.OrEmpty() {
		if line.DeletedAt != nil {
			continue
		}

		lineSvc, err := s.FromEntity(line)
		if err != nil {
			return fmt.Errorf("creating line service: %w", err)
		}

		if err := lineSvc.UpdateTotals(); err != nil {
			return fmt.Errorf("updating totals for line[%s]: %w", line.ID, err)
		}
	}

	// WARNING: Even if tempting to add discounts etc. here to the totals, we should always keep the logic as is.
	// The usageBasedLine will never be syncorinzed directly to stripe or other apps, only the detailed lines.
	//
	// Given that the external systems will have their own logic for calculating the totals, we cannot expect
	// any custom logic implemented here to be carried over to the external systems.

	// UBP line's value is the sum of all the children
	res := billing.Totals{}

	res = res.Add(lo.Map(line.Children.OrEmpty(), func(l *billing.Line, _ int) billing.Totals {
		// Deleted lines are not contributing to the totals
		if l.DeletedAt != nil {
			return billing.Totals{}
		}

		return l.Totals
	})...)

	line.LineBase.Totals = res

	return nil
}

type InvoicingCapabilityQueryInput struct {
	AsOf               time.Time
	ProgressiveBilling bool
}

type (
	CanBeInvoicedAsOfInput     = InvoicingCapabilityQueryInput
	ResolveBillablePeriodInput = InvoicingCapabilityQueryInput
)

type Line interface {
	LineBase

	Service() *Service

	// IsPeriodEmptyConsideringTruncations returns true if the line has an empty period. This is different from Period.IsEmpty() as
	// this method does any truncation for usage based lines.
	IsPeriodEmptyConsideringTruncations() bool

	Validate(context.Context, *billing.Invoice) error
	CanBeInvoicedAsOf(context.Context, CanBeInvoicedAsOfInput) (*billing.Period, error)
	SnapshotQuantity(context.Context, *billing.Invoice) error
	CalculateDetailedLines() error
	PrepareForCreate(context.Context) (Line, error)
	UpdateTotals() error
}

type Lines []Line

func (s Lines) ToEntities() []*billing.Line {
	return lo.Map(s, func(service Line, _ int) *billing.Line {
		return service.ToEntity()
	})
}

type LineWithBillablePeriod struct {
	Line
	BillablePeriod billing.Period
}

func (s Lines) ResolveBillablePeriod(ctx context.Context, in ResolveBillablePeriodInput) ([]LineWithBillablePeriod, error) {
	out := make([]LineWithBillablePeriod, 0, len(s))
	for _, lineSrv := range s {
		billablePeriod, err := lineSrv.CanBeInvoicedAsOf(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("checking if line can be invoiced: %w", err)
		}

		if billablePeriod != nil {
			out = append(out, LineWithBillablePeriod{
				Line:           lineSrv,
				BillablePeriod: *billablePeriod,
			})
		}
	}

	if len(out) == 0 {
		// We haven't requested explicit empty invoice, so we should have some pending lines
		return nil, billing.ValidationError{
			Err: billing.ErrInvoiceCreateNoLines,
		}
	}

	return out, nil
}
