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

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type BalanceSnapshotConnector interface {
	InvalidateAfter(ctx context.Context, owner NamespacedGrantOwner, at time.Time) error
	GetLatestValidAt(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (GrantBalanceSnapshot, error)
	Save(ctx context.Context, owner NamespacedGrantOwner, balances []GrantBalanceSnapshot) error

	entutils.TxCreator
	entutils.TxUser[BalanceSnapshotConnector]
}

// No balance has been saved since start of measurement for the owner
type GrantBalanceNoSavedBalanceForOwnerError struct {
	Owner NamespacedGrantOwner
	Time  time.Time
}

func (e GrantBalanceNoSavedBalanceForOwnerError) Error() string {
	return fmt.Sprintf("no saved balance for owner %s in namespace %s before %s", e.Owner.ID, e.Owner.Namespace, e.Time)
}
