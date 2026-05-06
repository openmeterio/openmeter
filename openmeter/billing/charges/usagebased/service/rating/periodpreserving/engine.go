package periodpreserving

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating/subtract"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/streaming"
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
	PriorPeriods  []PriorPeriod
}

func (i Input) Validate() error {
	var errs []error
	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	intentServicePeriod := i.Intent.ServicePeriod
	// Period-preserving rating only supports rating windows inside the charge's
	// service period. Otherwise the meter snapshot and output line period could
	// describe usage outside the charge intent.
	if !intentServicePeriod.ContainsPeriodInclusive(i.CurrentPeriod.ServicePeriod) {
		errs = append(errs, fmt.Errorf("current period service period must be contained in intent service period: [%s..%s] vs [%s..%s]",
			intentServicePeriod.From.Format(time.RFC3339), intentServicePeriod.To.Format(time.RFC3339),
			i.CurrentPeriod.ServicePeriod.From.Format(time.RFC3339), i.CurrentPeriod.ServicePeriod.To.Format(time.RFC3339)))
	}
	if err := i.CurrentPeriod.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("current period service period: %w", err))
	}

	for idx, priorPeriod := range i.PriorPeriods {
		// Prior period snapshots are re-rated against the same charge intent, so
		// they must also stay inside the charge service period.
		if !intentServicePeriod.ContainsPeriodInclusive(priorPeriod.ServicePeriod) {
			errs = append(errs, fmt.Errorf("prior period service period must be contained in intent service period: [%s..%s] vs [%s..%s]",
				intentServicePeriod.From.Format(time.RFC3339), intentServicePeriod.To.Format(time.RFC3339),
				priorPeriod.ServicePeriod.From.Format(time.RFC3339), priorPeriod.ServicePeriod.To.Format(time.RFC3339)))
		}
		if err := priorPeriod.ServicePeriod.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("prior periods[%d] service period: %w", idx, err))
		}

		// Meter snapshots are evaluated at the streaming minimum window
		// precision. If a prior service period collapses to an empty period at
		// that precision, it cannot produce a meaningful prior-period rating
		// bucket.
		if priorPeriod.ServicePeriod.Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
			errs = append(errs, fmt.Errorf("prior periods[%d] service period must not be empty when truncated to minimum streaming window size: [%s..%s]",
				idx,
				priorPeriod.ServicePeriod.From.Format(time.RFC3339Nano), priorPeriod.ServicePeriod.To.Format(time.RFC3339Nano)))
		}
	}

	for idx, priorPeriod := range i.PriorPeriods {
		// Each rating window must describe a distinct service-period bucket.
		// Overlaps would let multiple snapshots produce lines for the same period,
		// which can leak duplicate child unique reference IDs to persistence.
		if priorPeriod.ServicePeriod.Overlaps(i.CurrentPeriod.ServicePeriod) {
			errs = append(errs, fmt.Errorf("prior periods[%d] service period overlaps current period service period: [%s..%s] vs [%s..%s]",
				idx,
				priorPeriod.ServicePeriod.From.Format(time.RFC3339), priorPeriod.ServicePeriod.To.Format(time.RFC3339),
				i.CurrentPeriod.ServicePeriod.From.Format(time.RFC3339), i.CurrentPeriod.ServicePeriod.To.Format(time.RFC3339)))
		}

		for otherIdx := idx + 1; otherIdx < len(i.PriorPeriods); otherIdx++ {
			otherPriorPeriod := i.PriorPeriods[otherIdx]
			if priorPeriod.ServicePeriod.Overlaps(otherPriorPeriod.ServicePeriod) {
				errs = append(errs, fmt.Errorf("prior periods[%d] service period overlaps prior periods[%d] service period: [%s..%s] vs [%s..%s]",
					idx, otherIdx,
					priorPeriod.ServicePeriod.From.Format(time.RFC3339), priorPeriod.ServicePeriod.To.Format(time.RFC3339),
					otherPriorPeriod.ServicePeriod.From.Format(time.RFC3339), otherPriorPeriod.ServicePeriod.To.Format(time.RFC3339)))
			}
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// SortPriorPeriods sorts the prior periods by service period from ascending.
func (i *Input) SortPriorPeriods() {
	slices.SortStableFunc(i.PriorPeriods, func(a, b PriorPeriod) int {
		return cmp.Compare(a.ServicePeriod.From.UnixNano(), b.ServicePeriod.From.UnixNano())
	})
}

type CurrentPeriod struct {
	// MeteredQuantity is the metered quantity for [intent.ServicePeriodFrom ... servicePeriod.To) capped by StoredAtLT of the current run
	MeteredQuantity alpacadecimal.Decimal

	// ServicePeriod is the service period for the current period (from is intent.ServicePeriod.From if this is the first run or
	// the previous run's servicePeriod.To if this is not the first run)
	ServicePeriod timeutil.ClosedPeriod
}

type PriorPeriod struct {
	RunID usagebased.RealizationRunID

	// MeteredQuantity is the metered quantity for [intent.ServicePeriodFrom ... servicePeriod.To) capped by StoredAtLT of the current run
	MeteredQuantity alpacadecimal.Decimal

	// ServicePeriod is the service period for the prior period (from is intent.ServicePeriod.From, for the first item or
	// servicePeriod.From of the previous item)
	ServicePeriod timeutil.ClosedPeriod

	// DetailedLines are the detailed lines billed for the prior period
	DetailedLines usagebased.DetailedLines
}

type Result struct {
	DetailedLines usagebased.DetailedLines
}

type epochClosedPeriod struct {
	From int64
	To   int64
}

func (e epochClosedPeriod) AsClosedPeriod() timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: time.Unix(e.From, 0).In(time.UTC),
		To:   time.Unix(e.To, 0).In(time.UTC),
	}
}

func closedPeriodToEpochClosedPeriod(period timeutil.ClosedPeriod) epochClosedPeriod {
	return epochClosedPeriod{
		From: period.From.Unix(),
		To:   period.To.Unix(),
	}
}

type epochPeriodRatingInput struct {
	epochClosedPeriod
	Quantity alpacadecimal.Decimal
}

func (e Engine) Rate(ctx context.Context, in Input) (Result, error) {
	if err := in.Validate(); err != nil {
		return Result{}, err
	}

	detailedLinesByEpoch, err := e.buildDetailsByEpoch(ctx, in)
	if err != nil {
		return Result{}, fmt.Errorf("full rating with epochs: %w", err)
	}

	finalDetailedLines, err := flattenDetailedLinesByEpoch(detailedLinesByEpoch)
	if err != nil {
		return Result{}, err
	}

	return Result{
		DetailedLines: finalDetailedLines,
	}, nil
}

func (e Engine) buildDetailsByEpoch(ctx context.Context, in Input) (map[epochClosedPeriod]usagebased.DetailedLines, error) {
	// We need to first generate the expected detailed lines for each invoice without taking the already billed lines into account.
	fullRatingInput := make([]epochPeriodRatingInput, 0, len(in.PriorPeriods)+1)
	// Note: this is only to make sure that the input is sorted (it should be already sorted by the caller, but it's a safety measure in
	// case of future changes).
	in.SortPriorPeriods()

	for _, priorPeriod := range in.PriorPeriods {
		fullRatingInput = append(fullRatingInput, epochPeriodRatingInput{
			epochClosedPeriod: epochClosedPeriod{
				From: priorPeriod.ServicePeriod.From.Unix(),
				To:   priorPeriod.ServicePeriod.To.Unix(),
			},
			Quantity: priorPeriod.MeteredQuantity,
		})
	}

	fullRatingInput = append(fullRatingInput, epochPeriodRatingInput{
		epochClosedPeriod: epochClosedPeriod{
			From: in.CurrentPeriod.ServicePeriod.From.Unix(),
			To:   in.CurrentPeriod.ServicePeriod.To.Unix(),
		},
		Quantity: in.CurrentPeriod.MeteredQuantity,
	})

	// Let's start the rating process
	previouslyGeneratedDetailedLines := make(usagebased.DetailedLines, 0, 32) // 32 is a reasonable default size for most cases
	result := make(map[epochClosedPeriod]usagebased.DetailedLines, len(fullRatingInput))
	for _, epoch := range fullRatingInput {
		opts := []billingrating.GenerateDetailedLinesOption{
			billingrating.WithCreditsMutatorDisabled(),
		}
		// Minimum commitment is charged only on the final service-period snapshot.
		if epoch.To < in.Intent.ServicePeriod.To.Unix() {
			opts = append(opts, billingrating.WithMinimumCommitmentIgnored())
		}

		// Detailed lines contain the expected detailed lines for the whole usage between
		// [intent.ServicePeriodFrom ... servicePeriod.To) capped by StoredAtLT of the current run.
		//
		// This includes usage from any prior periods.
		billingDetailedLines, err := e.ratingService.GenerateDetailedLines(usagebased.RateableIntent{
			Intent:        in.Intent,
			ServicePeriod: epoch.epochClosedPeriod.AsClosedPeriod(),
			MeterValue:    epoch.Quantity,
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("generating detailed lines: %w", err)
		}

		detailedLinesWithUsageFromPriorPeriods := usagebased.NewDetailedLinesFromBilling(
			in.Intent,
			epoch.epochClosedPeriod.AsClosedPeriod(),
			billingDetailedLines.DetailedLines,
		)

		// Let's subtract the usage from the prior periods from the usage from the current period.
		//
		// This makes sure that we only bill the usage that is new for the current period.
		periodNewDetailedLines, err := subtract.SubtractRatedRunDetails(
			detailedLinesWithUsageFromPriorPeriods,
			previouslyGeneratedDetailedLines,
			generatedUniqueReferenceIDGenerator{},
			// This subtraction is intermediate period-preserving arithmetic. The
			// output is not persisted directly; final output IDs are period-stamped
			// before persistence, so duplicate child refs are allowed here.
			subtract.WithUniqueReferenceIDValidationIgnored(),
		)
		if err != nil {
			return nil, fmt.Errorf("subtracting detailed lines: %w", err)
		}

		for _, line := range periodNewDetailedLines {
			servicePeriod := closedPeriodToEpochClosedPeriod(line.ServicePeriod)
			result[servicePeriod] = append(result[servicePeriod], line)
		}

		// Now we should add these new detailed lines to the previously generated detailed lines, so that the next iteration will
		// also subtract the usage from the prior periods from the usage from the current period.
		previouslyGeneratedDetailedLines = append(previouslyGeneratedDetailedLines, periodNewDetailedLines...)
	}

	// Invariant: at this point result contains the detailed lines for each epoch, assuming nothing was actually billed yet.
	// At this point we need to subtract the already billed detailed lines from the detailed lines for the current period.

	alreadyBilledDetailedLinesByServicePeriod := make(map[epochClosedPeriod]usagebased.DetailedLines, len(in.PriorPeriods))
	runIDByServicePeriod := make(map[epochClosedPeriod]usagebased.RealizationRunID, len(in.PriorPeriods))
	for _, priorPeriod := range in.PriorPeriods {
		period := closedPeriodToEpochClosedPeriod(priorPeriod.ServicePeriod)
		runIDByServicePeriod[period] = priorPeriod.RunID

		for _, line := range priorPeriod.DetailedLines {
			// Credits are applied after rating. Strip them before comparing already-billed
			// lines to current rating output, otherwise credit allocation changes look like
			// pricing deltas.
			line = line.Clone()
			line.CreditsApplied = nil
			line.Totals.CreditsTotal = alpacadecimal.Zero
			line.Totals.Total = line.Totals.CalculateTotal()

			servicePeriod := closedPeriodToEpochClosedPeriod(line.ServicePeriod)
			alreadyBilledDetailedLinesByServicePeriod[servicePeriod] = append(alreadyBilledDetailedLinesByServicePeriod[servicePeriod], line)
		}
	}

	for servicePeriod, alreadyBilledDetailedLines := range alreadyBilledDetailedLinesByServicePeriod {
		alreadyBilledDetailedLines, err := alreadyBilledDetailedLines.StripServicePeriodFromUniqueReferenceID()
		if err != nil {
			return nil, fmt.Errorf("stripping already billed detailed line child unique reference ids: %w", err)
		}

		// Let's subtract the already billed detailed lines from the detailed lines for the prior period.
		periodRemainingDetailedLines, err := subtract.SubtractRatedRunDetails(
			result[servicePeriod],
			alreadyBilledDetailedLines,
			bookedCorrectionUniqueReferenceIDGenerator{},
		)
		if err != nil {
			return nil, fmt.Errorf("subtracting detailed lines: %w", err)
		}

		result[servicePeriod] = periodRemainingDetailedLines
	}

	for servicePeriod, runID := range runIDByServicePeriod {
		for idx := range result[servicePeriod] {
			result[servicePeriod][idx].CorrectsRunID = lo.ToPtr(runID.ID)
		}
	}

	return result, nil
}

func flattenDetailedLinesByEpoch(detailedLinesByEpoch map[epochClosedPeriod]usagebased.DetailedLines) (usagebased.DetailedLines, error) {
	// Output ordering is part of the persisted detailed-line contract. Keep period
	// corrections grouped by service-period start, then by the rating-generated
	// index, then by child reference. After sorting, restamp dense indexes for
	// this run the same way billing does when persisting generated details.
	periods := lo.Keys(detailedLinesByEpoch)
	slices.SortFunc(periods, compareEpochClosedPeriod)

	out := make(usagebased.DetailedLines, 0, len(detailedLinesByEpoch))
	for _, period := range periods {
		lines, err := detailedLinesByEpoch[period].WithServicePeriodFromUniqueReferenceID()
		if err != nil {
			return nil, fmt.Errorf("adding service period to detailed line child unique reference ids: %w", err)
		}

		out = append(out, lines...)
	}

	out.Sort()
	for idx := range out {
		out[idx].Index = lo.ToPtr(idx)
	}

	return out, nil
}

func compareEpochClosedPeriod(a, b epochClosedPeriod) int {
	if c := cmp.Compare(a.From, b.From); c != 0 {
		return c
	}

	return cmp.Compare(a.To, b.To)
}
