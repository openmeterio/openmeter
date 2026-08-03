package adapter

import (
	"context"
	"fmt"
	"slices"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"

	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeusagebasedrundetailedline "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedrundetailedline"
	dbchargeusagebasedruns "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedruns"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) FetchDetailedLines(ctx context.Context, charge usagebased.Charge) (usagebased.Charge, error) {
	if len(charge.Realizations) == 0 {
		return charge, nil
	}

	runIDs := lo.Map(charge.Realizations, func(run usagebased.RealizationRun, _ int) string {
		return run.ID.ID
	})

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.Charge, error) {
		dbLines, err := tx.db.ChargeUsageBasedRunDetailedLine.Query().
			Where(
				dbchargeusagebasedrundetailedline.NamespaceEQ(charge.Namespace),
				dbchargeusagebasedrundetailedline.ChargeIDEQ(charge.ID),
				dbchargeusagebasedrundetailedline.RunIDIn(runIDs...),
				dbchargeusagebasedrundetailedline.DeletedAtIsNil(),
			).
			All(ctx)
		if err != nil {
			return usagebased.Charge{}, err
		}

		dbRuns, err := tx.db.ChargeUsageBasedRuns.Query().
			Where(
				dbchargeusagebasedruns.NamespaceEQ(charge.Namespace),
				dbchargeusagebasedruns.ChargeIDEQ(charge.ID),
				dbchargeusagebasedruns.IDIn(runIDs...),
			).
			All(ctx)
		if err != nil {
			return usagebased.Charge{}, err
		}

		type detailedLinesMetadata struct {
			Present                  bool
			IncludeCreditAllocations bool
		}

		detailedLinesMetadataByRunID := make(map[string]detailedLinesMetadata, len(dbRuns))
		for _, dbRun := range dbRuns {
			detailedLinesMetadataByRunID[dbRun.ID] = detailedLinesMetadata{
				Present:                  dbRun.DetailedLinesPresent,
				IncludeCreditAllocations: dbRun.DetailedLinesIncludeCreditAllocations,
			}
		}

		linesByRunID := make(map[string]usagebased.DetailedLines, len(charge.Realizations))
		for _, dbLine := range dbLines {
			line, err := fromDBDetailedLine(dbLine)
			if err != nil {
				return usagebased.Charge{}, err
			}

			linesByRunID[dbLine.RunID] = append(linesByRunID[dbLine.RunID], line)
		}

		for idx, run := range charge.Realizations {
			lines := linesByRunID[run.ID.ID]
			slices.SortStableFunc(lines, stddetailedline.Compare[usagebased.DetailedLine])

			metadata, found := detailedLinesMetadataByRunID[run.ID.ID]
			if !found {
				charge.Realizations[idx].DetailedLines = mo.None[usagebased.DetailedLines]()
				continue
			}

			charge.Realizations[idx].DetailedLinesIncludeCreditAllocations = metadata.IncludeCreditAllocations

			// Safety measure: only mark detailed lines as expanded when the persisted
			// run records that detailed lines were written at least once. Treating
			// unknown detailed lines as an empty set can make
			// late-event rating overcharge.
			if !metadata.Present {
				charge.Realizations[idx].DetailedLines = mo.None[usagebased.DetailedLines]()
				continue
			}

			// Previously credit then invoice would not store credit allocations, but due to custom currency support
			// and consistency's sake, we should now start storing them.
			//
			// We are applying the credit allocations to the detailed lines dynamically here, once all data has been
			// migrated we can get rid of this code.
			if charge.Intent.GetSettlementMode() == productcatalog.CreditThenInvoiceSettlementMode &&
				!metadata.IncludeCreditAllocations {
				creditsApplied, err := run.CreditsAllocated.AsCreditsApplied()
				if err != nil {
					return usagebased.Charge{}, fmt.Errorf(
						"mapping legacy run credit allocations to detailed lines [run_id=%s]: %w",
						run.ID.ID,
						err,
					)
				}

				lines, err = lines.WithCreditsApplied(creditsApplied, charge.Intent.GetCurrency())
				if err != nil {
					return usagebased.Charge{}, fmt.Errorf(
						"applying legacy run credit allocations to detailed lines [run_id=%s]: %w",
						run.ID.ID,
						err,
					)
				}

				charge.Realizations[idx].DetailedLinesIncludeCreditAllocations = true
			}

			charge.Realizations[idx].DetailedLines = mo.Some(lines)
		}

		return charge, nil
	})
}

func (a *adapter) UpsertRunDetailedLines(
	ctx context.Context,
	input usagebased.UpsertRunDetailedLinesInput,
) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		createBuilders := make([]*entdb.ChargeUsageBasedRunDetailedLineCreate, 0, len(input.DetailedLines))

		for _, line := range input.DetailedLines {
			lineToPersist := line.Clone()
			lineToPersist.Namespace = input.RunID.Namespace
			lineToPersist.DeletedAt = nil

			create, err := buildDetailedLineCreate(tx.db, input.ChargeID, input.RunID, lineToPersist)
			if err != nil {
				return err
			}

			createBuilders = append(createBuilders, create)
		}

		now := clock.Now().In(time.UTC)
		deleteQuery := tx.db.ChargeUsageBasedRunDetailedLine.Update().
			Where(
				dbchargeusagebasedrundetailedline.NamespaceEQ(input.RunID.Namespace),
				dbchargeusagebasedrundetailedline.ChargeIDEQ(input.ChargeID.ID),
				dbchargeusagebasedrundetailedline.RunIDEQ(input.RunID.ID),
				dbchargeusagebasedrundetailedline.DeletedAtIsNil(),
			).
			SetDeletedAt(now)

		childRefsToKeep := lo.FilterMap(input.DetailedLines, func(line usagebased.DetailedLine, _ int) (string, bool) {
			if line.ChildUniqueReferenceID == "" {
				return "", false
			}

			return line.ChildUniqueReferenceID, true
		})
		if len(childRefsToKeep) > 0 {
			deleteQuery = deleteQuery.Where(
				dbchargeusagebasedrundetailedline.ChildUniqueReferenceIDNotIn(childRefsToKeep...),
			)
		}

		if _, err := deleteQuery.Save(ctx); err != nil {
			return err
		}

		if _, err := tx.db.ChargeUsageBasedRuns.Update().
			Where(
				dbchargeusagebasedruns.NamespaceEQ(input.RunID.Namespace),
				dbchargeusagebasedruns.ChargeIDEQ(input.ChargeID.ID),
				dbchargeusagebasedruns.ID(input.RunID.ID),
			).
			SetDetailedLinesPresent(true).
			SetDetailedLinesIncludeCreditAllocations(input.DetailedLinesIncludeCreditAllocations).
			Save(ctx); err != nil {
			return err
		}

		if len(createBuilders) == 0 {
			return nil
		}

		return tx.db.ChargeUsageBasedRunDetailedLine.CreateBulk(createBuilders...).
			OnConflict(
				sql.ConflictColumns(
					dbchargeusagebasedrundetailedline.FieldNamespace,
					dbchargeusagebasedrundetailedline.FieldChargeID,
					dbchargeusagebasedrundetailedline.FieldRunID,
					dbchargeusagebasedrundetailedline.FieldChildUniqueReferenceID,
				),
				sql.ConflictWhere(sql.IsNull(dbchargeusagebasedrundetailedline.FieldDeletedAt)),
				sql.ResolveWithNewValues(),
				sql.ResolveWith(func(u *sql.UpdateSet) {
					u.SetIgnore(dbchargeusagebasedrundetailedline.FieldCreatedAt)
					u.SetIgnore(dbchargeusagebasedrundetailedline.FieldID)
				}),
			).
			UpdateDescription().
			UpdateIndex().
			UpdatePricerReferenceID().
			UpdateCorrectsRunID().
			UpdateDeletedAt().
			UpdateInvoicingAppExternalID().
			UpdateChildUniqueReferenceID().
			UpdateCreditsApplied().
			UpdateAnnotations().
			UpdateMetadata().
			Exec(ctx)
	})
}

func buildDetailedLineCreate(db *entdb.Client, chargeID chargesmeta.ChargeID, runID usagebased.RealizationRunID, line usagebased.DetailedLine) (*entdb.ChargeUsageBasedRunDetailedLineCreate, error) {
	if line.ID == "" {
		line.ID = ulid.Make().String()
	}

	create := db.ChargeUsageBasedRunDetailedLine.Create().
		SetID(line.ID).
		SetNamespace(runID.Namespace).
		SetChargeID(chargeID.ID).
		SetRunID(runID.ID).
		SetPricerReferenceID(lo.CoalesceOrEmpty(line.PricerReferenceID, line.ChildUniqueReferenceID)).
		SetNillableCorrectsRunID(line.CorrectsRunID)

	create = stddetailedline.Create(create, line.Base)

	if len(line.CreditsApplied) > 0 {
		create = create.SetCreditsApplied(&line.CreditsApplied)
	}

	return create, nil
}

func fromDBDetailedLine(dbLine *entdb.ChargeUsageBasedRunDetailedLine) (usagebased.DetailedLine, error) {
	line := usagebased.DetailedLine{
		Base:              stddetailedline.FromDB(dbLine),
		PricerReferenceID: dbLine.PricerReferenceID,
		CorrectsRunID:     dbLine.CorrectsRunID,
	}

	return line, line.Validate()
}
