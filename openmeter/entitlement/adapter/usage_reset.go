// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package adapter

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	db_usagereset "github.com/openmeterio/openmeter/openmeter/ent/db/usagereset"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
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
