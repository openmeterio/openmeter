package billing

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingtaxcodevalidator "github.com/openmeterio/openmeter/openmeter/billing/validators/taxcode"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

type TaxCodeDeleteValidatorTestSuite struct {
	BaseSuite
}

func TestTaxCodeDeleteValidator(t *testing.T) {
	suite.Run(t, new(TaxCodeDeleteValidatorTestSuite))
}

func (s *TaxCodeDeleteValidatorTestSuite) TestProfileWorkflowConfigReferencesBlock() {
	// given:
	// - a billing profile whose workflow config's tax_code_id column points at a tax code
	// when:
	// - ValidateDeleteTaxCode is called for that tax code
	// then:
	// - a conflict error is returned

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("tc-del-val-profile")

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	// create a tax code and stamp it onto the profile's workflow config
	tc, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "test-code",
		Name:      "Test Code",
	})
	s.Require().NoError(err)

	s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxConfig{
		TaxCodeID: &tc.ID,
	})

	validator, err := billingtaxcodevalidator.NewValidator(s.DBClient)
	s.Require().NoError(err)

	err = validator.ValidateDeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{
		NamespacedID: models.NamespacedID{
			Namespace: ns,
			ID:        tc.ID,
		},
	})
	s.Require().Error(err)
	s.True(models.IsGenericConflictError(err), "expected a conflict error, got: %v", err)
}

func (s *TaxCodeDeleteValidatorTestSuite) TestCustomerOverrideReferencesBlock() {
	// given:
	// - a customer override whose tax_code_id column points at a tax code
	// when:
	// - ValidateDeleteTaxCode is called for that tax code
	// then:
	// - a conflict error is returned

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("tc-del-val-override")

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	cust := s.CreateTestCustomer(ns, "tc-del-val-override-cust")

	// create a tax code and reference it from a customer override
	tc, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "override-code",
		Name:      "Override Code",
	})
	s.Require().NoError(err)

	_, err = s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		Invoicing: billing.InvoicingOverrideConfig{
			DefaultTaxConfig: &productcatalog.TaxConfig{
				TaxCodeID: &tc.ID,
			},
		},
	})
	s.Require().NoError(err)

	validator, err := billingtaxcodevalidator.NewValidator(s.DBClient)
	s.Require().NoError(err)

	err = validator.ValidateDeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{
		NamespacedID: models.NamespacedID{
			Namespace: ns,
			ID:        tc.ID,
		},
	})
	s.Require().Error(err)
	s.True(models.IsGenericConflictError(err), "expected a conflict error, got: %v", err)
}

func (s *TaxCodeDeleteValidatorTestSuite) TestUnreferencedTaxCodeAllowed() {
	// given:
	// - a tax code that is not referenced by any billing profile or customer override
	// when:
	// - ValidateDeleteTaxCode is called for that tax code
	// then:
	// - nil is returned

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("tc-del-val-unreferenced")

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	tc, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "unreferenced-code",
		Name:      "Unreferenced Code",
	})
	s.Require().NoError(err)

	validator, err := billingtaxcodevalidator.NewValidator(s.DBClient)
	s.Require().NoError(err)

	err = validator.ValidateDeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{
		NamespacedID: models.NamespacedID{
			Namespace: ns,
			ID:        tc.ID,
		},
	})
	s.Require().NoError(err)
}
