package credits

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	creditpurchaseadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/adapter"
	creditpurchaseservice "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service"
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	creditgrant "github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	creditgrantservice "github.com/openmeterio/openmeter/openmeter/billing/creditgrant/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestCreditGrantTestSuite(t *testing.T) {
	suite.Run(t, new(CreditGrantTestSuite))
}

type CreditGrantTestSuite struct {
	BaseSuite

	CreditPurchaseService creditpurchase.Service
	CreditGrantService    creditgrant.Service
}

func (s *CreditGrantTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	logger := omtestutils.NewLogger(s.T())
	metaAdapter, err := metaadapter.New(metaadapter.Config{
		Client: s.DBClient,
		Logger: logger,
	})
	s.Require().NoError(err)

	lineageAdapter, err := lineageadapter.New(lineageadapter.Config{
		Client: s.DBClient,
	})
	s.Require().NoError(err)

	lineageService, err := lineageservice.New(lineageservice.Config{
		Adapter: lineageAdapter,
	})
	s.Require().NoError(err)

	creditPurchaseAdapter, err := creditpurchaseadapter.New(creditpurchaseadapter.Config{
		Client:      s.DBClient,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	s.Require().NoError(err)

	s.CreditPurchaseService, err = creditpurchaseservice.New(creditpurchaseservice.Config{
		Adapter:     creditPurchaseAdapter,
		Handler:     ledgerchargeadapter.NewCreditPurchaseHandler(s.Ledger, s.BalanceQuerier, s.LedgerResolver, s.LedgerAccountService),
		Lineage:     lineageService,
		MetaAdapter: metaAdapter,
	})
	s.Require().NoError(err)

	svc, err := creditgrantservice.New(creditgrantservice.Config{
		CreditPurchaseService: s.CreditPurchaseService,
		ChargesService:        s.Charges,
		CustomerService:       s.CustomerService,
	})
	s.Require().NoError(err)

	s.CreditGrantService = svc
}

func (s *CreditGrantTestSuite) TestCreateInvoiceFundedCreatesInvoiceArtifacts() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("creditgrant-service-invoice-funded")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	now := datetime.MustParseTimeInLocation(s.T(), "2026-04-17T11:23:53Z", time.UTC).AsTime()
	clock.SetTime(now)

	grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "$10.00 grant for $10.00 charge",
		Description:   lo.ToPtr("A $10.00 grant for $10.00 charge available immediately on grant."),
		Currency:      USD,
		Amount:        alpacadecimal.NewFromInt(10),
		Priority:      lo.ToPtr(int16(10)),
		FundingMethod: creditgrant.FundingMethodInvoice,
		Purchase: &creditgrant.PurchaseTerms{
			Currency:         USD,
			PerUnitCostBasis: lo.ToPtr(alpacadecimal.NewFromInt(1)),
		},
	})
	s.Require().NoError(err)
	s.Equal(creditpurchase.SettlementTypeInvoice, grant.Intent.Settlement.Type())
	s.Equal(creditpurchase.StatusActive, grant.Status)
	s.NotNil(grant.Realizations.CreditGrantRealization)

	standardInvoices, err := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
		Namespaces: []string{ns},
		Expand:     billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	s.Len(standardInvoices.Items, 1)

	invoice := standardInvoices.Items[0]
	s.Equal(cust.ID, invoice.Customer.CustomerID)
	s.Len(invoice.Lines.OrEmpty(), 1)
	s.Equal(grant.ID, *invoice.Lines.OrEmpty()[0].ChargeID)

	gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespaces: []string{ns},
		Customers:  []string{cust.ID},
		Expand:     []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
	})
	s.Require().NoError(err)
	s.Len(gatheringInvoices.Items, 0)
}

func (s *CreditGrantTestSuite) TestCreatePromotionalGrant() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("creditgrant-service-promotional")

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	now := datetime.MustParseTimeInLocation(s.T(), "2026-04-17T11:23:53Z", time.UTC).AsTime()
	clock.SetTime(now)

	grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "Promotional grant",
		Description:   lo.ToPtr("Promotional credit grant"),
		Currency:      USD,
		Amount:        alpacadecimal.NewFromInt(25),
		Priority:      lo.ToPtr(int16(15)),
		FundingMethod: creditgrant.FundingMethodNone,
	})
	s.Require().NoError(err)

	s.Equal(creditpurchase.SettlementTypePromotional, grant.Intent.Settlement.Type())
	s.Equal(creditpurchase.StatusFinal, grant.Status)
	s.NotNil(grant.Realizations.CreditGrantRealization)
	s.Nil(grant.Realizations.ExternalPaymentSettlement)
	s.Nil(grant.Realizations.InvoiceSettlement)

	gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespaces: []string{ns},
		Customers:  []string{cust.ID},
		Expand:     []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
	})
	s.Require().NoError(err)
	s.Len(gatheringInvoices.Items, 0)
}

func (s *CreditGrantTestSuite) TestCreateExternalGrantAndSettle() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("creditgrant-service-external")

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	now := datetime.MustParseTimeInLocation(s.T(), "2026-04-17T11:23:53Z", time.UTC).AsTime()
	clock.SetTime(now)

	grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "External grant",
		Description:   lo.ToPtr("External credit grant"),
		Currency:      USD,
		Amount:        alpacadecimal.NewFromInt(30),
		Priority:      lo.ToPtr(int16(20)),
		FundingMethod: creditgrant.FundingMethodExternal,
		Purchase: &creditgrant.PurchaseTerms{
			Currency:           USD,
			PerUnitCostBasis:   lo.ToPtr(alpacadecimal.NewFromFloat(0.5)),
			AvailabilityPolicy: lo.ToPtr(creditpurchase.CreatedInitialPaymentSettlementStatus),
		},
	})
	s.Require().NoError(err)

	s.Equal(creditpurchase.SettlementTypeExternal, grant.Intent.Settlement.Type())
	s.Equal(creditpurchase.StatusActive, grant.Status)
	s.NotNil(grant.Realizations.CreditGrantRealization)
	s.Nil(grant.Realizations.ExternalPaymentSettlement)

	grant, err = s.CreditGrantService.UpdateExternalSettlement(ctx, creditgrant.UpdateExternalSettlementInput{
		Namespace:    ns,
		CustomerID:   cust.ID,
		ChargeID:     grant.ID,
		TargetStatus: payment.StatusAuthorized,
	})
	s.Require().NoError(err)
	s.Equal(creditpurchase.StatusActive, grant.Status)
	s.NotNil(grant.Realizations.ExternalPaymentSettlement)
	s.Equal(payment.StatusAuthorized, grant.Realizations.ExternalPaymentSettlement.Status)

	grant, err = s.CreditGrantService.UpdateExternalSettlement(ctx, creditgrant.UpdateExternalSettlementInput{
		Namespace:    ns,
		CustomerID:   cust.ID,
		ChargeID:     grant.ID,
		TargetStatus: payment.StatusSettled,
	})
	s.Require().NoError(err)

	s.Equal(creditpurchase.StatusFinal, grant.Status)
	s.NotNil(grant.Realizations.ExternalPaymentSettlement)
	s.Equal(payment.StatusSettled, grant.Realizations.ExternalPaymentSettlement.Status)
}

func (s *CreditGrantTestSuite) TestListCreditGrants() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("creditgrant-service-list")

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	now := datetime.MustParseTimeInLocation(s.T(), "2026-04-17T11:23:53Z", time.UTC).AsTime()
	clock.SetTime(now)

	firstGrant := s.mustCreatePromotionalCreditGrant(ctx, ns, cust.GetID(), "list-grant-1", alpacadecimal.NewFromInt(10))
	secondGrant := s.mustCreatePromotionalCreditGrant(ctx, ns, cust.GetID(), "list-grant-2", alpacadecimal.NewFromInt(20))

	result, err := s.CreditGrantService.List(ctx, creditgrant.ListInput{
		Namespace:  ns,
		CustomerID: cust.ID,
	})
	s.Require().NoError(err)
	s.Len(result.Items, 2)

	ids := lo.Map(result.Items, func(item creditpurchase.Charge, _ int) string {
		return item.ID
	})
	s.Contains(ids, firstGrant.ID)
	s.Contains(ids, secondGrant.ID)
}

// TestCreateInvoiceFundedGrantPropagatesTaxConfigToInvoiceLine verifies that TaxConfig set on
// creditgrant.CreateInput is propagated to the standard invoice line. Credit purchases always bypass
// collection alignment (ServicePeriod = {Now, Now}), so the line is immediately converted from
// gathering to a standard invoice — the test checks the standard invoice line, not a gathering line.
func (s *CreditGrantTestSuite) TestCreateInvoiceFundedGrantPropagatesTaxConfigToInvoiceLine() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("creditgrant-taxconfig-propagation")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	now := datetime.MustParseTimeInLocation(s.T(), "2026-04-17T11:23:53Z", time.UTC).AsTime()
	clock.SetTime(now)

	tc, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "txcd-40000001",
		Name:      "Test Tax Code txcd-40000001",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_40000001"},
		},
	})
	s.Require().NoError(err)

	grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "$10.00 grant with tax config",
		Currency:      USD,
		Amount:        alpacadecimal.NewFromInt(10),
		Priority:      lo.ToPtr(int16(10)),
		FundingMethod: creditgrant.FundingMethodInvoice,
		Purchase: &creditgrant.PurchaseTerms{
			Currency:         USD,
			PerUnitCostBasis: lo.ToPtr(alpacadecimal.NewFromInt(1)),
		},
		TaxConfig: &productcatalog.TaxConfig{
			Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			TaxCodeID: &tc.ID,
		},
	})
	s.Require().NoError(err)
	s.Equal(creditpurchase.StatusActive, grant.Status)

	standardInvoices, err := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
		Namespaces: []string{ns},
		Expand:     billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	s.Require().Len(standardInvoices.Items, 1)

	lines := standardInvoices.Items[0].Lines.OrEmpty()
	s.Require().Len(lines, 1)
	line := lines[0]

	s.Require().NotNil(line.TaxConfig, "standard invoice line TaxConfig must be set from grant TaxConfig")
	s.Require().NotNil(line.TaxConfig.Behavior, "TaxBehavior must propagate to invoice line")
	s.Equal(productcatalog.InclusiveTaxBehavior, *line.TaxConfig.Behavior)
	s.Require().NotNil(line.TaxConfig.TaxCodeID, "TaxCodeID must propagate to invoice line")
	s.Equal(tc.ID, *line.TaxConfig.TaxCodeID)
	s.Require().NotNil(line.TaxConfig.Stripe, "Stripe.Code must be backfilled on invoice line via TaxCode edge")
	s.Equal("txcd_40000001", line.TaxConfig.Stripe.Code)
}

// TestCreateInvoiceFundedGrantNilTaxConfigInvoiceLineNil verifies that when TaxConfig is omitted
// from creditgrant.CreateInput the resulting standard invoice line has nil TaxConfig.
func (s *CreditGrantTestSuite) TestCreateInvoiceFundedGrantNilTaxConfigInvoiceLineNil() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("creditgrant-taxconfig-nil")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	now := datetime.MustParseTimeInLocation(s.T(), "2026-04-17T11:23:53Z", time.UTC).AsTime()
	clock.SetTime(now)

	_, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "$10.00 grant without tax config",
		Currency:      USD,
		Amount:        alpacadecimal.NewFromInt(10),
		Priority:      lo.ToPtr(int16(10)),
		FundingMethod: creditgrant.FundingMethodInvoice,
		Purchase: &creditgrant.PurchaseTerms{
			Currency:         USD,
			PerUnitCostBasis: lo.ToPtr(alpacadecimal.NewFromInt(1)),
		},
	})
	s.Require().NoError(err)

	standardInvoices, err := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
		Namespaces: []string{ns},
		Expand:     billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	s.Require().Len(standardInvoices.Items, 1)

	lines := standardInvoices.Items[0].Lines.OrEmpty()
	s.Require().Len(lines, 1)
	s.Nil(lines[0].TaxConfig, "standard invoice line TaxConfig must be nil when grant has no TaxConfig")
}

// TestCreateExternalGrantPropagatesTaxConfigToCharge verifies that TaxConfig set on
// creditgrant.CreateInput is carried through Service.Create → toIntent → charge persistence for
// external-funded grants. External grants produce no invoice line, so only the charge-level
// TaxConfig is asserted.
func (s *CreditGrantTestSuite) TestCreateExternalGrantPropagatesTaxConfigToCharge() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("creditgrant-external-taxconfig")

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	now := time.Date(2026, 4, 17, 11, 23, 53, 0, time.UTC)
	clock.SetTime(now)

	tc, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "txcd-50000001",
		Name:      "Test Tax Code txcd-50000001",
	})
	s.Require().NoError(err)

	grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "External grant with tax config",
		Currency:      USD,
		Amount:        alpacadecimal.NewFromInt(30),
		Priority:      lo.ToPtr(int16(20)),
		FundingMethod: creditgrant.FundingMethodExternal,
		Purchase: &creditgrant.PurchaseTerms{
			Currency:           USD,
			PerUnitCostBasis:   lo.ToPtr(alpacadecimal.NewFromFloat(0.5)),
			AvailabilityPolicy: lo.ToPtr(creditpurchase.CreatedInitialPaymentSettlementStatus),
		},
		TaxConfig: &productcatalog.TaxConfig{
			Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			TaxCodeID: &tc.ID,
		},
	})
	s.Require().NoError(err)
	s.Equal(creditpurchase.SettlementTypeExternal, grant.Intent.Settlement.Type())

	s.Require().NotNil(grant.Intent.TaxConfig, "charge intent TaxConfig must be set from CreateInput")
	s.Require().NotNil(grant.Intent.TaxConfig.Behavior, "TaxBehavior must propagate through toIntent to charge")
	s.Equal(productcatalog.ExclusiveTaxBehavior, *grant.Intent.TaxConfig.Behavior)
	s.Require().NotNil(grant.Intent.TaxConfig.TaxCodeID, "TaxCodeID must propagate through toIntent to charge")
	s.Equal(tc.ID, *grant.Intent.TaxConfig.TaxCodeID)
}

// TestCreatePromotionalGrantPropagatesTaxConfigToCharge verifies that TaxConfig set on
// creditgrant.CreateInput is carried through Service.Create → toIntent → charge persistence for
// promotional (FundingMethodNone) grants. Promotional grants produce no invoice line, so only the
// charge-level TaxConfig is asserted.
func (s *CreditGrantTestSuite) TestCreatePromotionalGrantPropagatesTaxConfigToCharge() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("creditgrant-promotional-taxconfig")

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	now := time.Date(2026, 4, 17, 11, 23, 53, 0, time.UTC)
	clock.SetTime(now)

	tc, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "txcd-50000002",
		Name:      "Test Tax Code txcd-50000002",
	})
	s.Require().NoError(err)

	grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "Promotional grant with tax config",
		Currency:      USD,
		Amount:        alpacadecimal.NewFromInt(25),
		Priority:      lo.ToPtr(int16(15)),
		FundingMethod: creditgrant.FundingMethodNone,
		TaxConfig: &productcatalog.TaxConfig{
			Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
			TaxCodeID: &tc.ID,
		},
	})
	s.Require().NoError(err)
	s.Equal(creditpurchase.SettlementTypePromotional, grant.Intent.Settlement.Type())
	s.Equal(creditpurchase.StatusFinal, grant.Status)

	s.Require().NotNil(grant.Intent.TaxConfig, "charge intent TaxConfig must be set from CreateInput")
	s.Require().NotNil(grant.Intent.TaxConfig.Behavior, "TaxBehavior must propagate through toIntent to charge")
	s.Equal(productcatalog.InclusiveTaxBehavior, *grant.Intent.TaxConfig.Behavior)
	s.Require().NotNil(grant.Intent.TaxConfig.TaxCodeID, "TaxCodeID must propagate through toIntent to charge")
	s.Equal(tc.ID, *grant.Intent.TaxConfig.TaxCodeID)
}

func (s *CreditGrantTestSuite) mustCreatePromotionalCreditGrant(ctx context.Context, namespace string, customerID customer.CustomerID, name string, amount alpacadecimal.Decimal) creditpurchase.Charge {
	s.T().Helper()

	grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     namespace,
		CustomerID:    customerID.ID,
		Name:          name,
		Description:   lo.ToPtr(name),
		Currency:      USD,
		Amount:        amount,
		Priority:      lo.ToPtr(int16(5)),
		FundingMethod: creditgrant.FundingMethodNone,
	})
	s.Require().NoError(err)

	return grant
}
