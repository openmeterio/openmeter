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

package creditdriver

import (
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handlers struct {
	GetFeature    GetFeatureHandler
	ListFeatures  ListFeaturesHandler
	CreateFeature CreateFeatureHandler
	DeleteFeature DeleteFeatureHandler

	// Ledger
	CreateLedger     CreateLedgerHandler
	ListLedgers      ListLedgersHandler
	GetLedgerHistory GetLedgerHistoryHandler

	// Reset
	ResetLedger ResetLedgerHandler

	// Grant
	ListLedgerGrants         ListLedgerGrantsHandler
	ListLedgerGrantsByLedger ListLedgerGrantsByLedgerHandler
	CreateLedgerGrant        CreateLedgerGrantHandler
	VoidLedgerGrant          VoidLedgerGrantHandler
	GetLedgerGrant           GetLedgerGrantHandler

	// Balances
	GetLedgerBalance GetLedgerBalanceHandler
}

func New(
	creditConnector credit.Connector,
	meterRepository meter.Repository,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) Handlers {
	builder := &builder{
		CreditConnector:  creditConnector,
		MeterRepository:  meterRepository,
		NamespaceDecoder: namespaceDecoder,
		Options:          options,
	}

	return Handlers{
		GetFeature:    builder.GetFeature(),
		ListFeatures:  builder.ListFeatures(),
		CreateFeature: builder.CreateFeature(),
		DeleteFeature: builder.DeleteFeature(),

		// Ledgers
		CreateLedger:     builder.CreateLedger(),
		ListLedgers:      builder.ListLedgers(),
		GetLedgerHistory: builder.GetLedgerHistory(),

		// Reset
		ResetLedger: builder.ResetLedger(),

		// Grants
		ListLedgerGrants:         builder.ListLedgerGrants(),
		ListLedgerGrantsByLedger: builder.ListLedgerGrantsByLedger(),
		CreateLedgerGrant:        builder.CreateLedgerGrant(),
		VoidLedgerGrant:          builder.VoidLedgerGrant(),
		GetLedgerGrant:           builder.GetLedgerGrant(),

		// Balances
		GetLedgerBalance: builder.GetLedgerBalance(),
	}
}

type builder struct {
	CreditConnector  credit.Connector
	MeterRepository  meter.Repository
	NamespaceDecoder namespacedriver.NamespaceDecoder
	Options          []httptransport.HandlerOption
}
