package output

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/benthosdev/benthos/v4/public/service"

	openmeter "github.com/openmeterio/openmeter/api/client/go"
)

const (
	urlField         = "url"
	tokenField       = "token"
	maxInFlightField = "max_in_flight"
	batchingField    = "batching"
)

func openmeterOutputConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		Beta().
		Categories("Services").
		Summary("Sends events the OpenMeter ingest API.").
		Description("").
		Fields(
			service.NewURLField(urlField).
				Description("OpenMeter API endpoint"),
			service.NewStringField(tokenField).
				Description("OpenMeter API token").
				Secret().
				Optional(),

			service.NewBatchPolicyField(batchingField),
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
			if maxInFlight, err = conf.FieldInt(maxInFlightField); err != nil {
				return
			}

			if batchPolicy, err = conf.FieldBatchPolicy(batchingField); err != nil {
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
	o := &openmeterOutput{}

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
	var events []any

	walkFn := func(_ int, msg *service.Message) error {
		if msg == nil {
			return errors.New("message is nil")
		}

		var e any
		e, err = msg.AsStructured()
		if err != nil {
			return fmt.Errorf("failed to convert message to structed data: %w", err)
		}
		events = append(events, e)

		return nil
	}
	if err = batch.WalkWithBatchedErrors(walkFn); err != nil {
		return fmt.Errorf("failed to process event: %w", err)
	}

	if len(events) == 0 {
		return errors.New("no valid messages found in batch")
	}

	var data any
	var contentType string
	if len(events) == 1 {
		contentType = "application/cloudevents+json"
		data = events[0]
	} else {
		contentType = "application/cloudevents-batch+json"
		data = events
	}

	var body bytes.Buffer
	err = json.NewEncoder(&body).Encode(data)
	if err != nil {
		return err
	}

	resp, err := o.client.IngestEventsWithBodyWithResponse(ctx, contentType, &body)
	if err != nil {
		return err
	}

	// TODO: improve error handling
	if resp.StatusCode() != http.StatusNoContent {
		if err = resp.ApplicationproblemJSON400; err != nil {
			return err
		} else if err = resp.ApplicationproblemJSONDefault; err != nil {
			return err
		} else {
			return fmt.Errorf("unknown error: %w", err)
		}
	}

	return nil
}

func (o *openmeterOutput) Close(_ context.Context) error {
	return nil
}
