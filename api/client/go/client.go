//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen --config=codegen.yaml ../../openapi.yaml
package openmeter

import (
	"context"
	"net/http"
)

// Sugar syntax top of generated client
func (c *Client) IngestEvent(ctx context.Context, event Event, reqEditors ...RequestEditorFn) (*http.Response, error) {
	return c.IngestEventsWithApplicationCloudeventsPlusJSONBody(ctx, event, reqEditors...)
}

func (c *Client) IngestEventBatchWithResponse(ctx context.Context, events []Event, reqEditors ...RequestEditorFn) (*http.Response, error) {
	return c.IngestEventsWithApplicationCloudeventsBatchPlusJSONBody(ctx, events, reqEditors...)
}

func (c *ClientWithResponses) IngestEventWithResponse(ctx context.Context, event Event, reqEditors ...RequestEditorFn) (*IngestEventsResponse, error) {
	return c.IngestEventsWithApplicationCloudeventsPlusJSONBodyWithResponse(ctx, event, reqEditors...)
}

func (c *ClientWithResponses) IngestEventBatchWithResponse(ctx context.Context, events []Event, reqEditors ...RequestEditorFn) (*IngestEventsResponse, error) {
	return c.IngestEventsWithApplicationCloudeventsBatchPlusJSONBodyWithResponse(ctx, events, reqEditors...)
}
