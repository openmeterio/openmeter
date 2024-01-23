package output

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/benthosdev/benthos/v4/public/service"

	openmeter "github.com/openmeterio/openmeter/api/client/go"
)

func openmeterOutputConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		Beta().
		Categories("Services").
		Summary("Sends events the OpenMeter ingest API.").
		Description("").
		Fields(
			service.NewURLField("url").
				Description("OpenMeter API endpoint"),
			service.NewStringField("token").
				Description("OpenMeter API token").
				Secret().
				Optional(),

			service.NewBatchPolicyField("batching"),
			service.NewOutputMaxInFlightField().Default(10),
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
			if maxInFlight, err = conf.FieldInt("max_in_flight"); err != nil {
				return
			}

			if batchPolicy, err = conf.FieldBatchPolicy("batching"); err != nil {
				return
			}

			output, err = newOpenMeterOutput(conf)

			return
		})
	if err != nil {
		panic(err)
	}
}

type openmeterOutput struct {
	client openmeter.ClientWithResponsesInterface
}

func newOpenMeterOutput(conf *service.ParsedConfig) (*openmeterOutput, error) {
	url, err := conf.FieldString("url")
	if err != nil {
		return nil, err
	}

	// TODO: custom HTTP client
	var client openmeter.ClientWithResponsesInterface

	if conf.Contains("token") {
		token, err := conf.FieldString("token")
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

	return &openmeterOutput{
		client: client,
	}, nil
}

func (out *openmeterOutput) Connect(_ context.Context) error {
	return nil
}

// TODO: add schema validation
func (out *openmeterOutput) WriteBatch(ctx context.Context, batch service.MessageBatch) error {
	// if there is only one message use the single message endpoint
	// otherwise use the batch endpoint
	// if validation is enabled, try to parse the message as cloudevents first
	//

	var contentType string
	var body io.Reader

	// No need to send a batch if there is only one message
	if len(batch) == 1 {
		contentType = "application/cloudevents+json"

		b, err := batch[0].AsBytes()
		if err != nil {
			return err
		}

		body = bytes.NewReader(b)
	} else {
		contentType = "application/cloudevents-batch+json"

		events := make([]any, 0, len(batch))

		err := batch.WalkWithBatchedErrors(func(_ int, msg *service.Message) error {
			e, err := msg.AsStructured()
			if err != nil {
				return err
			}

			events = append(events, e)

			return nil
		})
		if err != nil {
			return err
		}

		var b bytes.Buffer

		err = json.NewEncoder(&b).Encode(events)
		if err != nil {
			return err
		}

		body = &b
	}

	resp, err := out.client.IngestEventsWithBodyWithResponse(ctx, contentType, body)
	if err != nil {
		return err
	}

	// TODO: improve error handling
	if resp.StatusCode() != http.StatusNoContent {
		if err := resp.ApplicationproblemJSON400; err != nil {
			return err
		} else if err := resp.ApplicationproblemJSONDefault; err != nil {
			return err
		}

		return errors.New("unknown error")
	}

	return nil
}

func (out *openmeterOutput) Close(_ context.Context) error {
	return nil
}
