package postgres_connector

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
)

type resetWithRollovedGrants struct {
	credit.Reset
	rollovedGrants []credit.Grant
}

// Reset resets the ledger for the subject.
// Rolls over grants with rollover configuration.
func (c *PostgresConnector) Reset(ctx context.Context, reset credit.Reset) (credit.Reset, []credit.Grant, error) {
	ledgerID := credit.NewNamespacedLedgerID(reset.Namespace, reset.LedgerID)

	result, err := mutationTransaction(
		ctx,
		c,
		ledgerID,
		func(tx *db.Tx, ledgerEntity *db.Ledger) (*resetWithRollovedGrants, error) {
			var rollovedGrants []credit.Grant

			// Check if the reset is in the future
			err := checkAfterHighWatermark(reset.EffectiveAt, ledgerEntity)
			if err != nil {
				// Typed error do not wrap
				return nil, err
			}

			// Collect grants to rollover
			balance, err := c.GetBalance(ctx, ledgerID, reset.EffectiveAt)
			if err != nil {
				return nil, fmt.Errorf("failed to list grants: %w", err)
			}

			// Collect grants to rollover
			rolloverGrants := []credit.Grant{}
			for _, grantBalance := range balance.GrantBalances {
				grant := grantBalance.Grant

				// Do not rollover grants without rollover
				if grant.Rollover == nil {
					continue
				}

				switch grant.Rollover.Type {
				case credit.GrantRolloverTypeOriginalAmount:
					// Nothing to do, we rollover the original amount
				case credit.GrantRolloverTypeRemainingAmount:
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
					SetNamespace(reset.Namespace).
					SetLedgerID(string(reset.LedgerID)).
					SetEntryType(credit.EntryTypeReset).
					SetEffectiveAt(reset.EffectiveAt),
			}

			// Add new grants to the transaction
			for _, grant := range rolloverGrants {
				grantEntityCreate := tx.CreditEntry.Create().
					SetNamespace(reset.Namespace).
					SetLedgerID(string(grant.LedgerID)).
					SetEntryType(credit.EntryTypeGrant).
					SetType(grant.Type).
					SetNillableParentID((*string)(grant.ParentID)).
					SetNillableFeatureID((*string)(grant.FeatureID)).
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
		return credit.Reset{}, nil, err
	}

	return result.Reset, result.rollovedGrants, err
}

func mapResetEntity(entry *db.CreditEntry) (credit.Reset, error) {
	if entry.EntryType != credit.EntryTypeReset {
		return credit.Reset{}, fmt.Errorf("entry type must be reset: %s", entry.EntryType)
	}

	reset := credit.Reset{
		Namespace:   entry.Namespace,
		ID:          (*credit.GrantID)(&entry.ID),
		LedgerID:    credit.LedgerID(entry.LedgerID),
		EffectiveAt: entry.EffectiveAt.In(time.UTC),
	}

	return reset, nil
}
