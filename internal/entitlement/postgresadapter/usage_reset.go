package postgresadapter

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"

	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"

	"github.com/openmeterio/openmeter/internal/ent/db"
	db_usagereset "github.com/openmeterio/openmeter/internal/ent/db/usagereset"
	"github.com/openmeterio/openmeter/pkg/models"
)

type usageResetDBAdapter struct {
	db *db.Client
}

func NewPostgresUsageResetRepo(db *db.Client) meteredentitlement.UsageResetRepo {
	return &usageResetDBAdapter{
		db: db,
	}
}

func (a *usageResetDBAdapter) Save(ctx context.Context, usageResetTime meteredentitlement.UsageResetTime) error {
	_, err := a.db.UsageReset.Create().
		SetEntitlementID(usageResetTime.EntitlementID).
		SetNamespace(usageResetTime.Namespace).
		SetResetTime(usageResetTime.ResetTime).
		Save(ctx)
	return err
}

func (a *usageResetDBAdapter) GetLastAt(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (*meteredentitlement.UsageResetTime, error) {
	res, err := a.db.UsageReset.Query().
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
}

func (a *usageResetDBAdapter) GetBetween(ctx context.Context, entitlementID models.NamespacedID, from time.Time, to time.Time) ([]meteredentitlement.UsageResetTime, error) {
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

	return usageResetTimes, nil
}

func mapUsageResetTime(res *db.UsageReset) *meteredentitlement.UsageResetTime {
	return &meteredentitlement.UsageResetTime{
		EntitlementID: res.EntitlementID,
		ResetTime:     res.ResetTime,
	}
}
