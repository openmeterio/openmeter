package entitlement

import "github.com/openmeterio/openmeter/internal/entitlement"

type (
	SubTypeConnector              = entitlement.SubTypeConnector
	CreateEntitlementInputs       = entitlement.CreateEntitlementInputs
	Entitlement                   = entitlement.Entitlement
	EntitlementAlreadyExistsError = entitlement.AlreadyExistsError
	EntitlementConnector          = entitlement.Connector
	EntitlementRepo               = entitlement.EntitlementRepo
	EntitlementNotFoundError      = entitlement.NotFoundError
	EntitlementValue              = entitlement.EntitlementValue
	ListEntitlementsParams        = entitlement.ListEntitlementsParams
)
