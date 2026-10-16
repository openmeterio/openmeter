package errorsx

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestDiagnosticAttributesBoundsCyclicErrors(t *testing.T) {
	cycle := &cyclicDiagnosticError{}
	err := models.NewError("ledger_failed", "accounting failed", models.WithCause(cycle)).
		WithAttr("reason", "source_missing")

	attrs := diagnosticAttributes(err)

	require.Positive(t, cycle.visits)
	require.LessOrEqual(t, cycle.visits, 1024)
	require.Equal(t, []any{
		slog.Any("error_codes", []string{"ledger_failed"}),
		slog.Any("error_reasons", []string{"source_missing"}),
	}, attrs)
}

type cyclicDiagnosticError struct {
	visits int
}

var _ error = (*cyclicDiagnosticError)(nil)

func (*cyclicDiagnosticError) Error() string { return "cyclic error" }

func (e *cyclicDiagnosticError) Unwrap() []error {
	e.visits++
	return []error{e, e}
}
