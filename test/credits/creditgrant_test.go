package credits

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

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
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestCreditGrantTestSuite(t *testing.T) {
	suite.Run(t, new(CreditGrantTestSuite))
}

type CreditGrantTestSuite struct {
	CreditsTestSuite

	CreditPurchaseService creditpurchase.Service
	CreditGrantService    creditgrant.Service
}

func (s *CreditGrantTestSuite) SetupSuite() {
	s.CreditsTestSuite.SetupSuite()

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
	cust := s.createLedgerBackedCustomer(ns, "test-subject")

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

	cust := s.createLedgerBackedCustomer(ns, "test-subject")
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

	cust := s.createLedgerBackedCustomer(ns, "test-subject")
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

	cust := s.createLedgerBackedCustomer(ns, "test-subject")
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
