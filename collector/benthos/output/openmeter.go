package output

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/redpanda-data/benthos/v4/public/bloblang"
	"github.com/redpanda-data/benthos/v4/public/service"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	openmeter "github.com/openmeterio/openmeter/api/client/go"
)

const (
	urlField         = "url"
	tokenField       = "token"
	maxInFlightField = "max_in_flight"
	batchingField    = "batching"

	tracingAttrsMapField = "tracing_attrs_map"
)

const (
	otelName = "benthos-openmeter-output"
)

const defaultTracingAttrsMapField = `
root = {}
root.openmeter.event = this.with("id", "source", "subject")
`

func openmeterOutputConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		Beta().
		Categories("Services").
		Summary("Sends events the OpenMeter ingest API.").
		Description("This output is used to send events to the OpenMeter ingest API.").
		Fields(
			service.NewURLField(urlField).
				Description("OpenMeter API endpoint"),
			service.NewStringField(tokenField).
				Description("OpenMeter API token").
				Secret().
				Optional(),

			service.NewBatchPolicyField(batchingField),
			service.NewOutputMaxInFlightField().Default(10),
			service.NewBloblangField(tracingAttrsMapField).
				Description("An optional Bloblang mapping that can be defined in order to set attributes on tracing Span.").
				Optional().
				Advanced().
				Default(defaultTracingAttrsMapField),
		)
}

func init() {
	err := service.RegisterBatchOutput("openmeter", openmeterOutputConfig(),
		func(conf *service.ParsedConfig, mgr *service.Resources) (
			output service.BatchOutput,
			batchPolicy service.BatchPolicy,
			maxInFlight int,
			err error,
		) {
			if maxInFlight, err = conf.FieldInt(maxInFlightField); err != nil {
				return output, batchPolicy, maxInFlight, err
			}

			if batchPolicy, err = conf.FieldBatchPolicy(batchingField); err != nil {
				return output, batchPolicy, maxInFlight, err
			}

			output, err = newOpenMeterOutput(conf, mgr)

			return output, batchPolicy, maxInFlight, err
		})
	if err != nil {
		panic(err)
	}
}

type openmeterOutput struct {
	client openmeter.ClientWithResponsesInterface

	tracingAttrsMap *bloblang.Executor

	logger *service.Logger
	tracer trace.Tracer
}

func newOpenMeterOutput(conf *service.ParsedConfig, mgr *service.Resources) (*openmeterOutput, error) {
	o := &openmeterOutput{
		logger: mgr.Logger(),
		tracer: mgr.OtelTracer().Tracer(otelName),
	}

	url, err := conf.FieldString(urlField)
	if err != nil {
		return nil, err
	}

	// TODO: custom HTTP client
	var client openmeter.ClientWithResponsesInterface

	if conf.Contains(tokenField) {
		token, err := conf.FieldString(tokenField)
		if err != nil {
			return nil, err
		}

		client, err = openmeter.NewAuthClientWithResponses(url, token)
		if err != nil {
			return nil, err
		}
	} else {
		var err error

		client, err = openmeter.NewClientWithResponses(url)
		if err != nil {
			return nil, err
		}
	}
	o.client = client

	if conf.Contains(tracingAttrsMapField) {
		if o.tracingAttrsMap, err = conf.FieldBloblang(tracingAttrsMapField); err != nil {
			return nil, err
		}
	}

	return o, nil
}

func (o *openmeterOutput) Connect(_ context.Context) error {
	return nil
}

// TODO: add schema validation
func (o *openmeterOutput) WriteBatch(ctx context.Context, batch service.MessageBatch) error {
	// if there is only one message use the single message endpoint
	// otherwise use the batch endpoint
	// if validation is enabled, try to parse the message as cloudevents first
	//

	var err error

	ctx, span := o.tracer.Start(ctx, "output_openmeter_write_batch")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "batch write was successful")
		}
		span.End()
	}()

	o.logger.Debugf("received message batch [size=%d]", len(batch))

	if len(batch) == 0 {
		return nil
	}

	var events []any

	walkFn := func(_ int, msg *service.Message) error {
		if msg == nil {
			o.logger.Error("received nil message in batch")

			err = errors.New("received nil message in batch")

			return err
		}

		var e any
		e, err = msg.AsStructured()
		if err != nil {
			err = fmt.Errorf("failed to convert message to structed data: %w", err)

			return err
		}

		events = append(events, e)

		o.UpdateMessageSpan(ctx, msg)

		return nil
	}
	if err = batch.WalkWithBatchedErrors(walkFn); err != nil {
		return fmt.Errorf("failed to process event: %w", err)
	}

	if len(events) == 0 {
		o.logger.Error("no valid messages found in batch")

		err = errors.New("no valid messages found in batch")

		return err
	}

	var body bytes.Buffer
	err = json.NewEncoder(&body).Encode(events)
	if err != nil {
		return err
	}

	resp, err := o.client.IngestEventsWithBodyWithResponse(ctx, "application/json", &body)
	if err != nil {
		return err
	}

	span.SetAttributes(attribute.Int("openmeter.http.status_code", resp.StatusCode()))

	if resp.StatusCode() >= 400 {
		if resp.ApplicationproblemJSON400 != nil {
			return resp.ApplicationproblemJSON400
		}

		if resp.ApplicationproblemJSON401 != nil {
			return resp.ApplicationproblemJSON401
		}

		if resp.ApplicationproblemJSON403 != nil {
			return resp.ApplicationproblemJSON403
		}

		if resp.ApplicationproblemJSON500 != nil {
			return resp.ApplicationproblemJSON500
		}

		if resp.ApplicationproblemJSONDefault != nil {
			return resp.ApplicationproblemJSONDefault
		}

		return errors.New(http.StatusText(resp.StatusCode()))
	}

	return nil
}

func (o *openmeterOutput) Close(_ context.Context) error {
	return nil
}

func (o *openmeterOutput) UpdateMessageSpan(ctx context.Context, msg *service.Message) {
	// Add reference of write_batch Span to message Span
	msgSpan := trace.SpanFromContext(msg.Context())
	if msgSpan == nil {
		o.logger.Debug("no span found for message")

		return
	}

	msgSpan.AddLink(trace.LinkFromContext(ctx))

	// Enrich message Span with additional tracing information extracted from message itself

	// Return early if mapping expression is not provided
	if o.tracingAttrsMap != nil {
		return
	}

	spanAttrsMsg, err := msg.BloblangQuery(o.tracingAttrsMap)
	if err != nil {
		o.logger.Debugf("failed to extract tracing attributes from message: %v", err)

		return
	}

	var spanAttrsVal any
	if spanAttrsMsg != nil {
		if spanAttrsVal, err = spanAttrsMsg.AsStructured(); err != nil {
			o.logger.Debugf("failed to construct structured tracing data from message: %v", err)

			return
		}
	}

	if spanAttrsVal == nil {
		return
	}

	spanAttrMap, ok := spanAttrsVal.(map[string]interface{})
	if !ok {
		o.logger.Debugf("tracing attributes mapping resulted in a non-object mapping: %T", spanAttrsVal)

		return
	}

	var spanAttrs []attribute.KeyValue
	for k, v := range spanAttrMap {
		attrs := toAttrs(k, v)
		spanAttrs = append(spanAttrs, attrs...)
	}

	msgSpan.SetAttributes(spanAttrs...)
}

func toAttrs(prefix string, v interface{}) []attribute.KeyValue {
	var attrs []attribute.KeyValue

	switch value := v.(type) {
	case map[string]interface{}:
		for k, v := range value {
			a := toAttrs(fmt.Sprintf("%s.%s", prefix, k), v)
			attrs = append(attrs, a...)
		}
	case string:
		attrs = append(attrs, attribute.String(prefix, value))
	case fmt.Stringer:
		attrs = append(attrs, attribute.Stringer(prefix, value))
	case int:
		attrs = append(attrs, attribute.Int(prefix, value))
	case int64:
		attrs = append(attrs, attribute.Int64(prefix, value))
	case float32:
		attrs = append(attrs, attribute.Float64(prefix, float64(value)))
	case float64:
		attrs = append(attrs, attribute.Float64(prefix, value))
	case bool:
		attrs = append(attrs, attribute.Bool(prefix, value))
	default:
	}

	return attrs
}
