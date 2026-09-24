package credits

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
)

type creditTransactionRow struct {
	txType customerbalance.CreditTransactionType
	amount float64
	before float64
	after  float64
}

func (s *SanitySuite) TestCreditTransactionsListFlatFeeDeleteCorrectionAsRefund() {
	for _, tc := range []struct {
		name    string
		funding int64
		// Newest first.
		expected []creditTransactionRow
	}{
		{
			name:    "funded",
			funding: 30,
			expected: []creditTransactionRow{
				{txType: customerbalance.CreditTransactionTypeRefunded, amount: 30, before: 0, after: 30},
				{txType: customerbalance.CreditTransactionTypeConsumed, amount: -30, before: 30, after: 0},
				{txType: customerbalance.CreditTransactionTypeFunded, amount: 30, before: 0, after: 30},
			},
		},
		{
			name:    "partially funded",
			funding: 10,
			expected: []creditTransactionRow{
				{txType: customerbalance.CreditTransactionTypeRefunded, amount: 30, before: -20, after: 10},
				{txType: customerbalance.CreditTransactionTypeConsumed, amount: -20, before: 0, after: -20},
				{txType: customerbalance.CreditTransactionTypeConsumed, amount: -10, before: 10, after: 0},
				{txType: customerbalance.CreditTransactionTypeFunded, amount: 10, before: 0, after: 10},
			},
		},
		{
			name:    "advance only",
			funding: 0,
			expected: []creditTransactionRow{
				{txType: customerbalance.CreditTransactionTypeRefunded, amount: 30, before: -30, after: 0},
				{txType: customerbalance.CreditTransactionTypeConsumed, amount: -30, before: 0, after: -30},
			},
		},
	} {
		s.Run(tc.name, func() {
			setup := s.setupFlatFeeCreditOnlyDeleteCorrection("credit-transactions-flatfee-delete")
			customerID := setup.customer.GetID()

			clock.FreezeTime(setup.createAt)
			defer clock.UnFreeze()

			// given: a credit-only flat fee collected from the funding and, for any remainder, advance
			if tc.funding > 0 {
				s.CreatePromotionalCreditFunding(setup.ctx, CreatePromotionalCreditFundingInput{
					Namespace: setup.namespace,
					Customer:  customerID,
					Amount:    alpacadecimal.NewFromInt(tc.funding),
					At:        setup.createAt,
				})
			}
			created := s.createAndAdvanceCreditOnlyFlatFeeCharge(createCreditOnlyFlatFeeChargeInput{
				ctx:           setup.ctx,
				namespace:     setup.namespace,
				customer:      customerID,
				servicePeriod: setup.servicePeriod,
				createAt:      setup.createAt,
				advanceAt:     setup.servicePeriod.From,
				amount:        setup.amount,
				name:          setup.namespace,
			})

			// when: the charge is deleted with its credits corrected
			clock.FreezeTime(setup.servicePeriod.From.AddDate(0, 0, 10))
			s.deleteChargeWithRefundAsCredits(setup.ctx, customerID, created.id)

			// then: the correction is one refund booked with the original usage, and balances chain
			items := s.listAllCreditTransactions(customerID, nil, 100)
			s.requireCreditTransactionRows(items, tc.expected)
			s.Equal(created.id, items[0].Annotations[ledger.AnnotationChargeID])
			s.True(items[0].BookedAt.Equal(setup.servicePeriod.From))

			// then: type filtering and paging keep each row's balances
			refunds := s.listAllCreditTransactions(customerID, lo.ToPtr(customerbalance.CreditTransactionTypeRefunded), 100)
			s.requireCreditTransactionRows(refunds, tc.expected[:1])

			s.requireCreditTransactionRows(s.listAllCreditTransactions(customerID, nil, 1), tc.expected)
		})
	}
}

func (s *SanitySuite) TestCreditTransactionsListFlatFeeShrinkCorrectionAsRefund() {
	for _, tc := range []struct {
		name     string
		funding  int64
		expected []creditTransactionRow
	}{
		{
			name:    "funded",
			funding: 31,
			expected: []creditTransactionRow{
				{txType: customerbalance.CreditTransactionTypeRefunded, amount: 21, before: 0, after: 21},
				{txType: customerbalance.CreditTransactionTypeConsumed, amount: -31, before: 31, after: 0},
				{txType: customerbalance.CreditTransactionTypeFunded, amount: 31, before: 0, after: 31},
			},
		},
		{
			name:    "partially funded",
			funding: 10,
			expected: []creditTransactionRow{
				{txType: customerbalance.CreditTransactionTypeRefunded, amount: 21, before: -21, after: 0},
				{txType: customerbalance.CreditTransactionTypeConsumed, amount: -21, before: 0, after: -21},
				{txType: customerbalance.CreditTransactionTypeConsumed, amount: -10, before: 10, after: 0},
				{txType: customerbalance.CreditTransactionTypeFunded, amount: 10, before: 0, after: 10},
			},
		},
	} {
		s.Run(tc.name, func() {
			setup := s.setupFlatFeeCreditOnlyDeleteCorrection("credit-transactions-flatfee-shrink")
			customerID := setup.customer.GetID()
			shrunkTo := setup.servicePeriod.From.AddDate(0, 0, 10)

			clock.FreezeTime(setup.createAt)
			defer clock.UnFreeze()

			// given: a prorated credit-only flat fee of 31 for a 31-day period
			s.CreatePromotionalCreditFunding(setup.ctx, CreatePromotionalCreditFundingInput{
				Namespace: setup.namespace,
				Customer:  customerID,
				Amount:    alpacadecimal.NewFromInt(tc.funding),
				At:        setup.createAt,
			})
			created := s.createAndAdvanceCreditOnlyFlatFeeCharge(createCreditOnlyFlatFeeChargeInput{
				ctx:           setup.ctx,
				namespace:     setup.namespace,
				customer:      customerID,
				servicePeriod: setup.servicePeriod,
				createAt:      setup.createAt,
				advanceAt:     setup.servicePeriod.From,
				amount:        alpacadecimal.NewFromInt(31),
				name:          setup.namespace,
				proRating: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			})

			// when: a subscription edit shrinks the charge to the first 10 days
			clock.FreezeTime(shrunkTo)
			patch, err := meta.NewPatchShrink(meta.NewPatchShrinkInput{
				ChangeSource:           billing.ChangeSourceSystem,
				NewServicePeriodTo:     shrunkTo,
				NewFullServicePeriodTo: setup.servicePeriod.To,
				NewBillingPeriodTo:     shrunkTo,
				NewInvoiceAt:           shrunkTo,
			})
			s.Require().NoError(err)
			s.Require().NoError(s.Charges.ApplyPatches(setup.ctx, charges.ApplyPatchesInput{
				CustomerID:        customerID,
				PatchesByChargeID: map[string]charges.Patch{created.id: patch},
			}))

			// then: the prorated difference is refunded, taking back advance before real credit
			items := s.listAllCreditTransactions(customerID, nil, 100)
			s.requireCreditTransactionRows(items, tc.expected)
			s.Equal(created.id, items[0].Annotations[ledger.AnnotationChargeID])
		})
	}
}

func (s *SanitySuite) TestCreditTransactionsListBackfilledAdvanceCorrectionAsRefund() {
	setup := s.setupClosedPeriodUsageBasedCreditOnlyCollection("credit-transactions-backfill-correction")
	customerID := setup.customer.GetID()
	backfillAt := setup.createAt.Add(time.Hour)
	correctionAt := backfillAt.Add(time.Hour)
	expiresAt := setup.createAt.Add(7 * 24 * time.Hour)

	clock.FreezeTime(setup.createAt)
	defer clock.UnFreeze()

	// given: usage consumed 8 of advance
	s.MockStreamingConnector.AddSimpleEvent(setup.featureKey, setup.amount.InexactFloat64(), setup.servicePeriod.From.Add(24*time.Hour))
	chargeRes, err := s.Charges.Create(setup.ctx, charges.CreateInput{
		Namespace: setup.namespace,
		Intents: charges.NewCreateChargeIntents(
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:          customerID,
				Currency:          USD,
				ServicePeriod:     setup.servicePeriod,
				SettlementMode:    productcatalog.CreditOnlySettlementMode,
				Price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				Name:              "credit-transactions-backfill-correction-usage",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "credit-transactions-backfill-correction-usage",
				FeatureKey:        setup.featureKey,
			}),
		),
	})
	s.Require().NoError(err)
	s.Require().Len(chargeRes, 1)
	usageCharge, err := chargeRes[0].AsUsageBasedCharge()
	s.Require().NoError(err)

	// given: a later grant of 12 backfills the advance
	clock.FreezeTime(backfillAt)
	s.createPromotionalCreditGrant(setup.ctx, CreatePromotionalCreditFundingInput{
		Namespace: setup.namespace,
		Customer:  customerID,
		Amount:    alpacadecimal.NewFromInt(12),
		At:        backfillAt,
		ExpiresAt: &expiresAt,
	})

	// when: the usage is corrected, unwinding the backfill and restoring the purchased credit
	clock.FreezeTime(correctionAt)
	s.deleteChargeWithRefundAsCredits(setup.ctx, customerID, usageCharge.ID)

	// then: the offsetting correction postings are listed as one refund of the consumed 8
	s.requireCreditTransactionRows(s.listAllCreditTransactions(customerID, nil, 100), []creditTransactionRow{
		{txType: customerbalance.CreditTransactionTypeFunded, amount: 12, before: 0, after: 12},
		{txType: customerbalance.CreditTransactionTypeRefunded, amount: 8, before: -8, after: 0},
		{txType: customerbalance.CreditTransactionTypeConsumed, amount: -8, before: 0, after: -8},
	})
}

func (s *SanitySuite) listAllCreditTransactions(customerID customer.CustomerID, txType *customerbalance.CreditTransactionType, pageSize int) []customerbalance.CreditTransaction {
	s.T().Helper()

	var (
		items []customerbalance.CreditTransaction
		after *ledger.TransactionCursor
	)
	for {
		result, err := s.CustomerBalanceSvc.ListCreditTransactions(s.T().Context(), customerbalance.ListCreditTransactionsInput{
			CustomerID: customerID,
			Limit:      pageSize,
			After:      after,
			Type:       txType,
		})
		s.Require().NoError(err)

		items = append(items, result.Items...)
		if result.NextCursor == nil {
			return items
		}
		after = result.NextCursor
	}
}

func (s *SanitySuite) requireCreditTransactionRows(items []customerbalance.CreditTransaction, expected []creditTransactionRow) {
	s.T().Helper()

	actual := make([]creditTransactionRow, 0, len(items))
	for _, item := range items {
		actual = append(actual, creditTransactionRow{
			txType: item.Type,
			amount: item.Amount.InexactFloat64(),
			before: item.Balance.Before.InexactFloat64(),
			after:  item.Balance.After.InexactFloat64(),
		})
	}
	s.Require().Equal(expected, actual)
}
