package input

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/samber/lo"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/openmeterio/openmeter/collector/benthos/internal/message"
	"github.com/openmeterio/openmeter/collector/benthos/internal/shutdown"
)

// TODO: add batching config and policy

func otelLogInputConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		Beta().
		Categories("Services").
		Summary("Collect logs usgin the OTLP gRPC protocol.").
		Fields(
			service.NewStringField("address").
				Description("OTLP gRPC endpoint"),
			service.NewDurationField("timeout").
				Description("Timeout for requests. If a consumed messages takes longer than this to be delivered the connection is closed, but the message may still be delivered.").
				Default("5s"),
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

type otelLogInput struct {
	collogspb.UnimplementedLogsServiceServer

	address string
	server  *grpc.Server

	timeout time.Duration

	handlerWG    sync.WaitGroup
	transactions chan message.Transaction

	shutSig *shutdown.Signaller
}

func newOtelLogInput(conf *service.ParsedConfig) (*otelLogInput, error) {
	server := grpc.NewServer() // nosemgrep

	address, err := conf.FieldString("address")
	if err != nil {
		return nil, err
	}

	timeout, err := conf.FieldDuration("timeout")
	if err != nil {
		return nil, err
	}

	in := &otelLogInput{
		address: address,
		server:  server,

		timeout: timeout,

		transactions: make(chan message.Transaction),

		shutSig: shutdown.NewSignaller(),
	}

	collogspb.RegisterLogsServiceServer(server, in)

	return in, nil
}

func (in *otelLogInput) Connect(_ context.Context) error {
	ln, err := net.Listen("tcp", in.address)
	if err != nil {
		return err
	}

	// TODO: log listening

	go in.loop(ln)

	return nil
}

func (in *otelLogInput) ReadBatch(ctx context.Context) (service.MessageBatch, service.AckFunc, error) {
	select {
	case b, open := <-in.transactions:
		if open {
			return b.Payload, b.Ack, nil
		}
		return nil, nil, nil
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}

func (in *otelLogInput) Close(_ context.Context) error {
	in.server.Stop()

	return nil
}

func (in *otelLogInput) Export(ctx context.Context, request *collogspb.ExportLogsServiceRequest) (*collogspb.ExportLogsServiceResponse, error) {
	if in.shutSig.ShouldCloseAtLeisure() {
		return nil, status.Error(codes.Unavailable, "server closing")
	}

	in.handlerWG.Add(1)
	defer in.handlerWG.Done()

	// TODO: add rate limit

	msg, err := in.extractMessageFromRequest(request)
	if err != nil {
		// h.log.Warn("Request read failed: %v\n", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	resChan := make(chan error, 1)
	select {
	case in.transactions <- message.NewTransaction(msg, resChan):
	case <-time.After(in.timeout):
		return nil, status.Error(codes.DeadlineExceeded, "request timed out")
	case <-ctx.Done():
		return nil, status.Error(codes.DeadlineExceeded, "request timed out")
	case <-in.shutSig.CloseAtLeisureChan():
		return nil, status.Error(codes.Unavailable, "server closing")
	}

	select {
	case res, open := <-resChan:
		if !open {
			return nil, status.Error(codes.Unavailable, "server closing")
		} else if res != nil {
			var berr *service.BatchError
			if errors.As(res, &berr) && len(msg) > berr.IndexedErrors() {
				return &collogspb.ExportLogsServiceResponse{
					PartialSuccess: &collogspb.ExportLogsPartialSuccess{
						RejectedLogRecords: int64(berr.IndexedErrors()),
						ErrorMessage:       berr.Error(),
					},
				}, nil
			}

			return nil, status.Error(codes.Internal, berr.Error())
		}
	case <-time.After(in.timeout):
		return nil, status.Error(codes.DeadlineExceeded, "request timed out")
	case <-ctx.Done():
		return nil, status.Error(codes.DeadlineExceeded, "request timed out")
	case <-in.shutSig.CloseNowChan():
		return nil, status.Error(codes.Unavailable, "server closing")
	}

	return &collogspb.ExportLogsServiceResponse{}, nil
}

type record struct {
	Resource *resource  `json:"resource,omitempty"`
	Scope    *scope     `json:"scope,omitempty"`
	Record   *logRecord `json:"record,omitempty"`
}

type resource struct {
	Attributes             map[string]any `json:"attributes,omitempty"`
	DroppedAttributesCount uint32         `json:"dropped_attributes_count,omitempty"`
}

func resourceFrom(pb *resourcepb.Resource) *resource {
	if pb == nil {
		return nil
	}

	return &resource{
		Attributes: lo.SliceToMap(pb.GetAttributes(), func(item *commonpb.KeyValue) (string, any) {
			return item.GetKey(), anyFrom(item.GetValue())
		}),
		DroppedAttributesCount: pb.GetDroppedAttributesCount(),
	}
}

func anyFrom(pb *commonpb.AnyValue) any {
	if pb == nil {
		return nil
	}

	switch pb.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		return pb.GetStringValue()
	case *commonpb.AnyValue_BoolValue:
		return pb.GetBoolValue()
	case *commonpb.AnyValue_IntValue:
		return pb.GetIntValue()
	case *commonpb.AnyValue_DoubleValue:
		return pb.GetDoubleValue()
	case *commonpb.AnyValue_ArrayValue:
		return lo.Map(pb.GetArrayValue().GetValues(), func(v *commonpb.AnyValue, _ int) any {
			return anyFrom(v)
		})
	case *commonpb.AnyValue_KvlistValue:
		return lo.SliceToMap(pb.GetKvlistValue().GetValues(), func(item *commonpb.KeyValue) (string, any) {
			return item.GetKey(), anyFrom(item.GetValue())
		})
	case *commonpb.AnyValue_BytesValue:
		return pb.GetBytesValue()
	}

	return nil
}

type scope struct {
	Name                   string         `json:"name,omitempty"`
	Version                string         `json:"version,omitempty"`
	Attributes             map[string]any `json:"attributes,omitempty"`
	DroppedAttributesCount uint32         `json:"dropped_attributes_count,omitempty"`
}

func scopeFrom(pb *commonpb.InstrumentationScope) *scope {
	if pb == nil {
		return nil
	}

	return &scope{
		Name:    pb.GetName(),
		Version: pb.GetVersion(),
		Attributes: lo.SliceToMap(pb.GetAttributes(), func(item *commonpb.KeyValue) (string, any) {
			return item.GetKey(), anyFrom(item.GetValue())
		}),
		DroppedAttributesCount: pb.GetDroppedAttributesCount(),
	}
}

type logRecord struct {
	TimeUnixNano           uint64         `json:"time_unix_nano,omitempty"`
	ObservedTimeUnixNano   uint64         `json:"observed_time_unix_nano,omitempty"`
	SeverityNumber         int32          `json:"severity_number,omitempty"`
	SeverityText           string         `json:"severity_text,omitempty"`
	Body                   any            `json:"body,omitempty"`
	Attributes             map[string]any `json:"attributes,omitempty"`
	DroppedAttributesCount uint32         `json:"dropped_attributes_count,omitempty"`
	Flags                  uint32         `json:"flags,omitempty"`
	TraceId                []byte         `json:"trace_id,omitempty"`
	SpanId                 []byte         `json:"span_id,omitempty"`
}

func logRecordFrom(pb *logspb.LogRecord) *logRecord {
	if pb == nil {
		return nil
	}

	return &logRecord{
		TimeUnixNano:         pb.GetTimeUnixNano(),
		ObservedTimeUnixNano: pb.GetObservedTimeUnixNano(),
		SeverityNumber:       int32(pb.GetSeverityNumber()),
		SeverityText:         pb.GetSeverityText(),
		Body:                 anyFrom(pb.GetBody()),
		Attributes: lo.SliceToMap(pb.GetAttributes(), func(item *commonpb.KeyValue) (string, any) {
			return item.GetKey(), anyFrom(item.GetValue())
		}),
		DroppedAttributesCount: pb.GetDroppedAttributesCount(),
		Flags:                  pb.GetFlags(),
		TraceId:                pb.GetTraceId(),
		SpanId:                 pb.GetSpanId(),
	}
}

func (in *otelLogInput) extractMessageFromRequest(request *collogspb.ExportLogsServiceRequest) (service.MessageBatch, error) {
	var batch service.MessageBatch

	// TODO: improve message decoding
	for _, resourceLog := range request.GetResourceLogs() {
		r := resourceFrom(resourceLog.GetResource())

		for _, scopeLog := range resourceLog.GetScopeLogs() {
			s := scopeFrom(scopeLog.GetScope())

			for _, logRecord := range scopeLog.GetLogRecords() {
				record := record{
					Resource: r,
					Scope:    s,
					Record:   logRecordFrom(logRecord),
				}
				recordByte, err := json.Marshal(record)
				// recordByte, err := protojson.Marshal(record)
				if err != nil {
					return nil, err
				}
				msg := service.NewMessage(recordByte)
				batch = append(batch, msg)
			}
		}
	}

	return batch, nil
}

func (in *otelLogInput) loop(ln net.Listener) {
	defer func() {
		in.server.GracefulStop()

		in.handlerWG.Wait()

		close(in.transactions)
		in.shutSig.ShutdownComplete()
	}()

	go func() {
		// TODO: add TLS support
		if err := in.server.Serve(ln); err != nil {
			_ = err
			// in.log.Error("Server error: %v\n", err)
		}
	}()

	<-in.shutSig.CloseAtLeisureChan()
}
