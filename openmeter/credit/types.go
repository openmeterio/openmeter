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
	BalanceSnapshotRepo                     = balance.BalanceSnapshotRepo
	CreateGrantInput                        = credit.CreateGrantInput
	DBCreateGrantInput                      = grant.GrantRepoCreateGrantInput
	Engine                                  = engine.Engine
	ExpirationPeriod                        = grant.ExpirationPeriod
	ExpirationPeriodDuration                = grant.ExpirationPeriodDuration
	Grant                                   = grant.Grant
	GrantBalanceMap                         = balance.GrantBalanceMap
	GrantBalanceNoSavedBalanceForOwnerError = balance.GrantBalanceNoSavedBalanceForOwnerError
	GrantBalanceSnapshot                    = balance.GrantBalanceSnapshot
	GrantBurnDownHistory                    = engine.GrantBurnDownHistory
	GrantBurnDownHistorySegment             = engine.GrantBurnDownHistorySegment
	GrantConnector                          = credit.GrantConnector
	GrantRepo                               = grant.GrantRepo
	GrantNotFoundError                      = credit.GrantNotFoundError
	GrantOrderBy                            = grant.GrantOrderBy
	GrantOwner                              = grant.GrantOwner
	GrantUsage                              = engine.GrantUsage
	GrantUsageTerminationReason             = engine.GrantUsageTerminationReason
	ListGrantsParams                        = grant.ListGrantsParams
	NamespacedGrantOwner                    = grant.NamespacedGrantOwner
	OwnerConnector                          = grant.OwnerConnector
	QueryUsageFn                            = engine.QueryUsageFn
	SegmentTerminationReason                = engine.SegmentTerminationReason
)
