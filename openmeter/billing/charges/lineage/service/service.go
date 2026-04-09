package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Config struct {
	Adapter lineage.Adapter
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	return nil
}

func New(config Config) (lineage.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter: config.Adapter,
	}, nil
}

type service struct {
	adapter lineage.Adapter
}

func (s *service) CreateInitialLineages(ctx context.Context, input lineage.CreateInitialLineagesInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		specs, err := creditrealization.InitialLineageSpecs(input.Realizations)
		if err != nil {
			return fmt.Errorf("build initial credit realization lineage specs: %w", err)
		}
		if len(specs) == 0 {
			return nil
		}

		return s.adapter.CreateLineages(ctx, lineage.CreateLineagesInput{
			Namespace:  input.Namespace,
			ChargeID:   input.ChargeID,
			CustomerID: input.CustomerID,
			Currency:   input.Currency,
			Specs:      specs,
		})
	})
}

func (s *service) LoadActiveSegmentsByRealizationID(ctx context.Context, namespace string, realizationIDs []string) (lineage.ActiveSegmentsByRealizationID, error) {
	if len(realizationIDs) == 0 {
		return lineage.ActiveSegmentsByRealizationID{}, nil
	}

	segmentsByRealizationID, err := s.adapter.LoadActiveSegmentsByRealizationID(ctx, namespace, realizationIDs)
	if err != nil {
		return nil, fmt.Errorf("load active lineage segments: %w", err)
	}

	return segmentsByRealizationID, nil
}

func (s *service) PersistCorrectionLineageSegments(ctx context.Context, input lineage.PersistCorrectionLineageSegmentsInput) error {
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

		lineagesByRealizationID := make(map[string]lineage.Lineage, len(lineages))
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
			for _, segment := range lineage.SortCorrectionPersistSegments(entry.Segments) {
				if !remaining.IsPositive() {
					break
				}

				consumedAmount := lineage.MinDecimal(segment.Amount, remaining)
				if !consumedAmount.IsPositive() {
					continue
				}

				if err := s.adapter.CloseSegment(ctx, segment.ID, now); err != nil {
					return fmt.Errorf("close active lineage segment %s: %w", segment.ID, err)
				}

				remainder := segment.Amount.Sub(consumedAmount)
				if remainder.IsPositive() {
					if err := s.adapter.CreateSegment(ctx, lineage.CreateSegmentInput{
						LineageID:                 segment.LineageID,
						Amount:                    remainder,
						State:                     segment.State,
						BackingTransactionGroupID: segment.BackingTransactionGroupID,
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

func (s *service) BackfillAdvanceLineageSegments(ctx context.Context, input lineage.BackfillAdvanceLineageSegmentsInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		lineages, err := s.adapter.LockAdvanceLineagesForBackfill(ctx, input.Namespace, input.CustomerID, input.Currency)
		if err != nil {
			return fmt.Errorf("lock advance lineages for backfill: %w", err)
		}
		if len(lineages) == 0 {
			return nil
		}

		lineageIDs := make([]string, 0, len(lineages))
		for _, entry := range lineages {
			lineageIDs = append(lineageIDs, entry.ID)
		}

		state := creditrealization.LineageSegmentStateAdvanceUncovered
		segments, err := s.adapter.ListActiveSegments(ctx, lineage.ListActiveSegmentsInput{
			LineageIDs: lineageIDs,
			State:      &state,
		})
		if err != nil {
			return fmt.Errorf("query active uncovered advance lineage segments: %w", err)
		}

		now := clock.Now().Truncate(time.Microsecond)
		remaining := input.Amount

		for _, segment := range segments {
			if !remaining.IsPositive() {
				break
			}

			coveredAmount := lineage.MinDecimal(segment.Amount, remaining)
			if err := s.adapter.CloseSegment(ctx, segment.ID, now); err != nil {
				return fmt.Errorf("close uncovered advance lineage segment %s: %w", segment.ID, err)
			}

			remainder := segment.Amount.Sub(coveredAmount)
			if remainder.IsPositive() {
				if err := s.adapter.CreateSegment(ctx, lineage.CreateSegmentInput{
					LineageID: segment.LineageID,
					Amount:    remainder,
					State:     creditrealization.LineageSegmentStateAdvanceUncovered,
				}); err != nil {
					return fmt.Errorf("create uncovered advance lineage remainder for segment %s: %w", segment.ID, err)
				}
			}

			if err := s.adapter.CreateSegment(ctx, lineage.CreateSegmentInput{
				LineageID:                 segment.LineageID,
				Amount:                    coveredAmount,
				State:                     creditrealization.LineageSegmentStateAdvanceBackfilled,
				BackingTransactionGroupID: &input.BackingTransactionGroupID,
			}); err != nil {
				return fmt.Errorf("create backfilled advance lineage segment for segment %s: %w", segment.ID, err)
			}

			remaining = remaining.Sub(coveredAmount)
		}

		return nil
	})
}
