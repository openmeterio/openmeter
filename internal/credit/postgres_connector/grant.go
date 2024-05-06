package postgres_connector

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/oklog/ulid/v2"
	credit_model "github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_credit "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/creditentry"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/pgulid"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (c *PostgresConnector) CreateGrant(ctx context.Context, namespace string, grantIn credit_model.Grant) (credit_model.Grant, error) {
	grant, err := mutationTransaction(ctx, c, namespace, grantIn.LedgerID, func(tx *db.Tx, ledgerEntity *db.Ledger) (*credit_model.Grant, error) {
		// Check if the reset is in the future
		err := checkAfterHighWatermark(grantIn.EffectiveAt, ledgerEntity)
		if err != nil {
			// Typed error do not wrap
			return nil, err
		}

		q := tx.CreditEntry.Create().
			SetNamespace(namespace).
			SetNillableID(pgulid.Ptr(grantIn.ID)).
			SetLedgerID(pgulid.Wrap(grantIn.LedgerID)).
			SetEntryType(credit_model.EntryTypeGrant).
			SetType(grantIn.Type).
			SetNillableParentID(pgulid.Ptr(grantIn.ParentID)).
			SetNillableFeatureID(pgulid.Ptr(grantIn.FeatureID)).
			SetAmount(grantIn.Amount).
			SetPriority(grantIn.Priority).
			SetEffectiveAt(grantIn.EffectiveAt).
			SetExpirationPeriodDuration(grantIn.Expiration.Duration).
			SetExpirationPeriodCount(grantIn.Expiration.Count).
			SetExpirationAt(grantIn.Expiration.GetExpiration(grantIn.EffectiveAt)).
			SetMetadata(grantIn.Metadata)
		if grantIn.Rollover != nil {
			q = q.SetRolloverType(grantIn.Rollover.Type).
				SetNillableRolloverMaxAmount(grantIn.Rollover.MaxAmount)
		}
		entity, err := q.Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create grant: %w", err)
		}

		grant, err := mapGrantEntity(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to map grant entity: %w", err)
		}

		return &grant, nil
	})

	if err != nil {
		return credit_model.Grant{}, err
	}

	return *grant, nil
}

func (c *PostgresConnector) VoidGrant(ctx context.Context, namespace string, grantIn credit_model.Grant) (credit_model.Grant, error) {
	grant, err := mutationTransaction(ctx, c, namespace, grantIn.LedgerID, func(tx *db.Tx, ledgerEntity *db.Ledger) (*credit_model.Grant, error) {
		// Check if the reset is in the future
		err := checkAfterHighWatermark(grantIn.EffectiveAt, ledgerEntity)
		if err != nil {
			// Typed error do not wrap
			return nil, err
		}

		if grantIn.ID == nil {
			return nil, fmt.Errorf("grant ID is required")
		}

		entity, err := tx.CreditEntry.Query().
			Where(
				db_credit.Namespace(namespace),
				db_credit.ID(pgulid.Wrap(*grantIn.ID)),
			).
			Only(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return nil, &credit_model.GrantNotFoundError{GrantID: *grantIn.ID}
			}

			return nil, fmt.Errorf("failed to void grant: %w", err)
		}

		// create a new entry with parent ID and void type
		entity, err = tx.CreditEntry.Create().
			SetNamespace(entity.Namespace).
			SetParentID(entity.ID).
			SetLedgerID(entity.LedgerID).
			SetEntryType(credit_model.EntryTypeVoidGrant).
			SetType(*entity.Type).
			SetNillableFeatureID(entity.FeatureID).
			SetAmount(*entity.Amount).
			SetPriority(entity.Priority).
			SetEffectiveAt(entity.EffectiveAt).
			SetExpirationPeriodDuration(*entity.ExpirationPeriodDuration).
			SetExpirationPeriodCount(*entity.ExpirationPeriodCount).
			SetMetadata(entity.Metadata).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to void grant: %w", err)
		}

		grant, err := mapGrantEntity(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to map grant entity: %w", err)
		}
		return &grant, nil
	})

	if err != nil {
		return credit_model.Grant{}, err
	}

	return *grant, nil
}

func (c *PostgresConnector) ListGrants(ctx context.Context, namespace string, params credit_model.ListGrantsParams) ([]credit_model.Grant, error) {
	q := c.db.CreditEntry.Query().
		Where(
			db_credit.Namespace(namespace),
		)
	if len(params.LedgerIDs) > 0 {
		q = q.Where(db_credit.LedgerIDIn(slicesx.Map(params.LedgerIDs, func(id ulid.ULID) pgulid.ULID {
			return pgulid.Wrap(id)
		})...))
	}
	// equal?
	if params.From != nil {
		q = q.Where(db_credit.EffectiveAtGTE(*params.From))
	}
	if params.To != nil {
		q = q.Where(db_credit.EffectiveAtLT(*params.To))
	}
	if params.FromHighWatermark {
		t := sql.Table(db_credit.Table)

		// Define the subquery for the maximum reset date
		subQuery := sql.Select(
			sql.As(sql.Max(t.C(db_credit.FieldEffectiveAt)), "highwatermark"),
			t.C(db_credit.FieldLedgerID),
		).
			From(t).
			Where(
				sql.And(
					sql.EQ(t.C(db_credit.FieldNamespace), namespace),
					sql.EQ(t.C(db_credit.FieldEntryType), credit_model.EntryTypeReset),
				),
			).
			GroupBy(db_credit.FieldLedgerID)

		// include as subquery, and find the last reset for each subject
		// use the last reset as the high watermark
		q = q.Where(func(s *sql.Selector) {
			s.LeftJoin(subQuery).
				On(s.C(db_credit.FieldNamespace), t.C(db_credit.FieldNamespace)).
				On(s.C(db_credit.FieldLedgerID), t.C(db_credit.FieldLedgerID))

			// Ensure the effective date is greater than the last reset date
			s.Where(
				sql.Or(
					sql.IsNull(subQuery.C("highwatermark")),
					sql.ColumnsGTE(t.C(db_credit.FieldEffectiveAt), subQuery.C("highwatermark")),
				),
			)
		})
	}
	if params.IncludeVoid {
		// Has no void children or is void
		q = q.Where(
			db_credit.Or(
				db_credit.EntryTypeEQ(credit_model.EntryTypeVoidGrant),
				db_credit.And(
					db_credit.EntryTypeEQ(credit_model.EntryTypeGrant),
					db_credit.Not(db_credit.HasChildrenWith(
						db_credit.EntryTypeEQ(credit_model.EntryTypeVoidGrant),
					)),
				),
			),
		)
	} else {
		// Has no void children
		q = q.Where(
			db_credit.EntryTypeEQ(credit_model.EntryTypeGrant),
			db_credit.Not(db_credit.HasChildrenWith(
				db_credit.EntryTypeEQ(credit_model.EntryTypeVoidGrant),
			)),
		)
	}
	entities, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list grants: %w", err)
	}

	var list []credit_model.Grant
	for _, entity := range entities {
		grant, err := mapGrantEntity(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to map grant entity: %w", err)
		}
		list = append(list, grant)
	}

	return list, nil
}

func (c *PostgresConnector) GetGrant(ctx context.Context, namespace string, id ulid.ULID) (credit_model.Grant, error) {
	entity, err := c.db.CreditEntry.Query().Where(
		db_credit.Or(
			// grant
			db_credit.And(
				db_credit.Namespace(namespace),
				db_credit.ID(pgulid.Wrap(id)),
				db_credit.EntryTypeEQ(credit_model.EntryTypeGrant),
				db_credit.Not(db_credit.HasChildren()),
			),
			// void grant
			db_credit.And(
				db_credit.Namespace(namespace),
				db_credit.HasParentWith(db_credit.ID(pgulid.Wrap(id))),
				db_credit.EntryTypeEQ(credit_model.EntryTypeVoidGrant),
			),
		),
	).Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return credit_model.Grant{}, &credit_model.GrantNotFoundError{GrantID: id}
		}

		return credit_model.Grant{}, fmt.Errorf("failed to get grant: %w", err)
	}

	grant, err := mapGrantEntity(entity)
	if err != nil {
		return credit_model.Grant{}, fmt.Errorf("failed to map grant entity: %w", err)
	}
	return grant, nil
}

func mapGrantEntity(entry *db.CreditEntry) (credit_model.Grant, error) {
	if entry.EntryType != credit_model.EntryTypeGrant && entry.EntryType != credit_model.EntryTypeVoidGrant {
		return credit_model.Grant{}, fmt.Errorf("entry type must be grant: %s", entry.EntryType)
	}

	grant := credit_model.Grant{
		ID:          &entry.ID.ULID,
		ParentID:    entry.ParentID.ULIDPointer(),
		LedgerID:    entry.LedgerID.ULID,
		Type:        *entry.Type,
		FeatureID:   entry.FeatureID.ULIDPointer(),
		Amount:      *entry.Amount,
		Priority:    entry.Priority,
		EffectiveAt: entry.EffectiveAt.In(time.UTC),
		Expiration: credit_model.ExpirationPeriod{
			Duration: *entry.ExpirationPeriodDuration,
			Count:    *entry.ExpirationPeriodCount,
		},
		Metadata: entry.Metadata,
		Void:     entry.EntryType == credit_model.EntryTypeVoidGrant,
	}
	if entry.RolloverType != nil {
		grant.Rollover = &credit_model.GrantRollover{
			Type: *entry.RolloverType,
		}
		if entry.RolloverMaxAmount != nil {
			grant.Rollover.MaxAmount = entry.RolloverMaxAmount
		}
	}

	return grant, nil
}
