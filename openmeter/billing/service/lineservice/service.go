package lineservice

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
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

	featureCache map[string]*featureCacheItem
}

type Config struct {
	BillingAdapter     billing.Adapter
	FeatureService     feature.FeatureConnector
	MeterRepo          meter.Repository
	StreamingConnector streaming.Connector
}

func (c Config) Validate() error {
	if c.BillingAdapter == nil {
		return fmt.Errorf("adapter is required")
	}

	if c.FeatureService == nil {
		return fmt.Errorf("feature service is required")
	}

	if c.MeterRepo == nil {
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

		featureCache: make(map[string]*featureCacheItem),
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
	cacheKey := fmt.Sprintf("%s/%s", ns, featureKey)
	// Let's cache the results as we might need to resolve the same feature/meter multiple times (Validate, CanBeInvoicedAsOf, UpdateQty)
	if entry, found := s.featureCache[cacheKey]; found {
		return entry, nil
	}

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
	meter, err := s.MeterRepo.GetMeterByIDOrSlug(ctx, ns, *feat.MeterSlug)
	if err != nil {
		return nil, fmt.Errorf("fetching meter[%s]: %w", *feat.MeterSlug, err)
	}

	ent := &featureCacheItem{
		feature: *feat,
		meter:   meter,
	}

	s.featureCache[cacheKey] = ent
	return ent, nil
}

func (s *Service) UpsertLines(ctx context.Context, ns string, lines ...Line) (Lines, error) {
	if len(lines) == 0 {
		return nil, nil
	}

	newLines, err := s.BillingAdapter.UpsertInvoiceLines(
		ctx,
		billing.UpsertInvoiceLinesAdapterInput{
			Namespace: ns,
			Lines: lo.Map(lines, func(line Line, _ int) *billing.Line {
				return line.ToEntity()
			}),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("creating invoice lines: %w", err)
	}

	return slicesx.MapWithErr(newLines, func(line *billing.Line) (Line, error) {
		return s.FromEntity(line)
	})
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
