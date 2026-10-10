package errorsx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestIsOnly(t *testing.T) {
	allowed := errors.New("allowed")
	other := errors.New("other")
	issue := models.NewValidationIssue("allowed_issue", "allowed issue")

	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: true},
		{name: "single", err: allowed, want: true},
		{name: "wrapped", err: fmt.Errorf("context: %w", allowed), want: true},
		{name: "joined allowed", err: fmt.Errorf("context: %w", errors.Join(allowed, issue.WithField(models.NewFieldSelector("nested")))), want: true},
		{name: "joined mixed", err: errors.Join(allowed, other), want: false},
		{name: "other", err: other, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, errorsx.IsOnly(tc.err, allowed, issue))
		})
	}

	require.False(t, errorsx.IsOnly(allowed))
}
