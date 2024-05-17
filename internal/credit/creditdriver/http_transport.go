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
