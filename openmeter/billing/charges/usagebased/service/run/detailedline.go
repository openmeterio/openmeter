package run

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *Service) ensureDetailedLinesLoadedForRating(ctx context.Context, charge usagebased.Charge) (usagebased.Charge, error) {
	if len(charge.Realizations) == 0 {
		return charge, nil
	}

	if lo.EveryBy(charge.Realizations, func(run usagebased.RealizationRun) bool {
		return run.DetailedLines.IsPresent()
	}) {
		return charge, nil
	}

	expandedCharge, err := s.adapter.FetchDetailedLines(ctx, charge)
	if err != nil {
		return usagebased.Charge{}, fmt.Errorf("fetch detailed lines: %w", err)
	}

	return expandedCharge, nil
}

func mapRatingResultToRunDetailedLines(
	charge usagebased.Charge,
	run usagebased.RealizationRun,
	ratingResult usagebasedrating.GetDetailedRatingForUsageResult,
) usagebased.DetailedLines {
	return lo.Map(ratingResult.DetailedLines, func(line billingrating.DetailedLine, _ int) usagebased.DetailedLine {
		period := charge.Intent.ServicePeriod
		if line.Period != nil {
			period = *line.Period
		}

		category := line.Category
		if category == "" {
			category = stddetailedline.CategoryRegular
		}

		paymentTerm := lo.CoalesceOrEmpty(line.PaymentTerm, productcatalog.InArrearsPaymentTerm)

		return usagebased.DetailedLine{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: charge.Namespace,
				Name:      line.Name,
			}),
			ServicePeriod:          period,
			Currency:               charge.Intent.Currency,
			ChildUniqueReferenceID: line.ChildUniqueReferenceID,
			PaymentTerm:            paymentTerm,
			PerUnitAmount:          line.PerUnitAmount,
			Quantity:               line.Quantity,
			Category:               category,
			CreditsApplied:         line.CreditsApplied,
			Totals:                 line.Totals,
			TaxConfig:              cloneTaxConfig(charge.Intent.TaxConfig),
		}
	})
}

func cloneTaxConfig(cfg *productcatalog.TaxConfig) *productcatalog.TaxConfig {
	if cfg == nil {
		return nil
	}

	cloned := cfg.Clone()
	return &cloned
}

func (s *Service) PersistRunDetailedLines(
	ctx context.Context,
	charge usagebased.Charge,
	run usagebased.RealizationRun,
	ratingResult usagebasedrating.GetDetailedRatingForUsageResult,
) (usagebased.DetailedLines, error) {
	detailedLines := mapRatingResultToRunDetailedLines(charge, run, ratingResult)

	if err := s.adapter.UpsertRunDetailedLines(ctx, charge.GetChargeID(), run.ID, detailedLines); err != nil {
		return nil, fmt.Errorf("upsert run detailed lines: %w", err)
	}

	return detailedLines, nil
}
