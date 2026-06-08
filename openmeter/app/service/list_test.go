package appservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// minimalApp satisfies app.App without billing or customer operations.
type minimalApp struct {
	app.AppBase
}

func (m *minimalApp) GetEventAppData() (app.EventAppData, error)                     { return app.EventAppData{}, nil }
func (m *minimalApp) UpdateAppConfig(_ context.Context, _ app.AppConfigUpdate) error { return nil }
func (m *minimalApp) GetCustomerData(_ context.Context, _ app.GetAppInstanceCustomerDataInput) (app.CustomerData, error) {
	return nil, nil
}
func (m *minimalApp) UpsertCustomerData(_ context.Context, _ app.UpsertAppInstanceCustomerDataInput) error {
	return nil
}
func (m *minimalApp) DeleteCustomerData(_ context.Context, _ app.DeleteAppInstanceCustomerDataInput) error {
	return nil
}

// minimalFactory satisfies app.AppFactory for testing without billing wiring.
type minimalFactory struct{}

func (f *minimalFactory) NewApp(_ context.Context, base app.AppBase) (app.App, error) {
	return &minimalApp{AppBase: base}, nil
}

func (f *minimalFactory) UninstallApp(_ context.Context, _ app.UninstallAppInput) error {
	return nil
}

func newListTestService(t *testing.T) app.Service {
	t.Helper()

	db := testutils.InitPostgresDB(t)
	t.Cleanup(func() { db.Close(t) })

	client := db.EntDriver.Client()
	require.NoError(t, client.Schema.Create(t.Context()))

	adapter, err := appadapter.New(appadapter.Config{Client: client})
	require.NoError(t, err)

	svc, err := appservice.New(appservice.Config{
		Adapter:   adapter,
		Publisher: eventbus.NewMock(t),
	})
	require.NoError(t, err)

	for _, l := range []app.MarketplaceListing{
		{Type: app.AppTypeSandbox, Name: "Sandbox", Description: "test sandbox"},
		{Type: app.AppTypeCustomInvoicing, Name: "Custom Invoicing", Description: "test custom invoicing"},
	} {
		require.NoError(t, svc.RegisterMarketplaceListing(app.RegistryItem{
			Listing: l,
			Factory: &minimalFactory{},
		}))
	}

	return svc
}

func newNS() string { return ulid.Make().String() }

func makeApp(t *testing.T, svc app.Service, ns, name string, appType app.AppType) app.AppBase {
	t.Helper()
	base, err := svc.CreateApp(t.Context(), app.CreateAppInput{
		Namespace: ns, Name: name, Type: appType,
	})
	require.NoError(t, err)
	return base
}

func TestListApps_FilterByID(t *testing.T) {
	svc := newListTestService(t)
	ns := newNS()

	a1 := makeApp(t, svc, ns, "App One", app.AppTypeSandbox)
	_ = makeApp(t, svc, ns, "App Two", app.AppTypeSandbox)

	a1ID := a1.GetID().ID
	result, err := svc.ListApps(t.Context(), app.ListAppInput{
		Namespace: ns,
		Page:      pagination.NewPage(1, 20),
		ID:        &filter.FilterULID{FilterString: filter.FilterString{Eq: &a1ID}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.TotalCount)
	require.Equal(t, a1.GetID().ID, result.Items[0].GetID().ID)

	result, err = svc.ListApps(t.Context(), app.ListAppInput{
		Namespace: ns,
		Page:      pagination.NewPage(1, 20),
		ID:        &filter.FilterULID{FilterString: filter.FilterString{In: &[]string{a1.GetID().ID}}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.TotalCount)
}

func TestListApps_FilterByName(t *testing.T) {
	svc := newListTestService(t)
	ns := newNS()

	_ = makeApp(t, svc, ns, "Billing App", app.AppTypeSandbox)
	_ = makeApp(t, svc, ns, "Payment App", app.AppTypeSandbox)
	_ = makeApp(t, svc, ns, "Other App", app.AppTypeSandbox)

	result, err := svc.ListApps(t.Context(), app.ListAppInput{
		Namespace: ns,
		Page:      pagination.NewPage(1, 20),
		Name:      &filter.FilterString{Eq: lo.ToPtr("Billing App")},
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.TotalCount)
	require.Equal(t, "Billing App", result.Items[0].GetName())

	result, err = svc.ListApps(t.Context(), app.ListAppInput{
		Namespace: ns,
		Page:      pagination.NewPage(1, 20),
		Name:      &filter.FilterString{Contains: lo.ToPtr("app")},
	})
	require.NoError(t, err)
	require.Equal(t, 3, result.TotalCount)
}

func TestListApps_FilterByType(t *testing.T) {
	svc := newListTestService(t)
	ns := newNS()

	_ = makeApp(t, svc, ns, "Sandbox App", app.AppTypeSandbox)
	ci := makeApp(t, svc, ns, "Custom Invoicing App", app.AppTypeCustomInvoicing)

	result, err := svc.ListApps(t.Context(), app.ListAppInput{
		Namespace: ns,
		Page:      pagination.NewPage(1, 20),
		Type:      &filter.FilterString{Eq: lo.ToPtr(string(app.AppTypeCustomInvoicing))},
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.TotalCount)
	require.Equal(t, ci.GetID().ID, result.Items[0].GetID().ID)

	result, err = svc.ListApps(t.Context(), app.ListAppInput{
		Namespace: ns,
		Page:      pagination.NewPage(1, 20),
		Type:      &filter.FilterString{Eq: lo.ToPtr(string(app.AppTypeSandbox))},
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.TotalCount)
}

func TestListApps_FilterByStatus(t *testing.T) {
	svc := newListTestService(t)
	ns := newNS()

	a1 := makeApp(t, svc, ns, "Ready App", app.AppTypeSandbox)
	a2 := makeApp(t, svc, ns, "Unauthorized App", app.AppTypeSandbox)

	require.NoError(t, svc.UpdateAppStatus(t.Context(), app.UpdateAppStatusInput{
		ID:     a2.GetID(),
		Status: app.AppStatusUnauthorized,
	}))

	result, err := svc.ListApps(t.Context(), app.ListAppInput{
		Namespace: ns,
		Page:      pagination.NewPage(1, 20),
		Status:    &filter.FilterString{Eq: lo.ToPtr(string(app.AppStatusReady))},
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.TotalCount)
	require.Equal(t, a1.GetID().ID, result.Items[0].GetID().ID)

	result, err = svc.ListApps(t.Context(), app.ListAppInput{
		Namespace: ns,
		Page:      pagination.NewPage(1, 20),
		Status:    &filter.FilterString{Eq: lo.ToPtr(string(app.AppStatusUnauthorized))},
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.TotalCount)
	require.Equal(t, a2.GetID().ID, result.Items[0].GetID().ID)
}

func TestListApps_SortByIDDesc(t *testing.T) {
	svc := newListTestService(t)
	ns := newNS()

	a1 := makeApp(t, svc, ns, "App Alpha", app.AppTypeSandbox)
	a2 := makeApp(t, svc, ns, "App Beta", app.AppTypeSandbox)

	var result pagination.Result[app.App]
	assert.Eventually(t, func() bool {
		var err error
		result, err = svc.ListApps(t.Context(), app.ListAppInput{
			Namespace: ns,
			Page:      pagination.NewPage(1, 20),
			OrderBy:   app.AppOrderByID,
			Order:     sortx.OrderDesc,
		})
		require.Equal(t, 2, result.TotalCount)
		return err == nil && result.TotalCount == 2 &&
			result.Items[0].GetID().ID == a2.GetID().ID
	}, time.Second, time.Millisecond)
	require.Equal(t, a2.GetID().ID, result.Items[0].GetID().ID)
	require.Equal(t, a1.GetID().ID, result.Items[1].GetID().ID)
}

func TestListApps_SortByCreatedAtDesc(t *testing.T) {
	svc := newListTestService(t)
	ns := newNS()

	a1 := makeApp(t, svc, ns, "App First", app.AppTypeSandbox)
	a2 := makeApp(t, svc, ns, "App Second", app.AppTypeSandbox)

	var result pagination.Result[app.App]
	assert.Eventually(t, func() bool {
		var err error
		result, err = svc.ListApps(t.Context(), app.ListAppInput{
			Namespace: ns,
			Page:      pagination.NewPage(1, 20),
			OrderBy:   app.AppOrderByCreatedAt,
			Order:     sortx.OrderDesc,
		})
		require.Equal(t, 2, result.TotalCount)
		return err == nil && result.TotalCount == 2 &&
			result.Items[0].GetID().ID == a2.GetID().ID
	}, time.Second, time.Millisecond)
	require.Equal(t, a2.GetID().ID, result.Items[0].GetID().ID)
	require.Equal(t, a1.GetID().ID, result.Items[1].GetID().ID)
}

func TestListApps_DefaultSortCreatedAtAsc(t *testing.T) {
	svc := newListTestService(t)
	ns := newNS()

	a1 := makeApp(t, svc, ns, "App Oldest", app.AppTypeSandbox)
	a2 := makeApp(t, svc, ns, "App Newest", app.AppTypeSandbox)

	var result pagination.Result[app.App]
	assert.Eventually(t, func() bool {
		var err error
		result, err = svc.ListApps(t.Context(), app.ListAppInput{
			Namespace: ns,
			Page:      pagination.NewPage(1, 20),
		})
		require.Equal(t, 2, result.TotalCount)
		return err == nil && result.TotalCount == 2 &&
			result.Items[0].GetID().ID == a1.GetID().ID
	}, time.Second, time.Millisecond)
	require.Equal(t, a1.GetID().ID, result.Items[0].GetID().ID)
	require.Equal(t, a2.GetID().ID, result.Items[1].GetID().ID)
}
