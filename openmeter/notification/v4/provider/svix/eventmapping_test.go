package svix

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	svixmodels "github.com/svix/svix-webhooks/v2/go/models"
)

func TestFromAPIMessageAttemptOutResponseLimit(t *testing.T) {
	const responseLimit = 64 * 1024
	const endpointURL = "https://example.com/webhook"

	// Account for the complete stored envelope, not just the unescaped body.
	emptyResponse, err := json.Marshal(map[string]any{
		"status_code":        200,
		"body":               "",
		"duration_ms":        12,
		"url":                endpointURL,
		"response_truncated": false,
	})
	require.NoError(t, err)

	bodyCapacity := responseLimit - len(emptyResponse)
	quotedBody := strings.Repeat(`"`, bodyCapacity/2) + strings.Repeat("a", bodyCapacity%2)
	controlBody := strings.Repeat("\x00", bodyCapacity/6) + strings.Repeat("a", bodyCapacity%6)

	tests := []struct {
		name          string
		body          string
		url           string
		encodedSize   int
		wantTruncated bool
		wantMinimal   bool
	}{
		{
			name:        "one byte below limit",
			body:        strings.Repeat("a", bodyCapacity-1),
			encodedSize: responseLimit - 1,
		},
		{
			name:        "exactly at limit",
			body:        strings.Repeat("a", bodyCapacity),
			encodedSize: responseLimit,
		},
		{
			name:          "one byte above limit",
			body:          strings.Repeat("a", bodyCapacity+1),
			encodedSize:   responseLimit + 1,
			wantTruncated: true,
		},
		{
			name:        "escaped quotes exactly at limit",
			body:        quotedBody,
			encodedSize: responseLimit,
		},
		{
			name:          "escaped quotes exceed limit",
			body:          quotedBody + `"`,
			encodedSize:   responseLimit + 2,
			wantTruncated: true,
		},
		{
			name:        "escaped control characters exactly at limit",
			body:        controlBody,
			encodedSize: responseLimit,
		},
		{
			name:          "escaped control characters exceed limit",
			body:          controlBody + "\x00",
			encodedSize:   responseLimit + 6,
			wantTruncated: true,
		},
		{
			name:          "oversized URL requires minimal fallback after dropping body",
			body:          "response body",
			url:           endpointURL + strings.Repeat("a", responseLimit),
			wantTruncated: true,
			wantMinimal:   true,
		},
		{
			name:          "oversized URL requires minimal fallback with empty body",
			url:           endpointURL + strings.Repeat("a", responseLimit),
			wantTruncated: true,
			wantMinimal:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a response whose encoded size exercises the selected boundary.
			attempt := svixmodels.MessageAttemptOut{
				Status:             svixmodels.MESSAGESTATUS_SUCCESS,
				ResponseStatusCode: 200,
				ResponseDurationMs: 12,
				Response:           tt.body,
				Url:                endpointURL,
			}
			if tt.url != "" {
				attempt.Url = tt.url
			}

			response := map[string]any{
				"status_code":        attempt.ResponseStatusCode,
				"body":               attempt.Response,
				"duration_ms":        attempt.ResponseDurationMs,
				"url":                attempt.Url,
				"response_truncated": false,
			}
			encoded, err := json.Marshal(response)
			require.NoError(t, err)

			if tt.encodedSize > 0 {
				require.Len(t, encoded, tt.encodedSize)
			}

			require.Equal(t, tt.wantTruncated, len(encoded) > responseLimit)

			// When mapping the provider response for persistence.
			mapped, err := fromAPIMessageAttemptOut("test-namespace", attempt)
			require.NoError(t, err)
			require.NotNil(t, mapped)

			// Then the result is bounded JSON and marks truncation only when content is dropped.
			require.True(t, json.Valid(mapped.Response))
			require.LessOrEqual(t, len(mapped.Response), responseLimit)

			switch {
			case tt.wantMinimal:
				response = map[string]any{"response_truncated": true}
			case tt.wantTruncated:
				response["body"] = ""
				response["response_truncated"] = true
			default:
				require.Len(t, mapped.Response, len(encoded))
			}

			expected, err := json.Marshal(response)
			require.NoError(t, err)
			require.JSONEq(t, string(expected), string(mapped.Response))
		})
	}
}
