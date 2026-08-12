package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestatemachine "github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type stateMachine struct {
	*chargestatemachine.Machine[flatfee.Charge, flatfee.ChargeBase, flatfee.Status]

	Adapter      flatfee.Adapter
	Realizations *flatfeerealizations.Service
	Service      *service

	CreditNotesSupported bool
}

type StateMachine = chargestatemachine.StateMachine[flatfee.Charge]

type StateMachineConfig struct {
	Charge flatfee.Charge

	Adapter      flatfee.Adapter
	Realizations *flatfeerealizations.Service
	Service      *service

	CreditNotesSupported bool
}

func (c StateMachineConfig) Validate() error {
	var errs []error

	if err := c.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.Realizations == nil {
		errs = append(errs, errors.New("realizations service is required"))
	}

	if c.Service == nil {
		errs = append(errs, errors.New("service is required"))
	}

	return errors.Join(errs...)
}

func newStateMachineBase(config StateMachineConfig) (*stateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	out := &stateMachine{
		Adapter:              config.Adapter,
		Realizations:         config.Realizations,
		Service:              config.Service,
		CreditNotesSupported: config.CreditNotesSupported,
	}

	machine, err := chargestatemachine.New(chargestatemachine.Config[flatfee.Charge, flatfee.ChargeBase, flatfee.Status]{
		Charge: config.Charge,
		Persistence: chargestatemachine.Persistence[flatfee.Charge, flatfee.ChargeBase]{
			UpdateBase: func(ctx context.Context, base flatfee.ChargeBase) (flatfee.ChargeBase, error) {
				return out.Adapter.UpdateCharge(ctx, base)
			},
			Refetch: func(ctx context.Context, chargeID meta.ChargeID) (flatfee.Charge, error) {
				return out.Adapter.GetByID(ctx, flatfee.GetByIDInput{
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

// mutateIntentLayer mutates the requested intent layer, creating a new override
// layer first when the target is override and the charge has no override yet.
func (s *stateMachine) mutateIntentLayer(ctx context.Context, target meta.ChangeTarget, editFn func(*flatfee.IntentMutableFields)) error {
	switch target {
	case meta.ChangeTargetBase:
		if err := s.Charge.Intent.Mutate(meta.ChangeTargetBase, func(fields *flatfee.IntentMutableFields) {
			mutateFlatFeeIntentFields(fields, editFn)
		}); err != nil {
			return fmt.Errorf("mutating base intent: %w", err)
		}
	case meta.ChangeTargetOverride:
		if s.Charge.Intent.HasOverrideLayer() {
			if err := s.Charge.Intent.Mutate(meta.ChangeTargetOverride, func(fields *flatfee.IntentMutableFields) {
				mutateFlatFeeIntentFields(fields, editFn)
			}); err != nil {
				return fmt.Errorf("mutating override intent: %w", err)
			}

			return nil
		}

		effectiveIntent := s.Charge.Intent.GetEffectiveIntent()
		overrideFields := effectiveIntent.IntentMutableFields
		mutateFlatFeeIntentFields(&overrideFields, editFn)
		overrideFields = overrideFields.Normalized(effectiveIntent.Currency)
		if err := overrideFields.Validate(); err != nil {
			return fmt.Errorf("validating override intent: %w", err)
		}

		base, err := s.Adapter.CreateChargeOverride(ctx, s.Charge.ChargeBase, overrideFields)
		if err != nil {
			return fmt.Errorf("creating override intent: %w", err)
		}

		s.Charge.ChargeBase = base
	default:
		return fmt.Errorf("invalid change target: %s", target)
	}

	return nil
}

func mutateFlatFeeIntentFields(fields *flatfee.IntentMutableFields, editFn func(*flatfee.IntentMutableFields)) {
	editFn(fields)
	fields.PercentageDiscounts = fields.PercentageDiscounts.UpsertCorrelationID()
}

// setOverrideIntent replaces the complete mutable override snapshot while
// preserving the charge's base intent.
func (s *stateMachine) setOverrideIntent(ctx context.Context, patch flatfee.PatchSetOverride) error {
	target, err := patch.GetTargetLayer(s.Charge.Intent)
	if err != nil {
		return fmt.Errorf("getting patch target layer: %w", err)
	}

	fields := patch.GetIntentMutableFields()
	if err := s.mutateIntentLayer(ctx, target, func(current *flatfee.IntentMutableFields) {
		*current = fields
	}); err != nil {
		return fmt.Errorf("setting flat-fee override intent: %w", err)
	}

	return nil
}

// clearOverrideIntent removes the customer-facing layer so the latest source
// intent becomes effective. State-machine transitions own reconciling the
// restored intent's lifecycle, including deletion.
func (s *stateMachine) clearOverrideIntent(ctx context.Context) (bool, error) {
	if !s.Charge.Intent.HasOverrideLayer() {
		return false, nil
	}

	base, err := s.Adapter.DeleteChargeOverride(ctx, s.Charge.ChargeBase)
	if err != nil {
		return false, fmt.Errorf("deleting flat-fee override intent: %w", err)
	}

	s.Charge.ChargeBase = base
	return true, nil
}

func (s *stateMachine) IsBaseIntentDeleted() bool {
	return s.Charge.Intent.GetBaseIntent().IntentDeletedAt != nil
}

func (s *stateMachine) ClearOverrideFromDeletedBase(ctx context.Context, _ meta.PatchClearOverride) error {
	cleared, err := s.clearOverrideIntent(ctx)
	if err != nil {
		return err
	}
	if !cleared {
		return nil
	}

	if s.Charge.Intent.GetDeletedAt() == nil {
		return errors.New("clearing flat-fee override did not restore the deleted base intent")
	}

	return nil
}

func (s *stateMachine) UnsupportedSetOverrideOperation(_ context.Context, _ flatfee.PatchSetOverride) error {
	return models.NewGenericPreConditionFailedError(
		fmt.Errorf("cannot set override for flat-fee charge in status %s; retry after billing advances", s.Charge.Status),
	)
}

func (s *stateMachine) UnsupportedClearOverrideOperation(_ context.Context, _ meta.PatchClearOverride) error {
	return models.NewGenericPreConditionFailedError(
		fmt.Errorf("cannot clear override for flat-fee charge in status %s; retry after billing advances", s.Charge.Status),
	)
}

// rejectHiddenIntentTarget prevents lifecycle state machines from processing a
// hidden source intent. When an override layer exists, the override is the
// active customer-facing charge: it owns status transitions, realization runs,
// credit corrections, and invoice patches. Subscription-owned base/source
// changes must be applied before state-machine dispatch by service-level
// reconciliation, not interpreted as lifecycle events.
func (s *stateMachine) rejectHiddenIntentTarget(target meta.ChangeTarget) error {
	if target == meta.ChangeTargetBase && s.Charge.Intent.HasOverrideLayer() {
		return models.NewGenericPreConditionFailedError(
			fmt.Errorf("cannot mutate hidden base intent while override intent is active"),
		)
	}

	return nil
}

func (s *stateMachine) IsInsideServicePeriod() bool {
	return !clock.Now().Before(s.Charge.Intent.GetEffectiveServicePeriod().From)
}

func (s *stateMachine) IsInsideServicePeriodAndZeroAmount() bool {
	return s.IsInsideServicePeriod() && s.Charge.State.AmountAfterProration.IsZero()
}

func (s *stateMachine) IsInsideServicePeriodAndNonZeroAmount() bool {
	return s.IsInsideServicePeriod() && !s.Charge.State.AmountAfterProration.IsZero()
}

func (s *stateMachine) IsAfterInvoiceAt() bool {
	return !clock.Now().Before(s.Charge.Intent.GetEffectiveInvoiceAt())
}

func (s *stateMachine) IsAfterInvoiceAtAndZeroAmount() bool {
	return s.IsAfterInvoiceAt() && s.Charge.State.AmountAfterProration.IsZero()
}

func (s *stateMachine) IsAfterInvoiceAtAndNonZeroAmount() bool {
	return s.IsAfterInvoiceAt() && !s.Charge.State.AmountAfterProration.IsZero()
}

func (s *stateMachine) IsZeroAmount() bool {
	return s.Charge.State.AmountAfterProration.IsZero()
}

func (s *stateMachine) AdvanceAfterServicePeriodFrom(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(s.Charge.Intent.GetEffectiveServicePeriod().From))
	return nil
}

func (s *stateMachine) AdvanceAfterInvoiceAt(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(s.Charge.Intent.GetEffectiveInvoiceAt()))
	return nil
}

func (s *stateMachine) AdvanceAfterServicePeriodTo(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(s.Charge.Intent.GetEffectiveServicePeriod().To))
	return nil
}

func (s *stateMachine) ClearAdvanceAfter(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = nil
	return nil
}
