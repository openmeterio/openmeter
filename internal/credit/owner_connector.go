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

package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type EndCurrentUsagePeriodParams struct {
	At           time.Time
	RetainAnchor bool
}

type OwnerMeter struct {
	MeterSlug     string
	DefaultParams *streaming.QueryParams
	WindowSize    models.WindowSize
}

type OwnerConnector interface {
	GetMeter(ctx context.Context, owner NamespacedGrantOwner) (*OwnerMeter, error)
	GetStartOfMeasurement(ctx context.Context, owner NamespacedGrantOwner) (time.Time, error)
	GetPeriodStartTimesBetween(ctx context.Context, owner NamespacedGrantOwner, from, to time.Time) ([]time.Time, error)
	GetUsagePeriodStartAt(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (time.Time, error)

	//FIXME: this is a terrible hack
	EndCurrentUsagePeriodTx(ctx context.Context, tx *entutils.TxDriver, owner NamespacedGrantOwner, params EndCurrentUsagePeriodParams) error
	//FIXME: this is a terrible hack
	LockOwnerForTx(ctx context.Context, tx *entutils.TxDriver, owner NamespacedGrantOwner) error
}

type OwnerNotFoundError struct {
	Owner          NamespacedGrantOwner
	AttemptedOwner string
}

func (e OwnerNotFoundError) Error() string {
	return fmt.Sprintf("Owner %s not found in namespace %s, attempted to find as %s", e.Owner.ID, e.Owner.Namespace, e.AttemptedOwner)
}
