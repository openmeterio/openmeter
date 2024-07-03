package commonhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
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

// PlainTextResponseEncoder encodes a response as PlainText.
func PlainTextResponseEncoder[Response string](_ context.Context, w http.ResponseWriter, response Response) error {
	return plainTextResponseEncoder(w, http.StatusOK, response)
}

// PlainTextResponseEncoder encodes a response as PlainText.
func plainTextResponseEncoder[Response string](w http.ResponseWriter, statusCode int, response Response) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(statusCode)

	_, err := w.Write([]byte(response))
	if err != nil {
		return err
	}

	return nil
}

func EmptyResponseEncoder[Response any](statusCode int) httptransport.ResponseEncoder[Response] {
	return func(ctx context.Context, w http.ResponseWriter, r Response) error {
		w.WriteHeader(statusCode)
		return nil
	}
}
