package delta

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating/subtract"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Engine struct {
	ratingService billingrating.Service
}

func New(ratingService billingrating.Service) Engine {
	return Engine{
		ratingService: ratingService,
	}
}

type Input struct {
	Intent usagebased.Intent

	CurrentPeriod CurrentPeriod

	// AlreadyBilledDetailedLines contains all detailed lines already booked for
	// previous runs of the same charge. Credits are stripped before subtraction
	// because credit allocation is not part of rating.
	AlreadyBilledDetailedLines usagebased.DetailedLines
}

func (i Input) Validate() error {
	var errs []error
	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	intentServicePeriod := i.Intent.ServicePeriod
	if !intentServicePeriod.ContainsPeriodInclusive(i.CurrentPeriod.ServicePeriod) {
		errs = append(errs, fmt.Errorf("current period service period must be contained in intent service period: [%s..%s] vs [%s..%s]",
			intentServicePeriod.From.Format(time.RFC3339), intentServicePeriod.To.Format(time.RFC3339),
			i.CurrentPeriod.ServicePeriod.From.Format(time.RFC3339), i.CurrentPeriod.ServicePeriod.To.Format(time.RFC3339)))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CurrentPeriod struct {
	// MeteredQuantity is the metered quantity for [intent.ServicePeriodFrom ... servicePeriod.To) capped by StoredAtLT of the current run
	MeteredQuantity alpacadecimal.Decimal

	// ServicePeriod is the service period for the current period (from is intent.ServicePeriod.From if this is the first run or
	// the previous run's servicePeriod.To if this is not the first run)
	ServicePeriod timeutil.ClosedPeriod
}

type Result struct {
	DetailedLines usagebased.DetailedLines
}

// Rate is the temporary production rater for detailed lines:
// rate the cumulative current snapshot, subtract everything already booked, and
// book the remainder on the current run period. The period-preserving late-event
// rater stays separate until downstream invoicing can safely handle corrections.
func (e Engine) Rate(_ context.Context, in Input) (Result, error) {
	if err := in.Validate(); err != nil {
		return Result{}, err
	}

	opts := []billingrating.GenerateDetailedLinesOption{
		billingrating.WithCreditsMutatorDisabled(),
	}
	// Minimum commitment is charged only on the final service-period snapshot.
	if in.CurrentPeriod.ServicePeriod.To.Before(in.Intent.ServicePeriod.To) {
		opts = append(opts, billingrating.WithMinimumCommitmentIgnored())
	}

	billingDetailedLines, err := e.ratingService.GenerateDetailedLines(usagebased.RateableIntent{
		Intent:        in.Intent,
		ServicePeriod: in.CurrentPeriod.ServicePeriod,
		MeterValue:    in.CurrentPeriod.MeteredQuantity,
	}, opts...)
	if err != nil {
		return Result{}, fmt.Errorf("generating detailed lines: %w", err)
	}

	currentDetailedLines := usagebased.NewDetailedLinesFromBilling(
		in.Intent,
		in.CurrentPeriod.ServicePeriod,
		billingDetailedLines.DetailedLines,
	)

	alreadyBilledDetailedLines := make(usagebased.DetailedLines, 0, len(in.AlreadyBilledDetailedLines))
	for _, line := range in.AlreadyBilledDetailedLines {
		// Credits are allocated after rating. Remove them before subtraction so
		// credit allocation changes do not look like pricing or usage changes.
		line = line.Clone()
		line.CreditsApplied = nil
		line.Totals.CreditsTotal = alpacadecimal.Zero
		line.Totals.Total = line.Totals.CalculateTotal()
		alreadyBilledDetailedLines = append(alreadyBilledDetailedLines, line)
	}

	remainingDetailedLines, err := subtract.SubtractRatedRunDetails(
		currentDetailedLines,
		alreadyBilledDetailedLines,
		uniqueReferenceIDGenerator{},
	)
	if err != nil {
		return Result{}, fmt.Errorf("subtracting detailed lines: %w", err)
	}

	for idx := range remainingDetailedLines {
		// This rater intentionally books every delta on the current run period.
		// The period-preserving late-event rater owns correction metadata.
		remainingDetailedLines[idx].ServicePeriod = in.CurrentPeriod.ServicePeriod
		remainingDetailedLines[idx].CorrectsRunID = nil
	}

	remainingDetailedLines.Sort()
	for idx := range remainingDetailedLines {
		// Indexes are persisted as part of the generated detailed-line contract.
		index := idx
		remainingDetailedLines[idx].Index = &index
	}

	childUniqueReferenceIDs := lo.GroupBy(remainingDetailedLines, func(line usagebased.DetailedLine) string {
		return line.ChildUniqueReferenceID
	})

	for id, lines := range childUniqueReferenceIDs {
		if len(lines) > 1 {
			return Result{}, fmt.Errorf("duplicate child unique reference id: %s", id)
		}
	}

	return Result{
		DetailedLines: remainingDetailedLines,
	}, nil
}
