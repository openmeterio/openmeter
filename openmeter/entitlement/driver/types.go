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

package entitlementdriver

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
