package meteredentitlement

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type entitlementGrantOwner struct {
	featureRepo     feature.FeatureRepo
	entitlementRepo entitlement.EntitlementRepo
	usageResetRepo  UsageResetRepo
	meterService    meter.Service
	customerService customer.Service
	logger          *slog.Logger
	tracer          trace.Tracer
}

func NewEntitlementGrantOwnerAdapter(
	featureRepo feature.FeatureRepo,
	entitlementRepo entitlement.EntitlementRepo,
	usageResetRepo UsageResetRepo,
	meterService meter.Service,
	customerService customer.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
) grant.OwnerConnector {
	return &entitlementGrantOwner{
		featureRepo:     featureRepo,
		entitlementRepo: entitlementRepo,
		usageResetRepo:  usageResetRepo,
		meterService:    meterService,
		customerService: customerService,
		logger:          logger,
		tracer:          tracer,
	}
}

func (e *entitlementGrantOwner) DescribeOwner(ctx context.Context, id models.NamespacedID) (grant.Owner, error) {
	ctx, span := e.tracer.Start(ctx, "meteredentitlement.DescribeOwner", mTrace.WithOwner(id))
	defer span.End()

	var def grant.Owner

	// get feature of ent
	ent, err := e.entitlementRepo.GetEntitlement(ctx, id)
	if err != nil {
		if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
			return def, grant.NewOwnerNotFoundError(id, "entitlement")
		}

		return def, err
	}

	mEnt, err := ParseFromGenericEntitlement(ent)
	if err != nil {
		return def, err
	}

	feature, err := getRepoMaybeInTx(ctx, e.featureRepo, e.featureRepo).GetByIdOrKey(ctx, id.Namespace, ent.FeatureID, true)
	if err != nil {
		return def, fmt.Errorf("failed to get feature of entitlement: %w", err)
	}

	if feature.MeterSlug == nil {
		return def, fmt.Errorf("feature does not have a meter")
	}

	// meterrepo is not transactional
	met, err := e.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
		Namespace: id.Namespace,
		IDOrSlug:  *feature.MeterSlug,
	})
	if err != nil {
		return def, fmt.Errorf("failed to get meter: %w", err)
	}

	queryParams := streaming.QueryParams{
		FilterGroupBy: feature.MeterGroupByFilters,
	}

	cust, err := e.customerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: id.Namespace,
			ID:        ent.CustomerID,
		},
	})
	if err != nil {
		return def, fmt.Errorf("failed to get customer: %w", err)
	}

	// Require filtering by customer; error if missing
	if cust == nil || (cust.DeletedAt != nil && cust.DeletedAt.Before(clock.Now())) {
		return def, models.NewGenericValidationError(fmt.Errorf("customer not found for entitlement %s", id.ID))
	}

	var subjectKeys []string
	if cust.UsageAttribution != nil {
		subjectKeys = cust.UsageAttribution.SubjectKeys
	}

	streamingCustomer := ownerCustomer{
		id:          cust.ID,
		key:         cust.Key,
		subjectKeys: subjectKeys,
	}

	queryParams.FilterCustomer = []streaming.Customer{streamingCustomer}

	return grant.Owner{
		NamespacedID:       id,
		Meter:              met,
		DefaultQueryParams: queryParams,
		ResetBehavior: grant.ResetBehavior{
			PreserveOverage: mEnt.PreserveOverageAtReset,
		},
		StreamingCustomer: streamingCustomer,
	}, nil
}

func (e *entitlementGrantOwner) GetStartOfMeasurement(ctx context.Context, owner models.NamespacedID) (time.Time, error) {
	ctx, span := e.tracer.Start(ctx, "meteredentitlement.GetStartOfMeasurement", mTrace.WithOwner(owner))
	defer span.End()

	owningEntitlement, err := e.entitlementRepo.GetEntitlement(ctx, owner)
	if err != nil {
		if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
			return time.Time{}, grant.NewOwnerNotFoundError(owner, "entitlement")
		}

		return time.Time{}, fmt.Errorf("failed to get entitlement: %w", err)
	}

	metered, err := ParseFromGenericEntitlement(owningEntitlement)
	if err != nil {
		return time.Time{}, err
	}

	return metered.MeasureUsageFrom, nil
}

// The current usage period start time is either the current period start time, or if this is the first period then the start of measurement
func (e *entitlementGrantOwner) GetUsagePeriodStartAt(ctx context.Context, owner models.NamespacedID, at time.Time) (time.Time, error) {
	ctx, span := e.tracer.Start(ctx, "meteredentitlement.GetUsagePeriodStartAt", mTrace.WithOwner(owner), trace.WithAttributes(attribute.String("at", at.String())))
	defer span.End()

	ent, err := e.entitlementRepo.GetEntitlement(ctx, owner)
	if err != nil {
		return time.Time{}, err
	}

	metered, err := ParseFromGenericEntitlement(ent)
	if err != nil {
		return time.Time{}, err
	}

	// UsagePeriod handles all calculations correctly
	per, err := metered.UsagePeriod.GetCurrentPeriodAt(at)
	if err != nil {
		return time.Time{}, err
	}

	return per.From, nil
}

func (e *entitlementGrantOwner) GetResetTimelineInclusive(ctx context.Context, owner models.NamespacedID, period timeutil.ClosedPeriod) (timeutil.SimpleTimeline, error) {
	ctx, span := e.tracer.Start(ctx, "meteredentitlement.GetResetTimelineInclusive", mTrace.WithOwner(owner), mTrace.WithPeriod(period))
	defer span.End()

	var def timeutil.SimpleTimeline

	// Let's fetch the owner entitlement
	ent, err := e.entitlementRepo.GetEntitlement(ctx, owner)
	if err != nil {
		return def, err
	}

	if ent.UsagePeriod == nil {
		return def, fmt.Errorf("entitlement does not have a usage period")
	}

	initialResetOrStartOfMeasurement := ent.UsagePeriod.GetOriginalValueAsUsagePeriodInput().GetTime()

	// UsagePeriod handles all calculations correctly
	times, err := ent.UsagePeriod.GetResetTimelineInclusive(period)
	if err != nil {
		return def, err
	}

	// For backwards compatibility
	return timeutil.NewSimpleTimeline(lo.Uniq(append([]time.Time{initialResetOrStartOfMeasurement}, times.GetTimes()...))), nil
}

func (e *entitlementGrantOwner) EndCurrentUsagePeriod(ctx context.Context, owner models.NamespacedID, params grant.EndCurrentUsagePeriodParams) error {
	ctx, span := e.tracer.Start(ctx, "meteredentitlement.EndCurrentUsagePeriod", mTrace.WithOwner(owner), trace.WithAttributes(attribute.String("at", params.At.String())))
	defer span.End()

	// If we're not in a transaction this method should fail
	_, err := transaction.GetDriverFromContext(ctx)
	if err != nil {
		return fmt.Errorf("end current usage period must be called in a transaction: %w", err)
	}

	_, err = transaction.Run(ctx, e.featureRepo, func(ctx context.Context) (*interface{}, error) {
		// Check if time is after current start time. If so then we can end the period
		currentStartAt, err := e.GetUsagePeriodStartAt(ctx, owner, params.At)
		if err != nil {
			return nil, fmt.Errorf("failed to get current usage period start time: %w", err)
		}
		if params.At.Before(currentStartAt) {
			return nil, models.NewGenericValidationError(fmt.Errorf("cannot end usage period before current period start time"))
		}

		// Now let's see what the anchor is after the update
		entitlementEntity, err := e.entitlementRepo.GetEntitlement(ctx, owner)
		if err != nil {
			return nil, fmt.Errorf("failed to get entitlement: %w", err)
		}

		inpt, _, err := entitlementEntity.UsagePeriod.GetUsagePeriodInputAt(params.At)
		if err != nil {
			return nil, fmt.Errorf("failed to get current period at: %w", err)
		}

		anchor := inpt.GetValue().Anchor

		if !params.RetainAnchor {
			anchor = params.At
		}

		// Save usage reset
		if err := e.usageResetRepo.Save(ctx, UsageResetUpdate{
			NamespacedModel: models.NamespacedModel{
				Namespace: owner.Namespace,
			},
			EntitlementID:       owner.ID,
			ResetTime:           params.At,
			Anchor:              anchor,
			UsagePeriodInterval: inpt.GetValue().Interval.ISOString(),
		}); err != nil {
			return nil, fmt.Errorf("failed to save usage reset: %w", err)
		}

		// Now let's update the entitlement current usage period saved value
		// we refetch to get the new reset value
		entitlementEntity, err = e.entitlementRepo.GetEntitlement(ctx, owner)
		if err != nil {
			return nil, fmt.Errorf("failed to get entitlement: %w", err)
		}

		cup, err := entitlementEntity.UsagePeriod.GetCurrentPeriodAt(params.At)
		if err != nil {
			return nil, fmt.Errorf("failed to get new current usage period: %w", err)
		}

		if err := e.entitlementRepo.UpdateEntitlementUsagePeriod(
			ctx,
			owner,
			entitlement.UpdateEntitlementUsagePeriodParams{
				CurrentUsagePeriod: cup,
			}); err != nil {
			return nil, fmt.Errorf("failed to update entitlement usage period: %w", err)
		}

		return nil, nil
	})
	return err
}

func (e *entitlementGrantOwner) LockOwnerForTx(ctx context.Context, owner models.NamespacedID) error {
	ctx, span := e.tracer.Start(ctx, "meteredentitlement.LockOwnerForTx", mTrace.WithOwner(owner))
	defer span.End()

	// If we're not in a transaction this method has to fail
	tx, err := entutils.GetDriverFromContext(ctx)
	if err != nil {
		return fmt.Errorf("lock owner for tx must be called in a transaction: %w", err)
	}
	return e.entitlementRepo.LockEntitlementForTx(ctx, tx, owner)
}

// FIXME: this is a terrible hack to conditionally catch transactions
func getRepoMaybeInTx[T any](ctx context.Context, repo T, txUser entutils.TxUser[T]) T {
	if ctxTx, err := entutils.GetDriverFromContext(ctx); err == nil {
		// we're already in a tx
		return txUser.WithTx(ctx, ctxTx)
	} else {
		return repo
	}
}
