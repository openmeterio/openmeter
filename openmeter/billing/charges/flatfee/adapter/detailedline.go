package adapter

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeflatfeerundetailedline "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeerundetailedline"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ flatfee.ChargeDetailedLineAdapter = (*adapter)(nil)

func (a *adapter) FetchDetailedLines(ctx context.Context, charge flatfee.Charge) (flatfee.Charge, error) {
	runIDs := make([]string, 0, len(charge.Realizations.PriorRuns)+1)
	if charge.Realizations.CurrentRun != nil {
		runIDs = append(runIDs, charge.Realizations.CurrentRun.ID.ID)
	}

	for _, run := range charge.Realizations.PriorRuns {
		runIDs = append(runIDs, run.ID.ID)
	}

	if len(runIDs) == 0 {
		return charge, nil
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.Charge, error) {
		dbLines, err := tx.db.ChargeFlatFeeRunDetailedLine.Query().
			Where(
				dbchargeflatfeerundetailedline.NamespaceEQ(charge.Namespace),
				dbchargeflatfeerundetailedline.RunIDIn(runIDs...),
				dbchargeflatfeerundetailedline.DeletedAtIsNil(),
			).
			All(ctx)
		if err != nil {
			return flatfee.Charge{}, err
		}

		dbLinesByRunID := lo.GroupBy(dbLines, func(dbLine *entdb.ChargeFlatFeeRunDetailedLine) string {
			return dbLine.RunID
		})

		if charge.Realizations.CurrentRun != nil {
			lines := flatfee.DetailedLines(lo.Map(dbLinesByRunID[charge.Realizations.CurrentRun.ID.ID], func(dbLine *entdb.ChargeFlatFeeRunDetailedLine, _ int) flatfee.DetailedLine {
				return stddetailedline.FromDB(dbLine)
			}))
			sortDetailedLines(lines)
			charge.Realizations.CurrentRun.DetailedLines = mo.Some(lines)
		}

		for idx := range charge.Realizations.PriorRuns {
			lines := flatfee.DetailedLines(lo.Map(dbLinesByRunID[charge.Realizations.PriorRuns[idx].ID.ID], func(dbLine *entdb.ChargeFlatFeeRunDetailedLine, _ int) flatfee.DetailedLine {
				return stddetailedline.FromDB(dbLine)
			}))
			sortDetailedLines(lines)
			charge.Realizations.PriorRuns[idx].DetailedLines = mo.Some(lines)
		}

		return charge, nil
	})
}

func (a *adapter) UpsertDetailedLines(ctx context.Context, runID flatfee.RealizationRunID, lines flatfee.DetailedLines) error {
	if err := runID.Validate(); err != nil {
		return err
	}

	if err := lines.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		createBuilders := make([]*entdb.ChargeFlatFeeRunDetailedLineCreate, 0, len(lines))

		for _, line := range lines {
			lineToPersist := line.Clone()
			lineToPersist.Namespace = runID.Namespace
			lineToPersist.DeletedAt = nil

			create, err := buildDetailedLineCreate(tx.db, runID, lineToPersist)
			if err != nil {
				return err
			}

			createBuilders = append(createBuilders, create)
		}

		now := clock.Now().In(time.UTC)
		deleteQuery := tx.db.ChargeFlatFeeRunDetailedLine.Update().
			Where(
				dbchargeflatfeerundetailedline.NamespaceEQ(runID.Namespace),
				dbchargeflatfeerundetailedline.RunIDEQ(runID.ID),
				dbchargeflatfeerundetailedline.DeletedAtIsNil(),
			).
			SetDeletedAt(now)

		childRefsToKeep := lo.Map(lines, func(line flatfee.DetailedLine, _ int) string {
			return line.ChildUniqueReferenceID
		})
		if len(childRefsToKeep) > 0 {
			deleteQuery = deleteQuery.Where(
				dbchargeflatfeerundetailedline.ChildUniqueReferenceIDNotIn(childRefsToKeep...),
			)
		}

		if _, err := deleteQuery.Save(ctx); err != nil {
			return err
		}

		if len(createBuilders) == 0 {
			return nil
		}

		return tx.db.ChargeFlatFeeRunDetailedLine.CreateBulk(createBuilders...).
			OnConflict(
				sql.ConflictColumns(
					dbchargeflatfeerundetailedline.FieldNamespace,
					dbchargeflatfeerundetailedline.FieldRunID,
					dbchargeflatfeerundetailedline.FieldChildUniqueReferenceID,
				),
				sql.ConflictWhere(sql.IsNull(dbchargeflatfeerundetailedline.FieldDeletedAt)),
				sql.ResolveWithNewValues(),
				sql.ResolveWith(func(u *sql.UpdateSet) {
					u.SetIgnore(dbchargeflatfeerundetailedline.FieldCreatedAt)
					u.SetIgnore(dbchargeflatfeerundetailedline.FieldID)
				}),
			).
			UpdateDescription().
			UpdateIndex().
			UpdatePricerReferenceID().
			UpdateDeletedAt().
			UpdateInvoicingAppExternalID().
			UpdateChildUniqueReferenceID().
			UpdateCreditsApplied().
			UpdateAnnotations().
			UpdateMetadata().
			Exec(ctx)
	})
}

func buildDetailedLineCreate(db *entdb.Client, runID flatfee.RealizationRunID, line flatfee.DetailedLine) (*entdb.ChargeFlatFeeRunDetailedLineCreate, error) {
	if line.ID == "" {
		line.ID = ulid.Make().String()
	}

	create := db.ChargeFlatFeeRunDetailedLine.Create().
		SetID(line.ID).
		SetNamespace(runID.Namespace).
		SetRunID(runID.ID).
		SetPricerReferenceID(line.ChildUniqueReferenceID)

	create = stddetailedline.Create(create, line)

	if len(line.CreditsApplied) > 0 {
		create = create.SetCreditsApplied(&line.CreditsApplied)
	}

	return create, nil
}
