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
	"github.com/openmeterio/openmeter/openmeter/rte"
	rteadapter "github.com/openmeterio/openmeter/openmeter/rte/adapter"
	rteservice "github.com/openmeterio/openmeter/openmeter/rte/service"
)

var RTE = wire.NewSet(
	NewRTEService,
	RTEGroup,
)

func NewRTEService(client *db.Client, calculator customerbalance.Service, logger *slog.Logger, credits config.CreditsConfiguration) (rte.Service, error) {
	if !credits.Enabled {
		return nil, errors.New("RTE requires credits.enabled")
	}

	adapter, err := rteadapter.New(rteadapter.Config{Client: client, Logger: logger})
	if err != nil {
		return nil, err
	}

	return rteservice.New(rteservice.Config{Adapter: adapter, Calculator: calculator, Logger: logger})
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
