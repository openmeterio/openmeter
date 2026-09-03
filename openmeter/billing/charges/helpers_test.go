package charges

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingfeaturemeter "github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	featuremeterservice "github.com/openmeterio/openmeter/openmeter/billing/featuremeter/service"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestValidateStandardInvoiceCreatedFeatures(t *testing.T) {
	// given: a charge-owned standard line references a feature absent from the resolved collection
	line := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.ManagedResource{ID: "line-id"},
		},
		UsageBased: &billing.UsageBasedLine{
			FeatureKey: "missing-feature",
		},
	}
	input := billing.OnStandardInvoiceCreatedInput{
		StandardLineEventInput: billing.StandardLineEventInput{
			Lines: billing.StandardLines{line},
		},
		FeatureMeters: featuremeterservice.FeatureMeterCollection{
			ByKey: map[string]billingfeaturemeter.FeatureMeter{},
			ByID:  map[string]billingfeaturemeter.FeatureMeter{},
		},
	}

	// when: charge feature references are validated before lifecycle effects
	err := ValidateStandardInvoiceCreatedFeatures(input)

	// then: the missing reference is a line-scoped validation issue
	issues, systemErr := billing.ToValidationIssues(err)
	require.NoError(t, systemErr)
	require.Equal(t, billing.ValidationIssues{
		{
			Severity: billing.ValidationIssueSeverityCritical,
			Code:     billing.ErrInvoiceLineFeatureNotFound.Code,
			Message:  "feature[missing-feature]: invoice line: feature not found",
			Path:     "/lines/line-id",
		},
	}, issues)
}
