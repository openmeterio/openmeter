package adapter

import (
	"context"
	"errors"

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

func (a *usageResetDBAdapter) Save(ctx context.Context, usageResetTime meteredentitlement.UsageResetTime) error {
	_, err := entutils.TransactingRepo[interface{}, *usageResetDBAdapter](
		ctx,
		a,
		func(ctx context.Context, repo *usageResetDBAdapter) (interface{}, error) {
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

func mapUsageResetTime(res *db.UsageReset) (meteredentitlement.UsageResetTime, error) {
	if res == nil {
		return meteredentitlement.UsageResetTime{}, errors.New("usage reset is nil")
	}

	return meteredentitlement.UsageResetTime{
		EntitlementID:       res.EntitlementID,
		ResetTime:           res.ResetTime,
		Anchor:              res.Anchor,
		UsagePeriodInterval: res.UsagePeriodInterval,
	}, nil
}
