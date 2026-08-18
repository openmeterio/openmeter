package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type standardLinePopulationStage string

const (
	standardLinePopulationStageGatheringPreview     standardLinePopulationStage = "gathering_preview"
	standardLinePopulationStageInvoiceCreated       standardLinePopulationStage = "invoice_created"
	standardLinePopulationStageCollectionCompleted  standardLinePopulationStage = "collection_completed"
	standardLinePopulationStageInvoiceFinalizing    standardLinePopulationStage = "invoice_finalizing"
	standardLinePopulationStageManualAttachment     standardLinePopulationStage = "manual_attachment"
	standardLinePopulationStageIntentReconciliation standardLinePopulationStage = "intent_reconciliation"
)

func (s standardLinePopulationStage) Validate() error {
	switch s {
	case standardLinePopulationStageGatheringPreview,
		standardLinePopulationStageInvoiceCreated,
		standardLinePopulationStageCollectionCompleted,
		standardLinePopulationStageInvoiceFinalizing,
		standardLinePopulationStageManualAttachment,
		standardLinePopulationStageIntentReconciliation:
		return nil
	case "":
		return fmt.Errorf("standard line population stage is required")
	default:
		return fmt.Errorf("invalid standard line population stage: %q", s)
	}
}

type populateFlatFeeStandardLineFromRunInput struct {
	Charge flatfee.Charge
	Run    flatfee.RealizationRun
	Stage  standardLinePopulationStage
}

type calculateFiatOverageForRunResult struct {
	FiatCurrency          *currencyx.FiatCurrency
	FiatOverage           alpacadecimal.Decimal
	ShouldOmitInvoiceLine bool
}

// calculateFiatOverageForRun resolves the post-allocation custom-currency
// overage and the invoice-line omission decision from the same conversion.
func calculateFiatOverageForRun(charge flatfee.Charge, run flatfee.RealizationRun) (calculateFiatOverageForRunResult, error) {
	if !charge.Intent.GetCurrency().IsCustom() ||
		charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		return calculateFiatOverageForRunResult{}, nil
	}

	fiatOverage, err := charge.ConvertCustomCurrencyOverageToFiat(run.Totals)
	if err != nil {
		return calculateFiatOverageForRunResult{}, err
	}

	return calculateFiatOverageForRunResult{
		FiatCurrency:          fiatOverage.Currency,
		FiatOverage:           fiatOverage.Amount,
		ShouldOmitInvoiceLine: fiatOverage.Amount.IsZero(),
	}, nil
}

// resolveFiatOverageForLinePopulation reuses the gross FIAT amount persisted
// by invoice preparation. This keeps retries from repeating conversion after
// the durable accounting boundary has been crossed.
func resolveFiatOverageForLinePopulation(
	charge flatfee.Charge,
	run flatfee.RealizationRun,
	stage standardLinePopulationStage,
) (calculateFiatOverageForRunResult, error) {
	if stage != standardLinePopulationStageInvoiceFinalizing || run.AccruedUsage == nil {
		return calculateFiatOverageForRun(charge, run)
	}

	costBasisIntent := charge.Intent.GetCostBasisIntent()
	if costBasisIntent == nil {
		return calculateFiatOverageForRunResult{}, errors.New("cost basis intent is required for a custom-currency invoice")
	}

	fiatCurrency, err := costBasisIntent.GetFiatCurrency()
	if err != nil {
		return calculateFiatOverageForRunResult{}, err
	}

	grossFiatAmount := run.AccruedUsage.Totals.Total

	return calculateFiatOverageForRunResult{
		FiatCurrency:          fiatCurrency,
		FiatOverage:           grossFiatAmount,
		ShouldOmitInvoiceLine: grossFiatAmount.IsZero(),
	}, nil
}

func (i populateFlatFeeStandardLineFromRunInput) Validate() error {
	var errs []error

	if err := i.Stage.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	effectiveServicePeriod := i.Charge.Intent.GetEffectiveServicePeriod()
	if !i.Run.ServicePeriod.Equal(effectiveServicePeriod) {
		errs = append(errs, fmt.Errorf(
			"run service period does not match effective intent [run_id=%s run_period=%v intent_period=%v]",
			i.Run.ID.ID,
			i.Run.ServicePeriod,
			effectiveServicePeriod,
		))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func populateFlatFeeStandardLineFromRun(stdLine *billing.StandardLine, input populateFlatFeeStandardLineFromRunInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if stdLine.UsageBased == nil {
		return fmt.Errorf("standard line[%s]: usage based line is required", stdLine.ID)
	}

	rateableIntent, err := input.Charge.GetRateableIntent()
	if err != nil {
		return fmt.Errorf("getting rateable intent: %w", err)
	}

	stdLine.Name = rateableIntent.GetName()
	stdLine.Period = rateableIntent.GetServicePeriod()
	stdLine.UsageBased.Price = rateableIntent.GetPrice()
	stdLine.RateCardDiscounts = rateableIntent.GetRateCardDiscounts()

	invoiceCurrency, err := stdLine.Currency.AsFiatCurrency()
	if err != nil {
		return fmt.Errorf("creating currency calculator: %w", err)
	}

	if input.Charge.Intent.GetCurrency().IsCustom() {
		return populateCustomCurrencyOverageFromRun(stdLine, input)
	}

	creditsApplied, err := input.Run.CreditRealizations.AsCreditsApplied()
	if err != nil {
		return err
	}

	stdLine.CreditsApplied = creditsApplied

	mappedDetailedLines, err := mapFlatFeeDetailedLines(stdLine, input.Run)
	if err != nil {
		return fmt.Errorf("mapping run detailed lines: %w", err)
	}

	mappedDetailedLines, err = mappedDetailedLines.WithCreditsApplied(stdLine.CreditsApplied, invoiceCurrency)
	if err != nil {
		return fmt.Errorf("applying run credits to detailed lines: %w", err)
	}

	stdLine.DetailedLines = stdLine.DetailedLinesWithIDReuse(mappedDetailedLines)
	stdLine.Totals = stdLine.DetailedLines.SumTotals().RoundToPrecision(invoiceCurrency)

	expectedTotals := input.Run.Totals.RoundToPrecision(invoiceCurrency)
	if !stdLine.Totals.Equal(expectedTotals) {
		return fmt.Errorf("mapped line totals do not match run totals [line_id=%s run_id=%s line_total=%s run_total=%s]",
			stdLine.ID, input.Run.ID.ID, stdLine.Totals.Total.String(), expectedTotals.Total.String())
	}

	return nil
}

func populateCustomCurrencyOverageFromRun(
	stdLine *billing.StandardLine,
	input populateFlatFeeStandardLineFromRunInput,
) error {
	charge := input.Charge
	run := input.Run

	fiatOverage, err := resolveFiatOverageForLinePopulation(charge, run, input.Stage)
	if err != nil {
		return fmt.Errorf("custom currency charge[%s] converting overage to fiat: %w", charge.ID, err)
	}
	if fiatOverage.FiatCurrency == nil {
		return fmt.Errorf("custom currency charge[%s] does not have an invoiceable fiat overage", charge.ID)
	}

	if stdLine.Currency != fiatOverage.FiatCurrency.GetFiatCode() {
		return fmt.Errorf(
			"custom currency charge[%s] invoice currency mismatch: %s != %s",
			charge.ID,
			stdLine.Currency,
			fiatOverage.FiatCurrency.Details().Code,
		)
	}

	if stdLine.Annotations == nil {
		stdLine.Annotations = models.Annotations{}
	}
	stdLine.Annotations[billing.AnnotationKeyReason] = lo.ToPtr(billing.AnnotationValueReasonOverage)

	stdLine.RateCardDiscounts = billing.Discounts{}
	stdLine.Discounts = billing.StandardLineDiscounts{}
	stdLine.CreditsApplied = nil

	stdLine.UsageBased = &billing.UsageBasedLine{
		ConfigID:   stdLine.UsageBased.ConfigID,
		FeatureKey: stdLine.UsageBased.FeatureKey,
		Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      fiatOverage.FiatOverage,
			PaymentTerm: productcatalog.InArrearsPaymentTerm,
		}),
		MeteredQuantity:              lo.ToPtr(alpacadecimal.NewFromInt(1)),
		Quantity:                     lo.ToPtr(alpacadecimal.NewFromInt(1)),
		MeteredPreLinePeriodQuantity: lo.ToPtr(alpacadecimal.Zero),
		PreLinePeriodQuantity:        lo.ToPtr(alpacadecimal.Zero),
	}

	name := "overage"
	if stdLine.Name != "" {
		name = fmt.Sprintf("%s (overage)", stdLine.Name)
	}
	stdLine.Name = name

	stdLineWithDetails, err := creditpurchase.WithDetailedLines(creditpurchase.WithDetailedLinesInput{
		Line:              stdLine,
		Name:              name,
		CreditCurrency:    charge.Intent.GetCurrency(),
		CreditAmount:      run.Totals.Total,
		ResolvedCostBasis: charge.State.ResolvedCostBasis.CostBasis,
		FiatCurrency:      fiatOverage.FiatCurrency,
		FiatAmount:        fiatOverage.FiatOverage,
	})
	if err != nil {
		return fmt.Errorf("populating custom currency overage line: %w", err)
	}
	*stdLine = *stdLineWithDetails

	fiatCreditsApplied, err := run.FiatOverageCreditRealizations.AsCreditsApplied()
	if err != nil {
		return fmt.Errorf("mapping fiat overage credit realizations: %w", err)
	}
	stdLine.CreditsApplied = fiatCreditsApplied

	detailedLines, err := stdLine.DetailedLines.WithCreditsApplied(fiatCreditsApplied, fiatOverage.FiatCurrency)
	if err != nil {
		return fmt.Errorf("applying fiat overage credits to detailed lines: %w", err)
	}
	stdLine.DetailedLines = stdLine.DetailedLinesWithIDReuse(detailedLines)
	stdLine.Totals = stdLine.DetailedLines.SumTotals().RoundToPrecision(fiatOverage.FiatCurrency)

	if (input.Stage == standardLinePopulationStageGatheringPreview ||
		input.Stage == standardLinePopulationStageCollectionCompleted) &&
		fiatOverage.ShouldOmitInvoiceLine {
		stdLine.DeletedAt = lo.ToPtr(clock.Now())
	}

	return nil
}

func mapFlatFeeDetailedLines(stdLine *billing.StandardLine, run flatfee.RealizationRun) (billing.DetailedLines, error) {
	if run.DetailedLines.IsAbsent() {
		return nil, fmt.Errorf("run %s detailed lines must be expanded", run.ID.ID)
	}

	return lo.Map(run.DetailedLines.OrEmpty(), func(line flatfee.DetailedLine, _ int) billing.DetailedLine {
		base := line.Base.Clone()
		base.Namespace = stdLine.Namespace
		base.ID = ""
		base.CreatedAt = time.Time{}
		base.UpdatedAt = time.Time{}
		base.DeletedAt = nil

		return billing.DetailedLine{
			DetailedLineBase: billing.DetailedLineBase{
				Base:      base,
				InvoiceID: stdLine.InvoiceID,
			},
		}
	}), nil
}
