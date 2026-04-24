package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestatemachine "github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	usagebasedrun "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type stateMachine struct {
	*chargestatemachine.Machine[usagebased.Charge, usagebased.ChargeBase, usagebased.Status]

	Logger *slog.Logger

	Adapter usagebased.Adapter
	Rater   usagebasedrating.Service
	Runs    *usagebasedrun.Service

	CustomerOverride   billing.CustomerOverrideWithDetails
	FeatureMeter       feature.FeatureMeter
	CurrencyCalculator currencyx.Calculator
}

type StateMachine = chargestatemachine.StateMachine[usagebased.Charge]

type StateMachineConfig struct {
	Charge             usagebased.Charge
	Adapter            usagebased.Adapter
	Rater              usagebasedrating.Service
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
		Logger:             lo.CoalesceOrEmpty(config.Logger, slog.Default()),
		Adapter:            config.Adapter,
		Rater:              config.Rater,
		Runs:               config.Runs,
		CustomerOverride:   config.CustomerOverride,
		FeatureMeter:       config.FeatureMeter,
		CurrencyCalculator: config.CurrencyCalculator,
	}

	machine, err := chargestatemachine.New(chargestatemachine.Config[usagebased.Charge, usagebased.ChargeBase, usagebased.Status]{
		Charge: config.Charge,
		Persistence: chargestatemachine.Persistence[usagebased.Charge, usagebased.ChargeBase]{
			UpdateBase: func(ctx context.Context, base usagebased.ChargeBase) (usagebased.ChargeBase, error) {
				return out.Adapter.UpdateCharge(ctx, base)
			},
			Refetch: func(ctx context.Context, chargeID meta.ChargeID) (usagebased.Charge, error) {
				return out.Adapter.GetByID(ctx, usagebased.GetByIDInput{
					ChargeID: chargeID,
					Expands:  meta.Expands{meta.ExpandRealizations},
				})
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("new machine: %w", err)
	}

	out.Machine = machine

	return out, nil
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
	snapshotAfter, err := s.getCurrentRunSnapshotAfter()
	if err != nil {
		return err
	}

	s.Charge.State.AdvanceAfter = lo.ToPtr(snapshotAfter)

	return nil
}

func (s *stateMachine) IsAfterCollectionPeriod(ctx context.Context, _ ...any) bool {
	snapshotAfter, err := s.getCurrentRunSnapshotAfter()
	if err != nil {
		s.Logger.ErrorContext(ctx, "failed to get snapshot after", "error", err, "customerID", s.Charge.Intent.CustomerID)
		return false
	}

	return !clock.Now().Before(snapshotAfter)
}

func (s *stateMachine) getFinalRunStoredAtLT() (time.Time, error) {
	collectionPeriod := s.CustomerOverride.MergedProfile.WorkflowConfig.Collection.Interval
	storedAtLT, _ := collectionPeriod.AddTo(s.Charge.Intent.ServicePeriod.To)
	return meta.NormalizeTimestamp(storedAtLT), nil
}

func (s *stateMachine) getCurrentRunSnapshotAfter() (time.Time, error) {
	if s.Charge.State.CurrentRealizationRunID == nil {
		return time.Time{}, fmt.Errorf("no realization run in progress [charge_id=%s]", s.Charge.ID)
	}

	currentRun, err := s.Charge.Realizations.GetByID(*s.Charge.State.CurrentRealizationRunID)
	if err != nil {
		return time.Time{}, fmt.Errorf("get current realization run: %w", err)
	}

	return meta.NormalizeTimestamp(currentRun.StoredAtLT.Add(usagebased.InternalCollectionPeriod)), nil
}

func (s *stateMachine) ensureDetailedLinesLoadedForRating(ctx context.Context) error {
	if len(s.Charge.Realizations) == 0 {
		return nil
	}

	if lo.EveryBy(s.Charge.Realizations, func(run usagebased.RealizationRun) bool {
		return run.DetailedLines.IsPresent()
	}) {
		return nil
	}

	expandedCharge, err := s.Adapter.FetchDetailedLines(ctx, s.Charge)
	if err != nil {
		return fmt.Errorf("fetch detailed lines: %w", err)
	}

	s.Charge = expandedCharge

	return nil
}
