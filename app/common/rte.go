package common

import (
	"context"
	"errors"
	"log/slog"
	"syscall"

	"github.com/google/wire"
	"github.com/oklog/run"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/rte"
	rteadapter "github.com/openmeterio/openmeter/openmeter/rte/adapter"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	creditcalculator "github.com/openmeterio/openmeter/openmeter/rte/creditbalance/calculator"
	rtemetered "github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	meteredcalculator "github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement/calculator"
	rteservice "github.com/openmeterio/openmeter/openmeter/rte/service"
)

var RTE = wire.NewSet(
	NewRTEService,
	RTEGroup,
)

func NewRTEService(client *db.Client, source customerbalance.Service, entitlements *registry.Entitlement, logger *slog.Logger, credits config.CreditsConfiguration) (rte.Service, error) {
	if !credits.Enabled {
		return nil, errors.New("RTE requires credits.enabled")
	}

	creditStore, err := rteadapter.NewCreditBalanceStore(rteadapter.Config{Client: client, Logger: logger})
	if err != nil {
		return nil, err
	}

	creditCalculator, err := creditcalculator.New(creditcalculator.Config{Source: source, Logger: logger})
	if err != nil {
		return nil, err
	}

	meteredStore, err := rteadapter.NewMeteredEntitlementBalanceStore(rteadapter.Config{Client: client, Logger: logger})
	if err != nil {
		return nil, err
	}

	meteredCalculator, err := meteredcalculator.New(meteredcalculator.Config{Source: entitlements.MeteredEntitlement, Logger: logger})
	if err != nil {
		return nil, err
	}

	return rteservice.New(rteservice.Config{
		CreditBalances: rteservice.Registration[creditbalance.Key, creditbalance.Balance]{
			Calculator: creditCalculator,
			Store:      creditStore,
		},
		MeteredEntitlements: rteservice.Registration[rtemetered.Key, rtemetered.Balance]{
			Calculator: meteredCalculator,
			Store:      meteredStore,
		},
		Logger: logger,
	})
}

func RTEGroup(ctx context.Context, telemetryServer TelemetryServer) run.Group {
	var group run.Group

	group.Add(
		func() error { return telemetryServer.ListenAndServe() },
		func(error) { _ = telemetryServer.Shutdown(ctx) },
	)

	group.Add(run.SignalHandler(ctx, syscall.SIGINT, syscall.SIGTERM))

	return group
}
