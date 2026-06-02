package service

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/mutator"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func populateUsageBasedStandardLineFromRun(stdLine *billing.StandardLine, run usagebased.RealizationRun, runs usagebased.RealizationRuns) error {
	if stdLine.UsageBased == nil {
		stdLine.UsageBased = &billing.UsageBasedLine{}
	}

	currencyCalculator, err := stdLine.Currency.Calculator()
	if err != nil {
		return fmt.Errorf("creating currency calculator: %w", err)
	}

	billingMeteredQuantity, err := runs.MapToBillingMeteredQuantity(run)
	if err != nil {
		return fmt.Errorf("mapping run metered quantity to billing: %w", err)
	}

	stdLine.OverrideCollectionPeriodEnd = lo.ToPtr(run.StoredAtLT.Add(usagebased.InternalCollectionPeriod))
	stdLine.UsageBased.MeteredQuantity = lo.ToPtr(billingMeteredQuantity.LinePeriod)
	stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(billingMeteredQuantity.PreLinePeriod)

	// Charge runs store cumulative raw metered quantity. Billing lines expose the raw
	// metered values separately from net billable quantities and consumed usage discounts,
	// so reuse the standard billing usage-discount mutator contract here.
	discountedUsage, err := mutator.ApplyUsageDiscount(mutator.ApplyUsageDiscountInput{
		Usage: billingrating.Usage{
			Quantity:              billingMeteredQuantity.LinePeriod,
			PreLinePeriodQuantity: billingMeteredQuantity.PreLinePeriod,
		},
		RateCardDiscounts:     stdLine.RateCardDiscounts,
		StandardLineDiscounts: stdLine.Discounts,
	})
	if err != nil {
		return fmt.Errorf("applying usage discount: %w", err)
	}

	stdLine.UsageBased.Quantity = lo.ToPtr(discountedUsage.Usage.Quantity)
	stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(discountedUsage.Usage.PreLinePeriodQuantity)
	stdLine.Discounts = discountedUsage.StandardLineDiscounts

	creditsApplied, err := run.CreditsAllocated.AsCreditsApplied()
	if err != nil {
		return err
	}

	stdLine.CreditsApplied = creditsApplied

	mappedDetailedLines, err := mapUsageBasedDetailedLines(stdLine, run, currencyCalculator)
	if err != nil {
		return fmt.Errorf("mapping run detailed lines: %w", err)
	}

	stdLine.DetailedLines = stdLine.DetailedLinesWithIDReuse(mappedDetailedLines)
	stdLine.Totals = stdLine.DetailedLines.SumTotals().RoundToPrecision(currencyCalculator)

	expectedTotals := run.Totals.RoundToPrecision(currencyCalculator)
	if !stdLine.Totals.Equal(expectedTotals) {
		return fmt.Errorf("mapped line totals do not match run totals [line_id=%s run_id=%s line_total=%s run_total=%s]",
			stdLine.ID, run.ID.ID, stdLine.Totals.Total.String(), expectedTotals.Total.String())
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
