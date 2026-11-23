package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinggatheringinvoiceline"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestDynamicChargeCostBasis(t *testing.T) {
	suite.Run(t, new(dynamicChargeCostBasisSuite))
}

type dynamicChargeCostBasisSuite struct {
	BaseSuite
}

func (s *dynamicChargeCostBasisSuite) TestCreateRejectsBackdatedService() {
	// Given a first USD basis on January 15 and charges for service starting January 1.
	ctx := s.T().Context()
	firstUSDBasisAt := time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(firstUSDBasisAt)
	defer clock.UnFreeze()

	namespace := s.GetUniqueNamespace("backdated-charge-cost-basis")
	customer := s.CreateTestCustomer(namespace, "backdated-service")
	defaults := s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), namespace)
	currency := s.createTestCustomCurrency(ctx, namespace)
	s.createFeatureMeters(ctx, namespace, "requests")
	_, err := s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: currency.ID,
		FiatCode:   "USD",
		Rate:       alpacadecimal.NewFromInt(1),
	})
	s.Require().NoError(err)

	settlementMode := productcatalog.CreditThenInvoiceSettlementMode
	dynamic := costbasis.NewIntent(costbasis.DynamicIntent{FiatCurrency: s.newFiatCurrency("USD")})
	flatFee := s.newFlatFeeIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "flat-fee", settlementMode, &dynamic)
	usageBased := s.newUsageBasedIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "usage", "requests", settlementMode, &dynamic)
	creditPurchase := newCustomCurrencyInvoiceCreditPurchaseIntent(customer.ID, currency, defaults.CreditGrantTaxCodeID, dynamic)
	intents := []charges.ChargeIntent{
		charges.NewChargeIntent(flatFee),
		charges.NewChargeIntent(usageBased),
		charges.NewChargeIntent(creditPurchase),
	}

	// When each charge is submitted, the caller gets the missing historical rate as a validation issue.
	for _, intent := range intents {
		s.Run(string(intent.Type()), func() {
			_, err := s.Charges.Create(ctx, charges.CreateInput{
				Namespace: namespace,
				Intents:   charges.NewCreateChargeIntents(intent),
			})
			var issue models.ValidationIssue
			s.Require().ErrorAs(err, &issue)
			s.Equal(costbasis.ErrCodeChargeCostBasisUnavailable, issue.Code())
		})
	}

	// Then there is no charge to advance and no line to invoice.
	persisted, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{Namespace: namespace})
	s.Require().NoError(err)
	s.Empty(persisted.Items)

	hasLines, err := s.DBClient.BillingGatheringInvoiceLine.Query().
		Where(billinggatheringinvoiceline.Namespace(namespace)).
		Exist(ctx)
	s.Require().NoError(err)
	s.False(hasLines)
}
