package run

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditreconciliation"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *Service) createCreditRealizationLineages(
	ctx context.Context,
	charge usagebased.Charge,
	currency currencies.Currency,
	realizations creditrealization.Realizations,
) error {
	if err := s.lineage.CreateInitialLineages(ctx, lineage.CreateInitialLineagesInput{
		Namespace:    charge.Namespace,
		ChargeID:     charge.ID,
		CustomerID:   charge.Intent.GetCustomerID(),
		Currency:     currency,
		Features:     featuresForLineage(charge.Intent.GetFeatureKey()),
		Realizations: realizations,
	}); err != nil {
		return fmt.Errorf("create initial credit realization lineages: %w", err)
	}

	if err := s.lineage.PersistCorrectionLineageSegments(ctx, lineage.PersistCorrectionLineageSegmentsInput{
		Namespace:    charge.Namespace,
		Realizations: realizations,
	}); err != nil {
		return fmt.Errorf("persist correction lineage segments: %w", err)
	}

	return nil
}

func featuresForLineage(featureKey string) []string {
	if featureKey == "" {
		return nil
	}

	return []string{featureKey}
}

type CreditReconciliationHandlerInput struct {
	Charge     usagebased.Charge
	Run        usagebased.RealizationRun
	AllocateAt time.Time
}

func (i CreditReconciliationHandlerInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if i.AllocateAt.IsZero() {
		errs = append(errs, errors.New("allocate at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// chargeCurrencyCreditReconciliationHandler reconciles a usage-based run's
// allocations in the charge currency and preserves their realization lineage.
type chargeCurrencyCreditReconciliationHandler struct {
	service *Service
	CreditReconciliationHandlerInput
}

var _ creditreconciliation.Handler = (*chargeCurrencyCreditReconciliationHandler)(nil)

func (s *Service) NewChargeCurrencyCreditReconciliationHandler(
	input CreditReconciliationHandlerInput,
) creditreconciliation.Handler {
	return &chargeCurrencyCreditReconciliationHandler{
		service:                          s,
		CreditReconciliationHandlerInput: input,
	}
}

func (h *chargeCurrencyCreditReconciliationHandler) ValidateWith(currencyx.Currency) error {
	return h.CreditReconciliationHandlerInput.Validate()
}

func (h *chargeCurrencyCreditReconciliationHandler) Realizations() creditrealization.Realizations {
	return h.Run.CreditsAllocated
}

func (h *chargeCurrencyCreditReconciliationHandler) Allocate(
	ctx context.Context,
	amount alpacadecimal.Decimal,
) (creditrealization.CreateAllocationInputs, error) {
	creditAllocations, err := h.service.handler.OnCreditsOnlyUsageAccrued(ctx, usagebased.CreditsOnlyUsageAccruedInput{
		Charge:           h.Charge,
		Run:              h.Run,
		BookedAt:         h.AllocateAt,
		AmountToAllocate: amount,
	})
	if err != nil {
		return nil, fmt.Errorf("on credits only usage accrued: %w", err)
	}

	return creditAllocations, nil
}

func (h *chargeCurrencyCreditReconciliationHandler) Correct(
	ctx context.Context,
	request creditrealization.CorrectionRequest,
) (creditrealization.CreateCorrectionInputs, error) {
	lineageSegmentsByRealization, err := h.service.loadActiveCreditRealizationLineageSegments(
		ctx,
		h.Charge,
		h.Realizations(),
	)
	if err != nil {
		return nil, fmt.Errorf("load active lineage segments for run: %w", err)
	}

	return h.service.handler.OnCreditsOnlyUsageAccruedCorrection(ctx, usagebased.CreditsOnlyUsageAccruedCorrectionInput{
		Charge:                       h.Charge,
		Run:                          h.Run,
		BookedAt:                     h.AllocateAt,
		Corrections:                  request,
		LineageSegmentsByRealization: lineageSegmentsByRealization,
	})
}

func (h *chargeCurrencyCreditReconciliationHandler) Create(
	ctx context.Context,
	creditRealizations creditrealization.CreateInputs,
) (creditrealization.Realizations, error) {
	realizations, err := h.service.adapter.CreateChargeCurrencyCreditRealizations(ctx, usagebased.CreateCreditRealizationsInput{
		RunID:              h.Run.ID,
		CreditRealizations: creditRealizations,
	})
	if err != nil {
		return nil, fmt.Errorf("create charge currency credit realizations: %w", err)
	}

	if err := h.service.createCreditRealizationLineages(ctx, h.Charge, h.Charge.Intent.GetCurrency(), realizations); err != nil {
		return nil, err
	}

	return realizations, nil
}

// fiatOverageCreditReconciliationHandler reconciles a custom-currency run's
// overage allocations in settlement fiat and preserves their separate lineage.
type fiatOverageCreditReconciliationHandler struct {
	service *Service
	CreditReconciliationHandlerInput
}

var _ creditreconciliation.Handler = (*fiatOverageCreditReconciliationHandler)(nil)

func (s *Service) NewFiatOverageCreditReconciliationHandler(
	input CreditReconciliationHandlerInput,
) creditreconciliation.Handler {
	return &fiatOverageCreditReconciliationHandler{
		service:                          s,
		CreditReconciliationHandlerInput: input,
	}
}

func (h *fiatOverageCreditReconciliationHandler) ValidateWith(currency currencyx.Currency) error {
	var errs []error

	if err := h.CreditReconciliationHandlerInput.Validate(); err != nil {
		errs = append(errs, err)
	}

	if !h.Charge.Intent.GetCurrency().IsCustom() {
		errs = append(errs, errors.New("charge currency must be custom"))
	}

	if h.Charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		errs = append(errs, errors.New("settlement mode must be credit_then_invoice"))
	}

	costBasisIntent := h.Charge.Intent.GetCostBasisIntent()
	if costBasisIntent == nil {
		errs = append(errs, errors.New("cost basis intent is required"))
	} else {
		settlementCurrency, err := costBasisIntent.GetFiatCurrency()
		if err != nil {
			errs = append(errs, fmt.Errorf("get settlement fiat currency: %w", err))
		} else if currency != nil {
			fiatCurrency, err := currency.AsFiat()
			if err != nil {
				errs = append(errs, fmt.Errorf("currency calculator must be fiat: %w", err))
			} else if settlementCurrency.GetFiatCode() != fiatCurrency.GetFiatCode() {
				errs = append(errs, fmt.Errorf(
					"currency calculator must match settlement fiat currency: %s != %s",
					fiatCurrency.GetFiatCode(),
					settlementCurrency.GetFiatCode(),
				))
			}
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (h *fiatOverageCreditReconciliationHandler) Realizations() creditrealization.Realizations {
	return h.Run.FiatOverageCreditRealizations
}

func (h *fiatOverageCreditReconciliationHandler) Allocate(
	ctx context.Context,
	amount alpacadecimal.Decimal,
) (creditrealization.CreateAllocationInputs, error) {
	creditAllocations, err := h.service.handler.OnAllocateFiatOverageCredits(ctx, usagebased.AllocateFiatOverageCreditsInput{
		Charge:           h.Charge,
		Run:              h.Run,
		BookedAt:         h.AllocateAt,
		AmountToAllocate: amount,
	})
	if err != nil {
		return nil, fmt.Errorf("allocate fiat overage credits: %w", err)
	}

	return creditAllocations, nil
}

func (h *fiatOverageCreditReconciliationHandler) Correct(
	ctx context.Context,
	request creditrealization.CorrectionRequest,
) (creditrealization.CreateCorrectionInputs, error) {
	lineageSegmentsByRealization, err := h.service.loadActiveCreditRealizationLineageSegments(
		ctx,
		h.Charge,
		h.Realizations(),
	)
	if err != nil {
		return nil, fmt.Errorf("load active fiat overage lineage segments for run: %w", err)
	}

	return h.service.handler.OnCorrectFiatOverageCreditAllocations(ctx, usagebased.CorrectFiatOverageCreditAllocationsInput{
		Charge:                       h.Charge,
		Run:                          h.Run,
		BookedAt:                     h.AllocateAt,
		Corrections:                  request,
		LineageSegmentsByRealization: lineageSegmentsByRealization,
	})
}

func (h *fiatOverageCreditReconciliationHandler) Create(
	ctx context.Context,
	creditRealizations creditrealization.CreateInputs,
) (creditrealization.Realizations, error) {
	fiatCurrency, err := h.Charge.Intent.GetCostBasisIntent().GetFiatCurrency()
	if err != nil {
		return nil, fmt.Errorf("get settlement fiat currency: %w", err)
	}

	realizations, err := h.service.adapter.CreateFiatOverageCreditRealizations(ctx, usagebased.CreateCreditRealizationsInput{
		RunID:              h.Run.ID,
		CreditRealizations: creditRealizations,
	})
	if err != nil {
		return nil, fmt.Errorf("create fiat overage credit realizations: %w", err)
	}

	if err := h.service.createCreditRealizationLineages(ctx, h.Charge, currencies.Currency{Currency: fiatCurrency}, realizations); err != nil {
		return nil, err
	}

	return realizations, nil
}

// CorrectAllCreditRealizations reverses every active credit allocation for a
// usage-based run. Custom-currency credit-then-invoice runs unwind settlement
// fiat before the charge currency from which that overage was derived.
func (s *Service) CorrectAllCreditRealizations(
	ctx context.Context,
	input CreditReconciliationHandlerInput,
) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if input.Charge.Intent.GetSettlementMode() == productcatalog.CreditThenInvoiceSettlementMode &&
		input.Charge.Intent.GetCurrency().IsCustom() {
		costBasisIntent := input.Charge.Intent.GetCostBasisIntent()
		if costBasisIntent == nil {
			return errors.New("cost basis intent is required")
		}

		fiatCurrency, err := costBasisIntent.GetFiatCurrency()
		if err != nil {
			return fmt.Errorf("get settlement fiat currency: %w", err)
		}

		if _, err := creditreconciliation.CorrectAll(ctx, creditreconciliation.CorrectAllInput{
			CurrencyCalculator: fiatCurrency,
			Handler:            s.NewFiatOverageCreditReconciliationHandler(input),
		}); err != nil {
			return fmt.Errorf("correct all fiat overage credit realizations: %w", err)
		}
	}

	if _, err := creditreconciliation.CorrectAll(ctx, creditreconciliation.CorrectAllInput{
		CurrencyCalculator: input.Charge.Intent.GetCurrency(),
		Handler:            s.NewChargeCurrencyCreditReconciliationHandler(input),
	}); err != nil {
		return fmt.Errorf("correct all charge currency credit realizations: %w", err)
	}

	return nil
}

func (s *Service) loadActiveCreditRealizationLineageSegments(
	ctx context.Context,
	charge usagebased.Charge,
	realizations creditrealization.Realizations,
) (lineage.ActiveSegmentsByRealizationID, error) {
	realizationIDs := lo.Map(realizations, func(realization creditrealization.Realization, _ int) string {
		return realization.ID
	})

	return s.lineage.LoadActiveSegmentsByRealizationID(ctx, charge.Namespace, realizationIDs)
}
