package openmeter

import (
	"encoding/json"
	"fmt"
)

// APIError is returned for any non-2xx API response. It mirrors the API's
// RFC 7807-style problem body. When the body cannot be parsed as such, Title is
// left empty and RawBody carries the undecoded payload.
type APIError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int `json:"-"`

	// Status is the status code echoed in the problem body (usually equal to
	// StatusCode).
	Status int `json:"status"`
	// Title is a short, stable, human-readable summary of the problem.
	Title string `json:"title"`
	// Type is an optional machine-readable error type.
	Type string `json:"type,omitempty"`
	// Detail is a human-readable explanation specific to this occurrence.
	Detail string `json:"detail"`
	// Instance carries the correlation ID, formatted as kong:trace:<id>.
	Instance string `json:"instance"`

	// RawBody is the undecoded response body, always populated.
	RawBody []byte `json:"-"`
}

func newAPIError(statusCode int, body []byte) *APIError {
	e := &APIError{StatusCode: statusCode, RawBody: body}
	// Best-effort decode; a non-conforming body still yields a useful error via
	// StatusCode and RawBody.
	_ = json.Unmarshal(body, e)
	return e
}

func (e *APIError) Error() string {
	switch {
	case e.Title != "" && e.Detail != "":
		return fmt.Sprintf("openmeter: %d %s: %s", e.StatusCode, e.Title, e.Detail)
	case e.Title != "":
		return fmt.Sprintf("openmeter: %d %s", e.StatusCode, e.Title)
	default:
		return fmt.Sprintf("openmeter: unexpected status %d: %s", e.StatusCode, string(e.RawBody))
	}
}
