package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	chargesadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebased"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedruns"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ usagebased.RealizationRunCreditAllocationAdapter = (*adapter)(nil)

func (a *adapter) CreateRunCreditRealization(ctx context.Context, runID usagebased.RealizationRunID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error) {
	if err := runID.Validate(); err != nil {
		return nil, err
	}

	if err := creditAllocations.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		creates := lo.Map(creditAllocations, func(creditAllocation creditrealization.CreateInput, idx int) *entdb.ChargeUsageBasedRunCreditAllocationsCreate {
			create := tx.db.ChargeUsageBasedRunCreditAllocations.Create().
				SetRunID(runID.ID).
				SetNamespace(runID.Namespace)

			create = creditrealization.Create[*entdb.ChargeUsageBasedRunCreditAllocationsCreate](create, runID.Namespace, idx, creditAllocation)

			return create
		})

		dbEntities, err := tx.db.ChargeUsageBasedRunCreditAllocations.CreateBulk(creates...).Save(ctx)
		if err != nil {
			return nil, err
		}

		realizations := lo.Map(dbEntities, func(entity *entdb.ChargeUsageBasedRunCreditAllocations, _ int) creditrealization.Realization {
			return creditrealization.MapFromDB(entity)
		})

		if err := createInitialCreditRealizationLineagesForRun(ctx, tx, runID, realizations); err != nil {
			return nil, err
		}

		if err := chargesadapter.WritebackCorrectionLineageSegments(ctx, tx.db, runID.Namespace, realizations); err != nil {
			return nil, fmt.Errorf("write back correction lineage segments: %w", err)
		}

		chargesadapter.AttachInitialActiveLineageSegments(realizations)

		return realizations, nil
	})
}

func createInitialCreditRealizationLineagesForRun(ctx context.Context, tx *adapter, runID usagebased.RealizationRunID, realizations creditrealization.Realizations) error {
	specs, err := creditrealization.InitialLineageSpecs(realizations)
	if err != nil {
		return fmt.Errorf("build initial credit realization lineage specs: %w", err)
	}

	if len(specs) == 0 {
		return nil
	}

	run, err := tx.db.ChargeUsageBasedRuns.Query().
		Where(
			chargeusagebasedruns.Namespace(runID.Namespace),
			chargeusagebasedruns.ID(runID.ID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("get usage based run for credit realization lineage: %w", err)
	}

	charge, err := tx.db.ChargeUsageBased.Query().
		Where(
			chargeusagebased.Namespace(runID.Namespace),
			chargeusagebased.ID(run.ChargeID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("get usage based charge for credit realization lineage: %w", err)
	}

	rootCreates := make([]*entdb.CreditRealizationLineageCreate, 0, len(specs))
	segmentCreates := make([]*entdb.CreditRealizationLineageSegmentCreate, 0, len(specs))

	for _, spec := range specs {
		rootCreates = append(rootCreates, tx.db.CreditRealizationLineage.Create().
			SetID(spec.LineageID).
			SetNamespace(runID.Namespace).
			SetRootRealizationID(spec.RootRealizationID).
			SetCustomerID(charge.CustomerID).
			SetCurrency(charge.Currency).
			SetOriginKind(spec.OriginKind),
		)
		segmentCreates = append(segmentCreates, tx.db.CreditRealizationLineageSegment.Create().
			SetLineageID(spec.LineageID).
			SetAmount(spec.Amount).
			SetState(spec.InitialState),
		)
	}

	if _, err := tx.db.CreditRealizationLineage.CreateBulk(rootCreates...).Save(ctx); err != nil {
		return fmt.Errorf("create credit realization lineages: %w", err)
	}

	if _, err := tx.db.CreditRealizationLineageSegment.CreateBulk(segmentCreates...).Save(ctx); err != nil {
		return fmt.Errorf("create initial credit realization lineage segments: %w", err)
	}

	return nil
}
