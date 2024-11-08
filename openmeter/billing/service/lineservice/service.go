package lineservice

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
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

func (s *Service) FromEntity(line billingentity.Line) (Line, error) {
	base := lineBase{
		service: s,
		line:    line,
	}

	switch line.Type {
	case billingentity.InvoiceLineTypeFee:
		return feeLine{
			lineBase: base,
		}, nil
	case billingentity.InvoiceLineTypeUsageBased:
		return usageBasedLine{
			lineBase: base,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported line type: %s", line.Type)
	}
}

func (s *Service) FromEntities(line []billingentity.Line) (Lines, error) {
	return slicesx.MapWithErr(line, func(l billingentity.Line) (Line, error) {
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
		return nil, billingentity.ValidationError{
			Err: billingentity.ErrInvoiceLineFeatureHasNoMeters,
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

func (s *Service) CreateLines(ctx context.Context, lines ...Line) (Lines, error) {
	if len(lines) == 0 {
		return nil, nil
	}

	newLines, err := s.BillingAdapter.CreateInvoiceLines(
		ctx,
		lo.Map(lines, func(line Line, _ int) billingentity.Line {
			entity := line.ToEntity()
			return entity
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("creating invoice lines: %w", err)
	}

	return slicesx.MapWithErr(newLines.Lines, func(line billingentity.Line) (Line, error) {
		return s.FromEntity(line)
	})
}

func (s *Service) AssociateLinesToInvoice(ctx context.Context, invoice *billingentity.Invoice, lines Lines) (Lines, error) {
	lineEntities, err := s.BillingAdapter.AssociateLinesToInvoice(ctx, billing.AssociateLinesToInvoiceAdapterInput{
		Invoice: billingentity.InvoiceID{
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

	return s.FromEntities(lineEntities)
}

type snapshotQuantityResult struct {
	Line Line
	// TODO[OM-980]: Detailed lines should be returned here, that we are upserting based on the qty as described in README.md (see `Detailed Lines vs Splitting`)
}

type Line interface {
	LineBase

	Service() *Service

	Validate(ctx context.Context, invoice *billingentity.Invoice) error
	CanBeInvoicedAsOf(context.Context, time.Time) (*billingentity.Period, error)
	SnapshotQuantity(context.Context, *billingentity.Invoice) (*snapshotQuantityResult, error)
	PrepareForCreate(context.Context) (Line, error)
}

type Lines []Line

func (s Lines) ToEntities() []billingentity.Line {
	return lo.Map(s, func(service Line, _ int) billingentity.Line {
		return service.ToEntity()
	})
}

type LineWithBillablePeriod struct {
	Line
	BillablePeriod billingentity.Period
}

func (s Lines) ResolveBillablePeriod(ctx context.Context, asOf time.Time) ([]LineWithBillablePeriod, error) {
	out := make([]LineWithBillablePeriod, 0, len(s))
	for _, lineSrv := range s {
		billablePeriod, err := lineSrv.CanBeInvoicedAsOf(ctx, asOf)
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
		return nil, billingentity.ValidationError{
			Err: billingentity.ErrInvoiceCreateNoLines,
		}
	}

	return out, nil
}
