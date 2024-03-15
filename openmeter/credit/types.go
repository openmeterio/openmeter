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
	"github.com/openmeterio/openmeter/internal/credit/balance"
	"github.com/openmeterio/openmeter/internal/credit/engine"
	"github.com/openmeterio/openmeter/internal/credit/grant"
)

type (
	CreditConnector                         = credit.CreditConnector
	BalanceConnector                        = credit.BalanceConnector
	BalanceHistoryParams                    = credit.BalanceHistoryParams
	BalanceSnapshotRepo                     = balance.SnapshotRepo
	CreateGrantInput                        = credit.CreateGrantInput
	DBCreateGrantInput                      = grant.RepoCreateInput
	Engine                                  = engine.Engine
	ExpirationPeriod                        = grant.ExpirationPeriod
	ExpirationPeriodDuration                = grant.ExpirationPeriodDuration
	Grant                                   = grant.Grant
	GrantBalanceMap                         = balance.Map
	GrantBalanceNoSavedBalanceForOwnerError = balance.NoSavedBalanceForOwnerError
	GrantBalanceSnapshot                    = balance.Snapshot
	GrantBurnDownHistory                    = engine.GrantBurnDownHistory
	GrantBurnDownHistorySegment             = engine.GrantBurnDownHistorySegment
	GrantConnector                          = credit.GrantConnector
	GrantRepo                               = grant.Repo
	GrantNotFoundError                      = credit.GrantNotFoundError
	GrantOrderBy                            = grant.OrderBy
	GrantOwner                              = grant.Owner
	GrantUsage                              = engine.GrantUsage
	GrantUsageTerminationReason             = engine.GrantUsageTerminationReason
	ListGrantsParams                        = grant.ListParams
	NamespacedGrantOwner                    = grant.NamespacedOwner
	OwnerConnector                          = grant.OwnerConnector
	QueryUsageFn                            = engine.QueryUsageFn
	SegmentTerminationReason                = engine.SegmentTerminationReason
)
