package realizations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Service owns flat-fee realization mechanics: credit allocation/correction and
// realization lineage persistence. It must not make state-machine decisions.
type Service struct {
	adapter flatfee.Adapter
	handler flatfee.Handler
	lineage lineage.Service
}

type Config struct {
	Adapter flatfee.Adapter
	Handler flatfee.Handler
	Lineage lineage.Service
}

func (c Config) Validate() error {
	var errs []error

	if c.Adapter == nil {
		errs = append(errs, errors.New("adapter is required"))
	}

	if c.Handler == nil {
		errs = append(errs, errors.New("handler is required"))
	}

	if c.Lineage == nil {
		errs = append(errs, errors.New("lineage service is required"))
	}

	return errors.Join(errs...)
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter: config.Adapter,
		handler: config.Handler,
		lineage: config.Lineage,
	}, nil
}

func (s *Service) CreateCreditAllocations(ctx context.Context, charge flatfee.Charge, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error) {
	realizations, err := s.adapter.CreateCreditAllocations(ctx, charge.GetChargeID(), creditAllocations)
	if err != nil {
		return creditrealization.Realizations{}, err
	}

	if err := s.lineage.CreateInitialLineages(ctx, lineage.CreateInitialLineagesInput{
		Namespace:    charge.Namespace,
		ChargeID:     charge.ID,
		CustomerID:   charge.Intent.CustomerID,
		Currency:     charge.Intent.Currency,
		Realizations: realizations,
	}); err != nil {
		return creditrealization.Realizations{}, fmt.Errorf("create initial credit realization lineages: %w", err)
	}

	if err := s.lineage.PersistCorrectionLineageSegments(ctx, lineage.PersistCorrectionLineageSegmentsInput{
		Namespace:    charge.Namespace,
		Realizations: realizations,
	}); err != nil {
		return creditrealization.Realizations{}, fmt.Errorf("persist correction lineage segments: %w", err)
	}

	return realizations, nil
}

type AllocateCreditsOnlyInput struct {
	Charge             flatfee.Charge
	Amount             alpacadecimal.Decimal
	CurrencyCalculator currencyx.Calculator
}

func (i AllocateCreditsOnlyInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if i.Amount.IsNegative() {
		return fmt.Errorf("amount cannot be negative")
	}

	if err := i.CurrencyCalculator.Validate(); err != nil {
		return fmt.Errorf("currency calculator: %w", err)
	}

	return nil
}

type AllocateCreditsOnlyResult struct {
	Allocated    alpacadecimal.Decimal
	Realizations creditrealization.Realizations
}

func (s *Service) AllocateCreditsOnly(ctx context.Context, in AllocateCreditsOnlyInput) (AllocateCreditsOnlyResult, error) {
	in.Amount = in.CurrencyCalculator.RoundToPrecision(in.Amount)

	if err := in.Validate(); err != nil {
		return AllocateCreditsOnlyResult{}, err
	}

	if in.Amount.IsZero() {
		return AllocateCreditsOnlyResult{}, nil
	}

	input := flatfee.OnCreditsOnlyUsageAccruedInput{
		Charge:           in.Charge,
		AmountToAllocate: in.Amount,
	}
	if err := input.Validate(); err != nil {
		return AllocateCreditsOnlyResult{}, fmt.Errorf("validate input: %w", err)
	}

	creditAllocations, err := s.handler.OnCreditsOnlyUsageAccrued(ctx, input)
	if err != nil {
		return AllocateCreditsOnlyResult{}, fmt.Errorf("on credits only usage accrued: %w", err)
	}

	allocated := in.CurrencyCalculator.RoundToPrecision(creditAllocations.Sum())
	if !allocated.Equal(in.Amount) {
		return AllocateCreditsOnlyResult{}, models.NewGenericValidationError(
			fmt.Errorf("credit allocations do not match total [charge_id=%s, total=%s, allocations_sum=%s]",
				in.Charge.ID, in.Amount.String(), allocated.String()),
		)
	}

	result := AllocateCreditsOnlyResult{
		Allocated: allocated,
	}

	if len(creditAllocations) > 0 {
		realizations, err := s.CreateCreditAllocations(ctx, in.Charge, creditAllocations.AsCreateInputs())
		if err != nil {
			return AllocateCreditsOnlyResult{}, fmt.Errorf("create credit allocations: %w", err)
		}

		result.Realizations = realizations
	}

	return result, nil
}

type CorrectAllCreditRealizationsInput struct {
	Charge             flatfee.Charge
	AllocateAt         time.Time
	CurrencyCalculator currencyx.Calculator
}

func (i CorrectAllCreditRealizationsInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if i.AllocateAt.IsZero() {
		return fmt.Errorf("allocate at is required")
	}

	if err := i.CurrencyCalculator.Validate(); err != nil {
		return fmt.Errorf("currency calculator: %w", err)
	}

	return nil
}

type CorrectAllCreditRealizationsResult struct {
	Realizations creditrealization.Realizations
}

func (s *Service) CorrectAllCredits(ctx context.Context, in CorrectAllCreditRealizationsInput) (CorrectAllCreditRealizationsResult, error) {
	if err := in.Validate(); err != nil {
		return CorrectAllCreditRealizationsResult{}, err
	}

	realizationIDs := lo.Map(in.Charge.Realizations.CreditRealizations, func(realization creditrealization.Realization, _ int) string {
		return realization.ID
	})
	lineageSegmentsByRealization, err := s.lineage.LoadActiveSegmentsByRealizationID(ctx, in.Charge.Namespace, realizationIDs)
	if err != nil {
		return CorrectAllCreditRealizationsResult{}, fmt.Errorf("load active lineage segments: %w", err)
	}

	corrections, err := in.Charge.Realizations.CreditRealizations.CorrectAll(in.CurrencyCalculator, func(req creditrealization.CorrectionRequest) (creditrealization.CreateCorrectionInputs, error) {
		return s.handler.OnCreditsOnlyUsageAccruedCorrection(ctx, flatfee.CreditsOnlyUsageAccruedCorrectionInput{
			Charge:                       in.Charge,
			AllocateAt:                   in.AllocateAt,
			Corrections:                  req,
			LineageSegmentsByRealization: lineageSegmentsByRealization,
		})
	})
	if err != nil {
		return CorrectAllCreditRealizationsResult{}, fmt.Errorf("correct credits: %w", err)
	}

	result := CorrectAllCreditRealizationsResult{}
	if len(corrections) > 0 {
		realizations, err := s.CreateCreditAllocations(ctx, in.Charge, corrections)
		if err != nil {
			return CorrectAllCreditRealizationsResult{}, fmt.Errorf("create credit corrections: %w", err)
		}

		result.Realizations = realizations
	}

	return result, nil
}
