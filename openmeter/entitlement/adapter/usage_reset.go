package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type usageResetDBAdapter struct {
	db *db.Client
}

var (
	_ meteredentitlement.UsageResetRepo = (*usageResetDBAdapter)(nil)
	_ interface {
		transaction.Creator
		entutils.TxUser[*usageResetDBAdapter]
	} = (*usageResetDBAdapter)(nil)
)

func NewPostgresUsageResetRepo(db *db.Client) *usageResetDBAdapter {
	return &usageResetDBAdapter{
		db: db,
	}
}

func (a *usageResetDBAdapter) Save(ctx context.Context, usageResetTime meteredentitlement.UsageResetUpdate) error {
	_, err := entutils.TransactingRepo[interface{}, *usageResetDBAdapter](
		ctx,
		a,
		func(ctx context.Context, repo *usageResetDBAdapter) (interface{}, error) {
			if err := usageResetTime.Validate(); err != nil {
				return nil, err
			}

			_, err := repo.db.UsageReset.Create().
				SetEntitlementID(usageResetTime.EntitlementID).
				SetNamespace(usageResetTime.Namespace).
				SetResetTime(usageResetTime.ResetTime).
				SetAnchor(usageResetTime.Anchor).
				SetUsagePeriodInterval(usageResetTime.UsagePeriodInterval).
				Save(ctx)
			return nil, err
		},
	)
	return err
}
