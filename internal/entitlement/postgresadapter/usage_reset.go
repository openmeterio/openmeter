package postgresadapter

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db"
	db_usagereset "github.com/openmeterio/openmeter/internal/entitlement/postgresadapter/ent/db/usagereset"
	"github.com/openmeterio/openmeter/pkg/models"
)

type usageResetDBAdapter struct {
	db *db.Client
}

func NewPostgresUsageResetDBAdapter(db *db.Client) entitlement.UsageResetDBConnector {
	return &usageResetDBAdapter{
		db: db,
	}
}

func (a *usageResetDBAdapter) Save(ctx context.Context, usageResetTime entitlement.UsageResetTime) error {
	_, err := a.db.UsageReset.Create().
		SetEntitlementID(usageResetTime.EntitlementID).
		SetNamespace(usageResetTime.Namespace).
		SetResetTime(usageResetTime.ResetTime).
		Save(ctx)
	return err
}

func (a *usageResetDBAdapter) GetLastAt(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (*entitlement.UsageResetTime, error) {
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
			return nil, &entitlement.UsageResetNotFoundError{EntitlementID: entitlementID}
		}
		return nil, err
	}

	return mapUsageResetTime(res), nil
}

func (a *usageResetDBAdapter) GetBetween(ctx context.Context, entitlementID models.NamespacedID, from time.Time, to time.Time) ([]entitlement.UsageResetTime, error) {
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

	usageResetTimes := make([]entitlement.UsageResetTime, 0, len(res))
	for _, r := range res {
		usageResetTimes = append(usageResetTimes, *mapUsageResetTime(r))
	}

	return usageResetTimes, nil
}

func mapUsageResetTime(res *db.UsageReset) *entitlement.UsageResetTime {
	return &entitlement.UsageResetTime{
		EntitlementID: res.EntitlementID,
		ResetTime:     res.ResetTime,
	}
}
