//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen --config=codegen.yaml ../../openapi.yaml
package openmeter

import (
	"context"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
)

func (c *Client) IngestEvents(ctx context.Context, events ...event.Event) error {
	resp, err := c.IngestEventsWithApplicationCloudeventsBatchPlusJSONBody(ctx, nil, events, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode > 399 {
		return fmt.Errorf("error response from server: %d", resp.StatusCode)
	}

	return nil
}
