package e2e

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3CustomerCreditBalanceTimestampParamParsing(t *testing.T) {
	c := newV3Client(t)
	customerID := ulid.Make().String()

	t.Run("valid timestamp reaches endpoint handling", func(t *testing.T) {
		timestamp := time.Date(2026, 5, 11, 10, 30, 0, 0, time.UTC)

		_, err := c.Customers.Credits.Balance.Get(t.Context(), customerID, v3sdk.GetCustomerCreditBalanceParams{Timestamp: &timestamp})
		if err != nil {
			apiErr, ok := v3sdk.AsAPIError(err)
			require.True(t, ok, "unexpected non-API error: %v", err)
			require.NotEqual(t, http.StatusBadRequest, apiErr.StatusCode, "problem: %s", string(apiErr.RawBody))
		}
	})

	t.Run("invalid timestamp is rejected by query parsing", func(t *testing.T) {
		query := url.Values{
			"timestamp": {"not-a-date"},
		}

		status, _, problem := c.doMalformedRequest(http.MethodGet, "/customers/"+customerID+"/credits/balance?"+query.Encode(), nil)
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

	currency := v3sdk.BillingCurrencyCode("USD")
	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:      uniqueKey("credit_grant_taxcode_customer"),
		Name:     "Credit Grant Tax Code Test Customer",
		Currency: lo.ToPtr("USD"),
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)

	// A structurally valid but non-existent tax code ULID.
	missingTaxCode := ulid.Make().String()

	_, err = c.Customers.Credits.Grants.Create(t.Context(), customer.ID, v3sdk.CreateCreditGrantRequest{
		Name:          "grant with missing tax code",
		Amount:        v3sdk.Numeric("10"),
		Currency:      currency,
		FundingMethod: v3sdk.CreditFundingMethodNone,
		TaxConfig: &v3sdk.CreditGrantTaxConfig{
			TaxCode: &v3sdk.TaxCodeReference{ID: missingTaxCode},
		},
	})

	problem := requireProblem(t, err, http.StatusBadRequest)

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
	currency := v3sdk.BillingCurrencyCode("USD")

	createCustomer := func(prefix string) string {
		customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      uniqueKey(prefix),
			Name:     "Credit Grant Idempotency Test Customer",
			Currency: lo.ToPtr("USD"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, customer)
		return customer.ID
	}

	grant := func(key *string) v3sdk.CreateCreditGrantRequest {
		return v3sdk.CreateCreditGrantRequest{
			Name:          "idempotency grant",
			Amount:        v3sdk.Numeric("10"),
			Currency:      currency,
			FundingMethod: v3sdk.CreditFundingMethodNone,
			Key:           key,
		}
	}

	t.Run("reusing a key for the same customer returns 409", func(t *testing.T) {
		customerID := createCustomer("credit_grant_idem_conflict")
		key := ulid.Make().String()

		_, err := c.Customers.Credits.Grants.Create(t.Context(), customerID, grant(&key))
		c.requireStatus(http.StatusCreated, err)

		_, err = c.Customers.Credits.Grants.Create(t.Context(), customerID, grant(&key))
		requireProblem(t, err, http.StatusConflict)
	})

	t.Run("omitting the key allows duplicates", func(t *testing.T) {
		customerID := createCustomer("credit_grant_idem_nil")

		_, err := c.Customers.Credits.Grants.Create(t.Context(), customerID, grant(nil))
		c.requireStatus(http.StatusCreated, err)

		_, err = c.Customers.Credits.Grants.Create(t.Context(), customerID, grant(nil))
		c.requireStatus(http.StatusCreated, err)
	})

	t.Run("the same key conflicts across different customers in a namespace", func(t *testing.T) {
		key := ulid.Make().String()

		_, err := c.Customers.Credits.Grants.Create(t.Context(), createCustomer("credit_grant_idem_cust_a"), grant(&key))
		c.requireStatus(http.StatusCreated, err)

		_, err = c.Customers.Credits.Grants.Create(t.Context(), createCustomer("credit_grant_idem_cust_b"), grant(&key))
		requireProblem(t, err, http.StatusConflict)
	})

	t.Run("an over-length key is rejected with 400", func(t *testing.T) {
		customerID := createCustomer("credit_grant_idem_overlong")
		key := strings.Repeat("k", 257)

		_, err := c.Customers.Credits.Grants.Create(t.Context(), customerID, grant(&key))
		requireProblem(t, err, http.StatusBadRequest)
	})
}

func voidCreditGrantRaw(t testing.TB, c *v3Client, customerID, creditGrantID string, body any) (int, *v3sdk.CreditGrant, *v3Problem) {
	t.Helper()

	if body == nil {
		body = map[string]any{}
	}

	status, raw, problem := c.doMalformedRequest(http.MethodPost, "/customers/"+customerID+"/credits/grants/"+creditGrantID+"/void", body)
	if status != http.StatusOK {
		return status, nil, problem
	}

	var grant v3sdk.CreditGrant
	require.NoError(t, json.Unmarshal(raw, &grant))

	return status, &grant, nil
}

func TestV3VoidCreditGrant(t *testing.T) {
	c := newV3Client(t)
	currency := v3sdk.BillingCurrencyCode("USD")

	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:      uniqueKey("credit_grant_void_customer"),
		Name:     "Credit Grant Void Test Customer",
		Currency: lo.ToPtr("USD"),
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)
	customerID := customer.ID

	// given:
	// - an active promotional grant of 25 and a second grant that stays untouched
	grant, err := c.Customers.Credits.Grants.Create(t.Context(), customerID, v3sdk.CreateCreditGrantRequest{
		Name:          "grant to void",
		Amount:        v3sdk.Numeric("25"),
		Currency:      currency,
		FundingMethod: v3sdk.CreditFundingMethodNone,
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, grant)
	require.Equal(t, v3sdk.CreditGrantStatusActive, grant.Status)
	require.Nil(t, grant.VoidedAt)

	keptGrant, err := c.Customers.Credits.Grants.Create(t.Context(), customerID, v3sdk.CreateCreditGrantRequest{
		Name:          "grant to keep",
		Amount:        v3sdk.Numeric("40"),
		Currency:      currency,
		FundingMethod: v3sdk.CreditFundingMethodNone,
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, keptGrant)

	// when:
	// - the first grant is voided
	status, voided, problem := voidCreditGrantRaw(t, c, customerID, grant.ID, map[string]any{})
	require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
	require.NotNil(t, voided)

	// then:
	// - the response reads as voided with a voided_at timestamp
	require.Equal(t, v3sdk.CreditGrantStatusVoided, voided.Status)
	require.NotNil(t, voided.VoidedAt)

	t.Run("get derives the voided status", func(t *testing.T) {
		got, err := c.Customers.Credits.Grants.Get(t.Context(), customerID, grant.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, got)
		require.Equal(t, v3sdk.CreditGrantStatusVoided, got.Status)
		require.NotNil(t, got.VoidedAt)
		assert.True(t, voided.VoidedAt.Equal(*got.VoidedAt), "voided_at must be stable across reads")

		kept, err := c.Customers.Credits.Grants.Get(t.Context(), customerID, keptGrant.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, kept)
		require.Equal(t, v3sdk.CreditGrantStatusActive, kept.Status)
		require.Nil(t, kept.VoidedAt)
	})

	t.Run("retrying the void is an idempotent success", func(t *testing.T) {
		status, again, problem := voidCreditGrantRaw(t, c, customerID, grant.ID, map[string]any{})
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, again)
		require.Equal(t, v3sdk.CreditGrantStatusVoided, again.Status)
		require.NotNil(t, again.VoidedAt)
		assert.True(t, voided.VoidedAt.Equal(*again.VoidedAt), "a retry must return the original void time")
	})

	t.Run("credit balance only carries the untouched grant", func(t *testing.T) {
		balances, err := c.Customers.Credits.Balance.Get(t.Context(), customerID, v3sdk.GetCustomerCreditBalanceParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, balances)
		require.Len(t, balances.Balances, 1)
		assert.Equal(t, v3sdk.Numeric("40"), balances.Balances[0].Settled)
	})

	t.Run("status filter separates voided from active grants", func(t *testing.T) {
		voidedStatus := v3sdk.CreditGrantStatusVoided
		voidedList, err := c.Customers.Credits.Grants.List(t.Context(), customerID, v3sdk.CreditGrantListParams{
			Filter: &v3sdk.CreditGrantFilter{Status: &voidedStatus},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, voidedList)
		require.Len(t, voidedList.Data, 1)
		assert.Equal(t, grant.ID, voidedList.Data[0].ID)

		activeStatus := v3sdk.CreditGrantStatusActive
		activeList, err := c.Customers.Credits.Grants.List(t.Context(), customerID, v3sdk.CreditGrantListParams{
			Filter: &v3sdk.CreditGrantFilter{Status: &activeStatus},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, activeList)
		require.Len(t, activeList.Data, 1)
		assert.Equal(t, keptGrant.ID, activeList.Data[0].ID)
	})

	t.Run("transaction listing surfaces the forfeiture as voided", func(t *testing.T) {
		txType := v3sdk.CreditTransactionType("voided")
		transactions, err := c.Customers.Credits.Transactions.List(t.Context(), customerID, v3sdk.CreditTransactionListParams{
			Filter: &v3sdk.CreditTransactionFilter{Type: &txType},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, transactions)
		require.Len(t, transactions.Data, 1)
		assert.Equal(t, txType, transactions.Data[0].Type)
		assert.Equal(t, v3sdk.Numeric("-25"), transactions.Data[0].Amount)

		// The unfiltered listing carries both fundings and the void.
		all, err := c.Customers.Credits.Transactions.List(t.Context(), customerID, v3sdk.CreditTransactionListParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, all)

		typeCounts := map[v3sdk.CreditTransactionType]int{}
		for _, tx := range all.Data {
			typeCounts[tx.Type]++
		}
		assert.Equal(t, 2, typeCounts[v3sdk.CreditTransactionTypeFunded])
		assert.Equal(t, 1, typeCounts[txType])
	})

	t.Run("voiding an unknown grant is a 404", func(t *testing.T) {
		status, _, problem := voidCreditGrantRaw(t, c, customerID, ulid.Make().String(), map[string]any{})
		require.Equal(t, http.StatusNotFound, status, "problem: %+v", problem)
	})
}

func TestV3VoidCreditGrantPaymentAdjustmentNone(t *testing.T) {
	c := newV3Client(t)
	currency := v3sdk.BillingCurrencyCode("USD")

	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:      uniqueKey("credit_grant_void_adjustment_customer"),
		Name:     "Credit Grant Void Payment Adjustment Test Customer",
		Currency: lo.ToPtr("USD"),
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)

	grant, err := c.Customers.Credits.Grants.Create(t.Context(), customer.ID, v3sdk.CreateCreditGrantRequest{
		Name:          "grant to void with explicit payment adjustment",
		Amount:        v3sdk.Numeric("15"),
		Currency:      currency,
		FundingMethod: v3sdk.CreditFundingMethodNone,
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, grant)

	status, voided, problem := voidCreditGrantRaw(t, c, customer.ID, grant.ID, map[string]string{
		"payment_adjustment": "none",
	})
	require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
	require.NotNil(t, voided)
	require.Equal(t, v3sdk.CreditGrantStatusVoided, voided.Status)
	require.NotNil(t, voided.VoidedAt)
}

// TestV3CreditGrantKeyReadAndFilter verifies that the idempotency key is exposed
// on the get/list read responses and that list credit grants can be filtered by
// key. The key column carries a unique partial index per namespace, so an
// equality filter targets at most one grant.
func TestV3CreditGrantKeyReadAndFilter(t *testing.T) {
	c := newV3Client(t)
	currency := v3sdk.BillingCurrencyCode("USD")

	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:      uniqueKey("credit_grant_key_filter"),
		Name:     "Credit Grant Key Filter Customer",
		Currency: lo.ToPtr("USD"),
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)
	customerID := customer.ID

	grant := func(name string, key *string) v3sdk.CreateCreditGrantRequest {
		return v3sdk.CreateCreditGrantRequest{
			Name:          name,
			Amount:        v3sdk.Numeric("10"),
			Currency:      currency,
			FundingMethod: v3sdk.CreditFundingMethodNone,
			Key:           key,
		}
	}

	keyed := ulid.Make().String()

	// given:
	// - one grant created with an idempotency key and one without
	keyedGrant, err := c.Customers.Credits.Grants.Create(t.Context(), customerID, grant("keyed grant", &keyed))
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, keyedGrant)

	unkeyedGrant, err := c.Customers.Credits.Grants.Create(t.Context(), customerID, grant("unkeyed grant", nil))
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, unkeyedGrant)

	t.Run("create response echoes the key", func(t *testing.T) {
		require.Equal(t, &keyed, keyedGrant.Key)
		require.Nil(t, unkeyedGrant.Key)
	})

	t.Run("get response exposes the key", func(t *testing.T) {
		// when:
		// - the keyed grant is fetched by id
		got, err := c.Customers.Credits.Grants.Get(t.Context(), customerID, keyedGrant.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, got)
		// then:
		// - the read response carries the same key
		require.Equal(t, &keyed, got.Key)

		gotUnkeyed, err := c.Customers.Credits.Grants.Get(t.Context(), customerID, unkeyedGrant.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, gotUnkeyed)
		require.Nil(t, gotUnkeyed.Key)
	})

	t.Run("list filters by key", func(t *testing.T) {
		// when:
		// - listing grants filtered to the keyed grant's key
		list, err := c.Customers.Credits.Grants.List(t.Context(), customerID, v3sdk.CreditGrantListParams{
			Filter: &v3sdk.CreditGrantFilter{
				Key: &v3sdk.StringFilter{Eq: &keyed},
			},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, list)
		// then:
		// - exactly the keyed grant is returned, with its key populated
		require.Len(t, list.Data, 1)
		require.Equal(t, keyedGrant.ID, list.Data[0].ID)
		require.Equal(t, &keyed, list.Data[0].Key)
	})

	t.Run("list returns the key for unfiltered results", func(t *testing.T) {
		list, err := c.Customers.Credits.Grants.List(t.Context(), customerID, v3sdk.CreditGrantListParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, list)

		byID := make(map[string]v3sdk.CreditGrant, len(list.Data))
		for _, g := range list.Data {
			byID[g.ID] = g
		}

		require.Contains(t, byID, keyedGrant.ID)
		require.Equal(t, &keyed, byID[keyedGrant.ID].Key)
		require.Contains(t, byID, unkeyedGrant.ID)
		require.Nil(t, byID[unkeyedGrant.ID].Key)
	})

	t.Run("list key filter with no match returns empty", func(t *testing.T) {
		keyFilter := ulid.Make().String()
		list, err := c.Customers.Credits.Grants.List(t.Context(), customerID, v3sdk.CreditGrantListParams{
			Filter: &v3sdk.CreditGrantFilter{
				Key: &v3sdk.StringFilter{Eq: &keyFilter},
			},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, list)
		require.Empty(t, list.Data)
	})
}
