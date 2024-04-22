package postgres_connector

import (
	"context"
	"fmt"
	"math"

	"entgo.io/ent/dialect/sql"

	connector "github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	db_credit "github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/creditentry"
	credit_model "github.com/openmeterio/openmeter/pkg/credit"
)

func (c *PostgresConnector) CreateGrant(ctx context.Context, namespace string, grant credit_model.Grant) (credit_model.Grant, error) {
	// Lock ledger for the subject
	logger := c.logger.With("operation", "createGrant", "namespace", namespace, "subject", grant.Subject, "id", grant.ID)
	lock, err := c.lockManager.Obtain(ctx, namespace, grant.Subject)
	if err != nil {
		return credit_model.Grant{}, err
	}
	defer func() {
		err = c.lockManager.Release(ctx, lock)
		if err != nil {
			logger.Error("failed to release lock", "error", err)
		}
	}()

	q := c.db.CreditEntry.Create().
		SetNamespace(namespace).
		SetNillableID(grant.ID).
		SetSubject(grant.Subject).
		SetEntryType(credit_model.EntryTypeGrant).
		SetType(grant.Type).
		SetNillableParentID(grant.ParentID).
		SetNillableProductID(grant.ProductID).
		SetAmount(grant.Amount).
		SetPriority(grant.Priority).
		SetEffectiveAt(grant.EffectiveAt).
		SetExpirationPeriodDuration(grant.Expiration.Duration).
		SetExpirationPeriodCount(grant.Expiration.Count).
		SetMetadata(grant.Metadata)
	if grant.Rollover != nil {
		q = q.SetRolloverType(grant.Rollover.Type).
			SetNillableRolloverMaxAmount(grant.Rollover.MaxAmount)
	}
	entity, err := q.Save(ctx)
	if err != nil {
		return grant, fmt.Errorf("failed to create grant: %w", err)
	}

	grant, err = mapGrantEntity(entity)
	if err != nil {
		return grant, fmt.Errorf("failed to map grant entity: %w", err)
	}
	return grant, nil
}

// TODO: use grant ID as an argument to void grant
func (c *PostgresConnector) VoidGrant(ctx context.Context, namespace string, grant credit_model.Grant) (credit_model.Grant, error) {
	if grant.ID == nil {
		return grant, fmt.Errorf("grant ID is required")
	}

	// Lock ledger for the subject
	logger := c.logger.With("operation", "voidGrant", "namespace", namespace, "subject", grant.Subject, "id", grant.ID)
	lock, err := c.lockManager.Obtain(ctx, namespace, grant.Subject)
	if err != nil {
		return credit_model.Grant{}, err
	}
	defer func() {
		err = c.lockManager.Release(ctx, lock)
		if err != nil {
			logger.Error("failed to release lock", "error", err)
		}
	}()

	entity, err := c.db.CreditEntry.Query().
		Where(
			db_credit.Namespace(namespace),
			db_credit.ID(*grant.ID),
		).
		Only(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return grant, &credit_model.GrantNotFoundError{GrantID: *grant.ID}
		}

		return grant, fmt.Errorf("failed to void grant: %w", err)
	}

	// create a new entry with parent ID and void type
	entity, err = c.db.CreditEntry.Create().
		SetNamespace(entity.Namespace).
		SetParentID(entity.ID).
		SetSubject(entity.Subject).
		SetEntryType(credit_model.EntryTypeVoidGrant).
		SetType(*entity.Type).
		SetNillableProductID(entity.ProductID).
		SetAmount(*entity.Amount).
		SetPriority(entity.Priority).
		SetEffectiveAt(entity.EffectiveAt).
		SetExpirationPeriodDuration(*entity.ExpirationPeriodDuration).
		SetExpirationPeriodCount(*entity.ExpirationPeriodCount).
		SetMetadata(entity.Metadata).
		Save(ctx)
	if err != nil {
		return grant, fmt.Errorf("failed to void grant: %w", err)
	}

	grant, err = mapGrantEntity(entity)
	if err != nil {
		return grant, fmt.Errorf("failed to map grant entity: %w", err)
	}
	return grant, nil
}

func (c *PostgresConnector) ListGrants(ctx context.Context, namespace string, params connector.ListGrantsParams) ([]credit_model.Grant, error) {
	q := c.db.CreditEntry.Query().
		Where(
			db_credit.Namespace(namespace),
		)
	if len(params.Subjects) > 0 {
		q = q.Where(db_credit.SubjectIn(params.Subjects...))
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
			t.C(db_credit.FieldSubject),
		).
			From(t).
			Where(
				sql.And(
					sql.EQ(t.C(db_credit.FieldNamespace), namespace),
					sql.EQ(t.C(db_credit.FieldEntryType), credit_model.EntryTypeReset),
				),
			).
			GroupBy(db_credit.FieldSubject)

		// include as subquery, and find the last reset for each subject
		// use the last reset as the high watermark
		q = q.Where(func(s *sql.Selector) {
			s.LeftJoin(subQuery).
				On(s.C(db_credit.FieldNamespace), t.C(db_credit.FieldNamespace)).
				On(s.C(db_credit.FieldSubject), t.C(db_credit.FieldSubject))

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

func (c *PostgresConnector) GetGrant(ctx context.Context, namespace string, id string) (credit_model.Grant, error) {
	entity, err := c.db.CreditEntry.Query().Where(
		db_credit.Or(
			// grant
			db_credit.And(
				db_credit.Namespace(namespace),
				db_credit.ID(id),
				db_credit.EntryTypeEQ(credit_model.EntryTypeGrant),
				db_credit.Not(db_credit.HasChildren()),
			),
			// void grant
			db_credit.And(
				db_credit.Namespace(namespace),
				db_credit.HasParentWith(db_credit.ID(id)),
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

	grantOut, err := mapGrantEntity(entity)
	if err != nil {
		return credit_model.Grant{}, fmt.Errorf("failed to map grant entity: %w", err)
	}
	return grantOut, nil
}

// Reset resets the ledger for the subject.
// Rolls over grants with rollover configuration.
func (c *PostgresConnector) Reset(ctx context.Context, namespace string, reset credit_model.Reset) (credit_model.Reset, []credit_model.Grant, error) {
	var rollovedGrants []credit_model.Grant

	// Lock ledger for the subject
	logger := c.logger.With("operation", "reset", "namespace", namespace, "subject", reset.Subject, "id", reset.ID)
	lock, err := c.lockManager.Obtain(ctx, namespace, reset.Subject)
	if err != nil {
		return credit_model.Reset{}, rollovedGrants, err
	}
	defer func() {
		err = c.lockManager.Release(ctx, lock)
		if err != nil {
			logger.Error("failed to release lock", "error", err)
		}
	}()

	// Collect grants to rollover
	balance, err := c.GetBalance(ctx, namespace, reset.Subject, reset.EffectiveAt)
	if err != nil {
		return reset, rollovedGrants, fmt.Errorf("failed to list grants: %w", err)
	}

	// Collect grants to rollover
	rolloverGrants := []credit_model.Grant{}
	for _, grantBalance := range balance.GrantBalances {
		grant := grantBalance.Grant

		// Do not rollover grants without rollover
		if grant.Rollover == nil {
			continue
		}

		switch grant.Rollover.Type {
		case credit_model.GrantRolloverTypeOriginalAmount:
			// Nothing to do, we rollover the original amount
		case credit_model.GrantRolloverTypeRemainingAmount:
			// We rollover the remaining amount
			grant.Amount = grantBalance.Balance
		}
		if grant.Rollover.MaxAmount != nil {
			grant.Amount = math.Max(*grant.Rollover.MaxAmount, grant.Amount)
		}
		// Skip grants with zero amount, amount never goes negative
		if grant.Amount == 0 {
			continue
		}

		// Set the parent ID to the grant ID we are rolling over
		parentId := *grant.ID
		grant.ParentID = &parentId
		grant.EffectiveAt = reset.EffectiveAt

		// Append grant to rollover grants
		rolloverGrants = append(rolloverGrants, grant)
	}

	// Add reset entry to the transaction
	createEntities := []*db.CreditEntryCreate{
		c.db.CreditEntry.Create().
			SetNamespace(namespace).
			SetSubject(reset.Subject).
			SetEntryType(credit_model.EntryTypeReset).
			SetEffectiveAt(reset.EffectiveAt),
	}

	// Add new grants to the transaction
	for _, grant := range rolloverGrants {
		grantEntityCreate := c.db.CreditEntry.Create().
			SetNamespace(namespace).
			SetSubject(grant.Subject).
			SetEntryType(credit_model.EntryTypeGrant).
			SetType(grant.Type).
			SetNillableParentID(grant.ParentID).
			SetNillableProductID(grant.ProductID).
			SetAmount(grant.Amount).
			SetPriority(grant.Priority).
			SetEffectiveAt(grant.EffectiveAt).
			SetExpirationPeriodDuration(grant.Expiration.Duration).
			SetExpirationPeriodCount(grant.Expiration.Count).
			SetMetadata(grant.Metadata)

		if grant.Rollover != nil {
			grantEntityCreate = grantEntityCreate.
				SetNillableRolloverMaxAmount(grant.Rollover.MaxAmount).
				SetRolloverType(grant.Rollover.Type)
		}

		createEntities = append(createEntities, grantEntityCreate)

	}

	// Create the reset and grant entries
	entryEntities, err := c.db.CreditEntry.CreateBulk(createEntities...).Save(ctx)
	if err != nil {
		return reset, rollovedGrants, fmt.Errorf("failed to create grant entity: %w", err)
	}

	// Convert the entities to models
	resetEntity := entryEntities[0]
	reset, err = mapResetEntity(resetEntity)
	if err != nil {
		return reset, rollovedGrants, fmt.Errorf("failed to map reset entity: %w", err)
	}

	grantEntities := entryEntities[1:]
	for _, entity := range grantEntities {
		grant, err := mapGrantEntity(entity)
		if err != nil {
			return reset, rollovedGrants, fmt.Errorf("failed to map grant entity: %w", err)
		}
		rollovedGrants = append(rollovedGrants, grant)
	}
	return reset, rollovedGrants, nil
}

func mapGrantEntity(entry *db.CreditEntry) (credit_model.Grant, error) {
	if entry.EntryType != credit_model.EntryTypeGrant && entry.EntryType != credit_model.EntryTypeVoidGrant {
		return credit_model.Grant{}, fmt.Errorf("entry type must be grant: %s", entry.EntryType)
	}

	grant := credit_model.Grant{
		ID:          &entry.ID,
		ParentID:    entry.ParentID,
		Subject:     entry.Subject,
		Type:        *entry.Type,
		ProductID:   entry.ProductID,
		Amount:      *entry.Amount,
		Priority:    entry.Priority,
		EffectiveAt: entry.EffectiveAt,
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

func mapResetEntity(entry *db.CreditEntry) (credit_model.Reset, error) {
	if entry.EntryType != credit_model.EntryTypeReset {
		return credit_model.Reset{}, fmt.Errorf("entry type must be reset: %s", entry.EntryType)
	}

	reset := credit_model.Reset{
		ID:          &entry.ID,
		Subject:     entry.Subject,
		EffectiveAt: entry.EffectiveAt,
	}

	return reset, nil
}
