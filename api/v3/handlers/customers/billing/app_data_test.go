package customersbilling

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

type stubApp struct {
	app.App

	appType app.AppType

	customerData   app.CustomerData
	getCustomerErr error

	upsertedData []app.CustomerData
	deleteCalls  int
}

func (a *stubApp) GetType() app.AppType {
	return a.appType
}

func (a *stubApp) GetCustomerData(_ context.Context, _ app.GetAppInstanceCustomerDataInput) (app.CustomerData, error) {
	return a.customerData, a.getCustomerErr
}

func (a *stubApp) UpsertCustomerData(_ context.Context, input app.UpsertAppInstanceCustomerDataInput) error {
	a.upsertedData = append(a.upsertedData, input.Data)
	return nil
}

func (a *stubApp) DeleteCustomerData(_ context.Context, _ app.DeleteAppInstanceCustomerDataInput) error {
	a.deleteCalls++
	return nil
}

var testCustomerID = customer.CustomerID{Namespace: "test-ns", ID: "test-customer"}

func noStripeDataError() error {
	return app.NewAppCustomerPreConditionError(
		app.AppID{Namespace: testCustomerID.Namespace, ID: "test-app"},
		app.AppTypeStripe,
		&testCustomerID,
		"customer has no data for stripe app",
	)
}

func TestApplyAppData(t *testing.T) {
	t.Run("stripe", func(t *testing.T) {
		t.Run("omitted field deletes existing data", func(t *testing.T) {
			application := &stubApp{appType: app.AppTypeStripe}

			err := applyAppData(t.Context(), application, testCustomerID, "", api.UpsertAppCustomerDataRequest{})

			require.NoError(t, err)
			require.Equal(t, 1, application.deleteCalls)
			require.Empty(t, application.upsertedData)
		})

		t.Run("null deletes data", func(t *testing.T) {
			application := &stubApp{appType: app.AppTypeStripe}

			err := applyAppData(t.Context(), application, testCustomerID, "", api.UpsertAppCustomerDataRequest{
				Stripe: nullable.NewNullNullable[api.BillingAppCustomerDataStripe](),
			})

			require.NoError(t, err)
			require.Equal(t, 1, application.deleteCalls)
			require.Empty(t, application.upsertedData)
		})

		t.Run("value upserts mapped data", func(t *testing.T) {
			application := &stubApp{appType: app.AppTypeStripe}

			err := applyAppData(t.Context(), application, testCustomerID, "", api.UpsertAppCustomerDataRequest{
				Stripe: nullable.NewNullableWithValue(api.BillingAppCustomerDataStripe{
					CustomerId:             lo.ToPtr("cus_123"),
					DefaultPaymentMethodId: lo.ToPtr("pm_456"),
				}),
			})

			require.NoError(t, err)
			require.Equal(t, []app.CustomerData{appstripe.CustomerData{
				StripeCustomerID:             "cus_123",
				StripeDefaultPaymentMethodID: lo.ToPtr("pm_456"),
			}}, application.upsertedData)
			require.Zero(t, application.deleteCalls)
		})

		t.Run("value without customer id is a bad request", func(t *testing.T) {
			application := &stubApp{appType: app.AppTypeStripe}

			err := applyAppData(t.Context(), application, testCustomerID, "", api.UpsertAppCustomerDataRequest{
				Stripe: nullable.NewNullableWithValue(api.BillingAppCustomerDataStripe{}),
			})

			var apiErr *apierrors.BaseAPIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, http.StatusBadRequest, apiErr.Status)
			require.Equal(t, "stripe.customer_id", apiErr.InvalidParameters[0].Field)
			require.Empty(t, application.upsertedData)
			require.Zero(t, application.deleteCalls)
		})

		t.Run("the field prefix scopes the reported error field", func(t *testing.T) {
			application := &stubApp{appType: app.AppTypeStripe}

			err := applyAppData(t.Context(), application, testCustomerID, "app_data.", api.UpsertAppCustomerDataRequest{
				Stripe: nullable.NewNullableWithValue(api.BillingAppCustomerDataStripe{}),
			})

			var apiErr *apierrors.BaseAPIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, "app_data.stripe.customer_id", apiErr.InvalidParameters[0].Field)
		})
	})

	t.Run("custom invoicing", func(t *testing.T) {
		t.Run("omitted field deletes existing data", func(t *testing.T) {
			application := &stubApp{appType: app.AppTypeCustomInvoicing}

			err := applyAppData(t.Context(), application, testCustomerID, "", api.UpsertAppCustomerDataRequest{})

			require.NoError(t, err)
			require.Equal(t, 1, application.deleteCalls)
			require.Empty(t, application.upsertedData)
		})

		t.Run("null deletes data", func(t *testing.T) {
			application := &stubApp{appType: app.AppTypeCustomInvoicing}

			err := applyAppData(t.Context(), application, testCustomerID, "", api.UpsertAppCustomerDataRequest{
				ExternalInvoicing: nullable.NewNullNullable[api.BillingAppCustomerDataExternalInvoicing](),
			})

			require.NoError(t, err)
			require.Equal(t, 1, application.deleteCalls)
			require.Empty(t, application.upsertedData)
		})

		t.Run("value upserts labels", func(t *testing.T) {
			application := &stubApp{appType: app.AppTypeCustomInvoicing}

			err := applyAppData(t.Context(), application, testCustomerID, "", api.UpsertAppCustomerDataRequest{
				ExternalInvoicing: nullable.NewNullableWithValue(api.BillingAppCustomerDataExternalInvoicing{
					Labels: lo.ToPtr(api.Labels{"team": "billing"}),
				}),
			})

			require.NoError(t, err)
			require.Equal(t, []app.CustomerData{appcustominvoicing.CustomerData{
				Metadata: models.Metadata{"team": "billing"},
			}}, application.upsertedData)
		})

		t.Run("value without labels upserts empty data", func(t *testing.T) {
			application := &stubApp{appType: app.AppTypeCustomInvoicing}

			err := applyAppData(t.Context(), application, testCustomerID, "", api.UpsertAppCustomerDataRequest{
				ExternalInvoicing: nullable.NewNullableWithValue(api.BillingAppCustomerDataExternalInvoicing{}),
			})

			require.NoError(t, err)
			require.Equal(t, []app.CustomerData{appcustominvoicing.CustomerData{}}, application.upsertedData)
		})
	})

	t.Run("sandbox", func(t *testing.T) {
		t.Run("upserts the app customer relationship and ignores other app fields", func(t *testing.T) {
			application := &stubApp{appType: app.AppTypeSandbox}

			err := applyAppData(t.Context(), application, testCustomerID, "", api.UpsertAppCustomerDataRequest{
				Stripe: nullable.NewNullNullable[api.BillingAppCustomerDataStripe](),
			})

			require.NoError(t, err)
			require.Equal(t, []app.CustomerData{appsandbox.CustomerData{}}, application.upsertedData)
			require.Zero(t, application.deleteCalls)
		})
	})

	t.Run("invalid values are rejected regardless of the resolved app", func(t *testing.T) {
		for _, appType := range []app.AppType{app.AppTypeCustomInvoicing, app.AppTypeSandbox} {
			t.Run(string(appType), func(t *testing.T) {
				application := &stubApp{appType: appType}

				err := applyAppData(t.Context(), application, testCustomerID, "", api.UpsertAppCustomerDataRequest{
					Stripe: nullable.NewNullableWithValue(api.BillingAppCustomerDataStripe{}),
				})

				var apiErr *apierrors.BaseAPIError
				require.ErrorAs(t, err, &apiErr)
				require.Equal(t, http.StatusBadRequest, apiErr.Status)
				require.Zero(t, application.deleteCalls)
			})
		}
	})

	t.Run("valid values for other apps are ignored", func(t *testing.T) {
		application := &stubApp{appType: app.AppTypeCustomInvoicing}

		err := applyAppData(t.Context(), application, testCustomerID, "", api.UpsertAppCustomerDataRequest{
			Stripe: nullable.NewNullableWithValue(api.BillingAppCustomerDataStripe{
				CustomerId: lo.ToPtr("cus_123"),
			}),
		})

		require.NoError(t, err)
		require.Empty(t, application.upsertedData)
		require.Equal(t, 1, application.deleteCalls, "the resolved app's field is absent, so its data is removed")
	})
}

func TestBuildAppData(t *testing.T) {
	t.Run("stripe data is returned", func(t *testing.T) {
		application := &stubApp{
			appType: app.AppTypeStripe,
			customerData: appstripe.CustomerData{
				StripeCustomerID:             "cus_123",
				StripeDefaultPaymentMethodID: lo.ToPtr("pm_456"),
			},
		}

		appData, err := buildAppData(t.Context(), application, testCustomerID)

		require.NoError(t, err)
		require.Equal(t, &api.BillingAppCustomerData{
			Stripe: &api.BillingAppCustomerDataStripe{
				CustomerId:             lo.ToPtr("cus_123"),
				DefaultPaymentMethodId: lo.ToPtr("pm_456"),
			},
		}, appData)
	})

	t.Run("missing stripe data is omitted", func(t *testing.T) {
		application := &stubApp{
			appType:        app.AppTypeStripe,
			getCustomerErr: noStripeDataError(),
		}

		appData, err := buildAppData(t.Context(), application, testCustomerID)

		require.NoError(t, err)
		require.Equal(t, &api.BillingAppCustomerData{}, appData)
	})

	t.Run("other stripe errors are propagated", func(t *testing.T) {
		application := &stubApp{
			appType:        app.AppTypeStripe,
			getCustomerErr: errors.New("boom"),
		}

		_, err := buildAppData(t.Context(), application, testCustomerID)

		require.Error(t, err)
	})

	t.Run("custom invoicing labels are returned", func(t *testing.T) {
		application := &stubApp{
			appType: app.AppTypeCustomInvoicing,
			customerData: appcustominvoicing.CustomerData{
				Metadata: models.Metadata{"team": "billing"},
			},
		}

		appData, err := buildAppData(t.Context(), application, testCustomerID)

		require.NoError(t, err)
		require.Equal(t, &api.BillingAppCustomerData{
			ExternalInvoicing: &api.BillingAppCustomerDataExternalInvoicing{
				Labels: lo.ToPtr(api.Labels{"team": "billing"}),
			},
		}, appData)
	})

	t.Run("sandbox has no app data", func(t *testing.T) {
		application := &stubApp{appType: app.AppTypeSandbox}

		appData, err := buildAppData(t.Context(), application, testCustomerID)

		require.NoError(t, err)
		require.Equal(t, &api.BillingAppCustomerData{}, appData)
	})
}
