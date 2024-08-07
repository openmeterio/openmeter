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

import (
	"github.com/openmeterio/openmeter/api"
	httpdriver "github.com/openmeterio/openmeter/internal/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	EntitlementHandler        = httpdriver.EntitlementHandler
	MeteredEntitlementHandler = httpdriver.MeteredEntitlementHandler
)

type (
	CreateEntitlementHandler            = httpdriver.CreateEntitlementHandler
	CreateGrantHandler                  = httpdriver.CreateGrantHandler
	GetEntitlementBalanceHistoryHandler = httpdriver.GetEntitlementBalanceHistoryHandler
	GetEntitlementValueHandler          = httpdriver.GetEntitlementValueHandler
	GetEntitlementsOfSubjectHandler     = httpdriver.GetEntitlementsOfSubjectHandler
	ListEntitlementGrantsHandler        = httpdriver.ListEntitlementGrantsHandler
	ResetEntitlementUsageHandler        = httpdriver.ResetEntitlementUsageHandler
	ListEntitlementsHandler             = httpdriver.ListEntitlementsHandler
	GetEntitlementHandler               = httpdriver.GetEntitlementHandler
	GetEntitlementByIdHandler           = httpdriver.GetEntitlementByIdHandler
	DeleteEntitlementHandler            = httpdriver.DeleteEntitlementHandler
)

func NewEntitlementHandler(
	connector entitlement.EntitlementConnector,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) EntitlementHandler {
	return httpdriver.NewEntitlementHandler(connector, namespaceDecoder, options...)
}

func NewMeteredEntitlementHandler(
	entitlementConnector entitlement.EntitlementConnector,
	meteredEntitlementConnector meteredentitlement.Connector,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) MeteredEntitlementHandler {
	return httpdriver.NewMeteredEntitlementHandler(entitlementConnector, meteredEntitlementConnector, namespaceDecoder, options...)
}

func MapEntitlementValueToAPI(entitlementValue entitlement.EntitlementValue) (api.EntitlementValue, error) {
	return httpdriver.MapEntitlementValueToAPI(entitlementValue)
}
