package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3TaxCodeUpdateClearsOmittedFields(t *testing.T) {
	c := newV3Client(t)

	created, err := c.Tax.CreateCode(t.Context(), v3sdk.CreateTaxCodeRequest{
		Key:         uniqueKey("test_v3_tax_code_replace"),
		Name:        "Tax code replace",
		Description: lo.ToPtr("Original description"),
		Labels:      &map[string]string{"env": "prod"},
		AppMappings: []v3sdk.TaxCodeAppMapping{},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, created)
	require.NotNil(t, created.Description)
	require.NotEmpty(t, created.Labels)

	updated, err := c.Tax.UpsertCode(t.Context(), created.ID, v3sdk.UpsertTaxCodeRequest{
		Name:        created.Name,
		AppMappings: []v3sdk.TaxCodeAppMapping{},
	})
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, updated)

	assert.Nil(t, updated.Description, "omitted description must be cleared, not carried over")
	assert.Empty(t, updated.Labels, "omitted labels must be cleared, not carried over")

	fetched, err := c.Tax.GetCode(t.Context(), created.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, fetched)

	assert.Nil(t, fetched.Description)
	assert.Empty(t, fetched.Labels)
}
