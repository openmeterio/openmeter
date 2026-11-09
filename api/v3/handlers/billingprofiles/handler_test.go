package billingprofiles

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	taxcodedb "github.com/openmeterio/openmeter/openmeter/ent/db/taxcode"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

// countingTaxCodeResolver counts BatchResolve calls on top of the production resolver.
type countingTaxCodeResolver = pctestutils.CountingProductionTaxCodeResolver

type HandlerTaxConfigTestSuite struct {
	billingtest.BaseSuite

	namespace string
	appID     app.AppID
	resolver  *countingTaxCodeResolver
	handler   Handler
}

func TestHandlerTaxConfig(t *testing.T) {
	suite.Run(t, new(HandlerTaxConfigTestSuite))
}

func (s *HandlerTaxConfigTestSuite) SetupTest() {
	s.namespace = s.GetUniqueNamespace("profile-taxconfig-v3")
	s.appID = s.InstallSandboxApp(s.T(), s.namespace).GetID()
	s.resolver = &countingTaxCodeResolver{TransactionalTaxCodeResolver: s.TaxCodeResolver}
	s.handler = New(s.resolveNamespace, s.BillingService, s.resolver)
}

func (s *HandlerTaxConfigTestSuite) resolveNamespace(context.Context) (string, error) {
	return s.namespace, nil
}

func (s *HandlerTaxConfigTestSuite) serve(handler http.Handler, method string, body any) *httptest.ResponseRecorder {
	raw, err := json.Marshal(body)
	s.Require().NoError(err)

	request := httptest.NewRequest(method, "/", bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

func (s *HandlerTaxConfigTestSuite) supplier() api.BillingParty {
	return api.BillingParty{
		Name:      lo.ToPtr("Supplier"),
		Addresses: &api.BillingPartyAddresses{BillingAddress: api.Address{Country: lo.ToPtr(api.CountryCode("US"))}},
	}
}

func (s *HandlerTaxConfigTestSuite) workflow(defaultTaxConfig *api.BillingTaxConfig) api.BillingWorkflow {
	return api.BillingWorkflow{
		Tax: &api.BillingWorkflowTaxSettings{DefaultTaxConfig: defaultTaxConfig},
	}
}

func (s *HandlerTaxConfigTestSuite) update(profileID string, defaultTaxConfig *api.BillingTaxConfig) *httptest.ResponseRecorder {
	return s.serve(s.handler.UpdateBillingProfile().With(profileID), http.MethodPut, api.UpsertBillingProfileRequest{
		Name:     "Profile",
		Default:  true,
		Supplier: s.supplier(),
		Workflow: s.workflow(defaultTaxConfig),
	})
}

func (s *HandlerTaxConfigTestSuite) storedDefaultTaxConfig(profileID string) *productcatalog.TaxCodeConfig {
	profile, err := s.BillingService.GetProfile(s.T().Context(), billing.GetProfileInput{
		Profile: billing.ProfileID{Namespace: s.namespace, ID: profileID},
	})
	s.Require().NoError(err)

	return profile.WorkflowConfig.Invoicing.DefaultTaxConfig
}

func (s *HandlerTaxConfigTestSuite) taxCodeCount() int {
	count, err := s.DBClient.TaxCode.Query().Where(taxcodedb.Namespace(s.namespace)).Count(s.T().Context())
	s.Require().NoError(err)

	return count
}

func (s *HandlerTaxConfigTestSuite) TestCreateRejectsTaxCode() {
	createBody := api.CreateBillingProfileRequest{
		Name:     "Profile",
		Supplier: s.supplier(),
		Apps: api.BillingProfileAppReferences{
			Tax:       api.BillingAppReference{Id: s.appID.ID},
			Invoicing: api.BillingAppReference{Id: s.appID.ID},
			Payment:   api.BillingAppReference{Id: s.appID.ID},
		},
	}

	// given: a tax code that exists in the namespace
	taxCode := s.TaxCodeForStripeCode(s.T().Context(), s.namespace, "txcd_10000000")
	taxCodes := s.taxCodeCount()

	s.Run("deprecated stripe code", func() {
		// when: a profile is created with a deprecated stripe code
		body := createBody
		body.Workflow = s.workflow(&api.BillingTaxConfig{Stripe: &api.BillingTaxConfigStripe{Code: "txcd_20000000"}})
		rec := s.serve(s.handler.CreateBillingProfile(), http.MethodPost, body)

		// then: it's rejected and no tax code is created
		s.Require().Equal(http.StatusBadRequest, rec.Code, rec.Body.String())
		s.Equal(taxCodes, s.taxCodeCount())
	})

	s.Run("tax_code.id", func() {
		// when: a profile is created referencing an existing tax code
		body := createBody
		body.Workflow = s.workflow(&api.BillingTaxConfig{TaxCode: &api.TaxCodeReference{Id: taxCode.ID}})
		rec := s.serve(s.handler.CreateBillingProfile(), http.MethodPost, body)

		// then: it's rejected too
		s.Require().Equal(http.StatusBadRequest, rec.Code, rec.Body.String())
	})
}

func (s *HandlerTaxConfigTestSuite) TestUpdateGateAndResolution() {
	ctx := s.T().Context()

	// given: a legacy profile whose default references a tax code mapped to txcd_10000000
	taxCode := s.TaxCodeForStripeCode(ctx, s.namespace, "txcd_10000000")
	other := s.TaxCodeForStripeCode(ctx, s.namespace, "txcd_20000000")
	profile := s.ProvisionBillingProfile(ctx, s.namespace, s.appID)
	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: taxCode.ID,
	})
	taxCodes := s.taxCodeCount()

	s.Run("stripe-only echo", func() {
		// when: the deprecated stripe code is echoed alone
		rec := s.update(profile.ID, &api.BillingTaxConfig{
			Behavior: lo.ToPtr(api.BillingTaxBehaviorExclusive),
			Stripe:   &api.BillingTaxConfigStripe{Code: "txcd_10000000"},
		})
		s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())

		// then: the stored tax code is kept, rendered with its deprecated provider codes
		var resp api.BillingProfile
		s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &resp))
		s.Require().NotNil(resp.Workflow.Tax)
		s.Equal(&api.BillingTaxConfig{
			Behavior:          lo.ToPtr(api.BillingTaxBehaviorExclusive),
			ExternalInvoicing: &api.BillingTaxConfigExternalInvoicing{Code: "txcd_10000000"},
			Stripe:            &api.BillingTaxConfigStripe{Code: "txcd_10000000"},
			TaxCode:           &api.TaxCodeReference{Id: taxCode.ID},
			TaxCodeId:         lo.ToPtr(taxCode.ID),
		}, resp.Workflow.Tax.DefaultTaxConfig)

		s.Equal(taxCode.ID, s.storedDefaultTaxConfig(profile.ID).TaxCodeID)
		s.Equal(taxCodes, s.taxCodeCount())
	})

	s.Run("tax_code.id echo overriding a different tax_code_id", func() {
		// when: tax_code.id echoes the stored code while the deprecated tax_code_id names another
		rec := s.update(profile.ID, &api.BillingTaxConfig{
			TaxCode:   &api.TaxCodeReference{Id: taxCode.ID},
			TaxCodeId: lo.ToPtr(other.ID),
			Stripe:    &api.BillingTaxConfigStripe{Code: "txcd_10000000"},
		})
		s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())

		// then: tax_code.id wins and the stored code is kept
		s.Equal(&productcatalog.TaxCodeConfig{TaxCodeID: taxCode.ID}, s.storedDefaultTaxConfig(profile.ID))
	})

	s.Run("changed tax_code.id", func() {
		// when: tax_code.id names another tax code
		rec := s.update(profile.ID, &api.BillingTaxConfig{
			TaxCode: &api.TaxCodeReference{Id: other.ID},
			Stripe:  &api.BillingTaxConfigStripe{Code: "txcd_10000000"},
		})

		// then: the update is rejected and the stored code is kept
		s.Require().Equal(http.StatusBadRequest, rec.Code, rec.Body.String())
		s.Equal(taxCode.ID, s.storedDefaultTaxConfig(profile.ID).TaxCodeID)
	})

	s.Run("changed stripe code", func() {
		// when: the deprecated stripe code is changed
		rec := s.update(profile.ID, &api.BillingTaxConfig{
			Stripe: &api.BillingTaxConfigStripe{Code: "txcd_30000000"},
		})

		// then: the update is rejected before the code is resolved
		s.Require().Equal(http.StatusBadRequest, rec.Code, rec.Body.String())
		s.Equal(taxCodes, s.taxCodeCount())
	})

	s.Run("removal", func() {
		// when: the default is left out
		rec := s.update(profile.ID, nil)
		s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())

		// then: it's removed
		s.Nil(s.storedDefaultTaxConfig(profile.ID))
	})
}

func (s *HandlerTaxConfigTestSuite) TestListResolvesTaxCodesOnce() {
	ctx := s.T().Context()

	// given: two profiles whose defaults reference different tax codes
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

	wantStripeCodes := map[string]string{
		first.ID:  "txcd_10000000",
		second.ID: "txcd_20000000",
	}

	// when: the profiles are listed
	recorder := httptest.NewRecorder()
	s.handler.ListBillingProfiles().With(api.ListBillingProfilesParams{}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	s.Require().Equal(http.StatusOK, recorder.Code, recorder.Body.String())

	// then: each shows its deprecated stripe code, from one tax code lookup
	var resp api.BillingProfilePagePaginatedResponse
	s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
	s.Require().Len(resp.Data, 2)

	for _, item := range resp.Data {
		s.Require().NotNil(item.Workflow.Tax)
		s.Require().NotNil(item.Workflow.Tax.DefaultTaxConfig)
		s.Equal(&api.BillingTaxConfigStripe{Code: wantStripeCodes[item.Id]}, item.Workflow.Tax.DefaultTaxConfig.Stripe)
	}

	s.Equal(1, s.resolver.BatchResolveCalls)
}
