package credit

import (
	"github.com/openmeterio/openmeter/internal/credit"
)

type (
	BalanceConnector                        = credit.BalanceConnector
	BalanceHistoryParams                    = credit.BalanceHistoryParams
	BalanceSnapshotRepo                     = credit.BalanceSnapshotRepo
	CreateGrantInput                        = credit.CreateGrantInput
	DBCreateGrantInput                      = credit.GrantRepoCreateGrantInput
	Engine                                  = credit.Engine
	ExpirationPeriod                        = credit.ExpirationPeriod
	ExpirationPeriodDuration                = credit.ExpirationPeriodDuration
	Grant                                   = credit.Grant
	GrantBalanceMap                         = credit.GrantBalanceMap
	GrantBalanceNoSavedBalanceForOwnerError = credit.GrantBalanceNoSavedBalanceForOwnerError
	GrantBalanceSnapshot                    = credit.GrantBalanceSnapshot
	GrantBurnDownHistory                    = credit.GrantBurnDownHistory
	GrantBurnDownHistorySegment             = credit.GrantBurnDownHistorySegment
	GrantConnector                          = credit.GrantConnector
	GrantRepo                               = credit.GrantRepo
	GrantNotFoundError                      = credit.GrantNotFoundError
	GrantOrderBy                            = credit.GrantOrderBy
	GrantOwner                              = credit.GrantOwner
	GrantUsage                              = credit.GrantUsage
	GrantUsageTerminationReason             = credit.GrantUsageTerminationReason
	ListGrantsParams                        = credit.ListGrantsParams
	NamespacedGrantOwner                    = credit.NamespacedGrantOwner
	OwnerConnector                          = credit.OwnerConnector
	Pagination                              = credit.Pagination
	QueryUsageFn                            = credit.QueryUsageFn
	SegmentTerminationReason                = credit.SegmentTerminationReason
)
