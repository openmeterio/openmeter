package e2e

import (
	"net/http"
	"testing"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// Missing tax code references are rejected by the service input validation
// (resource_id_empty), not the converter — the request must fail with a 400
// carrying the domain code rather than a 500 or a free-text converter error.
func TestV3OrganizationDefaultTaxCodesMissingReferences(t *testing.T) {
	t.Run("both references missing", func(t *testing.T) {
		c := newV3Client(t)

		_, err := c.Defaults.UpdateOrganizationTaxCodes(t.Context(), v3sdk.UpdateOrganizationDefaultTaxCodesRequest{})
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertValidationCode(t, problem, "resource_id_empty")
	})

	t.Run("credit grant reference missing", func(t *testing.T) {
		c := newV3Client(t)

		_, err := c.Defaults.UpdateOrganizationTaxCodes(t.Context(), v3sdk.UpdateOrganizationDefaultTaxCodesRequest{
			InvoicingTaxCode: &v3sdk.TaxCodeReference{ID: "01JZZZZZZZZZZZZZZZZZZZZZZZ"},
		})
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertValidationCode(t, problem, "resource_id_empty")
	})
}
