package entitlement

import "github.com/openmeterio/openmeter/internal/entitlement"

type BalanceHistoryParams = entitlement.BalanceHistoryParams
type CreateEntitlementDBInputs = entitlement.EntitlementRepoCreateEntitlementInputs
type CreateEntitlementGrantInputs = entitlement.CreateEntitlementGrantInputs
type CreateEntitlementInputs = entitlement.CreateEntitlementInputs
type Entitlement = entitlement.Entitlement
type EntitlementAlreadyExistsError = entitlement.EntitlementAlreadyExistsError
type EntitlementBalance = entitlement.EntitlementBalance
type EntitlementBalanceConnector = entitlement.EntitlementBalanceConnector
type EntitlementBalanceHistoryWindow = entitlement.EntitlementBalanceHistoryWindow
type EntitlementConnector = entitlement.EntitlementConnector
type EntitlementRepo = entitlement.EntitlementRepo
type EntitlementGrant = entitlement.EntitlementGrant
type EntitlementNotFoundError = entitlement.EntitlementNotFoundError
type EntitlementValue = entitlement.EntitlementValue
type UsageResetRepo = entitlement.UsageResetRepo
type UsageResetNotFoundError = entitlement.UsageResetNotFoundError
type UsageResetTime = entitlement.UsageResetTime
type WindowSize = entitlement.WindowSize
