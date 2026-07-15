package e2e

import (
	"net/http"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// TestV3AppCatalogList exercises GET /api/v3/openmeter/apps/catalog. The
// catalog lists app types available to install, not installed app instances,
// so its contents are stable across environments regardless of config: all
// three app factories (sandbox, stripe, custom invoicing) register their
// marketplace listing unconditionally in app/common (see
// openmeter/app/sandbox/marketplace.go, openmeter/app/stripe/marketplace.go,
// openmeter/app/custominvoicing/factory.go) — a Stripe API key is only
// required to install a Stripe app instance, not for it to appear here.
func TestV3AppCatalogList(t *testing.T) {
	c := newV3Client(t)

	t.Run("Should list the app catalog with all built-in apps present", func(t *testing.T) {
		// TODO change it to app catalog get
		page, err := c.Apps.ListCatalog(t.Context(), v3sdk.AppCatalogItemListParams{})
		require.Equal(t, http.StatusOK, c.statuses.last())
		require.NoError(t, err)
		require.NotNil(t, page)

		assert.Equal(t, len(page.Data), page.Meta.Page.Total, "single-page catalog: total should match returned item count")

		for _, item := range page.Data {
			assert.NotEmpty(t, item.Name, "catalog item missing name: %+v", item)
			assert.NotEmpty(t, item.Description, "catalog item missing description: %+v", item)
		}

		sandbox, ok := lo.Find(page.Data, func(item v3sdk.AppCatalogItem) bool {
			return item.Type == v3sdk.AppTypeSandbox
		})
		require.True(t, ok, "sandbox app missing from catalog: %+v", page.Data)
		assert.Equal(t, "Sandbox", sandbox.Name)
		assert.Equal(t, "Sandbox can be used to test OpenMeter without external connections.", sandbox.Description)

		stripe, ok := lo.Find(page.Data, func(item v3sdk.AppCatalogItem) bool {
			return item.Type == v3sdk.AppTypeStripe
		})
		require.True(t, ok, "stripe app missing from catalog: %+v", page.Data)
		assert.Equal(t, "Stripe", stripe.Name)

		externalInvoicing, ok := lo.Find(page.Data, func(item v3sdk.AppCatalogItem) bool {
			return item.Type == v3sdk.AppTypeExternalInvoicing
		})
		require.True(t, ok, "external invoicing app missing from catalog: %+v", page.Data)
		assert.Equal(t, "Custom Invoicing", externalInvoicing.Name)
	})

	t.Run("Should paginate the app catalog", func(t *testing.T) {
		fullPage, err := c.Apps.ListCatalog(t.Context(), v3sdk.AppCatalogItemListParams{})
		require.Equal(t, http.StatusOK, c.statuses.last())
		require.NoError(t, err)
		require.NotNil(t, fullPage)
		require.NotEmpty(t, fullPage.Data)

		firstPage, err := c.Apps.ListCatalog(t.Context(), v3sdk.AppCatalogItemListParams{
			Page: &v3sdk.PageParams{
				Size:   lo.ToPtr(1),
				Number: lo.ToPtr(1),
			},
		})
		require.Equal(t, http.StatusOK, c.statuses.last())
		require.NoError(t, err)
		require.NotNil(t, firstPage)

		assert.Len(t, firstPage.Data, 1)
		assert.Equal(t, int(1), firstPage.Meta.Page.Number)
		assert.Equal(t, int(1), firstPage.Meta.Page.Size)
		assert.Equal(t, fullPage.Meta.Page.Total, firstPage.Meta.Page.Total, "total count should be independent of page size")
	})
}

func TestV3AppInstall(t *testing.T) {
	c := newV3Client(t)

	t.Run("Should install external invoicing app", func(t *testing.T) {
		req, err := v3sdk.InstallAppRequestFromInstallAppExternalInvoicing(v3sdk.InstallAppExternalInvoicing{
			Type:                 v3sdk.AppTypeExternalInvoicing,
			Name:                 gofakeit.LoremIpsumSentence(3),
			CreateBillingProfile: false,
		})
		require.NoError(t, err)
		resp, err := c.Apps.Install(t.Context(), req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, string(v3sdk.AppTypeExternalInvoicing), resp.App.Type)
	})
}
