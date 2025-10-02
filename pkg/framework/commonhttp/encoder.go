package commonhttp

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"

	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
	"github.com/openmeterio/openmeter/pkg/models"
)

// GetMediaType returns the media type of the request.
// If the media type is invalid, it defaults to JSON.
func GetMediaType(r *http.Request) (string, error) {
	var err error

	// Parse media type
	accept := r.Header.Get("Accept")
	if accept == "" {
		accept = "application/json"
	}

	mediatype, _, err := mime.ParseMediaType(accept)
	// Browser can send back media type Go marks as invalid
	// If that happens, default to JSON
	if err != nil {
		err = fmt.Errorf("invalid media type, default to json: %w", err)
		mediatype = "application/json"
	}

	return mediatype, err
}

// JSONResponseEncoder encodes a response as JSON.
func JSONResponseEncoder[Response any](_ context.Context, w http.ResponseWriter, _ *http.Request, response Response) error {
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

func JSONResponseEncoderWithStatus[Response any](statusCode int) encoder.ResponseEncoder[Response] {
	return func(ctx context.Context, w http.ResponseWriter, _ *http.Request, response Response) error {
		return jsonResponseEncoder(w, statusCode, response)
	}
}

// PlainTextResponseEncoder encodes a response as PlainText.
func PlainTextResponseEncoder[Response string](_ context.Context, w http.ResponseWriter, _ *http.Request, response Response) error {
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

// CSVResponse is a response that can be encoded as CSV.
type CSVResponse interface {
	FileName() string
	Records() [][]string
}

// CSVResponseEncoder encodes a response as CSV.
func CSVResponseEncoder[Response CSVResponse](_ context.Context, w http.ResponseWriter, _ *http.Request, response Response) error {
	return csvResponseEncoder(w, http.StatusOK, response)
}

// CSVResponseEncoder encodes a response as CSV.
func csvResponseEncoder[Response CSVResponse](w http.ResponseWriter, statusCode int, response Response) error {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", response.FileName()))
	w.WriteHeader(statusCode)

	// Write response
	writer := csv.NewWriter(w)
	err := writer.WriteAll(response.Records())
	if err != nil {
		return fmt.Errorf("writing record to csv: %w", err)
	}

	if err := writer.Error(); err != nil {
		return fmt.Errorf("writing csv: %w", err)
	}

	return nil
}

func EmptyResponseEncoder[Response any](statusCode int) encoder.ResponseEncoder[Response] {
	return func(_ context.Context, w http.ResponseWriter, _ *http.Request, resp Response) error {
		w.WriteHeader(statusCode)
		return nil
	}
}

// DummyErrorEncoder is a dummy error encoder that always returns a 400 status code with the received error.
func DummyErrorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, _ *http.Request) bool {
		NewHTTPError(http.StatusBadRequest, err).EncodeError(ctx, w)
		return true
	}
}

// GenericErrorEncoder is an error encoder that encodes the error as a generic error.
func GenericErrorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return HandleIssueIfHTTPStatusKnown(ctx, err, w) ||
			HandleErrorIfTypeMatches[*models.GenericConflictError](ctx, http.StatusConflict, err, w) ||
			HandleErrorIfTypeMatches[*models.GenericForbiddenError](ctx, http.StatusForbidden, err, w) ||
			HandleErrorIfTypeMatches[*models.GenericNotImplementedError](ctx, http.StatusNotImplemented, err, w) ||
			HandleErrorIfTypeMatches[*models.GenericValidationError](ctx, http.StatusBadRequest, err, w) ||
			HandleErrorIfTypeMatches[*models.GenericNotFoundError](ctx, http.StatusNotFound, err, w) ||
			HandleErrorIfTypeMatches[*models.GenericUnauthorizedError](ctx, http.StatusUnauthorized, err, w) ||
			HandleErrorIfTypeMatches[*models.GenericPreConditionFailedError](ctx, http.StatusPreconditionFailed, err, w)
	}
}

func RedirectResponseEncoder[Response string](statusCode int) encoder.ResponseEncoder[Response] {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, response Response) error {
		redirectURL := string(response)

		_, err := url.Parse(redirectURL)
		if err != nil {
			return fmt.Errorf("invalid redirect url: %w", err)
		}

		if statusCode < 300 || statusCode > 399 {
			return fmt.Errorf("invalid redirect status code: it must be in 3xx range: %w", err)
		}

		http.Redirect(w, r, redirectURL, statusCode)

		return nil
	}
}
