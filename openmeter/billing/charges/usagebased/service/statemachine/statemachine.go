package statemachine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/qmuntal/stateless"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
)

type StateMachine interface {
	statemachine.Methods[usagebased.Charge]
}

type Base struct {
	*statemachine.Base[usagebased.Charge]

	Logger *slog.Logger

	Service usagebased.Service
	Adapter usagebased.Adapter

	CustomerOverride billing.CustomerOverrideWithDetails
	FeatureMeter     feature.FeatureMeter
	Handler          usagebased.Handler
	RatingService    usagebasedrating.Service
	Lineage          lineage.Service
}

type Config struct {
	Charge           usagebased.Charge
	Service          usagebased.Service
	Adapter          usagebased.Adapter
	Logger           *slog.Logger
	CustomerOverride billing.CustomerOverrideWithDetails
	FeatureMeter     feature.FeatureMeter
	Handler          usagebased.Handler
	RatingService    usagebasedrating.Service
	Lineage          lineage.Service
}

func (c Config) Validate() error {
	var errs []error

	if err := c.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if c.Service == nil {
		errs = append(errs, errors.New("service is required"))
	}

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.CustomerOverride.Customer == nil {
		errs = append(errs, errors.New("expanded customer is required"))
	}

	if err := c.CustomerOverride.MergedProfile.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("merged profile is required: %w", err))
	}

	if c.FeatureMeter.Meter == nil {
		errs = append(errs, errors.New("feature meter is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.Handler == nil {
		errs = append(errs, errors.New("handler is required"))
	}

	if c.RatingService == nil {
		errs = append(errs, errors.New("rating service is required"))
	}

	if c.Lineage == nil {
		errs = append(errs, errors.New("lineage service is required"))
	}

	return errors.Join(errs...)
}

func newBase(config Config) (*Base, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	base, err := statemachine.New[usagebased.Charge](config.Charge, StateMutator{
		adapter: config.Adapter,
	})
	if err != nil {
		return nil, fmt.Errorf("new base: %w", err)
	}

	if base == nil {
		return nil, errors.New("base is nil")
	}

	out := &Base{
		Base:             base,
		Service:          config.Service,
		Logger:           config.Logger,
		Adapter:          config.Adapter,
		CustomerOverride: config.CustomerOverride,
		FeatureMeter:     config.FeatureMeter,
		Handler:          config.Handler,
		RatingService:    config.RatingService,
		Lineage:          config.Lineage,
	}

	stateMachine := stateless.NewStateMachineWithExternalStorage(
		func(ctx context.Context) (stateless.State, error) {
			return out.Charge.Status, nil
		},
		func(ctx context.Context, state stateless.State) error {
			newStatus := state.(usagebased.Status)
			if err := newStatus.Validate(); err != nil {
				return fmt.Errorf("invalid status: %w", err)
			}

			out.Charge.Status = newStatus
			return nil
		},
		stateless.FiringImmediate,
	)

	out.StateMachine = stateMachine

	return out, nil
}

// refetchCharge refetches the charge from the database and updates the state machine's charge.
// The adapter's modification functions should properly support updating the charge in memory, as
// a yearly charge with daily realization runs will have a lot of realizations thus a lot of data
// should be loaded.
//
// Use this where the final implementation is uncertain for now.
func (s *Base) refetchCharge(ctx context.Context) error {
	charge, err := s.Service.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: s.Charge.GetChargeID(),
		Expands:  meta.Expands{meta.ExpandRealizations},
	})
	if err != nil {
		return fmt.Errorf("get charge: %w", err)
	}

	s.Charge = charge

	return nil
}

func (s *Base) SyncFeatureIDFromFeatureMeter(ctx context.Context) error {
	s.Charge.State.FeatureID = s.FeatureMeter.Feature.ID
	return nil
}

func (s *Base) AdvanceAfterCollectionPeriodEnd(ctx context.Context) error {
	collectionPeriodEnd, err := s.getCurrentRunCollectionEnd()
	if err != nil {
		return err
	}

	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(collectionPeriodEnd.Add(usagebased.InternalCollectionPeriod)))

	return nil
}

func (s *Base) IsAfterCollectionPeriod(ctx context.Context, _ ...any) bool {
	collectionPeriodEnd, err := s.getCurrentRunCollectionEnd()
	if err != nil {
		s.Logger.ErrorContext(ctx, "failed to get collection period end", "error", err, "customerID", s.Charge.Intent.CustomerID)
		return false
	}

	return !clock.Now().Before(collectionPeriodEnd.Add(usagebased.InternalCollectionPeriod))
}

func (s *Base) GetCollectionPeriodEnd(_ context.Context) (time.Time, error) {
	collectionPeriod := s.CustomerOverride.MergedProfile.WorkflowConfig.Collection.Interval
	collectionPeriodEnd, _ := collectionPeriod.AddTo(s.Charge.Intent.ServicePeriod.To)
	return meta.NormalizeTimestamp(collectionPeriodEnd), nil
}

func (s *Base) getCurrentRunCollectionEnd() (time.Time, error) {
	if s.Charge.State.CurrentRealizationRunID == nil {
		return time.Time{}, fmt.Errorf("no realization run in progress [charge_id=%s]", s.Charge.ID)
	}

	currentRun, err := s.Charge.Realizations.GetByID(*s.Charge.State.CurrentRealizationRunID)
	if err != nil {
		return time.Time{}, fmt.Errorf("get current realization run: %w", err)
	}

	return meta.NormalizeTimestamp(currentRun.CollectionEnd), nil
}
