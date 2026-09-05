package realizations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditreconciliation"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreditReconciliationHandlerInput struct {
	Charge     flatfee.Charge
	Run        flatfee.RealizationRun
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

// chargeCurrencyCreditReconciliationHandler reconciles a flat-fee run's
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

func (h *chargeCurrencyCreditReconciliationHandler) Validate() error {
	return h.CreditReconciliationHandlerInput.Validate()
}

func (h *chargeCurrencyCreditReconciliationHandler) CurrencyCalculator() currencyx.Currency {
	return h.Charge.Intent.GetCurrency()
}

func (h *chargeCurrencyCreditReconciliationHandler) Realizations() creditrealization.Realizations {
	return h.Run.CreditRealizations
}

func (h *chargeCurrencyCreditReconciliationHandler) Allocate(
	ctx context.Context,
	amount alpacadecimal.Decimal,
) (creditrealization.CreateAllocationInputs, error) {
	allocations, err := h.service.handler.OnAllocateCredits(ctx, flatfee.OnAllocateCreditsInput{
		Charge:                 h.Charge,
		ServicePeriod:          h.Run.ServicePeriod,
		BookedAt:               h.AllocateAt,
		PreTaxAmountToAllocate: amount,
	})
	if err != nil {
		return nil, fmt.Errorf("allocate charge currency credits: %w", err)
	}

	return lo.Map(allocations, func(allocation creditrealization.CreateAllocationInput, _ int) creditrealization.CreateAllocationInput {
		allocation.LineID = h.Run.LineID
		return allocation
	}), nil
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
		return nil, fmt.Errorf("load active charge currency lineage segments for run: %w", err)
	}

	return h.service.handler.OnCorrectCreditAllocations(ctx, flatfee.CorrectCreditAllocationsInput{
		Charge:                       h.Charge,
		BookedAt:                     h.AllocateAt,
		Corrections:                  request,
		LineageSegmentsByRealization: lineageSegmentsByRealization,
	})
}

func (h *chargeCurrencyCreditReconciliationHandler) Create(
	ctx context.Context,
	creditRealizations creditrealization.CreateInputs,
) (creditrealization.Realizations, error) {
	realizations, err := h.service.adapter.CreateChargeCurrencyCreditRealizations(ctx, flatfee.CreateCreditRealizationsInput{
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

func (h *fiatOverageCreditReconciliationHandler) Validate() error {
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
	} else if _, err := costBasisIntent.GetFiatCurrency(); err != nil {
		errs = append(errs, fmt.Errorf("get settlement fiat currency: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (h *fiatOverageCreditReconciliationHandler) CurrencyCalculator() currencyx.Currency {
	costBasisIntent := h.Charge.Intent.GetCostBasisIntent()
	if costBasisIntent == nil {
		return nil
	}

	fiatCurrency, err := costBasisIntent.GetFiatCurrency()
	if err != nil {
		return nil
	}

	return fiatCurrency
}

func (h *fiatOverageCreditReconciliationHandler) Realizations() creditrealization.Realizations {
	return h.Run.FiatOverageCreditRealizations
}

func (h *fiatOverageCreditReconciliationHandler) Allocate(
	ctx context.Context,
	amount alpacadecimal.Decimal,
) (creditrealization.CreateAllocationInputs, error) {
	allocations, err := h.service.handler.OnAllocateFiatOverageCredits(ctx, flatfee.AllocateFiatOverageCreditsInput{
		Charge:           h.Charge,
		Run:              h.Run,
		BookedAt:         h.AllocateAt,
		AmountToAllocate: amount,
	})
	if err != nil {
		return nil, fmt.Errorf("allocate fiat overage credits: %w", err)
	}

	return lo.Map(allocations, func(allocation creditrealization.CreateAllocationInput, _ int) creditrealization.CreateAllocationInput {
		allocation.LineID = h.Run.LineID
		return allocation
	}), nil
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

	return h.service.handler.OnCorrectFiatOverageCreditAllocations(ctx, flatfee.CorrectFiatOverageCreditAllocationsInput{
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

	realizations, err := h.service.adapter.CreateFiatOverageCreditRealizations(ctx, flatfee.CreateCreditRealizationsInput{
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

type AllocateFiatOverageCreditsInput struct {
	Charge flatfee.Charge
	Run    flatfee.RealizationRun
	// AllowFiatCredits controls whether charge orchestration may invoke the
	// settlement-fiat allocation handler for this run.
	AllowFiatCredits bool
}

func (i AllocateFiatOverageCreditsInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if !i.Charge.Intent.GetCurrency().IsCustom() {
		errs = append(errs, errors.New("charge currency must be custom"))
	}

	if i.Charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		errs = append(errs, errors.New("settlement mode must be credit_then_invoice"))
	}

	if i.Charge.Realizations.CurrentRun == nil {
		errs = append(errs, errors.New("charge has no current realization run"))
	} else if i.Charge.Realizations.CurrentRun.ID != i.Run.ID {
		errs = append(errs, fmt.Errorf(
			"run is not the charge's current realization run [current_run_id=%s run_id=%s]",
			i.Charge.Realizations.CurrentRun.ID.ID,
			i.Run.ID.ID,
		))
	}

	if i.Run.ID.Namespace != i.Charge.Namespace {
		errs = append(errs, fmt.Errorf(
			"run namespace does not match charge namespace: %s != %s",
			i.Run.ID.Namespace,
			i.Charge.Namespace,
		))
	}

	if i.Run.AccruedUsage == nil {
		errs = append(errs, errors.New("run must have prepared invoice usage before fiat allocation"))
	}

	if i.Run.FiatOverageCreditAllocationCompleted {
		errs = append(errs, errors.New("fiat overage credit allocation already completed"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type AllocateFiatOverageCreditsResult struct {
	Charge flatfee.Charge
	Run    flatfee.RealizationRun
}

// AllocateFiatOverageCredits performs the one settlement-fiat allocation for a
// custom-currency run during invoice finalization. Fiat balance eligibility is
// evaluated at finalization, independently from charge-currency realization.
func (s *Service) AllocateFiatOverageCredits(
	ctx context.Context,
	in AllocateFiatOverageCreditsInput,
) (AllocateFiatOverageCreditsResult, error) {
	if err := in.Validate(); err != nil {
		return AllocateFiatOverageCreditsResult{}, err
	}

	fiatCurrency, err := in.Charge.GetInvoiceCurrency()
	if err != nil {
		return AllocateFiatOverageCreditsResult{}, fmt.Errorf("get invoice currency: %w", err)
	}
	grossFiatAmount := in.Run.AccruedUsage.Totals.Total

	run := in.Run
	if in.AllowFiatCredits && len(run.FiatOverageCreditRealizations) == 0 {
		allocationResult, err := creditreconciliation.Allocate(ctx, creditreconciliation.AllocateInput{
			Amount: grossFiatAmount,
			Handler: s.NewFiatOverageCreditReconciliationHandler(CreditReconciliationHandlerInput{
				Charge: in.Charge,
				Run:    run,
				// Fiat overage allocation is an invoice-finalization effect. Current
				// time makes the outstanding settlement-fiat balance eligible then.
				AllocateAt: clock.Now(),
			}),
		})
		if err != nil {
			return AllocateFiatOverageCreditsResult{}, fmt.Errorf("allocate fiat overage credits: %w", err)
		}

		run.FiatOverageCreditRealizations = allocationResult.Realizations
	}

	fiatCurrencyCalculator, err := currencyx.NewFiatCurrency(fiatCurrency)
	if err != nil {
		return AllocateFiatOverageCreditsResult{}, fmt.Errorf("create invoice currency calculator: %w", err)
	}
	allocated := fiatCurrencyCalculator.RoundToPrecision(run.FiatOverageCreditRealizations.Sum())
	if allocated.GreaterThan(grossFiatAmount) {
		return AllocateFiatOverageCreditsResult{}, fmt.Errorf("fiat overage credit allocations exceed prepared gross amount: %s > %s", allocated, grossFiatAmount)
	}
	remainingFiatOverage := fiatCurrencyCalculator.RoundToPrecision(grossFiatAmount.Sub(allocated))
	runBase, err := s.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
		ID:                                   run.ID,
		NoFiatTransactionRequired:            mo.Some(remainingFiatOverage.IsZero()),
		FiatOverageCreditAllocationCompleted: mo.Some(true),
	})
	if err != nil {
		return AllocateFiatOverageCreditsResult{}, fmt.Errorf("update realization run after fiat overage allocation: %w", err)
	}

	run.RealizationRunBase = runBase
	charge := in.Charge
	charge.Realizations.CurrentRun = &run

	return AllocateFiatOverageCreditsResult{
		Charge: charge,
		Run:    run,
	}, nil
}

// CorrectAllCreditRealizations reverses every active credit allocation for a
// flat-fee run without changing its accrued invoice usage.
func (s *Service) CorrectAllCreditRealizations(
	ctx context.Context,
	input CreditReconciliationHandlerInput,
) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if err := s.correctFiatOverageCreditRealizations(ctx, input); err != nil {
		return err
	}

	return s.correctChargeCurrencyCreditRealizations(ctx, input)
}

// CorrectPreparedCustomCurrencyInvoiceRealizations unwinds a prepared custom-
// currency invoice run in reverse accounting order while keeping accrued-usage
// correction separate from credit realization correction.
func (s *Service) CorrectPreparedCustomCurrencyInvoiceRealizations(
	ctx context.Context,
	input CreditReconciliationHandlerInput,
) (flatfee.RealizationRun, error) {
	if err := input.Validate(); err != nil {
		return flatfee.RealizationRun{}, err
	}

	accruedUsageInput := CorrectAccruedUsageInput{
		Charge: input.Charge,
		Run:    input.Run,
	}
	if err := accruedUsageInput.Validate(); err != nil {
		return flatfee.RealizationRun{}, err
	}

	if err := s.correctFiatOverageCreditRealizations(ctx, input); err != nil {
		return flatfee.RealizationRun{}, err
	}

	run, err := s.CorrectAccruedUsage(ctx, accruedUsageInput)
	if err != nil {
		return flatfee.RealizationRun{}, err
	}
	input.Run = run

	if err := s.correctChargeCurrencyCreditRealizations(ctx, input); err != nil {
		return flatfee.RealizationRun{}, err
	}

	runBase, err := s.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
		ID:                                   input.Run.ID,
		FiatOverageCreditAllocationCompleted: mo.Some(false),
	})
	if err != nil {
		return flatfee.RealizationRun{}, fmt.Errorf("reset invoice preparation state: %w", err)
	}
	input.Run.RealizationRunBase = runBase

	return input.Run, nil
}

func (s *Service) correctFiatOverageCreditRealizations(ctx context.Context, input CreditReconciliationHandlerInput) error {
	if input.Charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode ||
		!input.Charge.Intent.GetCurrency().IsCustom() {
		return nil
	}

	if _, err := creditreconciliation.CorrectAll(ctx, creditreconciliation.CorrectAllInput{
		Handler: s.NewFiatOverageCreditReconciliationHandler(input),
	}); err != nil {
		return fmt.Errorf("correct all fiat overage credit realizations: %w", err)
	}

	return nil
}

func (s *Service) correctChargeCurrencyCreditRealizations(ctx context.Context, input CreditReconciliationHandlerInput) error {
	if _, err := creditreconciliation.CorrectAll(ctx, creditreconciliation.CorrectAllInput{
		Handler: s.NewChargeCurrencyCreditReconciliationHandler(input),
	}); err != nil {
		return fmt.Errorf("correct all charge currency credit realizations: %w", err)
	}

	return nil
}

func (s *Service) loadActiveCreditRealizationLineageSegments(
	ctx context.Context,
	charge flatfee.Charge,
	realizations creditrealization.Realizations,
) (lineage.ActiveSegmentsByRealizationID, error) {
	realizations = realizations.LegacyLineageRealizations()
	if len(realizations) == 0 {
		return lineage.ActiveSegmentsByRealizationID{}, nil
	}
	realizationIDs := lo.Map(realizations, func(realization creditrealization.Realization, _ int) string {
		return realization.ID
	})

	return s.lineage.LoadActiveSegmentsByRealizationID(ctx, charge.Namespace, realizationIDs)
}
