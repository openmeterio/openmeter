package realizations

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
)

type RateResult struct {
	Intent        flatfee.RateableIntent
	DetailedLines flatfee.DetailedLines
	Totals        totals.Totals
}

func (s *Service) Rate(intent flatfee.RateableIntent) (RateResult, error) {
	if err := intent.Validate(); err != nil {
		return RateResult{}, fmt.Errorf("validating rateable intent: %w", err)
	}

	result, err := s.ratingService.GenerateDetailedLines(
		intent,
		billingrating.WithCreditsMutatorDisabled(),
	)
	if err != nil {
		return RateResult{}, fmt.Errorf("generating detailed lines: %w", err)
	}

	detailedLines := flatfee.NewDetailedLinesFromRating(intent.ServicePeriod, result.DetailedLines)
	if err := detailedLines.Validate(); err != nil {
		return RateResult{}, fmt.Errorf("validating detailed lines: %w", err)
	}

	return RateResult{
		Intent:        intent,
		DetailedLines: detailedLines,
		Totals:        result.Totals,
	}, nil
}
