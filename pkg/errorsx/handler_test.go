package errorsx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSlogHandlerCancellation(t *testing.T) {
	for _, test := range []struct {
		name     string
		err      error
		canceled bool
		level    string
	}{
		{"caller cancellation", fmt.Errorf("query: %w", context.Canceled), true, "WARN"},
		{"caller gRPC cancellation", fmt.Errorf("authorize: %w", status.Error(codes.Canceled, "request canceled")), true, "WARN"},
		{"internal cancellation", context.Canceled, false, "ERROR"},
		{"internal gRPC cancellation", status.Error(codes.Canceled, "request canceled"), false, "ERROR"},
		{"deadline", context.DeadlineExceeded, true, "ERROR"},
		{"gRPC deadline", status.Error(codes.DeadlineExceeded, "deadline exceeded"), true, "ERROR"},
		{"unrelated failure", errors.New("database unavailable"), true, "ERROR"},
		{"cancellation text", errors.New("context canceled"), true, "ERROR"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if test.canceled {
				cancel()
			}

			var output bytes.Buffer
			handler := NewSlogHandler(slog.New(slog.NewTextHandler(&output, nil)))
			handler.HandleContext(ctx, test.err)
			require.Contains(t, output.String(), "level="+test.level)
		})
	}
}

func TestSlogHandlerWithoutContextRecognizesGRPCCancellation(t *testing.T) {
	var output bytes.Buffer
	handler := NewSlogHandler(slog.New(slog.NewTextHandler(&output, nil)))
	handler.Handle(fmt.Errorf("authorize: %w", status.Error(codes.Canceled, "request canceled")))
	require.Contains(t, output.String(), "level=WARN")
}

func TestSlogHandlerErrorDiagnostics(t *testing.T) {
	base := models.NewValidationIssue("invalid_entry", "invalid entry").WithAttrs(models.Attributes{
		"reason": "old_reason",
	})
	issue := base.WithAttrs(models.Attributes{
		"reason":       "origin_mismatch",
		"request_data": "must not become a log attribute",
	})
	operational := models.NewError("ledger_collection_failed", "source exhausted", models.WithAttribute("reason", "ledger_source_exhausted"))
	err := fmt.Errorf("collect: %w", errors.Join(issue, &issue, operational, operational, errors.New("unclassified")))

	var output bytes.Buffer
	handler := NewSlogHandler(slog.New(slog.NewJSONHandler(&output, nil)))
	handler.HandleContext(t.Context(), err)

	var record map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &record))
	require.Equal(t, err.Error(), record["msg"])
	require.Equal(t, "ERROR", record["level"])
	require.Equal(t, []any{"invalid_entry", "ledger_collection_failed"}, record["error_codes"])
	require.Equal(t, []any{"origin_mismatch", "ledger_source_exhausted"}, record["error_reasons"])
	require.NotContains(t, output.String(), "old_reason")
	require.NotContains(t, output.String(), "request_data")
	require.NotContains(t, output.String(), "must not become a log attribute")
}

func TestSlogHandlerWarningDiagnostics(t *testing.T) {
	err := NewWarnError(models.NewError("ledger_collection_failed", "source exhausted", models.WithAttribute("reason", "ledger_source_exhausted")))

	var output bytes.Buffer
	handler := NewSlogHandler(slog.New(slog.NewJSONHandler(&output, nil)))
	handler.Handle(err)

	var record map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &record))
	require.Equal(t, "WARN", record["level"])
	require.Equal(t, err.Error(), record["msg"])
	require.Equal(t, []any{"ledger_source_exhausted"}, record["error_reasons"])
	require.Equal(t, []any{"ledger_collection_failed"}, record["error_codes"])
}

func TestSlogHandlerUnclassifiedError(t *testing.T) {
	var output bytes.Buffer
	handler := NewSlogHandler(slog.New(slog.NewJSONHandler(&output, nil)))
	handler.Handle(errors.New("unclassified failure"))

	var record map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &record))
	require.Equal(t, "ERROR", record["level"])
	require.Equal(t, "unclassified failure", record["msg"])
	require.NotContains(t, record, "error_codes")
	require.NotContains(t, record, "error_reasons")
}

func TestSlogHandlerStructuredCauseDiagnostics(t *testing.T) {
	cause := models.NewError("storage_failed", "storage unavailable").WithAttr("reason", "connection_lost")
	err := models.NewError("ledger_failed", "cannot load source", models.WithCause(cause)).
		WithAttr("reason", "old_reason").WithAttr("reason", "source_unavailable")

	var output bytes.Buffer
	handler := NewSlogHandler(slog.New(slog.NewJSONHandler(&output, nil)))
	handler.HandleContext(t.Context(), err)

	var record map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &record))
	require.Equal(t, []any{"ledger_failed", "storage_failed"}, record["error_codes"])
	require.Equal(t, []any{"source_unavailable", "connection_lost"}, record["error_reasons"])
	require.NotContains(t, output.String(), "old_reason")
}
