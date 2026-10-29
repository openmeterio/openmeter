package testutils_test

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/notification/v4/provider/svix/testutils"
)

func TestWebhookServer(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
		wantAllow  string
	}{
		{
			name:       "preserves raw JSON",
			method:     http.MethodPost,
			body:       "{\n  \"type\": \"invoice.created\"\n}\n",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "does not validate payload",
			method:     http.MethodPost,
			body:       "not JSON",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "accepts empty body",
			method:     http.MethodPost,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "accepts body at limit",
			method:     http.MethodPost,
			body:       strings.Repeat("a", 1<<20),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "rejects oversized body",
			method:     http.MethodPost,
			body:       strings.Repeat("a", (1<<20)+1),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "rejects GET",
			method:     http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a receiver with no captured deliveries.
			server := testutils.NewWebhookServer(t, "127.0.0.1")
			require.Empty(t, server.Requests())

			client := &http.Client{Timeout: 5 * time.Second}
			t.Cleanup(client.CloseIdleConnections)

			request, err := http.NewRequestWithContext(
				t.Context(), tt.method, server.URL, strings.NewReader(tt.body),
			)
			require.NoError(t, err)

			request.Header["Svix-Signature"] = []string{"v1,first", "v1,second"}

			// When a delivery reaches the receiver.
			response, err := client.Do(request)
			require.NoError(t, err)
			defer response.Body.Close()

			_, err = io.Copy(io.Discard, response.Body)
			require.NoError(t, err)

			// Then only accepted deliveries are captured, without transforming them.
			require.Equal(t, tt.wantStatus, response.StatusCode)
			require.Equal(t, tt.wantAllow, response.Header.Get("Allow"))

			requests := server.Requests()

			if tt.wantStatus != http.StatusNoContent {
				require.Empty(t, requests)

				return
			}

			require.Len(t, requests, 1)
			require.Equal(t, tt.body, string(requests[0].Body))
			require.Equal(t, []string{"v1,first", "v1,second"}, requests[0].Header.Values("Svix-Signature"))

			// Mutating a snapshot must not corrupt subsequent assertions.
			requests[0].Header["Svix-Signature"][0] = "changed"
			requests[0].Header.Set("Extra", "changed")

			if len(requests[0].Body) > 0 {
				requests[0].Body[0] = '!'
			}

			snapshot := server.Requests()
			require.Equal(t, tt.body, string(snapshot[0].Body))
			require.Equal(t, []string{"v1,first", "v1,second"}, snapshot[0].Header.Values("Svix-Signature"))
			require.Empty(t, snapshot[0].Header.Get("Extra"))
		})
	}
}

func TestWebhookServerSetResponse(t *testing.T) {
	type step struct {
		status     int
		body       string
		wantError  string
		wantStatus int
		wantBody   string
	}

	tests := []struct {
		name  string
		steps []step
	}{
		{
			name: "rejection then recovery",
			steps: []step{
				{
					status:     http.StatusServiceUnavailable,
					body:       "temporarily unavailable",
					wantStatus: http.StatusServiceUnavailable,
					wantBody:   "temporarily unavailable",
				},
				{
					status:     http.StatusNoContent,
					wantStatus: http.StatusNoContent,
				},
			},
		},
		{
			name: "invalid status preserves response",
			steps: []step{
				{
					status:     http.StatusServiceUnavailable,
					body:       "unavailable",
					wantStatus: http.StatusServiceUnavailable,
					wantBody:   "unavailable",
				},
				{
					status:     199,
					wantError:  "invalid response status code",
					wantStatus: http.StatusServiceUnavailable,
					wantBody:   "unavailable",
				},
				{
					status:     600,
					wantError:  "invalid response status code",
					wantStatus: http.StatusServiceUnavailable,
					wantBody:   "unavailable",
				},
			},
		},
		{
			name: "invalid body preserves response",
			steps: []step{
				{
					status:     http.StatusNoContent,
					body:       "not allowed",
					wantError:  "invalid response body",
					wantStatus: http.StatusNoContent,
				},
				{
					status:     http.StatusNotModified,
					body:       "not allowed",
					wantError:  "invalid response body",
					wantStatus: http.StatusNoContent,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a receiver whose response can change without losing request history.
			server := testutils.NewWebhookServer(t, "127.0.0.1")
			client := &http.Client{Timeout: 5 * time.Second}
			t.Cleanup(client.CloseIdleConnections)

			for i, step := range tt.steps {
				// When configuring the next delivery's response.
				err := server.SetResponse(step.status, step.body)
				if step.wantError != "" {
					require.ErrorContains(t, err, step.wantError)
				} else {
					require.NoError(t, err)
				}

				requestBody := fmt.Sprint(i)
				request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL, strings.NewReader(requestBody))
				require.NoError(t, err)

				response, err := client.Do(request)
				require.NoError(t, err)

				body, readErr := io.ReadAll(response.Body)
				closeErr := response.Body.Close()
				require.NoError(t, readErr)
				require.NoError(t, closeErr)

				// Then rejection is captured too, and invalid updates leave the response intact.
				require.Equal(t, step.wantStatus, response.StatusCode)
				require.Equal(t, step.wantBody, string(body))
				require.Len(t, server.Requests(), i+1)
				require.Equal(t, requestBody, string(server.Requests()[i].Body))
			}
		})
	}
}

func TestWebhookServerConcurrentRequests(t *testing.T) {
	tests := []struct {
		name           string
		count          int
		changeResponse bool
	}{
		{
			name:  "single delivery",
			count: 1,
		},
		{
			name:  "concurrent deliveries",
			count: 32,
		},
		{
			name:           "concurrent deliveries and response updates",
			count:          32,
			changeResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a receiver shared by concurrent deliveries and readers.
			server := testutils.NewWebhookServer(t, "127.0.0.1")

			client := &http.Client{Timeout: 5 * time.Second}
			t.Cleanup(client.CloseIdleConnections)

			results := make(chan error, tt.count)
			var workers sync.WaitGroup

			// When requests arrive while snapshots are read and modified.
			for i := range tt.count {
				workers.Go(func() {
					if tt.changeResponse {
						status, body := http.StatusNoContent, ""
						if i%2 == 0 {
							status, body = http.StatusServiceUnavailable, "unavailable"
						}

						if err := server.SetResponse(status, body); err != nil {
							results <- err

							return
						}
					}

					request, err := http.NewRequestWithContext(
						t.Context(), http.MethodPost, server.URL, strings.NewReader(fmt.Sprint(i)),
					)
					if err != nil {
						results <- err

						return
					}

					response, err := client.Do(request)
					if err != nil {
						results <- err

						return
					}
					defer response.Body.Close()

					body, err := io.ReadAll(response.Body)
					if err == nil {
						switch {
						case response.StatusCode == http.StatusNoContent && len(body) == 0:
						case tt.changeResponse && response.StatusCode == http.StatusServiceUnavailable && string(body) == "unavailable":
						default:
							err = fmt.Errorf("unexpected response: status=%d body=%q", response.StatusCode, body)
						}
					}

					for _, captured := range server.Requests() {
						captured.Body[0] = '!'
						captured.Header.Set("Extra", "changed")
					}

					results <- err
				})
			}

			workers.Wait()
			close(results)

			// Then all deliveries remain intact, regardless of arrival order.
			for err := range results {
				require.NoError(t, err)
			}

			requests := server.Requests()
			require.Len(t, requests, tt.count)

			bodies := make([]string, 0, tt.count)
			expected := make([]string, 0, tt.count)

			for i, request := range requests {
				bodies = append(bodies, string(request.Body))
				expected = append(expected, fmt.Sprint(i))
				require.Empty(t, request.Header.Get("Extra"))
			}

			require.ElementsMatch(t, expected, bodies)
		})
	}
}
