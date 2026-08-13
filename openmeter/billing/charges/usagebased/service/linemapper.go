package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/mutator"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type standardLinePopulationStage string

const (
	standardLinePopulationStageGatheringPreview    standardLinePopulationStage = "gathering_preview"
	standardLinePopulationStageInvoiceCreated      standardLinePopulationStage = "invoice_created"
	standardLinePopulationStageCollectionCompleted standardLinePopulationStage = "collection_completed"
	standardLinePopulationStageManualAttachment    standardLinePopulationStage = "manual_attachment"
)

func (s standardLinePopulationStage) Validate() error {
	switch s {
	case standardLinePopulationStageGatheringPreview,
		standardLinePopulationStageInvoiceCreated,
		standardLinePopulationStageCollectionCompleted,
		standardLinePopulationStageManualAttachment:
		return nil
	case "":
		return fmt.Errorf("standard line population stage is required")
	default:
		return fmt.Errorf("invalid standard line population stage: %q", s)
	}
}

func intentFromManualCreatedLine(
	ctx context.Context,
	invoice billing.GenericInvoiceReader,
	line billing.GenericInvoiceLineReader,
	defaultInvoicingTaxCodeResolver billing.DefaultTaxCodeResolver,
) (usagebased.Intent, error) {
	if invoice == nil {
		return usagebased.Intent{}, fmt.Errorf("invoice is required")
	}

	if line == nil {
		return usagebased.Intent{}, fmt.Errorf("line is required")
	}

	if line.GetID() == "" {
		return usagebased.Intent{}, fmt.Errorf("line id is required")
	}

	currency, err := line.GetCurrency().AsFiatCurrency()
	if err != nil {
		return usagebased.Intent{}, fmt.Errorf("resolving fiat currency %q: %w", line.GetCurrency(), err)
	}

	if chargeID := line.GetChargeID(); chargeID != nil && *chargeID != "" {
		return usagebased.Intent{}, fmt.Errorf("line[%s]: charge id must be empty for manual create", line.GetID())
	}

	price := line.GetPrice()
	if price == nil {
		return usagebased.Intent{}, fmt.Errorf("line[%s]: price is required", line.GetID())
	}

	if line.GetFeatureKey() == "" {
		return usagebased.Intent{}, fmt.Errorf("line[%s]: feature key is required", line.GetID())
	}

	annotations, err := line.GetAnnotations().Clone()
	if err != nil {
		return usagebased.Intent{}, fmt.Errorf("cloning line[%s] annotations: %w", line.GetID(), err)
	}

	servicePeriod := line.GetServicePeriod()
	invoiceAt := servicePeriod.To
	if invoiceAtAccessor, ok := line.(billing.InvoiceAtAccessor); ok {
		invoiceAt = invoiceAtAccessor.GetInvoiceAt()
	}

	taxConfig := productcatalog.TaxCodeConfig{}
	if lineTaxConfig := line.GetTaxConfig(); lineTaxConfig != nil {
		taxConfig = productcatalog.TaxCodeConfigFrom(lineTaxConfig.ToProductCatalog())
	}

	var unitConfig *productcatalog.UnitConfig
	if config := line.GetUnitConfig(); config != nil {
		unitConfig = lo.ToPtr(config.Clone())
	}

	intent := usagebased.Intent{
		Intent: meta.Intent{
			ManagedBy:   billing.ManuallyManagedLine,
			CustomerID:  invoice.GetCustomerID().ID,
			Annotations: annotations,
			Currency:    currencies.Currency{Currency: currency},
			TaxConfig:   taxConfig,
		},
		IntentMutableFields: usagebased.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              line.GetName(),
				Description:       line.GetDescription(),
				Metadata:          line.GetMetadata().Clone(),
				ServicePeriod:     servicePeriod,
				FullServicePeriod: servicePeriod,
				BillingPeriod:     servicePeriod,
			},
			InvoiceAt:  invoiceAt,
			Price:      *price.Clone(),
			Discounts:  line.GetRateCardDiscounts().Clone(),
			UnitConfig: unitConfig,
		},
		FeatureKey:     line.GetFeatureKey(),
		SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
	}

	intent = intent.Normalized()
	if intent.TaxConfig.TaxCodeID == "" {
		if defaultInvoicingTaxCodeResolver == nil {
			return usagebased.Intent{}, fmt.Errorf("line[%s]: default invoicing tax code resolver is required", line.GetID())
		}

		defaultTaxCodeID, err := defaultInvoicingTaxCodeResolver(ctx)
		if err != nil {
			return usagebased.Intent{}, fmt.Errorf("resolving default invoicing tax code: %w", err)
		}

		intent.TaxConfig.TaxCodeID = defaultTaxCodeID
	}

	if err := intent.Validate(); err != nil {
		return usagebased.Intent{}, err
	}

	return intent, nil
}

type populateStandardLineFromRunInput struct {
	Charge usagebased.Charge
	Run    usagebased.RealizationRun
	Stage  standardLinePopulationStage
}

type calculateFiatOverageForRunResult struct {
	FiatCurrency          *currencyx.FiatCurrency
	FiatOverage           alpacadecimal.Decimal
	ShouldOmitInvoiceLine bool
}

// calculateFiatOverageForRun resolves the post-allocation custom-currency
// overage and the invoice-line omission decision from the same conversion.
func calculateFiatOverageForRun(charge usagebased.Charge, run usagebased.RealizationRun) (calculateFiatOverageForRunResult, error) {
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

func (i populateStandardLineFromRunInput) Validate() error {
	var errs []error

	if err := i.Stage.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func populateStandardLineFromRun(stdLine *billing.StandardLine, input populateStandardLineFromRunInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if stdLine.UsageBased == nil {
		stdLine.UsageBased = &billing.UsageBasedLine{}
	}

	cur, err := stdLine.Currency.AsFiatCurrency()
	if err != nil {
		return fmt.Errorf("creating currency calculator: %w", err)
	}

	// Wait until StoredAtLT plus the internal collection period before collecting the line.
	// This ensures the usage snapshot bounded by StoredAtLT is no longer changing when billing reads it.
	stdLine.OverrideCollectionPeriodEnd = lo.ToPtr(input.Run.StoredAtLT.Add(usagebased.InternalCollectionPeriod))

	if input.Charge.Intent.GetCurrency().IsCustom() {
		return populateCustomCurrencyOverageFromRun(stdLine, input)
	}

	billingMeteredQuantity, err := input.Charge.Realizations.MapToBillingMeteredQuantity(input.Run)
	if err != nil {
		return fmt.Errorf("mapping run metered quantity to billing: %w", err)
	}

	stdLine.UsageBased.MeteredQuantity = lo.ToPtr(billingMeteredQuantity.LinePeriod)
	stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(billingMeteredQuantity.PreLinePeriod)

	// Charge runs store cumulative raw metered quantity. Billing lines expose the raw
	// metered values (MeteredQuantity above) separately from net billable quantities and
	// consumed usage discounts. Convert the raw quantity through the rate card's
	// unit_config before the discount — mirroring the rating pipeline's
	// [UnitConfig, DiscountUsage] order — so the displayed billable Quantity matches the
	// priced amount rather than staying in raw metered units. A nil unit_config is the
	// identity, so non-unit_config lines are unchanged.
	billableUsage := mutator.ApplyUnitConfig(billingrating.Usage{
		Quantity:              billingMeteredQuantity.LinePeriod,
		PreLinePeriodQuantity: billingMeteredQuantity.PreLinePeriod,
	}, stdLine.UsageBased.UnitConfig)

	discountedUsage, err := mutator.ApplyUsageDiscount(mutator.ApplyUsageDiscountInput{
		Usage:                 billableUsage,
		RateCardDiscounts:     stdLine.RateCardDiscounts,
		StandardLineDiscounts: stdLine.Discounts,
	})
	if err != nil {
		return fmt.Errorf("applying usage discount: %w", err)
	}

	stdLine.UsageBased.Quantity = lo.ToPtr(discountedUsage.Usage.Quantity)
	stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(discountedUsage.Usage.PreLinePeriodQuantity)
	stdLine.Discounts = discountedUsage.StandardLineDiscounts

	creditsApplied, err := input.Run.CreditsAllocated.AsCreditsApplied()
	if err != nil {
		return err
	}

	stdLine.CreditsApplied = creditsApplied

	mappedDetailedLines, err := mapUsageBasedDetailedLines(stdLine, input.Run, cur)
	if err != nil {
		return fmt.Errorf("mapping run detailed lines: %w", err)
	}

	stdLine.DetailedLines = stdLine.DetailedLinesWithIDReuse(mappedDetailedLines)
	stdLine.Totals = stdLine.DetailedLines.SumTotals().RoundToPrecision(cur)

	expectedTotals := input.Run.Totals.RoundToPrecision(cur)
	if !stdLine.Totals.Equal(expectedTotals) {
		return fmt.Errorf("mapped line totals do not match run totals [line_id=%s run_id=%s line_total=%s run_total=%s]",
			stdLine.ID, input.Run.ID.ID, stdLine.Totals.Total.String(), expectedTotals.Total.String())
	}

	return nil
}

func populateCustomCurrencyOverageFromRun(
	stdLine *billing.StandardLine,
	input populateStandardLineFromRunInput,
) error {
	charge := input.Charge
	run := input.Run

	fiatOverage, err := calculateFiatOverageForRun(charge, run)
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

func mapUsageBasedDetailedLines(
	stdLine *billing.StandardLine,
	run usagebased.RealizationRun,
	currencyCalculator currencyx.Currency,
) (billing.DetailedLines, error) {
	if currencyCalculator == nil {
		return nil, fmt.Errorf("currency calculator is required")
	}

	if run.DetailedLines.IsAbsent() {
		return nil, fmt.Errorf("run %s detailed lines must be expanded", run.ID.ID)
	}

	detailedLines := billing.DetailedLines(lo.Map(run.DetailedLines.OrEmpty(), func(line usagebased.DetailedLine, _ int) billing.DetailedLine {
		base := line.CloneStandardBase()
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
	}))

	detailedLines, err := detailedLines.WithCreditsApplied(stdLine.CreditsApplied, currencyCalculator)
	if err != nil {
		return nil, err
	}

	return detailedLines, nil
}
