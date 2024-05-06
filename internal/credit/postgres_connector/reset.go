package postgres_connector

import (
	"context"
	"fmt"
	"math"
	"time"

	credit_model "github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/pgulid"
)

type resetWithRollovedGrants struct {
	credit_model.Reset
	rollovedGrants []credit_model.Grant
}

// Reset resets the ledger for the subject.
// Rolls over grants with rollover configuration.
func (c *PostgresConnector) Reset(ctx context.Context, namespace string, reset credit_model.Reset) (credit_model.Reset, []credit_model.Grant, error) {
	result, err := mutationTransaction(ctx, c, namespace, reset.LedgerID, func(tx *db.Tx, ledgerEntity *db.Ledger) (*resetWithRollovedGrants, error) {
		var rollovedGrants []credit_model.Grant

		// Check if the reset is in the future
		err := checkAfterHighWatermark(reset.EffectiveAt, ledgerEntity)
		if err != nil {
			// Typed error do not wrap
			return nil, err
		}

		// Collect grants to rollover
		balance, err := c.GetBalance(ctx, namespace, reset.LedgerID, reset.EffectiveAt)
		if err != nil {
			return nil, fmt.Errorf("failed to list grants: %w", err)
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
			grant.ParentID = grant.ID
			grant.EffectiveAt = reset.EffectiveAt

			// Append grant to rollover grants
			rolloverGrants = append(rolloverGrants, grant)
		}

		// Add reset entry to the transaction
		createEntities := []*db.CreditEntryCreate{
			tx.CreditEntry.Create().
				SetNamespace(namespace).
				SetLedgerID(pgulid.Wrap(reset.LedgerID)).
				SetEntryType(credit_model.EntryTypeReset).
				SetEffectiveAt(reset.EffectiveAt),
		}

		// Add new grants to the transaction
		for _, grant := range rolloverGrants {
			grantEntityCreate := tx.CreditEntry.Create().
				SetNamespace(namespace).
				SetLedgerID(pgulid.Wrap(grant.LedgerID)).
				SetEntryType(credit_model.EntryTypeGrant).
				SetType(grant.Type).
				SetNillableParentID(pgulid.Ptr(grant.ParentID)).
				SetNillableFeatureID(pgulid.Ptr(grant.FeatureID)).
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
		entryEntities, err := tx.CreditEntry.CreateBulk(createEntities...).Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create grant entity: %w", err)
		}

		// Convert the entities to models
		resetEntity := entryEntities[0]
		reset, err = mapResetEntity(resetEntity)
		if err != nil {
			return nil, fmt.Errorf("failed to map reset entity: %w", err)
		}

		grantEntities := entryEntities[1:]
		for _, entity := range grantEntities {
			grant, err := mapGrantEntity(entity)
			if err != nil {
				return nil, fmt.Errorf("failed to map grant entity: %w", err)
			}
			rollovedGrants = append(rollovedGrants, grant)
		}

		// Update the ledger high watermark
		err = ledgerEntity.Update().SetHighwatermark(reset.EffectiveAt).Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to update ledger highwatermark: %w", err)
		}

		return &resetWithRollovedGrants{
			Reset:          reset,
			rollovedGrants: rollovedGrants,
		}, nil
	})

	if err != nil {
		return credit_model.Reset{}, nil, err
	}

	return result.Reset, result.rollovedGrants, err
}

func mapResetEntity(entry *db.CreditEntry) (credit_model.Reset, error) {
	if entry.EntryType != credit_model.EntryTypeReset {
		return credit_model.Reset{}, fmt.Errorf("entry type must be reset: %s", entry.EntryType)
	}

	reset := credit_model.Reset{
		ID:          &entry.ID.ULID,
		LedgerID:    entry.LedgerID.ULID,
		EffectiveAt: entry.EffectiveAt.In(time.UTC),
	}

	return reset, nil
}
