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
	"github.com/openmeterio/openmeter/internal/credit"
)

type BalanceConnector = credit.BalanceConnector
type BalanceHistoryParams = credit.BalanceHistoryParams
type BalanceSnapshotConnector = credit.BalanceSnapshotConnector
type CreateGrantInput = credit.CreateGrantInput
type DBCreateGrantInput = credit.GrantRepoCreateGrantInput
type Engine = credit.Engine
type ExpirationPeriod = credit.ExpirationPeriod
type ExpirationPeriodDuration = credit.ExpirationPeriodDuration
type Grant = credit.Grant
type GrantBalanceMap = credit.GrantBalanceMap
type GrantBalanceNoSavedBalanceForOwnerError = credit.GrantBalanceNoSavedBalanceForOwnerError
type GrantBalanceSnapshot = credit.GrantBalanceSnapshot
type GrantBurnDownHistory = credit.GrantBurnDownHistory
type GrantBurnDownHistorySegment = credit.GrantBurnDownHistorySegment
type GrantConnector = credit.GrantConnector
type GrantRepo = credit.GrantRepo
type GrantNotFoundError = credit.GrantNotFoundError
type GrantOrderBy = credit.GrantOrderBy
type GrantOwner = credit.GrantOwner
type GrantUsage = credit.GrantUsage
type GrantUsageTerminationReason = credit.GrantUsageTerminationReason
type ListGrantsParams = credit.ListGrantsParams
type NamespacedGrantOwner = credit.NamespacedGrantOwner
type OwnerConnector = credit.OwnerConnector
type Pagination = credit.Pagination
type QueryUsageFn = credit.QueryUsageFn
type SegmentTerminationReason = credit.SegmentTerminationReason
