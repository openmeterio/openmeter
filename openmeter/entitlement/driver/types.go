package httpdriver

import httpdriver "github.com/openmeterio/openmeter/internal/entitlement/driver"

// requests
type (
	CreateEntitlementHandlerRequest            = httpdriver.CreateEntitlementHandlerRequest
	CreateGrantHandlerRequest                  = httpdriver.CreateGrantHandlerRequest
	GetEntitlementBalanceHistoryHandlerRequest = httpdriver.GetEntitlementBalanceHistoryHandlerRequest
	GetEntitlementValueHandlerRequest          = httpdriver.GetEntitlementValueHandlerRequest
	GetEntitlementsOfSubjectHandlerRequest     = httpdriver.GetEntitlementsOfSubjectHandlerRequest
	ListEntitlementGrantHandlerRequest         = httpdriver.ListEntitlementGrantHandlerRequest
	ResetEntitlementUsageHandlerRequest        = httpdriver.ResetEntitlementUsageHandlerRequest
	ListEntitlementsHandlerRequest             = httpdriver.ListEntitlementsHandlerRequest
	GetEntitlementHandlerRequest               = httpdriver.GetEntitlementHandlerRequest
	GetEntitlementByIdHandlerRequest           = httpdriver.GetEntitlementByIdHandlerRequest
	DeleteEntitlementHandlerRequest            = httpdriver.DeleteEntitlementHandlerRequest
)

// responses
type (
	CreateEntitlementHandlerResponse            = httpdriver.CreateEntitlementHandlerResponse
	CreateGrantHandlerResponse                  = httpdriver.CreateGrantHandlerResponse
	GetEntitlementBalanceHistoryHandlerResponse = httpdriver.GetEntitlementBalanceHistoryHandlerResponse
	GetEntitlementValueHandlerResponse          = httpdriver.GetEntitlementValueHandlerResponse
	GetEntitlementsOfSubjectHandlerResponse     = httpdriver.GetEntitlementsOfSubjectHandlerResponse
	ListEntitlementGrantHandlerResponse         = httpdriver.ListEntitlementGrantHandlerResponse
	ResetEntitlementUsageHandlerResponse        = httpdriver.ResetEntitlementUsageHandlerResponse
	ListEntitlementsHandlerResponse             = httpdriver.ListEntitlementsHandlerResponse
	GetEntitlementHandlerResponse               = httpdriver.GetEntitlementHandlerResponse
	GetEntitlementByIdHandlerResponse           = httpdriver.GetEntitlementByIdHandlerResponse
	DeleteEntitlementHandlerResponse            = httpdriver.DeleteEntitlementHandlerResponse
)

// params
type (
	CreateEntitlementHandlerParams            = httpdriver.CreateEntitlementHandlerParams
	CreateGrantHandlerParams                  = httpdriver.CreateGrantHandlerParams
	GetEntitlementBalanceHistoryHandlerParams = httpdriver.GetEntitlementBalanceHistoryHandlerParams
	GetEntitlementValueHandlerParams          = httpdriver.GetEntitlementValueHandlerParams
	GetEntitlementsOfSubjectHandlerParams     = httpdriver.GetEntitlementsOfSubjectHandlerParams
	ListEntitlementGrantsHandlerParams        = httpdriver.ListEntitlementGrantsHandlerParams
	ResetEntitlementUsageHandlerParams        = httpdriver.ResetEntitlementUsageHandlerParams
	ListEntitlementsHandlerParams             = httpdriver.ListEntitlementsHandlerParams
	GetEntitlementHandlerParams               = httpdriver.GetEntitlementHandlerParams
	GetEntitlementByIdHandlerParams           = httpdriver.GetEntitlementByIdHandlerParams
	DeleteEntitlementHandlerParams            = httpdriver.DeleteEntitlementHandlerParams
)
