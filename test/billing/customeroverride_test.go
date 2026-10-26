package billing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	legacyapi "github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billinghttpdriver "github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/customer"
	namespacedriver "github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CustomerBillingProfileTestSuite struct {
	BaseSuite
}

func TestCustomerBillingProfile(t *testing.T) {
	suite.Run(t, new(CustomerBillingProfileTestSuite))
}

func (s *CustomerBillingProfileTestSuite) TestUpsertRejectsCrossNamespaceReferences() {
	// given:
	// - customers and billing profiles in two namespaces
	// when:
	// - an override is created using a customer or profile from the other namespace
	// then:
	// - the foreign reference is reported as missing and no override is persisted
	ctx := s.T().Context()
	targetNamespace := s.GetUniqueNamespace("customer-override-target")
	foreignNamespace := s.GetUniqueNamespace("customer-override-foreign")
	targetCustomer := s.CreateTestCustomer(targetNamespace, "target")
	foreignCustomer := s.CreateTestCustomer(foreignNamespace, "foreign")
	targetProfile := s.ProvisionBillingProfile(ctx, targetNamespace, s.InstallSandboxApp(s.T(), targetNamespace).GetID())
	foreignProfile := s.ProvisionBillingProfile(ctx, foreignNamespace, s.InstallSandboxApp(s.T(), foreignNamespace).GetID())

	result, err := s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{
		Customer:  customer.CustomerID{Namespace: targetNamespace, ID: targetCustomer.ID},
		ProfileID: foreignProfile.ID,
	})
	require.ErrorIs(s.T(), err, billing.ErrProfileNotFound)
	require.Empty(s.T(), result)

	result, err = s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{
		Customer:  customer.CustomerID{Namespace: targetNamespace, ID: foreignCustomer.ID},
		ProfileID: targetProfile.ID,
	})
	require.True(s.T(), models.IsGenericNotFoundError(err))
	require.Empty(s.T(), result)

	for _, customerID := range []string{targetCustomer.ID, foreignCustomer.ID} {
		override, err := s.BillingAdapter.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
			Customer: customer.CustomerID{Namespace: targetNamespace, ID: customerID},
		})
		require.NoError(s.T(), err)
		require.Nil(s.T(), override.AssignedProfile)
	}
}

func (s *CustomerBillingProfileTestSuite) TestGetRejectsPersistedCrossNamespaceProfile() {
	// given:
	// - a deliberately corrupted override that references a profile in another namespace
	// when:
	// - the customer override is read
	// then:
	// - the foreign profile is not returned and the profile is reported as missing
	ctx := s.T().Context()
	customerNamespace := s.GetUniqueNamespace("customer-override-read-customer")
	profileNamespace := s.GetUniqueNamespace("customer-override-read-profile")

	customerEntity := s.CreateTestCustomer(customerNamespace, "customer")
	foreignProfile := s.ProvisionBillingProfile(ctx, profileNamespace, s.InstallSandboxApp(s.T(), profileNamespace).GetID())

	_, err := s.DBClient.BillingCustomerOverride.Create().
		SetNamespace(customerNamespace).
		SetCustomerID(customerEntity.ID).
		SetBillingProfileID(foreignProfile.ID).
		Save(ctx)
	require.NoError(s.T(), err)

	result, err := s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
		Customer: customer.CustomerID{
			Namespace: customerNamespace,
			ID:        customerEntity.ID,
		},
	})

	require.ErrorIs(s.T(), err, billing.ErrProfileNotFound)
	require.Empty(s.T(), result)
}

func (s *CustomerBillingProfileTestSuite) TestGetIgnoresHistoricalCrossNamespaceTaxCode() {
	// given: a legacy row references a tax code from another namespace.
	ctx := s.T().Context()
	customerNamespace := s.GetUniqueNamespace("customer-override-tax-customer")
	taxNamespace := s.GetUniqueNamespace("customer-override-tax-code")
	customerEntity := s.CreateTestCustomer(customerNamespace, "customer")
	profile := s.ProvisionBillingProfile(ctx, customerNamespace, s.InstallSandboxApp(s.T(), customerNamespace).GetID())
	foreignTaxCode := s.TaxCodeEnv.ProvisionProviderDefaultTaxCode(s.T(), taxNamespace)

	_, err := s.DBClient.BillingCustomerOverride.Create().
		SetNamespace(customerNamespace).
		SetCustomerID(customerEntity.ID).
		SetTaxCodeID(foreignTaxCode.ID).
		Save(ctx)
	require.NoError(s.T(), err)

	// when: the effective profile is resolved without an active pin.
	result, err := s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
		Customer: customerEntity.GetID(),
	})
	// then: historical fields cannot change the namespace default's workflow.
	require.NoError(s.T(), err)
	require.Nil(s.T(), result.AssignedProfile)
	require.Equal(s.T(), profile.ID, result.EffectiveProfile.ID)
	require.Equal(s.T(), profile.WorkflowConfig, result.EffectiveProfile.WorkflowConfig)
}

func (s *CustomerBillingProfileTestSuite) TestBulkAssignValidatesReferencesAndDeduplicatesCustomers() {
	// given:
	// - customers and billing profiles in two namespaces
	// when:
	// - bulk assignment receives invalid references or a duplicated valid customer
	// then:
	// - invalid references write nothing and the valid customer is assigned once
	ctx := s.T().Context()
	targetNamespace := s.GetUniqueNamespace("customer-override-bulk-target")
	foreignNamespace := s.GetUniqueNamespace("customer-override-bulk-foreign")
	targetCustomer := s.CreateTestCustomer(targetNamespace, "target")
	foreignCustomer := s.CreateTestCustomer(foreignNamespace, "foreign")
	targetProfile := s.ProvisionBillingProfile(ctx, targetNamespace, s.InstallSandboxApp(s.T(), targetNamespace).GetID())
	foreignProfile := s.ProvisionBillingProfile(ctx, foreignNamespace, s.InstallSandboxApp(s.T(), foreignNamespace).GetID())

	err := s.BillingAdapter.BulkAssignCustomersToProfile(ctx, billing.BulkAssignCustomersToProfileInput{
		ProfileID:   foreignProfile.ProfileID(),
		CustomerIDs: []customer.CustomerID{targetCustomer.GetID()},
	})
	require.Error(s.T(), err)

	err = s.BillingAdapter.BulkAssignCustomersToProfile(ctx, billing.BulkAssignCustomersToProfileInput{
		ProfileID:   billing.ProfileID{Namespace: targetNamespace, ID: foreignProfile.ID},
		CustomerIDs: []customer.CustomerID{targetCustomer.GetID()},
	})
	require.ErrorIs(s.T(), err, billing.ErrProfileNotFound)

	err = s.BillingAdapter.BulkAssignCustomersToProfile(ctx, billing.BulkAssignCustomersToProfileInput{
		ProfileID:   targetProfile.ProfileID(),
		CustomerIDs: []customer.CustomerID{{Namespace: targetNamespace, ID: foreignCustomer.ID}},
	})
	require.ErrorIs(s.T(), err, billing.ErrCustomerNotFound)

	override, err := s.BillingAdapter.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
		Customer: targetCustomer.GetID(),
	})
	require.NoError(s.T(), err)
	require.Nil(s.T(), override.AssignedProfile)

	err = s.BillingAdapter.BulkAssignCustomersToProfile(ctx, billing.BulkAssignCustomersToProfileInput{
		ProfileID:   targetProfile.ProfileID(),
		CustomerIDs: []customer.CustomerID{targetCustomer.GetID(), targetCustomer.GetID()},
	})
	require.NoError(s.T(), err)

	override, err = s.BillingAdapter.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
		Customer: targetCustomer.GetID(),
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), override.AssignedProfile)
	require.Equal(s.T(), targetProfile.ID, override.Profile.ID)
}

func (s *CustomerBillingProfileTestSuite) TestFetchNonExistingCustomer() {
	// Given we have a non-existing customer
	nonExistingCustomerID := "non-existing-customer-id"
	ns := "test-ns"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	// When querying the customer's billing profile overrides
	customerEntity, err := s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
		Customer: customer.CustomerID{
			Namespace: ns,
			ID:        nonExistingCustomerID,
		},
	})

	// Then we get a customer not found error
	require.True(s.T(), models.IsGenericNotFoundError(err), "expect a customer not found error")
	require.Empty(s.T(), customerEntity)
}

func (s *CustomerBillingProfileTestSuite) TestCustomerIntegration() {
	// Given we have an existing customer and a default profile
	ns := "test-customer-integration"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	custKey := "johny-the-doe-4"
	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,

		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
			Key:  &custKey,
			BillingAddress: &models.Address{
				City:    lo.ToPtr("Berlin"),
				Country: lo.ToPtr(models.CountryCode("DE")),
			},
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), cust)

	// Let's fetch the customer again to make sure we have truncated timestamps
	cust, err = s.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: ns,
			ID:        cust.ID,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), cust)

	profileInput := MinimalCreateProfileInputTemplate(sandboxApp.GetID())
	profileInput.Namespace = ns

	defaultProfile, err := s.BillingService.CreateProfile(ctx, profileInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), defaultProfile)

	// When querying the customer's billing profile overrides
	customerProfile, err := s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
		Customer: customer.CustomerID{
			Namespace: ns,
			ID:        cust.ID,
		},
		Expand: billing.CustomerBillingProfileExpand{
			Customer: true,
		},
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), customerProfile)

	// Then we get the customer object, name and billing address gets overridden
	require.Equal(s.T(), *cust, *customerProfile.Customer)
}

func (s *CustomerBillingProfileTestSuite) TestGetCustomerApp() {
	ns := "test-get-customer-app"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	// Create a default profile
	defaultProfileCreateInput := MinimalCreateProfileInputTemplate(sandboxApp.GetID())
	defaultProfileCreateInput.Namespace = ns

	defaultProfile, err := s.BillingService.CreateProfile(ctx, defaultProfileCreateInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), defaultProfile)

	// Create a customer
	custKey := "johny-the-doe-6"
	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
			Key:  &custKey,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), cust)

	// Get the customer app
	customerApp, err := s.BillingService.GetCustomerApp(ctx, billing.GetCustomerAppInput{
		CustomerID: cust.GetID(),
		AppType:    sandboxApp.GetType(),
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), customerApp)

	// Then we get the customer app
	require.Equal(s.T(), sandboxApp.GetID(), customerApp.GetID())
}

func (s *CustomerBillingProfileTestSuite) TestListCustomerOverrides() {
	ns := "test-list-customer-overrides"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	// Given we have a default profile and an override profile

	defaultProfileCreateInput := MinimalCreateProfileInputTemplate(sandboxApp.GetID())
	defaultProfileCreateInput.Namespace = ns

	defaultProfile, err := s.BillingService.CreateProfile(ctx, defaultProfileCreateInput)
	require.NoError(s.T(), err)

	overrideProfileCreateInput := MinimalCreateProfileInputTemplate(sandboxApp.GetID())
	overrideProfileCreateInput.Namespace = ns
	overrideProfileCreateInput.Default = false

	overrideProfile, err := s.BillingService.CreateProfile(ctx, overrideProfileCreateInput)
	require.NoError(s.T(), err)

	customers := []*customer.Customer{}
	for _, name := range []string{"custNoOverride", "custOverride", "custPinnedToDefault"} {
		custKey := name
		cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: ns,
			CustomerMutate: customer.CustomerMutate{
				Name: name,
				Key:  &custKey,
			},
		})
		require.NoError(s.T(), err)
		customers = append(customers, cust)
	}

	// Given we have a customer with no override (uses default profile)

	custNoOverride := customers[0]

	// Given we have a customer with an override (uses override profile)

	custOverrideProfile := customers[1]
	_, err = s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{

		Customer:  customer.CustomerID{Namespace: ns, ID: custOverrideProfile.ID},
		ProfileID: overrideProfile.ID,
	})
	require.NoError(s.T(), err)

	// Given we have a customer with an override (uses *default* profile)

	custPinnedToDefaultProfile := customers[2]
	_, err = s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{

		Customer:  customer.CustomerID{Namespace: ns, ID: custPinnedToDefaultProfile.ID},
		ProfileID: defaultProfile.ID,
	})
	require.NoError(s.T(), err)

	// When listing the customer overrides

	tcs := []struct {
		name              string
		listInput         billing.ListCustomerBillingProfilesInput
		expectedCustomers []*customer.Customer
	}{
		{
			// When there's no filter only customers with overrides are returned
			name:      "no filter",
			listInput: billing.ListCustomerBillingProfilesInput{},
			expectedCustomers: []*customer.Customer{
				custOverrideProfile,
				custPinnedToDefaultProfile,
			},
		},
		{
			// When we require all customers, all customers are returned
			name: "all customers",
			listInput: billing.ListCustomerBillingProfilesInput{
				IncludeAllCustomers: true,
			},
			expectedCustomers: customers,
		},
		{
			// When we filter by profile ID, only customers with explicit overrides for that profile are returned
			name: "filter by profile ID",
			listInput: billing.ListCustomerBillingProfilesInput{
				BillingProfiles: []string{defaultProfile.ID},
			},
			expectedCustomers: []*customer.Customer{
				custPinnedToDefaultProfile,
			},
		},
		{
			// When we filter by profile ID, but enable IncludeAllCustomers, for default profile we get all customers that either explicitly use the default profile or not set any profile
			name: "filter by profile ID, include all customers",
			listInput: billing.ListCustomerBillingProfilesInput{
				BillingProfiles:     []string{defaultProfile.ID},
				IncludeAllCustomers: true,
			},
			expectedCustomers: []*customer.Customer{
				custNoOverride,
				custPinnedToDefaultProfile,
			},
		},
		{
			// When we filter by customer name, we get the customers that match the name
			name: "filter by customer name",
			listInput: billing.ListCustomerBillingProfilesInput{
				CustomerName: "override",
			},
			expectedCustomers: []*customer.Customer{
				custOverrideProfile,
			},
		},
		{
			// When we filter by customer name, but don't have any customers that match, we get an empty list
			name: "filter by customer name, no customers match",
			listInput: billing.ListCustomerBillingProfilesInput{
				CustomerName: "noOverride",
			},
			expectedCustomers: []*customer.Customer{},
		},
		{
			// When we filter by customer name and enable IncludeAllCustomers we get a match even if the customer has no override
			name: "filter by customer name, include all customers",
			listInput: billing.ListCustomerBillingProfilesInput{
				CustomerName:        "noOverride",
				IncludeAllCustomers: true,
			},
			expectedCustomers: []*customer.Customer{
				custNoOverride,
			},
		},
		{
			// When we filter by explicit pinnings we only get customers that don't have an explicit override
			name: "filter by explicit pinnings",
			listInput: billing.ListCustomerBillingProfilesInput{
				CustomersWithoutPinnedProfile: true,
			},
			expectedCustomers: []*customer.Customer{
				custNoOverride,
			},
		},
	}

	for _, tc := range tcs {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.listInput.Expand = billing.CustomerBillingProfileExpand{
				Customer: true,
			}

			tc.listInput.Namespace = ns

			customerOverrides, err := s.BillingService.ListCustomerBillingProfiles(ctx, tc.listInput)
			require.NoError(t, err)

			expectedCustomerNames := lo.Map(tc.expectedCustomers, func(c *customer.Customer, _ int) string {
				return c.Name
			})

			actualCustomerNames := lo.Map(customerOverrides.Items, func(co billing.CustomerBillingProfile, _ int) string {
				return co.Customer.Name
			})

			require.ElementsMatch(t, expectedCustomerNames, actualCustomerNames)
		})
	}
}

func (s *CustomerBillingProfileTestSuite) TestPinLifecycleAndDefaultChange() {
	// given: a default A, alternative B, and three customers with distinct selections.
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("profile-selection-lifecycle")
	application := s.InstallSandboxApp(s.T(), ns)
	profileA := s.ProvisionBillingProfile(ctx, ns, application.GetID())
	profileInput := MinimalCreateProfileInputTemplate(application.GetID())
	profileInput.Namespace = ns
	profileInput.Default = false
	profileB, err := s.BillingService.CreateProfile(ctx, profileInput)
	s.Require().NoError(err)
	unpinned := s.CreateTestCustomer(ns, "unpinned")
	pinnedDefault := s.CreateTestCustomer(ns, "pinned-default")
	pinnedOther := s.CreateTestCustomer(ns, "pinned-other")
	selection, err := s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{
		Customer:  pinnedDefault.GetID(),
		ProfileID: profileA.ID,
	})
	s.Require().NoError(err)
	s.Require().NotNil(selection.AssignedProfile)
	originalPin := *selection.AssignedProfile
	s.Equal(profileA.ID, selection.EffectiveProfile.ID)
	_, err = s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{
		Customer:  pinnedOther.GetID(),
		ProfileID: profileB.ID,
	})
	s.Require().NoError(err)

	// when: B becomes default with the same invoicing app type.
	profileB.Default = true
	profileB.AppReferences = nil
	_, err = s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(profileB.BaseProfile))
	s.Require().NoError(err)

	// then: only the unpinned customer follows the new default.
	cases := []struct {
		name        string
		customer    customer.CustomerID
		pinID       string
		effectiveID string
	}{
		{name: "no pin follows B", customer: unpinned.GetID(), effectiveID: profileB.ID},
		{name: "explicit default pin remains A", customer: pinnedDefault.GetID(), pinID: profileA.ID, effectiveID: profileA.ID},
		{name: "other pin remains B", customer: pinnedOther.GetID(), pinID: profileB.ID, effectiveID: profileB.ID},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			result, err := s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{Customer: tc.customer})
			s.Require().NoError(err)
			s.Equal(tc.customer, result.CustomerID)
			s.Equal(tc.effectiveID, result.EffectiveProfile.ID)
			if tc.pinID == "" {
				s.Nil(result.AssignedProfile)
			} else {
				s.Require().NotNil(result.AssignedProfile)
				s.Equal(tc.pinID, result.AssignedProfile.ProfileID)
			}
		})
	}

	// when: the former-default pin is cleared and subsequently set again.
	s.Require().NoError(s.BillingService.ClearCustomerBillingProfile(ctx, billing.ClearCustomerBillingProfileInput{Customer: pinnedDefault.GetID()}))
	cleared, err := s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{Customer: pinnedDefault.GetID()})
	s.Require().NoError(err)
	s.Nil(cleared.AssignedProfile)
	s.Equal(profileB.ID, cleared.EffectiveProfile.ID)
	s.ErrorIs(s.BillingService.ClearCustomerBillingProfile(ctx, billing.ClearCustomerBillingProfileInput{Customer: pinnedDefault.GetID()}), billing.ErrCustomerOverrideNotFound)
	restored, err := s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{
		Customer:  pinnedDefault.GetID(),
		ProfileID: profileA.ID,
	})
	// then: the existing row identity is retained and the pin is active again.
	s.Require().NoError(err)
	s.Require().NotNil(restored.AssignedProfile)
	s.Equal(originalPin.ID, restored.AssignedProfile.ID)
	s.Equal(originalPin.CreatedAt, restored.AssignedProfile.CreatedAt)
	s.Equal(profileA.ID, restored.EffectiveProfile.ID)
}

func (s *CustomerBillingProfileTestSuite) TestMissingDefaultAndInvalidSelection() {
	// given: a customer and a non-default profile, with no namespace default.
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("profile-selection-no-default")
	application := s.InstallSandboxApp(s.T(), ns)
	profileInput := MinimalCreateProfileInputTemplate(application.GetID())
	profileInput.Namespace = ns
	profileInput.Default = false
	profile, err := s.BillingService.CreateProfile(ctx, profileInput)
	s.Require().NoError(err)
	cust := s.CreateTestCustomer(ns, "customer")
	_, err = s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{Customer: cust.GetID()})
	s.ErrorIs(err, billing.ErrDefaultProfileNotFound)

	// when: the customer is pinned, then an empty profile ID is submitted.
	selected, err := s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{Customer: cust.GetID(), ProfileID: profile.ID})
	s.Require().NoError(err)
	s.Equal(profile.ID, selected.EffectiveProfile.ID)
	_, err = s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{Customer: cust.GetID()})
	s.Require().Error(err)

	// then: validation preserves the existing pin; clearing it does not require a default.
	selected, err = s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{Customer: cust.GetID()})
	s.Require().NoError(err)
	s.Equal(profile.ID, selected.AssignedProfile.ProfileID)
	s.Require().NoError(s.BillingService.ClearCustomerBillingProfile(ctx, billing.ClearCustomerBillingProfileInput{Customer: cust.GetID()}))
	_, err = s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{Customer: cust.GetID()})
	s.ErrorIs(err, billing.ErrDefaultProfileNotFound)
	result, err := s.BillingService.ListCustomerBillingProfiles(ctx, billing.ListCustomerBillingProfilesInput{
		Namespace:           ns,
		IncludeAllCustomers: true,
		BillingProfiles:     []string{profile.ID},
	})
	s.Require().NoError(err)
	s.Empty(result.Items)
}

func (s *CustomerBillingProfileTestSuite) TestHistoricalRowsAndListSemantics() {
	// given: no row, an empty legacy row, a deleted pin, and two active pins.
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("profile-selection-history")
	application := s.InstallSandboxApp(s.T(), ns)
	profileA := s.ProvisionBillingProfile(ctx, ns, application.GetID())
	input := MinimalCreateProfileInputTemplate(application.GetID())
	input.Namespace = ns
	input.Default = false
	profileB, err := s.BillingService.CreateProfile(ctx, input)
	s.Require().NoError(err)
	noRow := s.CreateTestCustomer(ns, "no-row")
	empty := s.CreateTestCustomer(ns, "empty")
	deleted := s.CreateTestCustomer(ns, "deleted")
	pinnedA := s.CreateTestCustomer(ns, "pinned-A")
	pinnedB := s.CreateTestCustomer(ns, "pinned-B")
	_, err = s.DBClient.BillingCustomerOverride.Create().SetNamespace(ns).SetCustomerID(empty.ID).
		SetLineCollectionPeriod(datetime.ISODurationString("PT999H")).Save(ctx)
	s.Require().NoError(err)
	for _, cust := range []customer.CustomerID{deleted.GetID(), pinnedB.GetID()} {
		_, err = s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{Customer: cust, ProfileID: profileB.ID})
		s.Require().NoError(err)
	}

	s.Require().NoError(s.BillingService.ClearCustomerBillingProfile(ctx, billing.ClearCustomerBillingProfileInput{Customer: deleted.GetID()}))
	selected, err := s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{Customer: pinnedA.GetID(), ProfileID: profileA.ID})
	s.Require().NoError(err)
	_, err = s.DBClient.BillingCustomerOverride.UpdateOneID(selected.AssignedProfile.ID).
		SetLineCollectionPeriod(datetime.ISODurationString("PT999H")).
		SetInvoiceAutoAdvance(!profileA.WorkflowConfig.Invoicing.AutoAdvance).
		Save(ctx)
	s.Require().NoError(err)

	// when: legacy row states are read, with and without app/customer expansion.
	for _, cust := range []customer.CustomerID{noRow.GetID(), empty.GetID(), deleted.GetID(), pinnedA.GetID(), pinnedB.GetID()} {
		for _, expand := range []billing.CustomerBillingProfileExpand{{}, billing.CustomerBillingProfileExpandAll} {
			result, err := s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{Customer: cust, Expand: expand})
			s.Require().NoError(err)
			if cust == pinnedA.GetID() || cust == pinnedB.GetID() {
				s.Require().NotNil(result.AssignedProfile)
				s.Equal(result.AssignedProfile.ProfileID, result.EffectiveProfile.ID)
			} else {
				s.Nil(result.AssignedProfile)
				s.Equal(profileA.ID, result.EffectiveProfile.ID)
				s.Equal(profileA.WorkflowConfig.Collection, result.EffectiveProfile.WorkflowConfig.Collection)
			}

			if cust == pinnedA.GetID() {
				s.Equal(profileA.WorkflowConfig, result.EffectiveProfile.WorkflowConfig, "historical fields must not modify the selected profile")
			}

			if expand.Apps {
				s.Require().NotNil(result.EffectiveProfile.Apps)
				s.Equal(application.GetID(), result.EffectiveProfile.Apps.Payment.GetID())
			}

			if expand.Customer {
				s.Require().NotNil(result.Customer)
				s.Equal(cust.ID, result.Customer.ID)
			}
		}
	}

	// then: lists use active pins, not row existence, and profile filters use the effective default when requested.
	cases := []struct {
		name  string
		input billing.ListCustomerBillingProfilesInput
		ids   []string
	}{
		{name: "all", input: billing.ListCustomerBillingProfilesInput{IncludeAllCustomers: true}, ids: []string{noRow.ID, empty.ID, deleted.ID, pinnedA.ID, pinnedB.ID}},
		{name: "unpinned", input: billing.ListCustomerBillingProfilesInput{CustomersWithoutPinnedProfile: true}, ids: []string{noRow.ID, empty.ID, deleted.ID}},
		{name: "pins only", input: billing.ListCustomerBillingProfilesInput{}, ids: []string{pinnedA.ID, pinnedB.ID}},
		{name: "effective default", input: billing.ListCustomerBillingProfilesInput{IncludeAllCustomers: true, BillingProfiles: []string{profileA.ID}}, ids: []string{noRow.ID, empty.ID, deleted.ID, pinnedA.ID}},
		{name: "explicit default", input: billing.ListCustomerBillingProfilesInput{BillingProfiles: []string{profileA.ID}}, ids: []string{pinnedA.ID}},
		{name: "other profile", input: billing.ListCustomerBillingProfilesInput{IncludeAllCustomers: true, BillingProfiles: []string{profileB.ID}}, ids: []string{pinnedB.ID}},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			tc.input.Namespace = ns
			result, err := s.BillingService.ListCustomerBillingProfiles(ctx, tc.input)
			s.Require().NoError(err)
			ids := lo.Map(result.Items, func(item billing.CustomerBillingProfile, _ int) string { return item.CustomerID.ID })
			s.ElementsMatch(tc.ids, ids)
		})
	}

	// when: automatic assignment encounters existing empty/deleted rows.
	s.Require().NoError(s.BillingAdapter.BulkAssignCustomersToProfile(ctx, billing.BulkAssignCustomersToProfileInput{
		ProfileID:   profileB.ProfileID(),
		CustomerIDs: []customer.CustomerID{empty.GetID(), deleted.GetID()},
	}))
	// then: both are active pins to B.
	for _, cust := range []customer.CustomerID{empty.GetID(), deleted.GetID()} {
		result, err := s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{Customer: cust})
		s.Require().NoError(err)
		s.Require().NotNil(result.AssignedProfile)
		s.Equal(profileB.ID, result.AssignedProfile.ProfileID)
	}
}

func (s *CustomerBillingProfileTestSuite) TestLegacyPUTSetsAndClearsPin() {
	// given: a default profile, a customer, and the production legacy handler.
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("legacy-profile-selection")
	profile := s.ProvisionBillingProfile(ctx, ns, s.InstallSandboxApp(s.T(), ns).GetID())
	cust := s.CreateTestCustomer(ns, "customer")
	handler := billinghttpdriver.New(testutils.NewDiscardLogger(s.T()), namespacedriver.StaticNamespaceDecoder(ns),
		config.BillingFeatureSwitchesConfiguration{}, s.BillingService, s.AppService, nil, nil, config.CreditsConfiguration{}, nil)

	// when: an explicit reference pins the default, then omission clears it twice.
	cases := []struct {
		name   string
		body   string
		pinned bool
	}{
		{name: "set", body: `{"billingProfileId":"` + profile.ID + `"}`, pinned: true},
		{name: "clear", body: `{}`},
		{name: "already clear", body: `{}`},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			request := httptest.NewRequest(http.MethodPut, "/api/v1/billing/customers/"+cust.ID, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			handler.UpsertCustomerOverride().With(billinghttpdriver.UpsertCustomerOverrideParams{CustomerId: cust.ID}).ServeHTTP(recorder, request)
			// then: the legacy envelope exposes the persisted pin and the effective profile independently.
			s.Require().Equal(http.StatusOK, recorder.Code, recorder.Body.String())
			var response legacyapi.BillingProfileCustomerOverrideWithDetails
			s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &response))
			s.Equal(profile.ID, response.BaseBillingProfileId)
			if tc.pinned {
				s.Require().NotNil(response.CustomerOverride)
				s.Equal(cust.ID, response.CustomerOverride.CustomerId)
				s.Equal(profile.ID, *response.CustomerOverride.BillingProfileId)
			} else {
				s.Nil(response.CustomerOverride)
			}
		})
	}
}

func (s *CustomerBillingProfileTestSuite) TestLegacyPUTMissingDefaultPreservesPin() {
	// given: a pinned customer in a namespace with no default profile.
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("legacy-profile-missing-default")
	application := s.InstallSandboxApp(s.T(), ns)
	profileInput := MinimalCreateProfileInputTemplate(application.GetID())
	profileInput.Namespace = ns
	profileInput.Default = false
	profile, err := s.BillingService.CreateProfile(ctx, profileInput)
	s.Require().NoError(err)
	cust := s.CreateTestCustomer(ns, "customer")
	_, err = s.BillingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{
		Customer:  cust.GetID(),
		ProfileID: profile.ID,
	})
	s.Require().NoError(err)
	handler := billinghttpdriver.New(testutils.NewDiscardLogger(s.T()), namespacedriver.StaticNamespaceDecoder(ns),
		config.BillingFeatureSwitchesConfiguration{}, s.BillingService, s.AppService, nil, nil, config.CreditsConfiguration{}, nil)

	// when: PUT clears the pin but cannot resolve a profile for its response.
	request := httptest.NewRequest(http.MethodPut, "/api/v1/billing/customers/"+cust.ID, strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.UpsertCustomerOverride().With(billinghttpdriver.UpsertCustomerOverrideParams{CustomerId: cust.ID}).ServeHTTP(recorder, request)

	// then: the request fails without changing the original pin.
	s.Require().Equal(http.StatusNotFound, recorder.Code, recorder.Body.String())
	selected, err := s.BillingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{Customer: cust.GetID()})
	s.Require().NoError(err)
	s.Require().NotNil(selected.AssignedProfile)
	s.Equal(profile.ID, selected.AssignedProfile.ProfileID)
}
