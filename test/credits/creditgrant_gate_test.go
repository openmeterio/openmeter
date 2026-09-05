package credits

import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	creditgrantservice "github.com/openmeterio/openmeter/openmeter/billing/creditgrant/service"
	"github.com/openmeterio/openmeter/openmeter/ent/db/charge"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
)

func (s *CreditGrantTestSuite) TestCustomCurrencyGrantRequiresRolloutFlag() {
	svc, err := creditgrantservice.New(creditgrantservice.Config{
		CreditPurchaseService: s.CreditPurchaseService,
		ChargesService:        s.Charges,
		BillingService:        s.BillingService,
		CustomerService:       s.CustomerService,
		CreditVoidService:     creditvoid.NewNoopService(),
		CurrencyResolver:      s.CurrencyResolver,
		TransactionManager:    enttx.NewCreator(s.DBClient),
		CreditsConfig:         config.CreditsConfiguration{Enabled: true, EnableCustomCurrencyCharge: false},
	})
	s.Require().NoError(err)

	for _, funding := range []creditgrant.FundingMethod{creditgrant.FundingMethodNone, creditgrant.FundingMethodExternal, creditgrant.FundingMethodInvoice} {
		s.Run(string(funding), func() {
			// given: credits are enabled, but custom-currency charges are disabled
			ctx := s.T().Context()
			ns := s.GetUniqueNamespace("custom-grant-disabled")
			s.ProvisionDefaultTaxCodes(ctx, ns)
			cust := s.CreateLedgerBackedCustomer(ns, "customer")
			tokens := s.CreateCustomCurrency(ns, "TOKENS")
			if funding == creditgrant.FundingMethodInvoice {
				invoicing := s.SetupCustomInvoicing(ns)
				s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID())
			}
			input := creditgrant.CreateInput{
				Namespace: ns, CustomerID: cust.ID, Name: "grant", Currency: tokens.GetCode(),
				Amount: alpacadecimal.NewFromInt(100), FundingMethod: funding,
			}
			if funding != creditgrant.FundingMethodNone {
				input.Purchase = &creditgrant.PurchaseTerms{
					Currency: USD, PerUnitCostBasis: lo.ToPtr(alpacadecimal.NewFromInt(1)),
					AvailabilityPolicy: lo.ToPtr(creditpurchase.CreatedInitialPaymentSettlementStatus),
				}
			}

			// when: any funding method attempts to issue custom credits
			_, err := svc.Create(ctx, input)

			// then: rejection leaves no durable charge or ledger posting
			s.Require().ErrorIs(err, meta.ErrCustomCurrencyNotSupported)
			chargeCount, err := s.DBClient.Charge.Query().Where(charge.NamespaceEQ(ns)).Count(ctx)
			s.Require().NoError(err)
			s.Zero(chargeCount)
			postingCount, err := s.DBClient.LedgerTransaction.Query().Where(ledgertransaction.NamespaceEQ(ns)).Count(ctx)
			s.Require().NoError(err)
			s.Zero(postingCount)

			// then: the same flag does not prevent fiat credit grants
			input.Currency = USD
			grant, err := svc.Create(ctx, input)
			s.Require().NoError(err)
			s.Equal(USD, grant.Intent.Currency.GetCode())
			s.Require().NotNil(grant.Realizations.CreditGrantRealization)
		})
	}
}
