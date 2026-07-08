package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/mutator"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

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
			Currency:    line.GetCurrency(),
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
	Run  usagebased.RealizationRun
	Runs usagebased.RealizationRuns
}

func populateStandardLineFromRun(stdLine *billing.StandardLine, input populateStandardLineFromRunInput) error {
	if stdLine.UsageBased == nil {
		stdLine.UsageBased = &billing.UsageBasedLine{}
	}

	currencyCalculator, err := stdLine.Currency.Calculator()
	if err != nil {
		return fmt.Errorf("creating currency calculator: %w", err)
	}

	billingMeteredQuantity, err := input.Runs.MapToBillingMeteredQuantity(input.Run)
	if err != nil {
		return fmt.Errorf("mapping run metered quantity to billing: %w", err)
	}

	stdLine.OverrideCollectionPeriodEnd = lo.ToPtr(input.Run.StoredAtLT.Add(usagebased.InternalCollectionPeriod))
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

	mappedDetailedLines, err := mapUsageBasedDetailedLines(stdLine, input.Run, currencyCalculator)
	if err != nil {
		return fmt.Errorf("mapping run detailed lines: %w", err)
	}

	stdLine.DetailedLines = stdLine.DetailedLinesWithIDReuse(mappedDetailedLines)
	stdLine.Totals = stdLine.DetailedLines.SumTotals().RoundToPrecision(currencyCalculator)

	expectedTotals := input.Run.Totals.RoundToPrecision(currencyCalculator)
	if !stdLine.Totals.Equal(expectedTotals) {
		return fmt.Errorf("mapped line totals do not match run totals [line_id=%s run_id=%s line_total=%s run_total=%s]",
			stdLine.ID, input.Run.ID.ID, stdLine.Totals.Total.String(), expectedTotals.Total.String())
	}

	return nil
}

func mapUsageBasedDetailedLines(
	stdLine *billing.StandardLine,
	run usagebased.RealizationRun,
	currencyCalculator currencyx.Calculator,
) (billing.DetailedLines, error) {
	if run.DetailedLines.IsAbsent() {
		return nil, fmt.Errorf("run %s detailed lines must be expanded", run.ID.ID)
	}

	detailedLines := billing.DetailedLines(lo.Map(run.DetailedLines.OrEmpty(), func(line usagebased.DetailedLine, _ int) billing.DetailedLine {
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
	}))

	detailedLines, err := detailedLines.WithCreditsApplied(stdLine.CreditsApplied, currencyCalculator)
	if err != nil {
		return nil, err
	}

	return detailedLines, nil
}
