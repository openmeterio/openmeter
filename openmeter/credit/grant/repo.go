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

package grant

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/recurrence"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type OrderBy string

const (
	OrderByCreatedAt   OrderBy = "created_at"
	OrderByUpdatedAt   OrderBy = "updated_at"
	OrderByExpiresAt   OrderBy = "expires_at"
	OrderByEffectiveAt OrderBy = "effective_at"
	OrderByOwner       OrderBy = "owner_id" // check
)

func (f OrderBy) Values() []OrderBy {
	return []OrderBy{
		OrderByCreatedAt,
		OrderByUpdatedAt,
		OrderByExpiresAt,
		OrderByEffectiveAt,
		OrderByOwner,
	}
}

func (f OrderBy) StrValues() []string {
	return slicesx.Map(f.Values(), func(v OrderBy) string {
		return string(v)
	})
}

type ListParams struct {
	Namespace        string
	OwnerID          *Owner
	IncludeDeleted   bool
	SubjectKeys      []string
	FeatureIdsOrKeys []string
	Page             pagination.Page
	OrderBy          OrderBy
	Order            sortx.Order
	// will be deprecated
	Limit int
	// will be deprecated
	Offset int
}

type RepoCreateInput struct {
	OwnerID          Owner
	Namespace        string
	Amount           float64
	Priority         uint8
	EffectiveAt      time.Time
	Expiration       ExpirationPeriod
	ExpiresAt        time.Time
	Metadata         map[string]string
	ResetMaxRollover float64
	ResetMinRollover float64
	Recurrence       *recurrence.Recurrence
}

type Repo interface {
	CreateGrant(ctx context.Context, grant RepoCreateInput) (*Grant, error)
	VoidGrant(ctx context.Context, grantID models.NamespacedID, at time.Time) error
	// For bw compatibility, if pagination is not provided we return a simple array
	ListGrants(ctx context.Context, params ListParams) (pagination.PagedResponse[Grant], error)
	// ListActiveGrantsBetween returns all grants that are active at any point between the given time range.
	ListActiveGrantsBetween(ctx context.Context, owner NamespacedOwner, from, to time.Time) ([]Grant, error)
	GetGrant(ctx context.Context, grantID models.NamespacedID) (Grant, error)

	entutils.TxCreator
	entutils.TxUser[Repo]
}
