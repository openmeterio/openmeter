package adapter

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	db_usagereset "github.com/openmeterio/openmeter/openmeter/ent/db/usagereset"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
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
		func(ctx context.Context, repo *usageResetDBAdapter) (*interface{}, error) {
			_, err := repo.db.UsageReset.Create().
				SetEntitlementID(usageResetTime.EntitlementID).
				SetNamespace(usageResetTime.Namespace).
				SetResetTime(usageResetTime.ResetTime).
				Save(ctx)
			return nil, err
		},
	)
	return err
}

func (a *usageResetDBAdapter) GetLastAt(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (*meteredentitlement.UsageResetTime, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *usageResetDBAdapter) (*meteredentitlement.UsageResetTime, error) {
			res, err := repo.db.UsageReset.Query().
				Where(
					db_usagereset.EntitlementID(entitlementID.ID),
					db_usagereset.Namespace(entitlementID.Namespace),
					db_usagereset.ResetTimeLTE(at),
				).
				Order(db_usagereset.ByResetTime(sql.OrderDesc())).
				First(ctx)
			if err != nil {
				if db.IsNotFound(err) {
					return nil, &meteredentitlement.UsageResetNotFoundError{EntitlementID: entitlementID}
				}
				return nil, err
			}

			return mapUsageResetTime(res), nil
		},
	)
}

func (a *usageResetDBAdapter) GetBetween(ctx context.Context, entitlementID models.NamespacedID, from time.Time, to time.Time) ([]meteredentitlement.UsageResetTime, error) {
	res, err := entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *usageResetDBAdapter) (*[]meteredentitlement.UsageResetTime, error) {
			res, err := a.db.UsageReset.Query().
				Where(
					db_usagereset.EntitlementID(entitlementID.ID),
					db_usagereset.Namespace(entitlementID.Namespace),
					db_usagereset.ResetTimeGTE(from),
					db_usagereset.ResetTimeLTE(to),
				).
				All(ctx)
			if err != nil {
				return nil, err
			}

			usageResetTimes := make([]meteredentitlement.UsageResetTime, 0, len(res))
			for _, r := range res {
				usageResetTimes = append(usageResetTimes, *mapUsageResetTime(r))
			}

			return &usageResetTimes, nil
		},
	)
	return defaultx.WithDefault(res, nil), err
}

func mapUsageResetTime(res *db.UsageReset) *meteredentitlement.UsageResetTime {
	return &meteredentitlement.UsageResetTime{
		EntitlementID: res.EntitlementID,
		ResetTime:     res.ResetTime,
	}
}
