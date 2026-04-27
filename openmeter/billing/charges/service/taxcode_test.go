package service

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestTaxCodePersistence(t *testing.T) {
	suite.Run(t, new(TaxCodePersistenceTestSuite))
}

type TaxCodePersistenceTestSuite struct {
	BaseSuite
}

func (s *TaxCodePersistenceTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *TaxCodePersistenceTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

// TestFlatFeeChargePersistsTaxConfig verifies that TaxCodeID and TaxBehavior set on a flat fee
// charge intent survive the create→read round-trip through the database.
func (s *TaxCodePersistenceTestSuite) TestFlatFeeChargePersistsTaxConfig() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-flatfee")

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	cust := s.CreateTestCustomer(ns, "test-subject")

	tc := s.createTestTaxCode(ctx, ns, "txcd-10000000")

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.SetTime(servicePeriod.From)

	s.Run("persists both behavior and tax code id", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					name:              "flat-fee-taxcode",
					managedBy:         billing.ManuallyManagedLine,
					uniqueReferenceID: "flat-fee-taxcode",
					taxConfig: &productcatalog.TaxConfig{
						Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
						TaxCodeID: &tc.ID,
					},
				}),
			},
		})
		s.NoError(err)
		s.Require().Len(res, 1)

		chargeID, err := res[0].GetChargeID()
		s.NoError(err)

		readBack := s.mustGetChargeByID(chargeID)
		flatFee, err := readBack.AsFlatFeeCharge()
		s.NoError(err)

		s.Require().NotNil(flatFee.Intent.TaxConfig, "TaxConfig must be populated on read")
		s.Require().NotNil(flatFee.Intent.TaxConfig.Behavior, "TaxBehavior must be persisted")
		s.Equal(productcatalog.InclusiveTaxBehavior, *flatFee.Intent.TaxConfig.Behavior)
		s.Require().NotNil(flatFee.Intent.TaxConfig.TaxCodeID, "TaxCodeID must be persisted as FK")
		s.Equal(tc.ID, *flatFee.Intent.TaxConfig.TaxCodeID)
	})

	s.Run("nil tax config reads back as nil", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					name:              "flat-fee-no-taxcode",
					managedBy:         billing.ManuallyManagedLine,
					uniqueReferenceID: "flat-fee-no-taxcode",
				}),
			},
		})
		s.NoError(err)
		s.Require().Len(res, 1)

		chargeID, err := res[0].GetChargeID()
		s.NoError(err)

		readBack := s.mustGetChargeByID(chargeID)
		flatFee, err := readBack.AsFlatFeeCharge()
		s.NoError(err)

		s.Nil(flatFee.Intent.TaxConfig, "TaxConfig must be nil when not set on intent")
	})
}

// TestUsageBasedChargePersistsTaxConfig verifies that TaxCodeID and TaxBehavior set on a
// usage-based charge intent survive the create→read round-trip through the database.
func (s *TaxCodePersistenceTestSuite) TestUsageBasedChargePersistsTaxConfig() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-usagebased")

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	cust := s.CreateTestCustomer(ns, "test-subject")
	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer apiRequestsTotal.Cleanup()

	tc := s.createTestTaxCode(ctx, ns, "txcd-10000001")

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.SetTime(servicePeriod.From)

	s.Run("persists both behavior and tax code id", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditOnlySettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(1),
					}),
					featureKey:        apiRequestsTotal.Feature.Key,
					name:              "usage-based-taxcode",
					managedBy:         billing.ManuallyManagedLine,
					uniqueReferenceID: "usage-based-taxcode",
					taxConfig: &productcatalog.TaxConfig{
						Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
						TaxCodeID: &tc.ID,
					},
				}),
			},
		})
		s.NoError(err)
		s.Require().Len(res, 1)

		chargeID, err := res[0].GetChargeID()
		s.NoError(err)

		readBack := s.mustGetChargeByID(chargeID)
		usageBased, err := readBack.AsUsageBasedCharge()
		s.NoError(err)

		s.Require().NotNil(usageBased.Intent.TaxConfig, "TaxConfig must be populated on read")
		s.Require().NotNil(usageBased.Intent.TaxConfig.Behavior, "TaxBehavior must be persisted")
		s.Equal(productcatalog.ExclusiveTaxBehavior, *usageBased.Intent.TaxConfig.Behavior)
		s.Require().NotNil(usageBased.Intent.TaxConfig.TaxCodeID, "TaxCodeID must be persisted as FK")
		s.Equal(tc.ID, *usageBased.Intent.TaxConfig.TaxCodeID)
	})

	s.Run("nil tax config reads back as nil", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditOnlySettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(1),
					}),
					featureKey:        apiRequestsTotal.Feature.Key,
					name:              "usage-based-no-taxcode",
					managedBy:         billing.ManuallyManagedLine,
					uniqueReferenceID: "usage-based-no-taxcode",
				}),
			},
		})
		s.NoError(err)
		s.Require().Len(res, 1)

		chargeID, err := res[0].GetChargeID()
		s.NoError(err)

		readBack := s.mustGetChargeByID(chargeID)
		usageBased, err := readBack.AsUsageBasedCharge()
		s.NoError(err)

		s.Nil(usageBased.Intent.TaxConfig, "TaxConfig must be nil when not set on intent")
	})
}

// TestCreditPurchaseChargePersistsTaxConfig verifies that TaxCodeID and TaxBehavior set on a
// credit purchase charge intent survive the create→read round-trip through the database.
func (s *TaxCodePersistenceTestSuite) TestCreditPurchaseChargePersistsTaxConfig() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-creditpurchase")

	cust := s.CreateTestCustomer(ns, "test-subject")

	tc := s.createTestTaxCode(ctx, ns, "txcd-10000002")

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.SetTime(servicePeriod.From)

	s.Run("persists both behavior and tax code id", func() {
		callback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		s.CreditPurchaseTestHandler.onPromotionalCreditPurchase = callback.Handler(s.T())

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				charges.NewChargeIntent(creditpurchase.Intent{
					Intent: meta.Intent{
						Name:              "credit-purchase-taxcode",
						ManagedBy:         billing.ManuallyManagedLine,
						CustomerID:        cust.GetID().ID,
						Currency:          USD,
						ServicePeriod:     servicePeriod,
						BillingPeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						TaxConfig: &productcatalog.TaxConfig{
							Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
							TaxCodeID: &tc.ID,
						},
					},
					CreditAmount: alpacadecimal.NewFromFloat(50),
					Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
				}),
			},
		})
		s.NoError(err)
		s.Require().Len(res, 1)

		chargeID, err := res[0].GetChargeID()
		s.NoError(err)

		readBack := s.mustGetChargeByID(chargeID)
		cp, err := readBack.AsCreditPurchaseCharge()
		s.NoError(err)

		s.Require().NotNil(cp.Intent.TaxConfig, "TaxConfig must be populated on read")
		s.Require().NotNil(cp.Intent.TaxConfig.Behavior, "TaxBehavior must be persisted")
		s.Equal(productcatalog.InclusiveTaxBehavior, *cp.Intent.TaxConfig.Behavior)
		s.Require().NotNil(cp.Intent.TaxConfig.TaxCodeID, "TaxCodeID must be persisted as FK")
		s.Equal(tc.ID, *cp.Intent.TaxConfig.TaxCodeID)
	})

	s.Run("nil tax config reads back as nil", func() {
		callback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		s.CreditPurchaseTestHandler.onPromotionalCreditPurchase = callback.Handler(s.T())

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				charges.NewChargeIntent(creditpurchase.Intent{
					Intent: meta.Intent{
						Name:              "credit-purchase-no-taxcode",
						ManagedBy:         billing.ManuallyManagedLine,
						CustomerID:        cust.GetID().ID,
						Currency:          USD,
						ServicePeriod:     servicePeriod,
						BillingPeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
					},
					CreditAmount: alpacadecimal.NewFromFloat(50),
					Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
				}),
			},
		})
		s.NoError(err)
		s.Require().Len(res, 1)

		chargeID, err := res[0].GetChargeID()
		s.NoError(err)

		readBack := s.mustGetChargeByID(chargeID)
		cp, err := readBack.AsCreditPurchaseCharge()
		s.NoError(err)

		s.Nil(cp.Intent.TaxConfig, "TaxConfig must be nil when not set on intent")
	})
}

// TestCreditPurchaseInvoiceSettlementPropagatesTaxConfigToGatheringLine verifies that TaxConfig
// set on a credit purchase intent is propagated to the gathering invoice line built by
// buildInvoiceCreditPurchaseGatheringLine. Guards the contract that the gathering line reads
// TaxConfig from intent.TaxConfig (single source of truth), not from the settlement.
func (s *TaxCodePersistenceTestSuite) TestCreditPurchaseInvoiceSettlementPropagatesTaxConfigToGatheringLine() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-creditpurchase-invoice")

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	cust := s.CreateTestCustomer(ns, "test-subject")

	tc := s.createTestTaxCode(ctx, ns, "txcd-10000003")

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	// Set clock before service period start so Create defers invoicing (no
	// onCreditPurchaseInitiated callback needed) and the gathering line stays
	// queryable via ListGatheringInvoices.
	clock.SetTime(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))

	taxConfig := &productcatalog.TaxConfig{
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		TaxCodeID: &tc.ID,
	}

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			charges.NewChargeIntent(creditpurchase.Intent{
				Intent: meta.Intent{
					Name:              "credit-purchase-invoice-taxcode",
					ManagedBy:         billing.ManuallyManagedLine,
					CustomerID:        cust.GetID().ID,
					Currency:          USD,
					ServicePeriod:     servicePeriod,
					BillingPeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
					TaxConfig:         taxConfig,
				},
				CreditAmount: alpacadecimal.NewFromFloat(100),
				Settlement: creditpurchase.NewSettlement(creditpurchase.InvoiceSettlement{
					GenericSettlement: creditpurchase.GenericSettlement{
						Currency:  USD,
						CostBasis: alpacadecimal.NewFromFloat(0.5),
					},
				}),
			}),
		},
	})
	s.NoError(err)
	s.Require().Len(res, 1)

	gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespaces: []string{ns},
		Customers:  []string{cust.ID},
		Currencies: []currencyx.Code{USD},
		Expand:     []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
	})
	s.NoError(err)
	s.Require().Len(gatheringInvoices.Items, 1)

	lines := gatheringInvoices.Items[0].Lines.OrEmpty()
	s.Require().Len(lines, 1)
	gatheringLine := lines[0]

	s.Require().NotNil(gatheringLine.TaxConfig, "gathering line TaxConfig must be set from intent")
	s.Require().NotNil(gatheringLine.TaxConfig.Behavior, "TaxBehavior must propagate to gathering line")
	s.Equal(productcatalog.ExclusiveTaxBehavior, *gatheringLine.TaxConfig.Behavior)
	s.Require().NotNil(gatheringLine.TaxConfig.TaxCodeID, "TaxCodeID must propagate to gathering line")
	s.Equal(tc.ID, *gatheringLine.TaxConfig.TaxCodeID)
}

// TestCreditPurchaseInvoiceSettlementNilTaxConfigDoesNotPropagateToGatheringLine verifies that
// when Intent.TaxConfig is nil the gathering line's TaxConfig is also nil.
func (s *TaxCodePersistenceTestSuite) TestCreditPurchaseInvoiceSettlementNilTaxConfigDoesNotPropagateToGatheringLine() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-creditpurchase-invoice-nil")

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	cust := s.CreateTestCustomer(ns, "test-subject")

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.SetTime(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			charges.NewChargeIntent(creditpurchase.Intent{
				Intent: meta.Intent{
					Name:              "credit-purchase-invoice-nil-taxcode",
					ManagedBy:         billing.ManuallyManagedLine,
					CustomerID:        cust.GetID().ID,
					Currency:          USD,
					ServicePeriod:     servicePeriod,
					BillingPeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
				},
				CreditAmount: alpacadecimal.NewFromFloat(100),
				Settlement: creditpurchase.NewSettlement(creditpurchase.InvoiceSettlement{
					GenericSettlement: creditpurchase.GenericSettlement{
						Currency:  USD,
						CostBasis: alpacadecimal.NewFromFloat(0.5),
					},
				}),
			}),
		},
	})
	s.NoError(err)
	s.Require().Len(res, 1)

	gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespaces: []string{ns},
		Customers:  []string{cust.ID},
		Currencies: []currencyx.Code{USD},
		Expand:     []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
	})
	s.NoError(err)
	s.Require().Len(gatheringInvoices.Items, 1)

	lines := gatheringInvoices.Items[0].Lines.OrEmpty()
	s.Require().Len(lines, 1)

	s.Nil(lines[0].TaxConfig, "gathering line TaxConfig must be nil when Intent.TaxConfig is nil")
}

func (s *TaxCodePersistenceTestSuite) createTestTaxCode(ctx context.Context, ns, key string) taxcode.TaxCode {
	s.T().Helper()
	tc, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       key,
		Name:      "Test Tax Code " + key,
	})
	s.Require().NoError(err, "creating test tax code must succeed")
	return tc
}
