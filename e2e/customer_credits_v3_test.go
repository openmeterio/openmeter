package e2e

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

func TestV3CustomerCreditBalanceTimestampParamParsing(t *testing.T) {
	c := newV3Client(t)
	customerID := ulid.Make().String()

	t.Run("valid timestamp reaches endpoint handling", func(t *testing.T) {
		query := url.Values{
			"timestamp": {time.Date(2026, 5, 11, 10, 30, 0, 0, time.UTC).Format(time.RFC3339)},
		}

		status, _, problem := c.do(http.MethodGet, "/customers/"+customerID+"/credits/balance?"+query.Encode(), nil)
		require.NotEqual(t, http.StatusBadRequest, status, "problem: %+v", problem)
	})

	t.Run("invalid timestamp is rejected by query parsing", func(t *testing.T) {
		query := url.Values{
			"timestamp": {"not-a-date"},
		}

		status, _, problem := c.do(http.MethodGet, "/customers/"+customerID+"/credits/balance?"+query.Encode(), nil)
		require.Equal(t, http.StatusBadRequest, status)
		require.NotNil(t, problem)
		assert.Contains(t, problem.Detail, "timestamp")
	})
}

// TestV3CreateCreditGrantMissingTaxCode verifies the documented contract for
// create-credit-grant: referencing a tax code that does not exist is rejected
// with HTTP 400 (a validation error), not a 412/500. The OpenAPI spec documents
// 400 for this operation, so this asserts the in-contract behavior end-to-end
// through the HTTP layer rather than only at the service boundary.
func TestV3CreateCreditGrantMissingTaxCode(t *testing.T) {
	c := newV3Client(t)

	currency := apiv3.CurrencyCode("USD")
	status, customer, problem := c.CreateCustomer(apiv3.CreateCustomerRequest{
		Key:      uniqueKey("credit_grant_taxcode_customer"),
		Name:     "Credit Grant Tax Code Test Customer",
		Currency: &currency,
	})
	require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
	require.NotNil(t, customer)

	// A structurally valid but non-existent tax code ULID.
	missingTaxCode := ulid.Make().String()

	status, _, problem = c.CreateCreditGrant(customer.Id, apiv3.CreateCreditGrantRequest{
		Name:          "grant with missing tax code",
		Amount:        apiv3.Numeric("10"),
		Currency:      currency,
		FundingMethod: apiv3.BillingCreditFundingMethodNone,
		TaxConfig: &apiv3.CreateCreditGrantTaxConfig{
			TaxCode: &apiv3.CreateResourceReference{Id: missingTaxCode},
		},
	})

	require.Equal(t, http.StatusBadRequest, status, "missing tax code must be a 400 validation error, problem: %+v", problem)
	require.NotNil(t, problem)

	// The offending tax code is named either in the top-level Detail or in the
	// structured validationErrors[] message, depending on which error layer
	// renders it. Accept either so the test asserts the contract, not the
	// rendering shape.
	mentionsTaxCode := strings.Contains(problem.Detail, "tax code")
	for _, ve := range problem.ValidationErrors() {
		if strings.Contains(ve.Message, "tax code") || strings.Contains(ve.Field, "tax_code") {
			mentionsTaxCode = true
		}
	}
	assert.True(t, mentionsTaxCode, "response should name the missing tax code, problem: %+v", problem)
}
