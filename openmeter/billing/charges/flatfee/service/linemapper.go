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
	standardLinePopulationStageManualAttachment     standardLinePopulationStage = "manual_attachment"
	standardLinePopulationStageIntentReconciliation standardLinePopulationStage = "intent_reconciliation"
)

func (s standardLinePopulationStage) Validate() error {
	switch s {
	case standardLinePopulationStageGatheringPreview,
		standardLinePopulationStageInvoiceCreated,
		standardLinePopulationStageCollectionCompleted,
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
		return populateCustomCurrencyOverageFromRun(stdLine, input, invoiceCurrency)
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
	invoiceCurrency currencyx.Currency,
) error {
	charge := input.Charge
	run := input.Run

	fiatOverage, err := charge.ConvertCustomCurrencyOverageToFiat(run.Totals)
	if err != nil {
		return fmt.Errorf("custom currency charge[%s] converting overage to fiat: %w", charge.ID, err)
	}

	if stdLine.Currency != fiatOverage.Currency.GetFiatCode() {
		return fmt.Errorf(
			"custom currency charge[%s] invoice currency mismatch: %s != %s",
			charge.ID,
			stdLine.Currency,
			fiatOverage.Currency.Details().Code,
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
			Amount:      fiatOverage.Amount,
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

	detailedLine, err := creditpurchase.NewDetailedLine(creditpurchase.NewDetailedLineInput{
		Namespace:            stdLine.Namespace,
		InvoiceID:            stdLine.InvoiceID,
		Name:                 name,
		ServicePeriod:        stdLine.Period,
		CustomCurrency:       charge.Intent.GetCurrency(),
		CustomCurrencyAmount: run.Totals.Total,
		ResolvedCostBasis:    charge.State.ResolvedCostBasis,
		FiatCurrency:         fiatOverage.Currency,
		FiatAmount:           fiatOverage.Amount,
	})
	if err != nil {
		return fmt.Errorf("creating custom currency overage detail: %w", err)
	}

	stdLine.DetailedLines = stdLine.DetailedLinesWithIDReuse(billing.DetailedLines{detailedLine})
	stdLine.Totals = stdLine.DetailedLines.SumTotals().RoundToPrecision(invoiceCurrency)

	if !stdLine.Totals.Total.Equal(fiatOverage.Amount) {
		return fmt.Errorf(
			"custom currency charge[%s] mapped overage total mismatch [line_id=%s run_id=%s line_total=%s overage_total=%s]",
			charge.ID,
			stdLine.ID,
			run.ID.ID,
			stdLine.Totals.Total.String(),
			fiatOverage.Amount.String(),
		)
	}

	if (input.Stage == standardLinePopulationStageGatheringPreview ||
		input.Stage == standardLinePopulationStageCollectionCompleted) &&
		isZeroFiatAmountOverageRun(charge, run) {
		stdLine.DeletedAt = lo.ToPtr(clock.Now())
	}

	return nil
}

func mapFlatFeeDetailedLines(stdLine *billing.StandardLine, run flatfee.RealizationRun) (billing.DetailedLines, error) {
	if run.DetailedLines.IsAbsent() {
		return nil, fmt.Errorf("run %s detailed lines must be expanded", run.ID.ID)
	}

	return lo.Map(run.DetailedLines.OrEmpty(), func(line flatfee.DetailedLine, _ int) billing.DetailedLine {
		base := line.Clone()
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
