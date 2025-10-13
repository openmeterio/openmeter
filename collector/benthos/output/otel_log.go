package output

import (
	"context"

	"github.com/redpanda-data/benthos/v4/public/service"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

func otelLogOutputConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		Beta().
		Categories("Services").
		Summary("Export logs to an OTLP log collector service.").
		Description("").
		Fields(
			service.NewStringField("address").
				Description("OTLP gRPC endpoint"),

			service.NewBatchPolicyField("batching"),
			service.NewOutputMaxInFlightField().Default(10),
		)
}

func init() {
	err := service.RegisterBatchOutput("otel_log", otelLogOutputConfig(),
		func(conf *service.ParsedConfig, mgr *service.Resources) (
			output service.BatchOutput,
			batchPolicy service.BatchPolicy,
			maxInFlight int,
			err error,
		) {
			if maxInFlight, err = conf.FieldInt("max_in_flight"); err != nil {
				return output, batchPolicy, maxInFlight, err
			}

			if batchPolicy, err = conf.FieldBatchPolicy("batching"); err != nil {
				return output, batchPolicy, maxInFlight, err
			}

			output, err = newOtelLogOutput(conf)

			return output, batchPolicy, maxInFlight, err
		})
	if err != nil {
		panic(err)
	}
}

type otelLogOutput struct {
	address string

	conn   *grpc.ClientConn
	client collogspb.LogsServiceClient
}

func newOtelLogOutput(conf *service.ParsedConfig) (*otelLogOutput, error) {
	address, err := conf.FieldString("address")
	if err != nil {
		return nil, err
	}

	return &otelLogOutput{
		address: address,
	}, nil
}

func (out *otelLogOutput) Connect(ctx context.Context) error {
	if out.conn != nil {
		out.conn.Close()
		out.conn = nil
	}

	conn, err := grpc.NewClient(out.address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	out.conn = conn
	out.client = collogspb.NewLogsServiceClient(conn)

	return nil
}

func (out *otelLogOutput) WriteBatch(ctx context.Context, batch service.MessageBatch) error {
	var resourceLogs []*logspb.ResourceLogs

	for _, msg := range batch {
		var resourceLog logspb.ResourceLogs

		b, err := msg.AsBytes()
		if err != nil {
			return err
		}

		err = protojson.Unmarshal(b, &resourceLog)
		if err != nil {
			return err
		}

		resourceLogs = append(resourceLogs, &resourceLog)
	}

	req := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: resourceLogs,
	}

	_, err := out.client.Export(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

func (out *otelLogOutput) Close(_ context.Context) error {
	out.conn.Close()

	return nil
}
