//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-slog/otelslog"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

type Application struct {
	common.GlobalInitializer

	Metadata common.Metadata

	MeterRepository  meter.Repository
	TelemetryServer  common.TelemetryServer
	FlushHandler     flushhandler.FlushEventHandler
	TopicProvisioner pkgkafka.TopicProvisioner

	Meter  metric.Meter
	Tracer trace.Tracer
}

func initializeApplication(ctx context.Context, conf config.Configuration, logger *slog.Logger) (Application, func(), error) {
	wire.Build(
		metadata,
		common.Config,
		common.Framework,
		common.Telemetry,
		common.NewDefaultTextMapPropagator,
		common.KafkaTopic,
		common.SinkWorkerProvisionTopics,
		common.WatermillNoPublisher,
		common.NewSinkWorkerPublisher,
		common.OpenMeter,
		common.NewFlushHandler,
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

// TODO: is this necessary? Do we need a logger first?
func initializeLogger(conf config.Configuration) *slog.Logger {
	wire.Build(metadata, common.Config, common.Logger)

	return new(slog.Logger)
}

func metadata(conf config.Configuration) common.Metadata {
	return common.Metadata{
		ServiceName:       "openmeter",
		Version:           version,
		Environment:       conf.Environment,
		OpenTelemetryName: "openmeter.io/sink-worker",
	}
}

// TODO: use the primary logger
func NewLogger(conf config.Configuration, res *resource.Resource) *slog.Logger {
	logger := slog.New(otelslog.NewHandler(conf.Telemetry.Log.NewHandler(os.Stdout)))
	logger = otelslog.WithResource(logger, res)

	return logger
}
