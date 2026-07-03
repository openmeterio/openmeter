package service

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/mutator"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type populateStandardLineFromRunInput struct {
	Run        usagebased.RealizationRun
	Runs       usagebased.RealizationRuns
	UnitConfig *productcatalog.UnitConfig
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
	}, input.UnitConfig)

	// Snapshot the config that produced the conversion above onto the line, so the
	// metered→invoiced derivation stays auditable and re-rating converts identically
	// even if the rate card's unit_config is edited after invoicing. Today the source
	// is the charge intent's effective config (a reconciliation-time copy of the rate
	// card); once unit_config is frozen onto the subscription item at subscription
	// creation, the intent — and therefore this snapshot — will carry that frozen value.
	stdLine.UsageBased.AppliedUnitConfig = input.UnitConfig

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
