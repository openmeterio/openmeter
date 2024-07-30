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
