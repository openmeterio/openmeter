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

package balance

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type SnapshotRepo interface {
	InvalidateAfter(ctx context.Context, owner grant.NamespacedOwner, at time.Time) error
	GetLatestValidAt(ctx context.Context, owner grant.NamespacedOwner, at time.Time) (Snapshot, error)
	Save(ctx context.Context, owner grant.NamespacedOwner, balances []Snapshot) error

	entutils.TxCreator
	entutils.TxUser[SnapshotRepo]
}

// No balance has been saved since start of measurement for the owner
type NoSavedBalanceForOwnerError struct {
	Owner grant.NamespacedOwner
	Time  time.Time
}

func (e NoSavedBalanceForOwnerError) Error() string {
	return fmt.Sprintf("no saved balance for owner %s in namespace %s before %s", e.Owner.ID, e.Owner.Namespace, e.Time)
}
