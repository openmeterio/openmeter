//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-slog/otelslog"
	"github.com/google/wire"
	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/common"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

type Application struct {
	common.GlobalInitializer

	FlushHandler     flushhandler.FlushEventHandler
	Logger           *slog.Logger
	Metadata         common.Metadata
	Meter            metric.Meter
	MeterService     meter.Service
	Streaming        streaming.Connector
	TelemetryServer  common.TelemetryServer
	TopicProvisioner pkgkafka.TopicProvisioner
	TopicResolver    *topicresolver.NamespacedTopicResolver
	Tracer           trace.Tracer
}

func initializeApplication(ctx context.Context, conf config.Configuration) (Application, func(), error) {
	wire.Build(
		wire.FieldsOf(new(config.Configuration), "Sink"),

		metadata,
		common.ClickHouse,
		common.Config,
		common.Framework,
		common.KafkaTopic,
		common.KafkaNamespaceResolver,
		common.MeterInMemory,
		common.NewDefaultTextMapPropagator,
		common.NewFlushHandler,
		common.NewSinkWorkerPublisher,
		common.SinkWorkerProvisionTopics,
		common.Streaming,
		common.Telemetry,
		common.WatermillNoPublisher,
		wire.Struct(new(Application), "*"),
	)
	return Application{}, nil, nil
}

func metadata(conf config.Configuration) common.Metadata {
	return common.NewMetadata(conf, version, "sink-worker")
}

// TODO: use the primary logger
func NewLogger(conf config.LogTelemetryConfig, res *resource.Resource) *slog.Logger {
	return slog.New(slogmulti.Pipe(
		otelslog.ResourceMiddleware(res),
		otelslog.NewHandler,
	).Handler(conf.NewHandler(os.Stdout)))
}
