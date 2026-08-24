package e2e

import (
	"net/http"
	"slices"
	"strings"
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

// externalInvoicingAppIDs returns the IDs of the page's apps, in page order.
// It requires every item to be an external-invoicing app, which holds for any
// list scoped to this suite's marker-named fixtures.
func externalInvoicingAppIDs(t *testing.T, page *v3sdk.AppPagePaginatedResponse) []string {
	t.Helper()

	ids := make([]string, 0, len(page.Data))
	for _, item := range page.Data {
		app, err := item.AsAppExternalInvoicing()
		require.NoError(t, err)
		ids = append(ids, app.ID)
	}

	return ids
}

// TestV3AppListFiltersAndSort exercises the filter[...] and sort query
// parameters of GET /api/v3/openmeter/apps. The e2e namespace is shared across
// tests and re-runs, so every fixture name embeds a unique marker and every
// assertion scopes the list with a filter[name][contains]=<marker> predicate
// (or a globally unique app ID) instead of relying on namespace isolation.
func TestV3AppListFiltersAndSort(t *testing.T) {
	c := newV3Client(t)

	marker := uniqueKey("applist")

	// Installed in this order; app IDs are ULIDs, so both the id and the
	// created_at orderings follow the install order.
	names := []string{
		"Billing App " + marker,
		"Payment App " + marker,
		"Other App " + marker,
	}

	installed := make([]v3sdk.AppExternalInvoicing, 0, len(names))
	for _, name := range names {
		req, err := v3sdk.InstallAppRequestFromInstallAppExternalInvoicing(v3sdk.InstallAppExternalInvoicing{
			Type:                 v3sdk.AppTypeExternalInvoicing,
			Name:                 name,
			CreateBillingProfile: false,
		})
		require.NoError(t, err)

		resp, err := c.Apps.Install(t.Context(), req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		app, err := resp.App.AsAppExternalInvoicing()
		require.NoError(t, err)
		installed = append(installed, *app)
	}

	idsByInstall := lo.Map(installed, func(a v3sdk.AppExternalInvoicing, _ int) string { return a.ID })

	listPage := &v3sdk.PageParams{Size: lo.ToPtr(1000)}
	nameHasMarker := &v3sdk.StringFilter{Contains: lo.ToPtr(marker)}

	t.Run("Should filter by id with eq and oeq", func(t *testing.T) {
		page, err := c.Apps.List(t.Context(), v3sdk.AppListParams{
			Page:   listPage,
			Filter: &v3sdk.AppFilter{ID: &v3sdk.StringExactFilter{Eq: lo.ToPtr(installed[0].ID)}},
		})
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, 1, page.Meta.Page.Total)
		require.Equal(t, idsByInstall[:1], externalInvoicingAppIDs(t, page))

		page, err = c.Apps.List(t.Context(), v3sdk.AppListParams{
			Page:   listPage,
			Filter: &v3sdk.AppFilter{ID: &v3sdk.StringExactFilter{Oeq: []string{installed[0].ID, installed[1].ID}}},
		})
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, 2, page.Meta.Page.Total)
	})

	t.Run("Should filter by name with eq and case-insensitive contains", func(t *testing.T) {
		page, err := c.Apps.List(t.Context(), v3sdk.AppListParams{
			Page:   listPage,
			Filter: &v3sdk.AppFilter{Name: &v3sdk.StringFilter{Eq: lo.ToPtr(names[0])}},
		})
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, 1, page.Meta.Page.Total)
		require.Equal(t, idsByInstall[:1], externalInvoicingAppIDs(t, page))

		// contains translates to ILIKE, so the upper-cased marker must still
		// match the lower-cased fixture names.
		page, err = c.Apps.List(t.Context(), v3sdk.AppListParams{
			Page:   listPage,
			Filter: &v3sdk.AppFilter{Name: &v3sdk.StringFilter{Contains: lo.ToPtr(strings.ToUpper(marker))}},
		})
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, 3, page.Meta.Page.Total)
		assert.ElementsMatch(t, idsByInstall, externalInvoicingAppIDs(t, page))
	})

	t.Run("Should filter by type", func(t *testing.T) {
		page, err := c.Apps.List(t.Context(), v3sdk.AppListParams{
			Page: listPage,
			Filter: &v3sdk.AppFilter{
				Type: &v3sdk.StringExactFilter{Eq: lo.ToPtr(string(v3sdk.AppTypeExternalInvoicing))},
				Name: nameHasMarker,
			},
		})
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, 3, page.Meta.Page.Total)

		// The marker fixtures are all external-invoicing apps, so a different
		// type must filter every one of them out.
		page, err = c.Apps.List(t.Context(), v3sdk.AppListParams{
			Page: listPage,
			Filter: &v3sdk.AppFilter{
				Type: &v3sdk.StringExactFilter{Eq: lo.ToPtr(string(v3sdk.AppTypeSandbox))},
				Name: nameHasMarker,
			},
		})
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, 0, page.Meta.Page.Total)
	})

	t.Run("Should filter by status", func(t *testing.T) {
		// Freshly installed apps are ready; no public API can flip an app to
		// unauthorized (that transition belongs to the Stripe credential
		// checks), so the equality predicate is exercised through a matching
		// and a non-matching status value instead.
		page, err := c.Apps.List(t.Context(), v3sdk.AppListParams{
			Page: listPage,
			Filter: &v3sdk.AppFilter{
				Status: &v3sdk.StringExactFilter{Eq: lo.ToPtr(string(v3sdk.AppStatusReady))},
				Name:   nameHasMarker,
			},
		})
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, 3, page.Meta.Page.Total)

		page, err = c.Apps.List(t.Context(), v3sdk.AppListParams{
			Page: listPage,
			Filter: &v3sdk.AppFilter{
				Status: &v3sdk.StringExactFilter{Eq: lo.ToPtr(string(v3sdk.AppStatusUnauthorized))},
				Name:   nameHasMarker,
			},
		})
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, 0, page.Meta.Page.Total)
	})

	newestFirst := slices.Clone(idsByInstall)
	slices.Reverse(newestFirst)

	t.Run("Should sort by id desc", func(t *testing.T) {
		page, err := c.Apps.List(t.Context(), v3sdk.AppListParams{
			Page:   listPage,
			Filter: &v3sdk.AppFilter{Name: nameHasMarker},
			Sort:   &v3sdk.Sort{By: "id", Order: v3sdk.SortOrderDesc},
		})
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, newestFirst, externalInvoicingAppIDs(t, page))
	})

	t.Run("Should sort by created_at desc", func(t *testing.T) {
		page, err := c.Apps.List(t.Context(), v3sdk.AppListParams{
			Page:   listPage,
			Filter: &v3sdk.AppFilter{Name: nameHasMarker},
			Sort:   &v3sdk.Sort{By: "created_at", Order: v3sdk.SortOrderDesc},
		})
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, newestFirst, externalInvoicingAppIDs(t, page))
	})

	t.Run("Should default to created_at asc without a sort parameter", func(t *testing.T) {
		page, err := c.Apps.List(t.Context(), v3sdk.AppListParams{
			Page:   listPage,
			Filter: &v3sdk.AppFilter{Name: nameHasMarker},
		})
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, idsByInstall, externalInvoicingAppIDs(t, page))
	})

	t.Run("Should reject a malformed ULID in the id filter", func(t *testing.T) {
		_, err := c.Apps.List(t.Context(), v3sdk.AppListParams{
			Page:   listPage,
			Filter: &v3sdk.AppFilter{ID: &v3sdk.StringExactFilter{Eq: lo.ToPtr("not-a-ulid")}},
		})
		// pkg/filter's FilterULID rejects non-ULID values (ulid.ParseStrict)
		// when ListAppInput is validated in the app service.
		requireProblem(t, err, http.StatusBadRequest)
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

func TestV3AppUninstall(t *testing.T) {
	c := newV3Client(t)

	t.Run("Should uninstall an installed app", func(t *testing.T) {
		req, err := v3sdk.InstallAppRequestFromInstallAppExternalInvoicing(v3sdk.InstallAppExternalInvoicing{
			Type:                 v3sdk.AppTypeExternalInvoicing,
			Name:                 gofakeit.LoremIpsumSentence(3),
			CreateBillingProfile: false,
		})
		require.NoError(t, err)
		installResp, err := c.Apps.Install(t.Context(), req)
		require.NoError(t, err)
		require.NotNil(t, installResp)

		installed, err := installResp.App.AsAppExternalInvoicing()
		require.NoError(t, err)
		require.NotEmpty(t, installed.ID)

		c.requireStatus(http.StatusNoContent, c.Apps.Uninstall(t.Context(), installed.ID))

		// UninstallApp soft-deletes external-invoicing apps (sets deleted_at) rather
		// than removing the row, and GetApp does not filter on deleted_at, so the
		// app is still readable afterwards — just marked deleted. This mirrors the
		// v1 behavior for this app type; other app types (e.g. Stripe) hard-delete
		// their sub-table data on uninstall and surface a not-found error instead.
		getResp, err := c.Apps.Get(t.Context(), installed.ID)
		require.Equal(t, http.StatusOK, c.statuses.last())
		require.NoError(t, err)
		require.NotNil(t, getResp)

		gotten, err := getResp.AsAppExternalInvoicing()
		require.NoError(t, err)
		require.NotNil(t, gotten.DeletedAt, "expected deleted_at to be set after uninstall")
	})
}
