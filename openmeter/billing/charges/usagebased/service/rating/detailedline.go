package rating

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func mapBillingRatingDetailedLinesToUsageBasedDetailedLines(
	intent usagebased.Intent,
	defaultServicePeriod timeutil.ClosedPeriod,
	lines billingrating.DetailedLines,
) usagebased.DetailedLines {
	return usagebased.NewDetailedLinesFromBilling(intent, defaultServicePeriod, lines)
}

func (s *service) ensureDetailedLinesLoadedForRating(ctx context.Context, charge usagebased.Charge, servicePeriodTo time.Time) (usagebased.Charge, error) {
	if len(charge.Realizations) == 0 {
		return charge, nil
	}

	if !lo.EveryBy(charge.Realizations, func(run usagebased.RealizationRun) bool {
		return !run.ServicePeriodTo.Before(servicePeriodTo) || run.DetailedLines.IsPresent()
	}) {
		expandedCharge, err := s.detailedLinesFetcher.FetchDetailedLines(ctx, charge)
		if err != nil {
			return usagebased.Charge{}, fmt.Errorf("fetch detailed lines: %w", err)
		}

		charge = expandedCharge
	}

	for idx, run := range charge.Realizations {
		// Extra safety: the fetcher contract should return all prior-run detailed
		// lines, but rating must not proceed with incomplete prior runs as we will overcharge
		// customers.
		if run.ServicePeriodTo.Before(servicePeriodTo) && !run.DetailedLines.IsPresent() {
			return usagebased.Charge{}, fmt.Errorf("prior runs[%d]: detailed lines must be expanded", idx)
		}
	}

	return charge, nil
}
