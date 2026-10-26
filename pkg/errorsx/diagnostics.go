package errorsx

import (
	"log/slog"
	"slices"

	"github.com/openmeterio/openmeter/pkg/models"
)

// Keep all reasons from joined errors, but don't copy arbitrary error attributes
// into logs. They can contain request data or unbounded provider values.
func diagnosticAttributes(err error) []any {
	// Bound the whole traversal so cycles and branching chains cannot recurse forever.
	remaining := 1024
	var codes, reasons []string
	appendUnique := func(values *[]string, value string) {
		if value != "" && !slices.Contains(*values, value) {
			*values = append(*values, value)
		}
	}

	var visit func(error)
	visit = func(err error) {
		if err == nil || remaining == 0 {
			return
		}

		remaining--

		if issue, ok := err.(interface {
			Code() models.ErrorCode
			Attributes() models.Attributes
		}); ok {
			appendUnique(&codes, string(issue.Code()))
			if reason, ok := issue.Attributes()["reason"].(string); ok {
				appendUnique(&reasons, reason)
			}

			// Skip earlier copies whose attributes were replaced, but keep the cause.
			if caused, ok := err.(interface{ Cause() error }); ok {
				visit(caused.Cause())
			}

			return
		}

		switch wrapped := err.(type) {
		case interface{ Unwrap() []error }:
			for _, child := range wrapped.Unwrap() {
				if remaining == 0 {
					break
				}

				visit(child)
			}
		case interface{ Unwrap() error }:
			visit(wrapped.Unwrap())
		}
	}
	visit(err)

	var attrs []any
	if len(codes) > 0 {
		attrs = append(attrs, slog.Any("error_codes", codes))
	}

	if len(reasons) > 0 {
		attrs = append(attrs, slog.Any("error_reasons", reasons))
	}

	return attrs
}
