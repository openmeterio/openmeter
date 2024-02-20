package input

import (
	"context"

	"github.com/benthosdev/benthos/v4/public/service"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	_ "go.opentelemetry.io/proto/otlp/logs/v1"
)

// TODO: add batching config and policy

func otelLogInputConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		Beta().
		Categories("Services").
		Summary("List objects in Kubernetes.").
		Fields(
			service.NewObjectField(
				"resource",
				service.NewStringField("group").
					Description("Kubernetes API group.").
					Optional(),
				service.NewStringField("version").
					Description("Kubernetes API group version.").
					Example("v1"),
				service.NewStringField("name").
					Description("Kubernetes API resource name.").
					Example("pods"),
			).
				Description("Kubernetes resource details.").
				Advanced(),
			service.NewStringListField("namespaces").
				Description("List of namespaces to list objects from."),
			service.NewStringField("label_selector").
				Description("Label selector applied to each list operation.").
				Optional(),
		)
}

func init() {
	err := service.RegisterBatchInput("otel_log", otelLogInputConfig(), func(conf *service.ParsedConfig, mgr *service.Resources) (service.BatchInput, error) {
		return newOtelLogInput(conf)
	})
	if err != nil {
		panic(err)
	}
}

type otelLogInput struct{}

func newOtelLogInput(conf *service.ParsedConfig) (*otelLogInput, error) {
	return &otelLogInput{}, nil
}

func (in *otelLogInput) Connect(_ context.Context) error {
	return nil
}

func (in *otelLogInput) ReadBatch(ctx context.Context) (service.MessageBatch, service.AckFunc, error) {
	return nil, func(context.Context, error) error { return nil }, nil
}

func (in *otelLogInput) Close(_ context.Context) error {
	return nil
}

type OtelLogsServiceServer struct {
	collogspb.UnimplementedLogsServiceServer
}

func (OtelLogsServiceServer) Export(ctx context.Context, request *collogspb.ExportLogsServiceRequest) (*collogspb.ExportLogsServiceResponse, error) {
	resourceLogs := request.GetResourceLogs()
	return nil, nil
}

type transaction struct {
	msg *collogspb.ExportLogsServiceRequest
	res chan error
}
