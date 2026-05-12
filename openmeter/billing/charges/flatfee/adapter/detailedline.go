package adapter

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeflatfeerundetailedline "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeerundetailedline"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ flatfee.ChargeDetailedLineAdapter = (*adapter)(nil)

func (a *adapter) FetchDetailedLines(ctx context.Context, charge flatfee.Charge) (flatfee.Charge, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.Charge, error) {
		run, err := queryCurrentRunByChargeID(tx.db, charge.GetChargeID()).Only(ctx)
		if err != nil {
			return flatfee.Charge{}, err
		}

		dbLines, err := tx.db.ChargeFlatFeeRunDetailedLine.Query().
			Where(
				dbchargeflatfeerundetailedline.NamespaceEQ(charge.Namespace),
				dbchargeflatfeerundetailedline.RunIDEQ(run.ID),
				dbchargeflatfeerundetailedline.DeletedAtIsNil(),
			).
			WithTaxCode().
			All(ctx)
		if err != nil {
			return flatfee.Charge{}, err
		}

		lines := make(flatfee.DetailedLines, 0, len(dbLines))
		for _, dbLine := range dbLines {
			line, err := mapRunDetailedLineFromDB(dbLine)
			if err != nil {
				return flatfee.Charge{}, err
			}

			lines = append(lines, line)
		}

		sortDetailedLines(lines)
		charge.Realizations.DetailedLines = mo.Some(lines)

		return charge, nil
	})
}

func (a *adapter) UpsertDetailedLines(ctx context.Context, chargeID meta.ChargeID, lines flatfee.DetailedLines) error {
	if err := chargeID.Validate(); err != nil {
		return err
	}

	if err := lines.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		run, err := tx.currentRunByChargeID(ctx, chargeID)
		if err != nil {
			return err
		}

		createBuilders := make([]*entdb.ChargeFlatFeeRunDetailedLineCreate, 0, len(lines))

		for _, line := range lines {
			lineToPersist := line.Clone()
			lineToPersist.Namespace = chargeID.Namespace
			lineToPersist.DeletedAt = nil

			create, err := buildDetailedLineCreate(tx.db, chargeID, run.ID, lineToPersist)
			if err != nil {
				return err
			}

			createBuilders = append(createBuilders, create)
		}

		now := clock.Now().In(time.UTC)
		deleteQuery := tx.db.ChargeFlatFeeRunDetailedLine.Update().
			Where(
				dbchargeflatfeerundetailedline.NamespaceEQ(chargeID.Namespace),
				dbchargeflatfeerundetailedline.RunIDEQ(run.ID),
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
			UpdateTaxConfig().
			UpdateTaxCodeID().
			UpdateTaxBehavior().
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

func buildDetailedLineCreate(db *entdb.Client, chargeID meta.ChargeID, runID string, line flatfee.DetailedLine) (*entdb.ChargeFlatFeeRunDetailedLineCreate, error) {
	if line.ID == "" {
		line.ID = ulid.Make().String()
	}

	create := db.ChargeFlatFeeRunDetailedLine.Create().
		SetID(line.ID).
		SetNamespace(chargeID.Namespace).
		SetRunID(runID).
		SetPricerReferenceID(line.ChildUniqueReferenceID)

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
