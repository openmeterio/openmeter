package billingcommon

import (
	"encoding/json"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestToAPIValidationIssues(t *testing.T) {
	t.Run("empty issues are omitted", func(t *testing.T) {
		issues := ToAPIValidationIssues(nil)
		require.Nil(t, issues)

		body, err := json.Marshal(struct {
			ValidationIssues *[]apiv3models.BillingValidationIssue `json:"validation_issues,omitempty"`
		}{ValidationIssues: issues})
		require.NoError(t, err)
		require.JSONEq(t, `{}`, string(body))
	})

	t.Run("maps public issue fields", func(t *testing.T) {
		issues := ToAPIValidationIssues(billing.ValidationIssues{
			{
				Severity:   billing.ValidationIssueSeverityWarning,
				Message:    "line needs attention",
				Code:       "line_context",
				Component:  billing.ComponentName("openmeter.invoicing"),
				Path:       "/lines/0",
				Attributes: models.Annotations{"invoice": "invoice-1", "line": "line-1"},
			},
		})
		require.NotNil(t, issues)
		require.Equal(t, []apiv3models.BillingValidationIssue{
			{
				Severity:   apiv3models.BillingValidationIssueSeverityWarning,
				Message:    "line needs attention",
				Code:       "line_context",
				Component:  lo.ToPtr("openmeter.invoicing"),
				Field:      lo.ToPtr("/lines/0"),
				Attributes: lo.ToPtr(map[string]any{"invoice": "invoice-1", "line": "line-1"}),
			},
		}, *issues)
	})
}
