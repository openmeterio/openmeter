//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=codegen.yaml ../../openapi.cloud.yaml
package openmeter

import (
	"context"
	"fmt"
	"net/http"
)

func NewAuthClientWithResponses(server string, apiSecret string, opts ...ClientOption) (*ClientWithResponses, error) {
	o := []ClientOption{WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiSecret))
		return nil
	})}
	o = append(opts, o...)

	return NewClientWithResponses(server, o...)
}

func NewAuthClient(server string, apiSecret string, opts ...ClientOption) (*Client, error) {
	o := []ClientOption{WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiSecret))
		return nil
	})}
	o = append(opts, o...)

	return NewClient(server, o...)
}

// IngestEvents is a wrapper around generated client's IngestEventsWithApplicationCloudeventsPlusJSONBody
func (c *Client) IngestEvent(ctx context.Context, event Event, reqEditors ...RequestEditorFn) (*http.Response, error) {
	return c.IngestEventsWithApplicationCloudeventsPlusJSONBody(ctx, event, reqEditors...)
}

// IngestEvents is a wrapper around generated client's IngestEventsWithApplicationCloudeventsBatchPlusJSONBody
func (c *Client) IngestEventBatch(ctx context.Context, events []Event, reqEditors ...RequestEditorFn) (*http.Response, error) {
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
