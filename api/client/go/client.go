//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen --config=codegen.yaml ../../openapi.yaml
package openmeter

import (
	"context"
	"net/http"
)

// IngestEvents is a wrapper around generated client's IngestEventsWithApplicationCloudeventsPlusJSONBody
func (c *Client) IngestEvent(ctx context.Context, event Event, reqEditors ...RequestEditorFn) (*http.Response, error) {
	return c.IngestEventsWithApplicationCloudeventsPlusJSONBody(ctx, event, reqEditors...)
}

// IngestEvents is a wrapper around generated client's IngestEventsWithApplicationCloudeventsBatchPlusJSONBody
func (c *Client) IngestEventBatchWithResponse(ctx context.Context, events []Event, reqEditors ...RequestEditorFn) (*http.Response, error) {
	return c.IngestEventsWithApplicationCloudeventsBatchPlusJSONBody(ctx, events, reqEditors...)
}

// IngestEventsWithResponse is a wrapper around generated client's IngestEventsWithApplicationCloudeventsPlusJSONBodyWithResponse
func (c *ClientWithResponses) IngestEventWithResponse(ctx context.Context, event Event, reqEditors ...RequestEditorFn) (*IngestEventsResponse, error) {
	return c.IngestEventsWithApplicationCloudeventsPlusJSONBodyWithResponse(ctx, event, reqEditors...)
}

// IngestEventsWithResponse is a wrapper around generated client's IngestEventsWithApplicationCloudeventsBatchPlusJSONBodyWithResponse
func (c *ClientWithResponses) IngestEventBatchWithResponse(ctx context.Context, events []Event, reqEditors ...RequestEditorFn) (*IngestEventsResponse, error) {
	return c.IngestEventsWithApplicationCloudeventsBatchPlusJSONBodyWithResponse(ctx, events, reqEditors...)
}
