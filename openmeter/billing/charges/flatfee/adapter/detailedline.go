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
	dbchargeflatfeedetailedline "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeedetailedline"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ flatfee.ChargeDetailedLineAdapter = (*adapter)(nil)

func (a *adapter) FetchDetailedLines(ctx context.Context, charge flatfee.Charge) (flatfee.Charge, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.Charge, error) {
		dbLines, err := tx.db.ChargeFlatFeeDetailedLine.Query().
			Where(
				dbchargeflatfeedetailedline.NamespaceEQ(charge.Namespace),
				dbchargeflatfeedetailedline.ChargeIDEQ(charge.ID),
				dbchargeflatfeedetailedline.DeletedAtIsNil(),
			).
			WithTaxCode().
			All(ctx)
		if err != nil {
			return flatfee.Charge{}, err
		}

		lines := make(flatfee.DetailedLines, 0, len(dbLines))
		for _, dbLine := range dbLines {
			line, err := mapDetailedLineFromDB(dbLine)
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
		createBuilders := make([]*entdb.ChargeFlatFeeDetailedLineCreate, 0, len(lines))

		for _, line := range lines {
			lineToPersist := line.Clone()
			lineToPersist.Namespace = chargeID.Namespace
			lineToPersist.DeletedAt = nil

			create, err := buildDetailedLineCreate(tx.db, chargeID, lineToPersist)
			if err != nil {
				return err
			}

			createBuilders = append(createBuilders, create)
		}

		now := clock.Now().In(time.UTC)
		deleteQuery := tx.db.ChargeFlatFeeDetailedLine.Update().
			Where(
				dbchargeflatfeedetailedline.NamespaceEQ(chargeID.Namespace),
				dbchargeflatfeedetailedline.ChargeIDEQ(chargeID.ID),
				dbchargeflatfeedetailedline.DeletedAtIsNil(),
			).
			SetDeletedAt(now)

		childRefsToKeep := lo.Map(lines, func(line flatfee.DetailedLine, _ int) string {
			return line.ChildUniqueReferenceID
		})
		if len(childRefsToKeep) > 0 {
			deleteQuery = deleteQuery.Where(
				dbchargeflatfeedetailedline.ChildUniqueReferenceIDNotIn(childRefsToKeep...),
			)
		}

		if _, err := deleteQuery.Save(ctx); err != nil {
			return err
		}

		if len(createBuilders) == 0 {
			return nil
		}

		return tx.db.ChargeFlatFeeDetailedLine.CreateBulk(createBuilders...).
			OnConflict(
				sql.ConflictColumns(
					dbchargeflatfeedetailedline.FieldNamespace,
					dbchargeflatfeedetailedline.FieldChargeID,
					dbchargeflatfeedetailedline.FieldChildUniqueReferenceID,
				),
				sql.ConflictWhere(sql.IsNull(dbchargeflatfeedetailedline.FieldDeletedAt)),
				sql.ResolveWithNewValues(),
				sql.ResolveWith(func(u *sql.UpdateSet) {
					u.SetIgnore(dbchargeflatfeedetailedline.FieldCreatedAt)
					u.SetIgnore(dbchargeflatfeedetailedline.FieldID)
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

func buildDetailedLineCreate(db *entdb.Client, chargeID meta.ChargeID, line flatfee.DetailedLine) (*entdb.ChargeFlatFeeDetailedLineCreate, error) {
	if line.ID == "" {
		line.ID = ulid.Make().String()
	}

	create := db.ChargeFlatFeeDetailedLine.Create().
		SetID(line.ID).
		SetNamespace(chargeID.Namespace).
		SetChargeID(chargeID.ID)

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
