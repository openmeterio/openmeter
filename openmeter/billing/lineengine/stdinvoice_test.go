package lineengine

import (
	"errors"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCalculateLinesPreservesResultsWithValidationWarnings(t *testing.T) {
	line := standardLineForLineEngineOverrideTest(t, lineEngineOverrideTestPeriod())
	ratingResult := rating.GenerateDetailedLinesResult{
		FinalUsage: &rating.Usage{
			Quantity:              alpacadecimal.NewFromInt(2),
			PreLinePeriodQuantity: alpacadecimal.Zero,
		},
		Totals: totals.Totals{
			Amount: alpacadecimal.NewFromInt(2),
			Total:  alpacadecimal.NewFromInt(2),
		},
	}
	engine := &Engine{
		ratingService: calculateLinesRatingService{
			result: ratingResult,
			err: billing.ValidationWithComponent(
				billing.ValidationComponentBillingRating,
				billing.WarnNegativeMeteredQuantityClamped,
			),
		},
	}

	lines, err := engine.CalculateLines(billing.CalculateLinesInput{
		Invoice: billing.StandardInvoice{StandardInvoiceBase: billing.StandardInvoiceBase{ID: "invoice-id"}},
		Lines:   billing.StandardLines{line},
	})

	issues, systemErr := billing.ToValidationIssues(err)
	require.NoError(t, systemErr)
	require.Equal(t, billing.ValidationIssues{{
		Severity:  billing.ValidationIssueSeverityWarning,
		Message:   billing.WarnNegativeMeteredQuantityClamped.Message,
		Code:      billing.WarnNegativeMeteredQuantityClamped.Code,
		Component: billing.ValidationComponentBillingRating,
		Attributes: models.Annotations{
			billing.AttributeKeyLineID: line.ID,
		},
	}}, issues)
	require.Equal(t, billing.StandardLines{line}, lines)
	require.Equal(t, float64(2), line.UsageBased.Quantity.InexactFloat64())
	require.Equal(t, float64(2), line.Totals.Amount.InexactFloat64())
	require.Equal(t, float64(2), line.Totals.Total.InexactFloat64())
}

func TestCalculateLinesRejectsSystemErrors(t *testing.T) {
	ratingErr := errors.New("rating unavailable")
	engine := &Engine{
		ratingService: calculateLinesRatingService{err: ratingErr},
	}

	lines, err := engine.CalculateLines(billing.CalculateLinesInput{
		Invoice: billing.StandardInvoice{StandardInvoiceBase: billing.StandardInvoiceBase{ID: "invoice-id"}},
		Lines: billing.StandardLines{
			standardLineForLineEngineOverrideTest(t, lineEngineOverrideTestPeriod()),
		},
	})

	require.ErrorIs(t, err, ratingErr)
	require.Nil(t, lines)
}

type calculateLinesRatingService struct {
	result rating.GenerateDetailedLinesResult
	err    error
}

func (s calculateLinesRatingService) GenerateDetailedLines(rating.StandardLineAccessor, ...rating.GenerateDetailedLinesOption) (rating.GenerateDetailedLinesResult, error) {
	return s.result, s.err
}

func (calculateLinesRatingService) ResolveBillablePeriod(rating.ResolveBillablePeriodInput) (billing.IsLineBillableAsOfResult, error) {
	return billing.IsLineBillableAsOfResult{}, nil
}
