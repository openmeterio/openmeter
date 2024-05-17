package postgres_connector

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_credit "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/creditentry"
	"github.com/openmeterio/openmeter/pkg/convertx"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (c *PostgresConnector) CreateGrant(ctx context.Context, grantIn credit.Grant) (credit.Grant, error) {
	grant, err := mutationTransaction(ctx, c, credit.NewNamespacedLedgerID(grantIn.Namespace, grantIn.LedgerID), func(tx *db.Tx, ledgerEntity *db.Ledger) (*credit.Grant, error) {
		// Check if the reset is in the future
		err := checkAfterHighWatermark(grantIn.EffectiveAt, ledgerEntity)
		if err != nil {
			// Typed error do not wrap
			return nil, err
		}

		// TODO: validate namespace for not being empty (at all places)

		q := tx.CreditEntry.Create().
			SetNamespace(grantIn.Namespace).
			SetNillableID((*string)(grantIn.ID)).
			SetLedgerID(string(grantIn.LedgerID)).
			SetEntryType(credit.EntryTypeGrant).
			SetType(grantIn.Type).
			SetNillableParentID((*string)(grantIn.ParentID)).
			SetNillableFeatureID((*string)(grantIn.FeatureID)).
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
		return credit.Grant{}, err
	}

	return *grant, nil
}

func (c *PostgresConnector) VoidGrant(ctx context.Context, grantIn credit.Grant) (credit.Grant, error) {
	grant, err := mutationTransaction(ctx, c, credit.NewNamespacedLedgerID(grantIn.Namespace, grantIn.LedgerID), func(tx *db.Tx, ledgerEntity *db.Ledger) (*credit.Grant, error) {
		// Check if the reset is in the future
		err := checkAfterHighWatermark(grantIn.EffectiveAt, ledgerEntity)
		if err != nil {
			// Typed error do not wrap
			return nil, err
		}

		// Get balance to check if grant can be voided: not partially or fully used yet
		balance, err := c.GetBalance(ctx, credit.NewNamespacedLedgerID(grantIn.Namespace, grantIn.LedgerID), time.Now())
		if err != nil {
			return nil, err
		}

		if grantIn.ID == nil {
			return nil, errors.New("grant ID is required")
		}

		for _, entry := range balance.GrantBalances {
			if *entry.Grant.ID == *grantIn.ID {
				if entry.Balance != grantIn.Amount {
					return nil, fmt.Errorf("grant has been used, cannot void: %s", *grantIn.ID)
				}
				break
			}

		}

		if grantIn.ID == nil {
			return nil, fmt.Errorf("grant ID is required")
		}

		entity, err := tx.CreditEntry.Query().
			Where(
				db_credit.Namespace(grantIn.Namespace),
				db_credit.ID(string(*grantIn.ID)),
			).
			Only(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return nil, &credit.GrantNotFoundError{GrantID: *grantIn.ID}
			}

			return nil, fmt.Errorf("failed to void grant: %w", err)
		}

		// create a new entry with parent ID and void type
		entity, err = tx.CreditEntry.Create().
			SetNamespace(entity.Namespace).
			SetParentID(entity.ID).
			SetLedgerID(entity.LedgerID).
			SetEntryType(credit.EntryTypeVoidGrant).
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
		return credit.Grant{}, err
	}

	return *grant, nil
}

func (c *PostgresConnector) ListGrants(ctx context.Context, params credit.ListGrantsParams) ([]credit.Grant, error) {
	q := c.db.CreditEntry.Query().
		Where(
			db_credit.Namespace(params.Namespace),
		)
	if len(params.LedgerIDs) > 0 {
		q = q.Where(db_credit.LedgerIDIn(slicesx.Map(params.LedgerIDs, func(id credit.LedgerID) string {
			return string(id)
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
					sql.EQ(t.C(db_credit.FieldNamespace), params.Namespace),
					sql.EQ(t.C(db_credit.FieldEntryType), credit.EntryTypeReset),
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
				db_credit.EntryTypeEQ(credit.EntryTypeVoidGrant),
				db_credit.And(
					db_credit.EntryTypeEQ(credit.EntryTypeGrant),
					db_credit.Not(db_credit.HasChildrenWith(
						db_credit.EntryTypeEQ(credit.EntryTypeVoidGrant),
					)),
				),
			),
		)
	} else {
		// Has no void children
		q = q.Where(
			db_credit.EntryTypeEQ(credit.EntryTypeGrant),
			db_credit.Not(db_credit.HasChildrenWith(
				db_credit.EntryTypeEQ(credit.EntryTypeVoidGrant),
			)),
		)
	}
	entities, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list grants: %w", err)
	}

	var list []credit.Grant
	for _, entity := range entities {
		grant, err := mapGrantEntity(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to map grant entity: %w", err)
		}
		list = append(list, grant)
	}

	return list, nil
}

func (c *PostgresConnector) GetGrant(ctx context.Context, grantID credit.NamespacedGrantID) (credit.Grant, error) {
	entity, err := c.db.CreditEntry.Query().Where(
		db_credit.Or(
			// grant
			db_credit.And(
				db_credit.Namespace(grantID.Namespace),
				db_credit.ID(string(grantID.ID)),
				db_credit.EntryTypeEQ(credit.EntryTypeGrant),
				db_credit.Not(db_credit.HasChildren()),
			),
			// void grant
			db_credit.And(
				db_credit.Namespace(grantID.Namespace),
				db_credit.HasParentWith(db_credit.ID(string(grantID.ID))),
				db_credit.EntryTypeEQ(credit.EntryTypeVoidGrant),
			),
		),
	).Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return credit.Grant{}, &credit.GrantNotFoundError{GrantID: grantID.ID}
		}

		return credit.Grant{}, fmt.Errorf("failed to get grant: %w", err)
	}

	grant, err := mapGrantEntity(entity)
	if err != nil {
		return credit.Grant{}, fmt.Errorf("failed to map grant entity: %w", err)
	}
	return grant, nil
}

func mapGrantEntity(entry *db.CreditEntry) (credit.Grant, error) {
	if entry.EntryType != credit.EntryTypeGrant && entry.EntryType != credit.EntryTypeVoidGrant {
		return credit.Grant{}, fmt.Errorf("entry type must be grant: %s", entry.EntryType)
	}

	grant := credit.Grant{
		Namespace:   entry.Namespace,
		ID:          (*credit.GrantID)(&entry.ID),
		ParentID:    (*credit.GrantID)(entry.ParentID),
		LedgerID:    credit.LedgerID(entry.LedgerID),
		Type:        *entry.Type,
		FeatureID:   (*credit.FeatureID)(entry.FeatureID),
		Amount:      *entry.Amount,
		Priority:    entry.Priority,
		EffectiveAt: entry.EffectiveAt.In(time.UTC),
		Expiration: credit.ExpirationPeriod{
			Duration: *entry.ExpirationPeriodDuration,
			Count:    *entry.ExpirationPeriodCount,
		},
		Metadata:  entry.Metadata,
		Void:      entry.EntryType == credit.EntryTypeVoidGrant,
		CreatedAt: convertx.ToPointer(entry.CreatedAt.In(time.UTC)),
		UpdatedAt: convertx.ToPointer(entry.UpdatedAt.In(time.UTC)),
	}
	if entry.RolloverType != nil {
		grant.Rollover = &credit.GrantRollover{
			Type: *entry.RolloverType,
		}
		if entry.RolloverMaxAmount != nil {
			grant.Rollover.MaxAmount = entry.RolloverMaxAmount
		}
	}

	return grant, nil
}
