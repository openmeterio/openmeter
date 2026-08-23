package taxcodes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Missing tax code references convert to empty IDs; rejecting them is the
// service input validation's job (resource_id_empty), not the converter's.
func TestFromAPIUpdateOrganizationDefaultTaxCodesRequestMissingReferences(t *testing.T) {
	input, err := FromAPIUpdateOrganizationDefaultTaxCodesRequest("test-ns", api.UpdateOrganizationDefaultTaxCodesRequest{})
	require.NoError(t, err)
	assert.Empty(t, input.InvoicingTaxCodeID)
	assert.Empty(t, input.CreditGrantTaxCodeID)

	issues, err := models.AsValidationIssues(input.Validate())
	require.NoError(t, err)

	var codes []models.ErrorCode
	for _, issue := range issues {
		codes = append(codes, issue.Code())
	}
	assert.Contains(t, codes, taxcode.ErrCodeResourceIDEmpty)
}
