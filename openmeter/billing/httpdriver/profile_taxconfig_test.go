package httpdriver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	taxcodedb "github.com/openmeterio/openmeter/openmeter/ent/db/taxcode"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

// countingTaxCodeResolver counts BatchResolve calls on top of the production resolver.
type countingTaxCodeResolver = pctestutils.CountingProductionTaxCodeResolver

type ProfileTaxConfigTestSuite struct {
	billingtest.BaseSuite

	namespace string
	appID     app.AppID
	resolver  *countingTaxCodeResolver
	handler   Handler
}

func TestProfileTaxConfig(t *testing.T) {
	suite.Run(t, new(ProfileTaxConfigTestSuite))
}

func (s *ProfileTaxConfigTestSuite) SetupTest() {
	s.namespace = s.GetUniqueNamespace("profile-taxconfig-v1")
	s.appID = s.InstallSandboxApp(s.T(), s.namespace).GetID()
	s.resolver = &countingTaxCodeResolver{TransactionalTaxCodeResolver: s.TaxCodeResolver}
	s.handler = New(testutils.NewDiscardLogger(s.T()), namespacedriver.StaticNamespaceDecoder(s.namespace),
		config.BillingFeatureSwitchesConfiguration{}, s.BillingService, s.AppService, nil, nil,
		config.CreditsConfiguration{}, nil, s.resolver)
}

func (s *ProfileTaxConfigTestSuite) supplier() api.BillingParty {
	return api.BillingParty{
		Name:      lo.ToPtr("Supplier"),
		Addresses: &[]api.Address{{Country: lo.ToPtr("US")}},
	}
}

func (s *ProfileTaxConfigTestSuite) serve(handler http.Handler, method string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader

	if body != nil {
		raw, err := json.Marshal(body)
		s.Require().NoError(err)

		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}

	request := httptest.NewRequest(method, "/", reader)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

func (s *ProfileTaxConfigTestSuite) updateBody(isDefault bool, defaultTaxConfig *api.TaxConfig) api.BillingProfileReplaceUpdateWithWorkflow {
	return api.BillingProfileReplaceUpdateWithWorkflow{
		Name:     "Profile",
		Default:  isDefault,
		Supplier: s.supplier(),
		Workflow: api.BillingWorkflow{
			Invoicing: &api.BillingWorkflowInvoicingSettings{DefaultTaxConfig: defaultTaxConfig},
		},
	}
}

func (s *ProfileTaxConfigTestSuite) updateDefaultTaxConfig(profileID string, defaultTaxConfig *api.TaxConfig) *httptest.ResponseRecorder {
	return s.serve(s.handler.UpdateProfile().With(profileID), http.MethodPut, s.updateBody(true, defaultTaxConfig))
}

func (s *ProfileTaxConfigTestSuite) createBody(defaultTaxConfig *api.TaxConfig) api.BillingProfileCreate {
	return api.BillingProfileCreate{
		Name:     "Profile",
		Supplier: s.supplier(),
		Apps: api.BillingProfileAppsCreate{
			Tax:       s.appID.ID,
			Invoicing: s.appID.ID,
			Payment:   s.appID.ID,
		},
		Workflow: api.BillingWorkflowCreate{
			Invoicing: &api.BillingWorkflowInvoicingSettings{DefaultTaxConfig: defaultTaxConfig},
		},
	}
}

func (s *ProfileTaxConfigTestSuite) storedDefaultTaxConfig(profileID string) *productcatalog.TaxCodeConfig {
	profile, err := s.BillingService.GetProfile(s.T().Context(), billing.GetProfileInput{
		Profile: billing.ProfileID{Namespace: s.namespace, ID: profileID},
	})
	s.Require().NoError(err)

	return profile.WorkflowConfig.Invoicing.DefaultTaxConfig
}

func (s *ProfileTaxConfigTestSuite) taxCodeCount() int {
	count, err := s.DBClient.TaxCode.Query().Where(taxcodedb.Namespace(s.namespace)).Count(s.T().Context())
	s.Require().NoError(err)

	return count
}

// seedLegacyProfile provisions the default profile with a default tax config referencing taxCodeID,
// the way rows written before the deprecation look.
func (s *ProfileTaxConfigTestSuite) seedLegacyProfile(taxCodeID string) *billing.Profile {
	ctx := s.T().Context()

	profile := s.ProvisionBillingProfile(ctx, s.namespace, s.appID)
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: taxCodeID,
	})

	return profile
}

func (s *ProfileTaxConfigTestSuite) TestUpdateDeprecationGateAndResolution() {
	// given: a legacy profile whose default references a tax code mapped to txcd_10000000
	taxCode := s.TaxCodeForStripeCode(s.T().Context(), s.namespace, "txcd_10000000")
	profile := s.seedLegacyProfile(taxCode.ID)
	taxCodes := s.taxCodeCount()

	s.Run("legacy stripe-only echo", func() {
		// when: a legacy client echoes only stripe.code
		rec := s.updateDefaultTaxConfig(profile.ID, &api.TaxConfig{
			Behavior: lo.ToPtr(api.TaxBehaviorInclusive),
			Stripe:   &api.StripeTaxConfig{Code: "txcd_10000000"},
		})
		s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())

		// then: the stored tax code is kept and shown with its stripe code; nothing is created
		var resp api.BillingProfile
		s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &resp))
		s.Equal(&api.TaxConfig{
			Behavior:  lo.ToPtr(api.TaxBehaviorInclusive),
			Stripe:    &api.StripeTaxConfig{Code: "txcd_10000000"},
			TaxCodeId: lo.ToPtr(taxCode.ID),
		}, resp.Workflow.Invoicing.DefaultTaxConfig)

		s.Equal(&productcatalog.TaxCodeConfig{
			Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			TaxCodeID: taxCode.ID,
		}, s.storedDefaultTaxConfig(profile.ID))
		s.Equal(taxCodes, s.taxCodeCount())
	})

	s.Run("changed stripe code", func() {
		// when: stripe.code is changed
		rec := s.updateDefaultTaxConfig(profile.ID, &api.TaxConfig{
			Stripe: &api.StripeTaxConfig{Code: "txcd_20000000"},
		})

		// then: the update is rejected before the code is resolved, so no tax code is created
		s.Require().Equal(http.StatusBadRequest, rec.Code, rec.Body.String())
		s.Contains(rec.Body.String(), "deprecated")

		s.Equal(taxCode.ID, s.storedDefaultTaxConfig(profile.ID).TaxCodeID)
		s.Equal(taxCodes, s.taxCodeCount())
	})

	s.Run("behavior change with full echo", func() {
		// when: taxCodeId and stripe.code are echoed with a new behavior
		rec := s.updateDefaultTaxConfig(profile.ID, &api.TaxConfig{
			Behavior:  lo.ToPtr(api.TaxBehaviorExclusive),
			Stripe:    &api.StripeTaxConfig{Code: "txcd_10000000"},
			TaxCodeId: lo.ToPtr(taxCode.ID),
		})
		s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())

		// then: only the behavior changes
		s.Equal(&productcatalog.TaxCodeConfig{
			Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			TaxCodeID: taxCode.ID,
		}, s.storedDefaultTaxConfig(profile.ID))
	})

	s.Run("stripe code dropped, tax code ID echoed", func() {
		// when: stripe.code is dropped while taxCodeId is echoed
		rec := s.updateDefaultTaxConfig(profile.ID, &api.TaxConfig{
			Behavior:  lo.ToPtr(api.TaxBehaviorExclusive),
			TaxCodeId: lo.ToPtr(taxCode.ID),
		})
		s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())

		// then: the pair is removed, the behavior is kept
		s.Equal(&productcatalog.TaxCodeConfig{
			Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		}, s.storedDefaultTaxConfig(profile.ID))
	})
}

func (s *ProfileTaxConfigTestSuite) TestStripeOnlyEchoKeepsDeletedStoredTaxCode() {
	ctx := s.T().Context()

	// given: a legacy profile whose default tax code was deleted afterwards
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), s.namespace)
	taxCode := s.TaxCodeForStripeCode(ctx, s.namespace, "txcd_10000000")
	profile := s.seedLegacyProfile(taxCode.ID)
	s.Require().NoError(s.TaxCodeEnv.Service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{NamespacedID: taxCode.NamespacedID}))
	taxCodes := s.taxCodeCount()

	// when: a legacy client echoes the stripe code it read
	rec := s.updateDefaultTaxConfig(profile.ID, &api.TaxConfig{
		Behavior: lo.ToPtr(api.TaxBehaviorInclusive),
		Stripe:   &api.StripeTaxConfig{Code: "txcd_10000000"},
	})

	// then: the no-op update succeeds, keeps the deleted tax code and creates no new one
	s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())
	s.Equal(taxCode.ID, s.storedDefaultTaxConfig(profile.ID).TaxCodeID)
	s.Equal(taxCodes, s.taxCodeCount())
}

func (s *ProfileTaxConfigTestSuite) TestStripeOnlyEchoKeepsStoredTaxCodeWithSharedMapping() {
	ctx := s.T().Context()

	// given: an older tax code with the same Stripe mapping, which a lookup by mapping prefers,
	// and a legacy profile referencing the newer one
	mapping := taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"}}
	preferred := s.TaxCodeEnv.CreateTaxCode(s.T(), s.namespace, taxcode.CreateTaxCodeInput{Key: "preferred", AppMappings: mapping})
	stored := s.TaxCodeEnv.CreateTaxCode(s.T(), s.namespace, taxcode.CreateTaxCodeInput{Key: "stored", AppMappings: mapping})

	byMapping, err := s.TaxCodeEnv.Service.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput{
		Namespace: s.namespace, AppType: app.AppTypeStripe, TaxCode: "txcd_10000000",
	})
	s.Require().NoError(err)
	s.Require().Equal(preferred.ID, byMapping.ID, "precondition: the mapping lookup prefers the other tax code")

	profile := s.seedLegacyProfile(stored.ID)
	taxCodes := s.taxCodeCount()

	// when: a legacy client echoes the stripe code it read
	rec := s.updateDefaultTaxConfig(profile.ID, &api.TaxConfig{
		Behavior: lo.ToPtr(api.TaxBehaviorInclusive),
		Stripe:   &api.StripeTaxConfig{Code: "txcd_10000000"},
	})

	// then: the profile keeps its own tax code and nothing is created
	s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())
	s.Equal(stored.ID, s.storedDefaultTaxConfig(profile.ID).TaxCodeID)
	s.Equal(taxCodes, s.taxCodeCount())
}

func (s *ProfileTaxConfigTestSuite) TestFailedUpdateLeavesNoTaxCode() {
	ctx := s.T().Context()

	// given: a legacy default profile
	taxCode := s.TaxCodeForStripeCode(ctx, s.namespace, "txcd_10000000")
	profile := s.seedLegacyProfile(taxCode.ID)
	taxCodes := s.taxCodeCount()

	s.Run("service error after the tax config is mapped", func() {
		// when: the update echoes the stripe code but tries to unset the default flag, which the
		// service rejects after the tax config was mapped
		rec := s.serve(s.handler.UpdateProfile().With(profile.ID), http.MethodPut, s.updateBody(false, &api.TaxConfig{
			Stripe: &api.StripeTaxConfig{Code: "txcd_10000000"},
		}))

		// then: the update fails and leaves the profile and tax codes as they were
		s.Require().Equal(http.StatusBadRequest, rec.Code, rec.Body.String())
		s.Equal(taxCode.ID, s.storedDefaultTaxConfig(profile.ID).TaxCodeID)
		s.Equal(taxCodes, s.taxCodeCount())
	})

	s.Run("tax code resolved in WithResolver, then rejected by the service", func() {
		// when: a new stripe code is resolved inside the handlers' transaction and the resulting
		// ID is then rejected by the service's ID gate
		err := s.TaxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
			created, err := resolver.ResolveOrBackfillTaxCode(ctx, productcatalog.ResolveOrBackfillTaxCodeInput{
				Namespace:  s.namespace,
				StripeCode: "txcd_30000000",
			})
			if err != nil {
				return err
			}

			update := profile.BaseProfile
			update.AppReferences = nil
			update.WorkflowConfig.Invoicing.DefaultTaxConfig = &productcatalog.TaxCodeConfig{TaxCodeID: created.ID}

			_, err = s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(update))

			return err
		})

		// then: the created tax code is rolled back with the failed update
		var validationErr billing.ValidationError
		s.Require().True(errors.As(err, &validationErr), "expected billing.ValidationError, got %v", err)
		s.Equal(taxCodes, s.taxCodeCount())
		s.Equal(taxCode.ID, s.storedDefaultTaxConfig(profile.ID).TaxCodeID)
	})
}

func (s *ProfileTaxConfigTestSuite) TestCreateRejectsTaxCode() {
	s.Run("stripe code on create", func() {
		// when: a stripe.code is sent on create
		rec := s.serve(s.handler.CreateProfile(), http.MethodPost, s.createBody(&api.TaxConfig{
			Stripe: &api.StripeTaxConfig{Code: "txcd_10000000"},
		}))

		// then: it's rejected and no tax code is created
		s.Require().Equal(http.StatusBadRequest, rec.Code, rec.Body.String())
		s.Contains(rec.Body.String(), "deprecated")
		s.Equal(0, s.taxCodeCount())
	})

	s.Run("behavior only with an empty stripe code", func() {
		// when: a behavior is sent with an empty stripe.code
		rec := s.serve(s.handler.CreateProfile(), http.MethodPost, s.createBody(&api.TaxConfig{
			Behavior: lo.ToPtr(api.TaxBehaviorExclusive),
			Stripe:   &api.StripeTaxConfig{Code: ""},
		}))
		s.Require().Equal(http.StatusCreated, rec.Code, rec.Body.String())

		// then: only the behavior is stored and no tax code is created
		var resp api.BillingProfile
		s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &resp))
		s.Equal(&api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, resp.Workflow.Invoicing.DefaultTaxConfig)
		s.Equal(&productcatalog.TaxCodeConfig{
			Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		}, s.storedDefaultTaxConfig(resp.Id))
		s.Equal(0, s.taxCodeCount())
	})
}

func (s *ProfileTaxConfigTestSuite) TestListsResolveTaxCodesOnce() {
	ctx := s.T().Context()

	// given: two profiles with defaults referencing different tax codes, and a customer on each
	first := s.ProvisionBillingProfile(ctx, s.namespace, s.appID)
	firstTaxCode := s.TaxCodeForStripeCode(ctx, s.namespace, "txcd_10000000")
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, first.ProfileID(), &productcatalog.TaxCodeConfig{TaxCodeID: firstTaxCode.ID})

	secondInput := billingtest.MinimalCreateProfileInputTemplate(s.appID)
	secondInput.Namespace = s.namespace
	secondInput.Default = false
	second, err := s.BillingService.CreateProfile(ctx, secondInput)
	s.Require().NoError(err)
	secondTaxCode := s.TaxCodeForStripeCode(ctx, s.namespace, "txcd_20000000")
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, second.ProfileID(), &productcatalog.TaxCodeConfig{TaxCodeID: secondTaxCode.ID})

	s.CreateTestCustomer(s.namespace, "first")
	pinned := s.CreateTestCustomer(s.namespace, "second")
	_, err = s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{
		Customer:  pinned.GetID(),
		ProfileID: second.ID,
	})
	s.Require().NoError(err)

	wantStripeCodes := map[string]string{
		first.ID:  "txcd_10000000",
		second.ID: "txcd_20000000",
	}

	s.Run("profiles", func() {
		// when: profiles are listed
		s.resolver.BatchResolveCalls = 0

		recorder := httptest.NewRecorder()
		s.handler.ListProfiles().With(api.ListBillingProfilesParams{}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		s.Require().Equal(http.StatusOK, recorder.Code, recorder.Body.String())

		// then: each shows its stripe code, from one tax code lookup
		var resp api.BillingProfilePaginatedResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Require().Len(resp.Items, 2)

		for _, item := range resp.Items {
			s.Require().NotNil(item.Workflow.Invoicing.DefaultTaxConfig)
			s.Equal(&api.StripeTaxConfig{Code: wantStripeCodes[item.Id]}, item.Workflow.Invoicing.DefaultTaxConfig.Stripe)
		}

		s.Equal(1, s.resolver.BatchResolveCalls)
	})

	s.Run("customer profiles", func() {
		// when: customer profiles are listed
		s.resolver.BatchResolveCalls = 0

		recorder := httptest.NewRecorder()
		s.handler.ListCustomerOverrides().With(api.ListBillingProfileCustomerOverridesParams{
			IncludeAllCustomers: lo.ToPtr(true),
		}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		s.Require().Equal(http.StatusOK, recorder.Code, recorder.Body.String())

		// then: each effective profile shows its stripe code, from one tax code lookup
		var resp api.BillingProfileCustomerOverrideWithDetailsPaginatedResponse
		s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
		s.Require().Len(resp.Items, 2)

		for _, item := range resp.Items {
			s.Require().NotNil(item.CustomerProfile)
			s.Require().NotNil(item.CustomerProfile.Workflow.Invoicing.DefaultTaxConfig)
			s.Equal(&api.StripeTaxConfig{Code: wantStripeCodes[item.BaseBillingProfileId]}, item.CustomerProfile.Workflow.Invoicing.DefaultTaxConfig.Stripe)
		}

		s.Equal(1, s.resolver.BatchResolveCalls)
	})
}
