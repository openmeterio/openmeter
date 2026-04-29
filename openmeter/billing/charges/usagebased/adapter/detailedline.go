package adapter

import (
	"context"
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
			WithTaxCode().
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

		detailedLinesPresentByRunID := make(map[string]bool, len(dbRuns))
		for _, dbRun := range dbRuns {
			detailedLinesPresentByRunID[dbRun.ID] = dbRun.DetailedLinesPresent
		}

		linesByRunID := make(map[string]usagebased.DetailedLines, len(charge.Realizations))
		for _, dbLine := range dbLines {
			line, err := mapDetailedLineFromDB(dbLine)
			if err != nil {
				return usagebased.Charge{}, err
			}

			linesByRunID[dbLine.RunID] = append(linesByRunID[dbLine.RunID], line)
		}

		for idx, run := range charge.Realizations {
			lines := linesByRunID[run.ID.ID]
			slices.SortStableFunc(lines, stddetailedline.Compare[usagebased.DetailedLine])

			detailedLinesPresent, found := detailedLinesPresentByRunID[run.ID.ID]
			if !found {
				continue
			}

			// Safety measure: only mark detailed lines as expanded when the persisted
			// run records that detailed lines were written at least once. Treating
			// unknown detailed lines as an empty set can make
			// late-event rating overcharge.
			if detailedLinesPresent {
				charge.Realizations[idx].DetailedLines = mo.Some(lines)
			} else {
				charge.Realizations[idx].DetailedLines = mo.None[usagebased.DetailedLines]()
			}
		}

		return charge, nil
	})
}

func (a *adapter) UpsertRunDetailedLines(ctx context.Context, chargeID chargesmeta.ChargeID, runID usagebased.RealizationRunID, lines usagebased.DetailedLines) error {
	if err := chargeID.Validate(); err != nil {
		return err
	}

	if err := runID.Validate(); err != nil {
		return err
	}

	if err := lines.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		createBuilders := make([]*entdb.ChargeUsageBasedRunDetailedLineCreate, 0, len(lines))

		for _, line := range lines {
			lineToPersist := line.Clone()
			lineToPersist.Namespace = runID.Namespace
			lineToPersist.DeletedAt = nil

			create, err := buildDetailedLineCreate(tx.db, chargeID, runID, lineToPersist)
			if err != nil {
				return err
			}

			createBuilders = append(createBuilders, create)
		}

		now := clock.Now().In(time.UTC)
		deleteQuery := tx.db.ChargeUsageBasedRunDetailedLine.Update().
			Where(
				dbchargeusagebasedrundetailedline.NamespaceEQ(runID.Namespace),
				dbchargeusagebasedrundetailedline.ChargeIDEQ(chargeID.ID),
				dbchargeusagebasedrundetailedline.RunIDEQ(runID.ID),
				dbchargeusagebasedrundetailedline.DeletedAtIsNil(),
			).
			SetDeletedAt(now)

		childRefsToKeep := lo.FilterMap(lines, func(line usagebased.DetailedLine, _ int) (string, bool) {
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
				dbchargeusagebasedruns.NamespaceEQ(runID.Namespace),
				dbchargeusagebasedruns.ChargeIDEQ(chargeID.ID),
				dbchargeusagebasedruns.ID(runID.ID),
			).
			SetDetailedLinesPresent(true).
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
			UpdateTaxConfig().
			UpdateTaxCodeID().
			UpdateTaxBehavior().
			UpdateIndex().
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
		SetRunID(runID.ID)

	create = stddetailedline.Create(create, line)

	if len(line.CreditsApplied) > 0 {
		create = create.SetCreditsApplied(&line.CreditsApplied)
	}

	if line.TaxConfig != nil {
		create = create.SetTaxConfig(*line.TaxConfig).
			SetNillableTaxCodeID(line.TaxConfig.TaxCodeID).
			SetNillableTaxBehavior(line.TaxConfig.Behavior)
	}

	return create, nil
}

func mapDetailedLineFromDB(dbLine *entdb.ChargeUsageBasedRunDetailedLine) (usagebased.DetailedLine, error) {
	line := stddetailedline.FromDB(
		dbLine,
		stddetailedline.BackfillTaxConfig(
			lo.EmptyableToPtr(dbLine.TaxConfig),
			dbLine.TaxBehavior,
			taxCodeIDFromEnt(dbLine.Edges.TaxCode),
		),
	)

	return line, line.Validate()
}

func taxCodeIDFromEnt(resolvedTaxCode *entdb.TaxCode) *string {
	if resolvedTaxCode == nil {
		return nil
	}

	return lo.ToPtr(resolvedTaxCode.ID)
}
