package lineservice

import (
	"context"
	"time"

	"github.com/samber/lo"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	productcatalogmodel "github.com/openmeterio/openmeter/openmeter/productcatalog/model"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ Line = usageBasedLine{}

type usageBasedLine struct {
	lineBase
}

func (l usageBasedLine) PrepareForCreate(context.Context) (Line, error) {
	l.line.Period = l.line.Period.Truncate(billingentity.DefaultMeterResolution)

	return l, nil
}

func (l usageBasedLine) Validate(ctx context.Context, targetInvoice *billingentity.Invoice) error {
	if _, err := l.service.resolveFeatureMeter(ctx, l.line.Namespace, l.line.UsageBased.FeatureKey); err != nil {
		return err
	}

	if err := l.lineBase.Validate(ctx, targetInvoice); err != nil {
		return err
	}

	if len(targetInvoice.Customer.UsageAttribution.SubjectKeys) == 0 {
		return billingentity.ValidationError{
			Err: billingentity.ErrInvoiceCreateUBPLineCustomerHasNoSubjects,
		}
	}

	if l.line.LineBase.Period.Truncate(billingentity.DefaultMeterResolution).IsEmpty() {
		return billingentity.ValidationError{
			Err: billingentity.ErrInvoiceCreateUBPLinePeriodIsEmpty,
		}
	}

	return nil
}

func (l usageBasedLine) CanBeInvoicedAsOf(ctx context.Context, asof time.Time) (*billingentity.Period, error) {
	if l.line.UsageBased.Price.Type() == productcatalogmodel.TieredPriceType {
		tiered, err := l.line.UsageBased.Price.AsTiered()
		if err != nil {
			return nil, err
		}

		if tiered.Mode == productcatalogmodel.GraduatedTieredPrice {
			// Graduated tiers are only billable if we have all the data acquired, as otherwise
			// we might overcharge the customer (if we are at tier bundaries)
			if !asof.Before(l.line.InvoiceAt) {
				return &l.line.Period, nil
			}
			return nil, nil
		}
	}

	meterAndFactory, err := l.service.resolveFeatureMeter(ctx, l.line.Namespace, l.line.UsageBased.FeatureKey)
	if err != nil {
		return nil, err
	}

	meter := meterAndFactory.meter

	asOfTruncated := asof.Truncate(billingentity.DefaultMeterResolution)

	switch meter.Aggregation {
	case models.MeterAggregationSum, models.MeterAggregationCount,
		models.MeterAggregationMax, models.MeterAggregationUniqueCount:

		periodStartTrucated := l.line.Period.Start.Truncate(billingentity.DefaultMeterResolution)

		if !periodStartTrucated.Before(asOfTruncated) {
			return nil, nil
		}

		candidatePeriod := billingentity.Period{
			Start: periodStartTrucated,
			End:   asOfTruncated,
		}

		if candidatePeriod.IsEmpty() {
			return nil, nil
		}

		return &candidatePeriod, nil
	default:
		// Other types need to be billed arrears truncated by window size
		if !asOfTruncated.Before(l.line.InvoiceAt) {
			return &l.line.Period, nil
		}
		return nil, nil
	}
}

func (l usageBasedLine) SnapshotQuantity(ctx context.Context, invoice *billingentity.Invoice) (*snapshotQuantityResult, error) {
	featureMeter, err := l.service.resolveFeatureMeter(ctx, l.line.Namespace, l.line.UsageBased.FeatureKey)
	if err != nil {
		return nil, err
	}

	usage, err := l.service.getFeatureUsage(ctx,
		getFeatureUsageInput{
			Line:       &l.line,
			ParentLine: l.line.ParentLine,
			Feature:    featureMeter.feature,
			Meter:      featureMeter.meter,
			Subjects:   invoice.Customer.UsageAttribution.SubjectKeys,
		},
	)
	if err != nil {
		return nil, err
	}

	updatedLineEntity := l.line
	updatedLineEntity.UsageBased.Quantity = lo.ToPtr(usage.LinePeriodQty)

	updatedLine, err := l.service.FromEntity(updatedLineEntity)
	if err != nil {
		return nil, err
	}

	// TODO[OM-980]: yield detailed lines here

	return &snapshotQuantityResult{
		Line: updatedLine,
	}, nil
}
