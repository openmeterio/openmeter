package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Config struct {
	Adapter legacylineage.Adapter
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	return nil
}

func New(config Config) (legacylineage.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter: config.Adapter,
	}, nil
}

type service struct {
	adapter legacylineage.Adapter
}

func (s *service) CreateInitialLineages(ctx context.Context, input legacylineage.CreateInitialLineagesInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		specs, err := creditrealization.InitialLineageSpecs(input.Realizations)
		if err != nil {
			return fmt.Errorf("build initial credit realization lineage specs: %w", err)
		}
		for idx := range specs {
			if specs[idx].OriginKind == creditrealization.LineageOriginKindAdvance {
				specs[idx].AdvanceFeatures = input.Features
			}
		}
		if len(specs) == 0 {
			return nil
		}

		return s.adapter.CreateLineages(ctx, legacylineage.CreateLineagesInput{
			Namespace:  input.Namespace,
			ChargeID:   input.ChargeID,
			CustomerID: input.CustomerID,
			Currency:   input.Currency.Reference(),
			Specs:      specs,
		})
	})
}

func (s *service) LoadActiveSegmentsByRealizationID(ctx context.Context, namespace string, realizationIDs []string) (legacylineage.ActiveSegmentsByRealizationID, error) {
	if len(realizationIDs) == 0 {
		return legacylineage.ActiveSegmentsByRealizationID{}, nil
	}

	segmentsByRealizationID, err := s.adapter.LoadActiveSegmentsByRealizationID(ctx, namespace, realizationIDs)
	if err != nil {
		return nil, fmt.Errorf("load active lineage segments: %w", err)
	}

	return segmentsByRealizationID, nil
}

func (s *service) LoadLineagesByCustomer(ctx context.Context, input legacylineage.LoadLineagesByCustomerInput) ([]legacylineage.Lineage, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return s.adapter.LoadLineagesByCustomer(ctx, input)
}

func (s *service) PersistCorrectionLineageSegments(ctx context.Context, input legacylineage.PersistCorrectionLineageSegmentsInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		correctionAmountsByRealizationID := make(map[string]alpacadecimal.Decimal, len(input.Realizations))
		correctionOrder := make([]string, 0)

		for _, realization := range input.Realizations {
			if realization.Type != creditrealization.TypeCorrection || realization.CorrectsRealizationID == nil {
				continue
			}

			correctsRealizationID := *realization.CorrectsRealizationID
			if _, ok := correctionAmountsByRealizationID[correctsRealizationID]; !ok {
				correctionOrder = append(correctionOrder, correctsRealizationID)
			}

			correctionAmountsByRealizationID[correctsRealizationID] = correctionAmountsByRealizationID[correctsRealizationID].Add(realization.Amount.Abs())
		}

		if len(correctionOrder) == 0 {
			return nil
		}

		lineages, err := s.adapter.LockCorrectionLineages(ctx, input.Namespace, correctionOrder)
		if err != nil {
			return fmt.Errorf("lock lineages for correction persistence: %w", err)
		}

		lineagesByRealizationID := make(map[string]legacylineage.Lineage, len(lineages))
		for _, entry := range lineages {
			lineagesByRealizationID[entry.RootRealizationID] = entry
		}

		now := clock.Now().Truncate(time.Microsecond)

		for _, realizationID := range correctionOrder {
			entry, ok := lineagesByRealizationID[realizationID]
			if !ok {
				continue
			}

			remaining := correctionAmountsByRealizationID[realizationID]
			for _, segment := range legacylineage.SortCorrectionPersistSegments(entry.Segments) {
				if !remaining.IsPositive() {
					break
				}

				consumedAmount := legacylineage.MinDecimal(segment.Amount, remaining)
				if !consumedAmount.IsPositive() {
					continue
				}

				if err := s.adapter.CloseSegment(ctx, segment.ID, now); err != nil {
					return fmt.Errorf("close active lineage segment %s: %w", segment.ID, err)
				}

				remainder := segment.Amount.Sub(consumedAmount)
				if remainder.IsPositive() {
					if err := s.adapter.CreateSegment(ctx, legacylineage.CreateSegmentInput{
						LineageID:                       segment.LineageID,
						Amount:                          remainder,
						State:                           segment.State,
						BackingTransactionGroupID:       segment.BackingTransactionGroupID,
						SourceState:                     segment.SourceState,
						SourceBackingTransactionGroupID: segment.SourceBackingTransactionGroupID,
					}); err != nil {
						return fmt.Errorf("create lineage segment remainder for %s: %w", segment.ID, err)
					}
				}

				remaining = remaining.Sub(consumedAmount)
			}

			if remaining.IsPositive() {
				return fmt.Errorf("correction amount %s exceeds active lineage coverage for realization %s", remaining.String(), realizationID)
			}
		}

		return nil
	})
}

func (s *service) BackfillAdvanceLineageSegments(ctx context.Context, input legacylineage.BackfillAdvanceLineageSegmentsInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if len(input.Allocations) == 0 {
		return nil
	}
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		lineages, err := s.adapter.LockAdvanceLineagesForBackfill(ctx, input.Namespace, input.CustomerID, input.Currency.Reference())
		if err != nil {
			return fmt.Errorf("lock advance lineages for backfill: %w", err)
		}
		lineages = legacylineage.FilterAdvanceLineagesForBackfill(lineages, input.FeatureFilters)

		segmentsByID := make(map[string]legacylineage.Segment)
		for _, root := range lineages {
			for _, segment := range root.Segments {
				segmentsByID[segment.ID] = segment
			}
		}

		now := clock.Now().Truncate(time.Microsecond)
		for _, allocation := range input.Allocations {
			segment, ok := segmentsByID[allocation.SegmentID]
			if !ok || allocation.Amount.GreaterThan(segment.Amount) {
				return fmt.Errorf("backfill allocation exceeds active eligible segment %s", allocation.SegmentID)
			}
			coveredAmount := allocation.Amount
			if err := s.adapter.CloseSegment(ctx, segment.ID, now); err != nil {
				return fmt.Errorf("close uncovered advance lineage segment %s: %w", segment.ID, err)
			}

			remainder := segment.Amount.Sub(coveredAmount)
			if remainder.IsPositive() {
				if err := s.adapter.CreateSegment(ctx, legacylineage.CreateSegmentInput{
					LineageID: segment.LineageID,
					Amount:    remainder,
					State:     creditrealization.LineageSegmentStateAdvanceUncovered,
				}); err != nil {
					return fmt.Errorf("create uncovered advance lineage remainder for segment %s: %w", segment.ID, err)
				}
			}

			if err := s.adapter.CreateSegment(ctx, legacylineage.CreateSegmentInput{
				LineageID:                 segment.LineageID,
				Amount:                    coveredAmount,
				State:                     creditrealization.LineageSegmentStateAdvanceBackfilled,
				BackingTransactionGroupID: &input.BackingTransactionGroupID,
			}); err != nil {
				return fmt.Errorf("create backfilled advance lineage segment for segment %s: %w", segment.ID, err)
			}
		}

		return nil
	})
}

func (s *service) CloseSegment(ctx context.Context, segmentID string, closedAt time.Time) error {
	return s.adapter.CloseSegment(ctx, segmentID, closedAt)
}

func (s *service) CreateSegment(ctx context.Context, input legacylineage.CreateSegmentInput) error {
	return s.adapter.CreateSegment(ctx, input)
}
