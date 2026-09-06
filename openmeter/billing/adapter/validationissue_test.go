package billingadapter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestIssueDedupeHashIncludesAttributes(t *testing.T) {
	base := billing.ValidationIssue{
		Severity: billing.ValidationIssueSeverityWarning,
		Message:  "invoice line needs attention",
		Attributes: models.Annotations{
			"invoice": "invoice-1",
			"line":    "line-1",
		},
	}

	baseHash, err := issueDedupeHash(base)
	require.NoError(t, err)

	sameHash, err := issueDedupeHash(billing.ValidationIssue{
		Severity: base.Severity,
		Message:  base.Message,
		Attributes: models.Annotations{
			"line":    "line-1",
			"invoice": "invoice-1",
		},
	})
	require.NoError(t, err)
	require.Equal(t, baseHash, sameHash)

	changed := base
	changed.Attributes = models.Annotations{"invoice": "invoice-1", "line": "line-2"}
	changedHash, err := issueDedupeHash(changed)
	require.NoError(t, err)
	require.NotEqual(t, baseHash, changedHash)
}
