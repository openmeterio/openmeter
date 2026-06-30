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

func TestV3CreateCreditGrantIdempotencyKey(t *testing.T) {
	c := newV3Client(t)
	currency := apiv3.CurrencyCode("USD")

	createCustomer := func(prefix string) string {
		status, customer, problem := c.CreateCustomer(apiv3.CreateCustomerRequest{
			Key:      uniqueKey(prefix),
			Name:     "Credit Grant Idempotency Test Customer",
			Currency: &currency,
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, customer)
		return customer.Id
	}

	grant := func(key *string) apiv3.CreateCreditGrantRequest {
		return apiv3.CreateCreditGrantRequest{
			Name:          "idempotency grant",
			Amount:        apiv3.Numeric("10"),
			Currency:      currency,
			FundingMethod: apiv3.BillingCreditFundingMethodNone,
			Key:           key,
		}
	}

	t.Run("reusing a key for the same customer returns 409", func(t *testing.T) {
		customerID := createCustomer("credit_grant_idem_conflict")
		key := ulid.Make().String()

		status, _, problem := c.CreateCreditGrant(customerID, grant(&key))
		require.Equal(t, http.StatusCreated, status, "first create must succeed, problem: %+v", problem)

		status, _, problem = c.CreateCreditGrant(customerID, grant(&key))
		require.Equal(t, http.StatusConflict, status, "reusing an idempotency key must be a 409, problem: %+v", problem)
	})

	t.Run("omitting the key allows duplicates", func(t *testing.T) {
		customerID := createCustomer("credit_grant_idem_nil")

		status, _, problem := c.CreateCreditGrant(customerID, grant(nil))
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)

		status, _, problem = c.CreateCreditGrant(customerID, grant(nil))
		require.Equal(t, http.StatusCreated, status, "grants without a key must not collide, problem: %+v", problem)
	})

	t.Run("the same key conflicts across different customers in a namespace", func(t *testing.T) {
		key := ulid.Make().String()

		status, _, problem := c.CreateCreditGrant(createCustomer("credit_grant_idem_cust_a"), grant(&key))
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)

		status, _, problem = c.CreateCreditGrant(createCustomer("credit_grant_idem_cust_b"), grant(&key))
		require.Equal(t, http.StatusConflict, status, "the key is unique per namespace, not per customer, problem: %+v", problem)
	})

	t.Run("an over-length key is rejected with 400", func(t *testing.T) {
		customerID := createCustomer("credit_grant_idem_overlong")
		key := strings.Repeat("k", 257)

		status, _, problem := c.CreateCreditGrant(customerID, grant(&key))
		require.Equal(t, http.StatusBadRequest, status, "an over-length key must be a 400, problem: %+v", problem)
	})
}

// TestV3CreditGrantKeyReadAndFilter verifies that the idempotency key is exposed
// on the get/list read responses and that list credit grants can be filtered by
// key. The key column carries a unique partial index per namespace, so an
// equality filter targets at most one grant.
func TestV3CreditGrantKeyReadAndFilter(t *testing.T) {
	c := newV3Client(t)
	currency := apiv3.CurrencyCode("USD")

	status, customer, problem := c.CreateCustomer(apiv3.CreateCustomerRequest{
		Key:      uniqueKey("credit_grant_key_filter"),
		Name:     "Credit Grant Key Filter Customer",
		Currency: &currency,
	})
	require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
	require.NotNil(t, customer)
	customerID := customer.Id

	grant := func(name string, key *string) apiv3.CreateCreditGrantRequest {
		return apiv3.CreateCreditGrantRequest{
			Name:          name,
			Amount:        apiv3.Numeric("10"),
			Currency:      currency,
			FundingMethod: apiv3.BillingCreditFundingMethodNone,
			Key:           key,
		}
	}

	keyed := ulid.Make().String()

	// given:
	// - one grant created with an idempotency key and one without
	status, keyedGrant, problem := c.CreateCreditGrant(customerID, grant("keyed grant", &keyed))
	require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
	require.NotNil(t, keyedGrant)

	status, unkeyedGrant, problem := c.CreateCreditGrant(customerID, grant("unkeyed grant", nil))
	require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
	require.NotNil(t, unkeyedGrant)

	t.Run("create response echoes the key", func(t *testing.T) {
		require.Equal(t, &keyed, keyedGrant.Key)
		require.Nil(t, unkeyedGrant.Key)
	})

	t.Run("get response exposes the key", func(t *testing.T) {
		// when:
		// - the keyed grant is fetched by id
		status, got, problem := c.GetCreditGrant(customerID, keyedGrant.Id)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, got)
		// then:
		// - the read response carries the same key
		require.Equal(t, &keyed, got.Key)

		status, gotUnkeyed, problem := c.GetCreditGrant(customerID, unkeyedGrant.Id)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, gotUnkeyed)
		require.Nil(t, gotUnkeyed.Key)
	})

	t.Run("list filters by key", func(t *testing.T) {
		// when:
		// - listing grants filtered to the keyed grant's key
		status, list, problem := c.ListCreditGrants(customerID, keyed)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, list)
		// then:
		// - exactly the keyed grant is returned, with its key populated
		require.Len(t, list.Data, 1)
		require.Equal(t, keyedGrant.Id, list.Data[0].Id)
		require.Equal(t, &keyed, list.Data[0].Key)
	})

	t.Run("list returns the key for unfiltered results", func(t *testing.T) {
		status, list, problem := c.ListCreditGrants(customerID, "")
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, list)

		byID := make(map[string]apiv3.BillingCreditGrant, len(list.Data))
		for _, g := range list.Data {
			byID[g.Id] = g
		}

		require.Contains(t, byID, keyedGrant.Id)
		require.Equal(t, &keyed, byID[keyedGrant.Id].Key)
		require.Contains(t, byID, unkeyedGrant.Id)
		require.Nil(t, byID[unkeyedGrant.Id].Key)
	})

	t.Run("list key filter with no match returns empty", func(t *testing.T) {
		status, list, problem := c.ListCreditGrants(customerID, ulid.Make().String())
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, list)
		require.Empty(t, list.Data)
	})
}
