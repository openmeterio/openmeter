package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	chargesadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (a *adapter) CreateCreditAllocations(ctx context.Context, chargeID meta.ChargeID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error) {
	if err := creditAllocations.Validate(); err != nil {
		return creditrealization.Realizations{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		dbEntities, err := tx.db.ChargeFlatFeeCreditAllocations.CreateBulk(
			lo.Map(creditAllocations, func(creditAllocation creditrealization.CreateInput, idx int) *db.ChargeFlatFeeCreditAllocationsCreate {
				create := tx.db.ChargeFlatFeeCreditAllocations.Create().
					SetChargeID(chargeID.ID)

				create = creditrealization.Create(create, chargeID.Namespace, idx, creditAllocation)

				return create
			})...,
		).Save(ctx)
		if err != nil {
			return creditrealization.Realizations{}, err
		}

		realizations, err := slicesx.MapWithErr(dbEntities, func(entity *db.ChargeFlatFeeCreditAllocations) (creditrealization.Realization, error) {
			return creditrealization.MapFromDB(entity), nil
		})
		if err != nil {
			return creditrealization.Realizations{}, err
		}

		if err := createInitialCreditRealizationLineagesForCharge(ctx, tx, chargeID, realizations); err != nil {
			return creditrealization.Realizations{}, err
		}

		if err := chargesadapter.WritebackCorrectionLineageSegments(ctx, tx.db, chargeID.Namespace, realizations); err != nil {
			return creditrealization.Realizations{}, fmt.Errorf("write back correction lineage segments: %w", err)
		}

		chargesadapter.AttachInitialActiveLineageSegments(realizations)

		return realizations, nil
	})
}

func createInitialCreditRealizationLineagesForCharge(ctx context.Context, tx *adapter, chargeID meta.ChargeID, realizations creditrealization.Realizations) error {
	specs, err := creditrealization.InitialLineageSpecs(realizations)
	if err != nil {
		return fmt.Errorf("build initial credit realization lineage specs: %w", err)
	}

	if len(specs) == 0 {
		return nil
	}

	charge, err := tx.db.ChargeFlatFee.Query().
		Where(
			chargeflatfee.Namespace(chargeID.Namespace),
			chargeflatfee.ID(chargeID.ID),
		).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("get flat fee charge for credit realization lineage: %w", err)
	}

	rootCreates := make([]*db.CreditRealizationLineageCreate, 0, len(specs))
	segmentCreates := make([]*db.CreditRealizationLineageSegmentCreate, 0, len(specs))

	for _, spec := range specs {
		rootCreates = append(rootCreates, tx.db.CreditRealizationLineage.Create().
			SetID(spec.LineageID).
			SetNamespace(chargeID.Namespace).
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
