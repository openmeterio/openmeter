package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3CustomerUpdateClearsOmittedFields(t *testing.T) {
	c := newV3Client(t)

	created, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:          uniqueKey("test_v3_customer_replace"),
		Name:         "Customer replace",
		Description:  lo.ToPtr("Original description"),
		PrimaryEmail: lo.ToPtr("original@example.com"),
		Currency:     lo.ToPtr("USD"),
		BillingAddress: &v3sdk.Address{
			City:       lo.ToPtr("Budapest"),
			Country:    lo.ToPtr("HU"),
			Line1:      lo.ToPtr("Fő utca 1."),
			PostalCode: lo.ToPtr("1011"),
			State:      lo.ToPtr("Budapest"),
		},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, created)
	require.NotNil(t, created.PrimaryEmail)
	require.NotNil(t, created.Currency)
	require.NotNil(t, created.BillingAddress)

	t.Run("Should replace a supplied billing address as a whole", func(t *testing.T) {
		updated, err := c.Customers.Upsert(t.Context(), created.ID, v3sdk.UpsertCustomerRequest{
			Name: created.Name,
			BillingAddress: &v3sdk.Address{
				City: lo.ToPtr("Berlin"),
			},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated)

		require.NotNil(t, updated.BillingAddress)
		assert.Equal(t, lo.ToPtr("Berlin"), updated.BillingAddress.City)
		assert.Nil(t, updated.BillingAddress.Country, "address lines omitted from a supplied address must be cleared")
		assert.Nil(t, updated.BillingAddress.Line1)
		assert.Nil(t, updated.BillingAddress.PostalCode)
		assert.Nil(t, updated.BillingAddress.State)
	})

	t.Run("Should clear omitted top-level fields", func(t *testing.T) {
		updated, err := c.Customers.Upsert(t.Context(), created.ID, v3sdk.UpsertCustomerRequest{
			Name: created.Name,
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updated)

		assert.Nil(t, updated.Description, "omitted description must be cleared, not carried over")
		assert.Nil(t, updated.PrimaryEmail, "omitted primary email must be cleared, not carried over")
		assert.Nil(t, updated.Currency, "omitted currency must be cleared, not carried over")
		assert.Nil(t, updated.BillingAddress, "omitted billing address must be cleared, not carried over")

		fetched, err := c.Customers.Get(t.Context(), created.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, fetched)

		assert.Nil(t, fetched.Description)
		assert.Nil(t, fetched.PrimaryEmail)
		assert.Nil(t, fetched.Currency)
		assert.Nil(t, fetched.BillingAddress)
		assert.Equal(t, created.Key, fetched.Key, "the key is not part of the update body and must survive the replace")
	})
}
