package billing

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/app/billingprofile"
	apphttpdriver "github.com/openmeterio/openmeter/openmeter/app/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (s *ProfileTestSuite) TestCustomInvoicingInstallProfileDefaults() {
	for _, tc := range []struct {
		Name                 string
		ExistingDefault      bool
		CreateBillingProfile bool
	}{
		{Name: "no existing default", CreateBillingProfile: true},
		{Name: "preserve existing default", ExistingDefault: true, CreateBillingProfile: true},
		{Name: "explicit opt out"},
	} {
		s.Run(tc.Name, func() {
			// given: a namespace with either no default or an existing Sandbox default.
			ctx := s.T().Context()
			ns := s.GetUniqueNamespace("custom_install_profile")
			var existingDefault *billing.Profile
			if tc.ExistingDefault {
				existingDefault = s.ProvisionBillingProfile(ctx, ns, s.InstallSandboxApp(s.T(), ns).GetID())
			}

			// when: Custom Invoicing is installed through the shared provisioning path.
			result, err := s.AppService.InstallApp(ctx, app.InstallAppV3Input{
				MarketplaceListingID:        app.MarketplaceListingID{Type: app.AppTypeCustomInvoicing},
				Namespace:                   ns,
				Name:                        "My Invoicing",
				CreateDefaultBillingProfile: tc.CreateBillingProfile,
				CreateDefaultBillingProfileFn: func(ctx context.Context, installedApp app.App) ([]app.CapabilityType, error) {
					return billingprofile.CreateDefault(ctx, s.BillingService, nil, installedApp)
				},
			})
			require.NoError(s.T(), err)

			// then: provisioning matches Cloud's preset, without changing the namespace default.
			require.Empty(s.T(), result.DefaultCapabilies)
			defaultProfile, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: ns})
			require.NoError(s.T(), err)
			if existingDefault == nil {
				require.Nil(s.T(), defaultProfile)
			} else {
				require.Equal(s.T(), existingDefault.ID, defaultProfile.ID)
			}

			profiles, err := s.BillingService.ListProfiles(ctx, billing.ListProfilesInput{
				Namespace: ns,
				Page:      pagination.NewPage(1, 20),
			})
			require.NoError(s.T(), err)
			expectedCount := 0
			if tc.ExistingDefault {
				expectedCount++
			}

			if !tc.CreateBillingProfile {
				require.Equal(s.T(), expectedCount, profiles.TotalCount)
				return
			}

			require.Equal(s.T(), expectedCount+1, profiles.TotalCount)
			profile, found := lo.Find(profiles.Items, func(p billing.Profile) bool { return p.Name == "My Invoicing (Auto Collection)" })
			require.True(s.T(), found)
			require.False(s.T(), profile.Default)
			require.Equal(s.T(), billing.DefaultWorkflowConfig, profile.WorkflowConfig)
			require.Equal(s.T(), "OpenMeter", profile.Supplier.Name)
			require.Equal(s.T(), models.Address{
				Country:    lo.ToPtr(models.CountryCode("US")),
				PostalCode: lo.ToPtr("94114"),
			}, profile.Supplier.Address)
			appID := result.App.GetID()
			require.Equal(s.T(), appID, profile.AppReferences.Tax)
			require.Equal(s.T(), appID, profile.AppReferences.Invoicing)
			require.Equal(s.T(), appID, profile.AppReferences.Payment)
		})
	}
}

func (s *ProfileTestSuite) TestCustomInvoicingInstallProfileRollback() {
	// given: a namespace without installed apps or billing profiles.
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("custom_install_rollback")
	failure := errors.New("profile provisioning failed")

	// when: provisioning fails after persisting the new profile.
	_, err := s.AppService.InstallApp(ctx, app.InstallAppV3Input{
		MarketplaceListingID:        app.MarketplaceListingID{Type: app.AppTypeCustomInvoicing},
		Namespace:                   ns,
		Name:                        "My Invoicing",
		CreateDefaultBillingProfile: true,
		CreateDefaultBillingProfileFn: func(ctx context.Context, installedApp app.App) ([]app.CapabilityType, error) {
			_, err := billingprofile.CreateDefault(ctx, s.BillingService, nil, installedApp)
			if err != nil {
				return nil, err
			}

			return nil, failure
		},
	})
	require.ErrorIs(s.T(), err, failure)

	// then: the enclosing installation transaction rolls back both writes.
	apps, err := s.AppService.ListApps(ctx, app.ListAppInput{Namespace: ns, Page: pagination.NewPage(1, 20)})
	require.NoError(s.T(), err)
	require.Empty(s.T(), apps.Items)
	profiles, err := s.BillingService.ListProfiles(ctx, billing.ListProfilesInput{Namespace: ns, Page: pagination.NewPage(1, 20)})
	require.NoError(s.T(), err)
	require.Empty(s.T(), profiles.Items)
}

func (s *ProfileTestSuite) TestV1CustomInvoicingInstallProfileFlag() {
	for _, tc := range []struct {
		Name             string
		Body             string
		ExpectedProfiles int
	}{
		{Name: "omitted flag", Body: `{}`, ExpectedProfiles: 1},
		{Name: "explicit true", Body: `{"createBillingProfile":true}`, ExpectedProfiles: 1},
		{Name: "explicit false", Body: `{"createBillingProfile":false}`},
	} {
		s.Run(tc.Name, func() {
			// given: a Custom Invoicing request using the v1 API's defaulting contract.
			ns := s.GetUniqueNamespace("custom_install_v1")
			handler := apphttpdriver.New(slog.Default(), namespacedriver.StaticNamespaceDecoder(ns), s.AppService, nil, s.BillingService, nil)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/marketplace/listings/custom_invoicing/install", strings.NewReader(tc.Body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			// when: the production HTTP handler installs the app and optionally its profile.
			handler.MarketplaceAppInstall().With(api.AppTypeCustomInvoicing).ServeHTTP(recorder, request)

			// then: all requests succeed, with omitted and true flags provisioning a profile.
			require.Equal(s.T(), http.StatusOK, recorder.Code, recorder.Body.String())
			profiles, err := s.BillingService.ListProfiles(s.T().Context(), billing.ListProfilesInput{Namespace: ns, Page: pagination.NewPage(1, 20)})
			require.NoError(s.T(), err)
			require.Equal(s.T(), tc.ExpectedProfiles, profiles.TotalCount)
		})
	}
}
