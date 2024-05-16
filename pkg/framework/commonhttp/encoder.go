package commonhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

// JSONResponseEncoder encodes a response as JSON.
func JSONResponseEncoder[Response any](_ context.Context, w http.ResponseWriter, response Response) error {
	return jsonResponseEncoder(w, http.StatusOK, response)
}

// JSONResponseEncoder encodes a response as JSON.
func jsonResponseEncoder[Response any](w http.ResponseWriter, statusCode int, response Response) error {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)

	if err := enc.Encode(response); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_, err := w.Write(buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func JSONResponseEncoderWithStatus[Response any](statusCode int) httptransport.ResponseEncoder[Response] {
	return func(ctx context.Context, w http.ResponseWriter, r Response) error {
		return jsonResponseEncoder(w, statusCode, r)
	}
}

func EmptyResponseEncoder[Response any](statusCode int) httptransport.ResponseEncoder[Response] {
	return func(ctx context.Context, w http.ResponseWriter, r Response) error {
		w.WriteHeader(statusCode)
		return nil
	}
}

type ErrorWithHTTPStatusCode struct {
	error
	StatusCode int
}

func NewHTTPError(statusCode int, err error) ErrorWithHTTPStatusCode {
	return ErrorWithHTTPStatusCode{
		StatusCode: statusCode,
		error:      err,
	}
}

func (e ErrorWithHTTPStatusCode) EncodeError(ctx context.Context, w http.ResponseWriter) bool {
	models.NewStatusProblem(ctx, e.error, e.StatusCode).Respond(w)
	return true
}

// ErrorEncoder encodes an error as HTTP 500 Internal Server Error.
func ErrorEncoder(ctx context.Context, _ error, w http.ResponseWriter) bool {
	models.NewStatusProblem(ctx, errors.New("something went wrong"), http.StatusInternalServerError).Respond(w)

	return false
}
