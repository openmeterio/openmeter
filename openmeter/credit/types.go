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
type Period = credit.Period
type QueryUsageFn = credit.QueryUsageFn
type SegmentTerminationReason = credit.SegmentTerminationReason
