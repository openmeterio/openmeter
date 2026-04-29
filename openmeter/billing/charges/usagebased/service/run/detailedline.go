package run

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
)

func mapRatingResultToRunDetailedLines(
	ratingResult usagebasedrating.GetDetailedRatingForUsageResult,
) usagebased.DetailedLines {
	return ratingResult.DetailedLines
}

func (s *Service) PersistRunDetailedLines(
	ctx context.Context,
	charge usagebased.Charge,
	run usagebased.RealizationRun,
	ratingResult usagebasedrating.GetDetailedRatingForUsageResult,
) (usagebased.DetailedLines, error) {
	detailedLines := mapRatingResultToRunDetailedLines(ratingResult)

	if err := s.adapter.UpsertRunDetailedLines(ctx, charge.GetChargeID(), run.ID, detailedLines); err != nil {
		return nil, fmt.Errorf("upsert run detailed lines: %w", err)
	}

	return detailedLines, nil
}
