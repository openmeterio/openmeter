package commonhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/models"
)

// JSONResponseEncoder encodes a response as JSON.
func JSONResponseEncoder[Response any](_ context.Context, w http.ResponseWriter, response Response) error {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)

	if err := enc.Encode(response); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	_, err := w.Write(buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}

// ErrorEncoder encodes an error as HTTP 500 Internal Server Error.
func ErrorEncoder(ctx context.Context, _ error, w http.ResponseWriter) bool {
	models.NewStatusProblem(ctx, errors.New("something went wrong"), http.StatusInternalServerError).Respond(w, nil)

	return false
}
