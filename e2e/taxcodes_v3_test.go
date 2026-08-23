package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// Missing tax code references are rejected by the service input validation
// (resource_id_empty per missing field), not the converter — the request must
// fail with a 400 naming exactly the missing fields rather than a 500 or a
// free-text converter error.
func TestV3OrganizationDefaultTaxCodesMissingReferences(t *testing.T) {
	fieldsByCode := func(problem *v3Problem, code string) []string {
		var fields []string
		for _, e := range problem.ValidationErrors() {
			if e.Code == code {
				fields = append(fields, e.Field)
			}
		}
		return fields
	}

	t.Run("both references missing", func(t *testing.T) {
		c := newV3Client(t)

		_, err := c.Defaults.UpdateOrganizationTaxCodes(t.Context(), v3sdk.UpdateOrganizationDefaultTaxCodesRequest{})
		problem := requireProblem(t, err, http.StatusBadRequest)
		assert.ElementsMatch(t,
			[]string{"$.invoicing_tax_code_id", "$.credit_grant_tax_code_id"},
			fieldsByCode(problem, "resource_id_empty"),
			"problem: %+v", problem)
	})

	t.Run("credit grant reference missing", func(t *testing.T) {
		c := newV3Client(t)

		_, err := c.Defaults.UpdateOrganizationTaxCodes(t.Context(), v3sdk.UpdateOrganizationDefaultTaxCodesRequest{
			InvoicingTaxCode: &v3sdk.TaxCodeReference{ID: "01JZZZZZZZZZZZZZZZZZZZZZZZ"},
		})
		problem := requireProblem(t, err, http.StatusBadRequest)
		assert.ElementsMatch(t,
			[]string{"$.credit_grant_tax_code_id"},
			fieldsByCode(problem, "resource_id_empty"),
			"problem: %+v", problem)
	})
}
