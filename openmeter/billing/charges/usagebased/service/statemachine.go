package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/qmuntal/stateless"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	usagebasedrun "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type stateMachine struct {
	*stateless.StateMachine

	Charge usagebased.Charge

	Logger *slog.Logger

	Adapter usagebased.Adapter
	Rater   *usagebasedrating.Service
	Runs    *usagebasedrun.Service

	CustomerOverride   billing.CustomerOverrideWithDetails
	FeatureMeter       feature.FeatureMeter
	CurrencyCalculator currencyx.Calculator
}

type StateMachine interface {
	AdvanceUntilStateStable(ctx context.Context) (*usagebased.Charge, error)
	CanFire(ctx context.Context, trigger meta.Trigger) (bool, error)
	FireAndActivate(ctx context.Context, trigger meta.Trigger, args ...any) error
	GetCharge() usagebased.Charge
}

type StateMachineConfig struct {
	Charge             usagebased.Charge
	Adapter            usagebased.Adapter
	Rater              *usagebasedrating.Service
	Runs               *usagebasedrun.Service
	Logger             *slog.Logger
	CustomerOverride   billing.CustomerOverrideWithDetails
	FeatureMeter       feature.FeatureMeter
	CurrencyCalculator currencyx.Calculator
}

func (c StateMachineConfig) Validate() error {
	var errs []error

	if err := c.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.Rater == nil {
		errs = append(errs, errors.New("rater is required"))
	}

	if c.Runs == nil {
		errs = append(errs, errors.New("run service is required"))
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

	if err := c.CurrencyCalculator.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency calculator: %w", err))
	}

	return errors.Join(errs...)
}

func newStateMachineBase(config StateMachineConfig) (*stateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	out := &stateMachine{
		Charge:             config.Charge,
		Logger:             lo.CoalesceOrEmpty(config.Logger, slog.Default()),
		Adapter:            config.Adapter,
		Rater:              config.Rater,
		Runs:               config.Runs,
		CustomerOverride:   config.CustomerOverride,
		FeatureMeter:       config.FeatureMeter,
		CurrencyCalculator: config.CurrencyCalculator,
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
// a yearly charge with daily realization lifecycle will have a lot of realizations thus a lot of data
// should be loaded.
//
// Use this where the final implementation is uncertain for now.
func (s *stateMachine) refetchCharge(ctx context.Context) error {
	charge, err := s.Adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: s.Charge.GetChargeID(),
		Expands:  meta.Expands{meta.ExpandRealizations},
	})
	if err != nil {
		return fmt.Errorf("get charge: %w", err)
	}

	s.Charge = charge

	return nil
}

func (s *stateMachine) GetCharge() usagebased.Charge {
	return s.Charge
}

func (s *stateMachine) IsInsideServicePeriod() bool {
	return !clock.Now().Before(s.Charge.Intent.ServicePeriod.From)
}

func (s *stateMachine) IsAfterServicePeriod() bool {
	return !clock.Now().Before(s.Charge.Intent.ServicePeriod.To)
}

func (s *stateMachine) AdvanceAfterServicePeriodTo(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(s.Charge.Intent.ServicePeriod.To))
	return nil
}

func (s *stateMachine) SyncFeatureIDFromFeatureMeter(ctx context.Context) error {
	s.Charge.State.FeatureID = s.FeatureMeter.Feature.ID
	return nil
}

func (s *stateMachine) AdvanceAfterServicePeriodFrom(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(s.Charge.Intent.ServicePeriod.From))
	return nil
}

func (s *stateMachine) AdvanceAfterCollectionPeriodEnd(ctx context.Context) error {
	collectionPeriodEnd, err := s.getCurrentRunCollectionEnd()
	if err != nil {
		return err
	}

	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(collectionPeriodEnd.Add(usagebased.InternalCollectionPeriod)))

	return nil
}

func (s *stateMachine) IsAfterCollectionPeriod(ctx context.Context, _ ...any) bool {
	collectionPeriodEnd, err := s.getCurrentRunCollectionEnd()
	if err != nil {
		s.Logger.ErrorContext(ctx, "failed to get collection period end", "error", err, "customerID", s.Charge.Intent.CustomerID)
		return false
	}

	return !clock.Now().Before(collectionPeriodEnd.Add(usagebased.InternalCollectionPeriod))
}

func (s *stateMachine) GetCollectionPeriodEnd(_ context.Context) (time.Time, error) {
	collectionPeriod := s.CustomerOverride.MergedProfile.WorkflowConfig.Collection.Interval
	collectionPeriodEnd, _ := collectionPeriod.AddTo(s.Charge.Intent.ServicePeriod.To)
	return meta.NormalizeTimestamp(collectionPeriodEnd), nil
}

func (s *stateMachine) getCurrentRunCollectionEnd() (time.Time, error) {
	if s.Charge.State.CurrentRealizationRunID == nil {
		return time.Time{}, fmt.Errorf("no realization run in progress [charge_id=%s]", s.Charge.ID)
	}

	currentRun, err := s.Charge.Realizations.GetByID(*s.Charge.State.CurrentRealizationRunID)
	if err != nil {
		return time.Time{}, fmt.Errorf("get current realization run: %w", err)
	}

	return meta.NormalizeTimestamp(currentRun.CollectionEnd), nil
}

func (s *stateMachine) FireAndActivate(ctx context.Context, trigger meta.Trigger, args ...any) error {
	if err := s.StateMachine.FireCtx(ctx, trigger, args...); err != nil {
		return err
	}

	return s.StateMachine.ActivateCtx(ctx)
}

func (s *stateMachine) CanFire(ctx context.Context, trigger meta.Trigger) (bool, error) {
	return s.StateMachine.CanFireCtx(ctx, trigger)
}

func (s *stateMachine) AdvanceUntilStateStable(ctx context.Context) (*usagebased.Charge, error) {
	var advanced bool

	for {
		canFire, err := s.StateMachine.CanFireCtx(ctx, meta.TriggerNext)
		if err != nil {
			return nil, err
		}

		if !canFire {
			if !advanced {
				return nil, nil
			}

			charge := s.Charge
			return &charge, nil
		}

		if err := s.FireAndActivate(ctx, meta.TriggerNext); err != nil {
			return nil, fmt.Errorf("cannot transition to the next status [current_status=%s]: %w", s.Charge.Status, err)
		}

		updatedChargeBase, err := s.Adapter.UpdateCharge(ctx, s.Charge.ChargeBase)
		if err != nil {
			return nil, fmt.Errorf("persist charge: %w", err)
		}

		s.Charge.ChargeBase = updatedChargeBase

		advanced = true
	}
}
