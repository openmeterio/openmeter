package service

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
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
					taxConfig: &productcatalog.TaxCodeConfig{
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
					taxConfig: &productcatalog.TaxCodeConfig{
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
						TaxConfig: &productcatalog.TaxCodeConfig{
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

	customInvoicing := s.SetupCustomInvoicing(ns)
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithManualApproval(),
	)
	cust := s.CreateTestCustomer(ns, "test-subject")

	const stripeCode = "txcd_10000003"
	tc := s.createTestTaxCodeWithStripeMapping(ctx, ns, "txcd-10000003", stripeCode)

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	// Set clock before service period start so Create defers invoicing (no
	// onCreditPurchaseInitiated callback needed) and the gathering line stays
	// queryable via ListGatheringInvoices.
	clock.SetTime(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))

	taxConfig := &productcatalog.TaxCodeConfig{
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
	// Stripe.Code is backfilled on read via BackfillTaxConfig + TaxCode edge (dual-write invariant).
	s.Require().NotNil(gatheringLine.TaxConfig.Stripe, "Stripe.Code must be backfilled on gathering line via TaxCode edge")
	s.Equal(stripeCode, gatheringLine.TaxConfig.Stripe.Code)

	// Advance to service period start, invoice, approve, and settle — then verify that the
	// standard invoice line carries Stripe.Code resolved from the TaxCode entity's app mapping.
	initiatedCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
	s.CreditPurchaseTestHandler.onCreditPurchaseInitiated = initiatedCallback.Handler(s.T())

	clock.SetTime(servicePeriod.From)
	now := clock.Now()
	createdInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.NoError(err)
	s.Require().Len(createdInvoices, 1)
	invoiceID := createdInvoices[0].GetInvoiceID()
	s.Equal(1, initiatedCallback.nrInvocations)

	_, err = s.BillingService.ApproveInvoice(ctx, invoiceID)
	s.NoError(err)

	authorizedCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
	s.CreditPurchaseTestHandler.onCreditPurchasePaymentAuthorized = authorizedCallback.Handler(s.T())
	settledCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
	s.CreditPurchaseTestHandler.onCreditPurchasePaymentSettled = settledCallback.Handler(s.T())

	_, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
		InvoiceID: invoiceID,
		Trigger:   billing.TriggerPaid,
	})
	s.NoError(err)
	s.Equal(1, authorizedCallback.nrInvocations)
	s.Equal(1, settledCallback.nrInvocations)

	invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: invoiceID,
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.NoError(err)

	stdLines := invoice.Lines.OrEmpty()
	s.Require().Len(stdLines, 1)
	line := stdLines[0]

	s.Require().NotNil(line.TaxConfig, "standard invoice line must have TaxConfig")
	s.Require().NotNil(line.TaxConfig.TaxCodeID, "TaxCodeID must be on standard invoice line")
	s.Equal(tc.ID, *line.TaxConfig.TaxCodeID)
	s.Require().NotNil(line.TaxConfig.Stripe, "Stripe.Code must be backfilled on standard invoice line via TaxCode edge")
	s.Equal(stripeCode, line.TaxConfig.Stripe.Code)
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

// TestFlatFeeCreditOnlyHandlerReceivesTaxConfig verifies that when a credit-only flat-fee charge
// is advanced to final realization, the ledger handler receives the correct TaxConfig. This guards
// the full path: intent → DB persistence → charge reconstruction → state machine → handler call.
func (s *TaxCodePersistenceTestSuite) TestFlatFeeCreditOnlyHandlerReceivesTaxConfig() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-flatfee-creditonly")

	cust := s.CreateTestCustomer(ns, "test-subject")

	tc := s.createTestTaxCode(ctx, ns, "txcd-20000000")

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	invoiceAt := servicePeriod.From
	clock.FreezeTime(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(100),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				name:              "flat-fee-creditonly-taxconfig",
				managedBy:         billing.ManuallyManagedLine,
				uniqueReferenceID: "flat-fee-creditonly-taxconfig",
				taxConfig: &productcatalog.TaxCodeConfig{
					Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
					TaxCodeID: &tc.ID,
				},
			}),
		},
	})
	s.NoError(err)
	s.Require().Len(res, 1)

	var capturedInput flatfee.OnAllocateCreditsInput
	s.FlatFeeTestHandler.onAllocateCredits = func(_ context.Context, input flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
		capturedInput = input
		return creditrealization.CreateAllocationInputs{
			{
				ServicePeriod: input.Charge.Intent.ServicePeriod,
				Amount:        input.PreTaxAmountToAllocate,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
			},
		}, nil
	}

	clock.FreezeTime(invoiceAt)

	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: cust.GetID()})
	s.NoError(err)
	s.Require().Len(advancedCharges, 1)

	s.Require().NotNil(capturedInput.Charge.Intent.TaxConfig, "handler must receive TaxConfig after DB roundtrip")
	s.Require().NotNil(capturedInput.Charge.Intent.TaxConfig.Behavior)
	s.Equal(productcatalog.InclusiveTaxBehavior, *capturedInput.Charge.Intent.TaxConfig.Behavior)
	s.Require().NotNil(capturedInput.Charge.Intent.TaxConfig.TaxCodeID)
	s.Equal(tc.ID, *capturedInput.Charge.Intent.TaxConfig.TaxCodeID)
}

// TestUsageBasedCreditOnlyHandlerReceivesTaxConfig verifies that when a credit-only usage-based
// charge reaches final realization, the ledger handler receives the correct TaxConfig. This guards
// the full path: intent → DB persistence → charge reconstruction → state machine → handler call.
func (s *TaxCodePersistenceTestSuite) TestUsageBasedCreditOnlyHandlerReceivesTaxConfig() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-usagebased-creditonly")

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	cust := s.CreateTestCustomer(ns, "test-subject")
	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer apiRequestsTotal.Cleanup()
	meterSlug := apiRequestsTotal.Feature.Key

	tc := s.createTestTaxCode(ctx, ns, "txcd-20000001")

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.FreezeTime(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

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
				featureKey:        meterSlug,
				name:              "usage-based-creditonly-taxconfig",
				managedBy:         billing.ManuallyManagedLine,
				uniqueReferenceID: "usage-based-creditonly-taxconfig",
				taxConfig: &productcatalog.TaxCodeConfig{
					Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					TaxCodeID: &tc.ID,
				},
			}),
		},
	})
	s.NoError(err)
	s.Require().Len(res, 1)

	// Advance into active state at service period start.
	clock.FreezeTime(servicePeriod.From)
	_, err = s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: cust.GetID()})
	s.NoError(err)

	// Add usage so the final realization produces a non-zero amount.
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 5, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	var capturedInput usagebased.CreditsOnlyUsageAccruedInput
	s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued = func(_ context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
		capturedInput = input
		return creditrealization.CreateAllocationInputs{
			{
				ServicePeriod: input.Charge.Intent.ServicePeriod,
				Amount:        input.AmountToAllocate,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
			},
		}, nil
	}

	// Advance past service period end to trigger final realization.
	clock.FreezeTime(time.Date(2026, 2, 3, 0, 1, 0, 0, time.UTC))
	_, err = s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: cust.GetID()})
	s.NoError(err)

	s.Require().NotNil(capturedInput.Charge.Intent.TaxConfig, "handler must receive TaxConfig after DB roundtrip")
	s.Require().NotNil(capturedInput.Charge.Intent.TaxConfig.Behavior)
	s.Equal(productcatalog.ExclusiveTaxBehavior, *capturedInput.Charge.Intent.TaxConfig.Behavior)
	s.Require().NotNil(capturedInput.Charge.Intent.TaxConfig.TaxCodeID)
	s.Equal(tc.ID, *capturedInput.Charge.Intent.TaxConfig.TaxCodeID)
}

// TestFlatFeeInvoiceSettlementPopulatesStripeCodeOnStandardInvoice verifies the dual-write
// invariant for flat-fee credit_then_invoice charges: after payment is settled, the standard
// invoice line carries both TaxCodeID (FK) and Stripe.Code resolved from the TaxCode entity.
func (s *TaxCodePersistenceTestSuite) TestFlatFeeInvoiceSettlementPopulatesStripeCodeOnStandardInvoice() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-flatfee-invoice-settled")

	customInvoicing := s.SetupCustomInvoicing(ns)
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithManualApproval(),
	)
	cust := s.CreateTestCustomer(ns, "test-subject")

	const stripeCode = "txcd_20000005"
	tc := s.createTestTaxCodeWithStripeMapping(ctx, ns, "txcd-20000005", stripeCode)

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.SetTime(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))

	_, err := s.Charges.Create(ctx, charges.CreateInput{
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
				name:              "flat-fee-invoice-stripe-taxcode",
				managedBy:         billing.ManuallyManagedLine,
				uniqueReferenceID: "flat-fee-invoice-stripe-taxcode",
				taxConfig: &productcatalog.TaxCodeConfig{
					Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					TaxCodeID: &tc.ID,
				},
			}),
		},
	})
	s.NoError(err)

	s.FlatFeeTestHandler.onAllocateCredits = func(_ context.Context, input flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
		return creditrealization.CreateAllocationInputs{}, nil
	}

	clock.SetTime(servicePeriod.From)
	now := clock.Now()
	createdInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.NoError(err)
	s.Require().Len(createdInvoices, 1)
	invoiceID := createdInvoices[0].GetInvoiceID()

	s.FlatFeeTestHandler.onInvoiceUsageAccrued = func(_ context.Context, _ flatfee.OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error) {
		return ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()}, nil
	}
	s.FlatFeeTestHandler.onPaymentAuthorized = func(_ context.Context, _ flatfee.OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error) {
		return ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()}, nil
	}
	s.FlatFeeTestHandler.onPaymentSettled = func(_ context.Context, _ flatfee.OnPaymentSettledInput) (ledgertransaction.GroupReference, error) {
		return ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()}, nil
	}

	_, err = s.BillingService.ApproveInvoice(ctx, invoiceID)
	s.NoError(err)

	_, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
		InvoiceID: invoiceID,
		Trigger:   billing.TriggerPaid,
	})
	s.NoError(err)

	invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: invoiceID,
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.NoError(err)

	lines := invoice.Lines.OrEmpty()
	s.Require().Len(lines, 1)
	line := lines[0]

	// Stripe.Code must be backfilled from the TaxCode entity's app mapping at invoice snapshot time.
	s.Require().NotNil(line.TaxConfig, "standard invoice line must have TaxConfig")
	s.Require().NotNil(line.TaxConfig.TaxCodeID, "TaxCodeID must be on standard invoice line")
	s.Equal(tc.ID, *line.TaxConfig.TaxCodeID)
	s.Require().NotNil(line.TaxConfig.Stripe, "Stripe.Code must be backfilled on standard invoice line via TaxCode edge")
	s.Equal(stripeCode, line.TaxConfig.Stripe.Code)
}

// TestTaxConfigInListCharges verifies that ListCharges returns charges with tax_config populated
// for both flat-fee and usage-based charge types — the domain-layer counterpart to the API
// conversion tested in convert_test.go.
func (s *TaxCodePersistenceTestSuite) TestTaxConfigInListCharges() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-list")

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	cust := s.CreateTestCustomer(ns, "test-subject")
	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer apiRequestsTotal.Cleanup()

	tc := s.createTestTaxCode(ctx, ns, "txcd-10000010")

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.SetTime(servicePeriod.From)

	tcID := tc.ID
	taxConfigFlat := &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: &tcID,
	}
	taxConfigUsage := &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID: &tcID,
	}

	_, err := s.Charges.Create(ctx, charges.CreateInput{
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
				name:              "flat-fee-list-taxcode",
				managedBy:         billing.ManuallyManagedLine,
				uniqueReferenceID: "flat-fee-list-taxcode",
				taxConfig:         taxConfigFlat,
			}),
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(1),
				}),
				featureKey:        apiRequestsTotal.Feature.Key,
				name:              "usage-based-list-taxcode",
				managedBy:         billing.ManuallyManagedLine,
				uniqueReferenceID: "usage-based-list-taxcode",
				taxConfig:         taxConfigUsage,
			}),
		},
	})
	s.Require().NoError(err)

	// Also seed a flat-fee charge without tax config to verify nil round-trips correctly.
	_, err = s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(50),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				name:              "flat-fee-list-no-taxcode",
				managedBy:         billing.ManuallyManagedLine,
				uniqueReferenceID: "flat-fee-list-no-taxcode",
			}),
		},
	})
	s.Require().NoError(err)

	result, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:   ns,
		CustomerIDs: []string{cust.GetID().ID},
		ChargeTypes: []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased},
		Expands:     meta.Expands{meta.ExpandRealizations},
		Page:        pagination.Page{PageSize: 20, PageNumber: 1},
	})
	s.Require().NoError(err)
	s.Require().Len(result.Items, 3, "all three charges must appear in list")

	for _, charge := range result.Items {
		switch charge.Type() {
		case meta.ChargeTypeFlatFee:
			ff, err := charge.AsFlatFeeCharge()
			s.Require().NoError(err)

			if ff.Intent.Intent.UniqueReferenceID != nil && *ff.Intent.Intent.UniqueReferenceID == "flat-fee-list-no-taxcode" {
				s.Nil(ff.Intent.TaxConfig, "flat fee charge without tax config must list with nil TaxConfig")
			} else {
				s.Require().NotNil(ff.Intent.TaxConfig, "flat fee charge must carry TaxConfig in list response")
				s.Require().NotNil(ff.Intent.TaxConfig.Behavior)
				s.Equal(productcatalog.InclusiveTaxBehavior, *ff.Intent.TaxConfig.Behavior)
				s.Require().NotNil(ff.Intent.TaxConfig.TaxCodeID)
				s.Equal(tc.ID, *ff.Intent.TaxConfig.TaxCodeID)
			}

		case meta.ChargeTypeUsageBased:
			ub, err := charge.AsUsageBasedCharge()
			s.Require().NoError(err)
			s.Require().NotNil(ub.Intent.TaxConfig, "usage-based charge must carry TaxConfig in list response")
			s.Require().NotNil(ub.Intent.TaxConfig.Behavior)
			s.Equal(productcatalog.InclusiveTaxBehavior, *ub.Intent.TaxConfig.Behavior)
			s.Require().NotNil(ub.Intent.TaxConfig.TaxCodeID)
			s.Equal(tc.ID, *ub.Intent.TaxConfig.TaxCodeID)

		default:
			s.Failf("unexpected charge type", "type=%s", string(charge.Type()))
		}
	}
}

// TestFlatFeeInvoiceSettlementPropagatesTaxConfigToGatheringLine verifies that TaxConfig set on a
// flat-fee CreditThenInvoice intent is propagated to the gathering invoice line built by
// gatheringLineFromFlatFeeCharge. Guards the single-source-of-truth contract: gathering line reads
// TaxConfig from intent.TaxConfig, and Stripe.Code is backfilled via the TaxCode entity edge.
func (s *TaxCodePersistenceTestSuite) TestFlatFeeInvoiceSettlementPropagatesTaxConfigToGatheringLine() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-flatfee-gathering")

	customInvoicing := s.SetupCustomInvoicing(ns)
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithManualApproval(),
	)
	cust := s.CreateTestCustomer(ns, "test-subject")

	const stripeCode = "txcd_30000001"
	tc := s.createTestTaxCodeWithStripeMapping(ctx, ns, "txcd-30000001", stripeCode)

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	// Clock before invoiceAt (= servicePeriod.From for InAdvance) keeps the gathering line
	// pending so ListGatheringInvoices can observe it without invoicing.
	clock.SetTime(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))

	_, err := s.Charges.Create(ctx, charges.CreateInput{
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
				name:              "flat-fee-gathering-taxcode",
				managedBy:         billing.ManuallyManagedLine,
				uniqueReferenceID: "flat-fee-gathering-taxcode",
				taxConfig: &productcatalog.TaxCodeConfig{
					Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					TaxCodeID: &tc.ID,
				},
			}),
		},
	})
	s.NoError(err)

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
	s.Require().NotNil(gatheringLine.TaxConfig.Stripe, "Stripe.Code must be backfilled on gathering line via TaxCode edge")
	s.Equal(stripeCode, gatheringLine.TaxConfig.Stripe.Code)
}

// TestFlatFeeInvoiceSettlementNilTaxConfigDoesNotPropagateToGatheringLine verifies that when
// Intent.TaxConfig is nil the flat-fee CreditThenInvoice gathering line's TaxConfig is also nil.
func (s *TaxCodePersistenceTestSuite) TestFlatFeeInvoiceSettlementNilTaxConfigDoesNotPropagateToGatheringLine() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-flatfee-gathering-nil")

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	cust := s.CreateTestCustomer(ns, "test-subject")

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.SetTime(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))

	_, err := s.Charges.Create(ctx, charges.CreateInput{
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
				name:              "flat-fee-gathering-nil-taxcode",
				managedBy:         billing.ManuallyManagedLine,
				uniqueReferenceID: "flat-fee-gathering-nil-taxcode",
			}),
		},
	})
	s.NoError(err)

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

// TestUsageBasedCreditThenInvoicePropagatesTaxConfigToGatheringLine verifies that TaxConfig set on
// a usage-based CreditThenInvoice intent is propagated to the gathering invoice line built by
// gatheringLineFromUsageBasedCharge. Guards the same single-source-of-truth contract as the flat-fee
// equivalent, covering the usage-based charge type path.
func (s *TaxCodePersistenceTestSuite) TestUsageBasedCreditThenInvoicePropagatesTaxConfigToGatheringLine() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-usagebased-gathering")

	customInvoicing := s.SetupCustomInvoicing(ns)
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithManualApproval(),
	)
	cust := s.CreateTestCustomer(ns, "test-subject")
	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer apiRequestsTotal.Cleanup()

	const stripeCode = "txcd_30000002"
	tc := s.createTestTaxCodeWithStripeMapping(ctx, ns, "txcd-30000002", stripeCode)

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	// Clock before service period keeps the gathering line pending.
	clock.SetTime(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))

	_, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(1),
				}),
				featureKey:        apiRequestsTotal.Feature.Key,
				name:              "usage-based-gathering-taxcode",
				managedBy:         billing.ManuallyManagedLine,
				uniqueReferenceID: "usage-based-gathering-taxcode",
				taxConfig: &productcatalog.TaxCodeConfig{
					Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
					TaxCodeID: &tc.ID,
				},
			}),
		},
	})
	s.NoError(err)

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
	s.Equal(productcatalog.InclusiveTaxBehavior, *gatheringLine.TaxConfig.Behavior)
	s.Require().NotNil(gatheringLine.TaxConfig.TaxCodeID, "TaxCodeID must propagate to gathering line")
	s.Equal(tc.ID, *gatheringLine.TaxConfig.TaxCodeID)
	s.Require().NotNil(gatheringLine.TaxConfig.Stripe, "Stripe.Code must be backfilled on gathering line via TaxCode edge")
	s.Equal(stripeCode, gatheringLine.TaxConfig.Stripe.Code)
}

// TestUsageBasedInvoiceSettlementPopulatesStripeCodeOnStandardInvoice verifies the dual-write
// invariant for usage-based credit_then_invoice charges: after payment is settled, the standard
// invoice line carries both TaxCodeID (FK) and Stripe.Code resolved from the TaxCode entity.
// Mirrors TestFlatFeeInvoiceSettlementPopulatesStripeCodeOnStandardInvoice for the usage-based path.
func (s *TaxCodePersistenceTestSuite) TestUsageBasedInvoiceSettlementPopulatesStripeCodeOnStandardInvoice() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-usagebased-invoice-settled")

	customInvoicing := s.SetupCustomInvoicing(ns)
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)
	cust := s.CreateTestCustomer(ns, "test-subject")
	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer apiRequestsTotal.Cleanup()
	meterSlug := apiRequestsTotal.Feature.Key

	const stripeCode = "txcd_30000010"
	tc := s.createTestTaxCodeWithStripeMapping(ctx, ns, "txcd-30000010", stripeCode)

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.SetTime(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))

	_, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:          cust.GetID(),
				currency:          USD,
				servicePeriod:     servicePeriod,
				settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
				price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(1)}),
				featureKey:        meterSlug,
				name:              "usage-based-invoice-stripe-taxcode",
				managedBy:         billing.ManuallyManagedLine,
				uniqueReferenceID: "usage-based-invoice-stripe-taxcode",
				taxConfig: &productcatalog.TaxCodeConfig{
					Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					TaxCodeID: &tc.ID,
				},
			}),
		},
	})
	s.NoError(err)

	// Return empty allocations — no credits in balance, so nothing to apply.
	s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued = func(_ context.Context, _ usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
		return creditrealization.CreateAllocationInputs{}, nil
	}

	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 5, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	clock.SetTime(servicePeriod.To.Add(time.Second))
	createdInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     lo.ToPtr(servicePeriod.To),
	})
	s.NoError(err)
	s.Require().Len(createdInvoices, 1)
	invoice := createdInvoices[0]
	invoiceID := invoice.GetInvoiceID()

	clock.SetTime(invoice.DefaultCollectionAtForStandardInvoice())
	_, err = s.BillingService.AdvanceInvoice(ctx, invoiceID)
	s.NoError(err)

	s.UsageBasedTestHandler.onInvoiceUsageAccrued = func(_ context.Context, _ usagebased.OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error) {
		return ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()}, nil
	}
	s.UsageBasedTestHandler.onPaymentAuthorized = func(_ context.Context, _ usagebased.OnPaymentAuthorizedInput) (ledgertransaction.GroupReference, error) {
		return ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()}, nil
	}
	s.UsageBasedTestHandler.onPaymentSettled = func(_ context.Context, _ usagebased.OnPaymentSettledInput) (ledgertransaction.GroupReference, error) {
		return ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()}, nil
	}

	_, err = s.BillingService.ApproveInvoice(ctx, invoiceID)
	s.NoError(err)

	_, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
		InvoiceID: invoiceID,
		Trigger:   billing.TriggerPaid,
	})
	s.NoError(err)

	finalInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: invoiceID,
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.NoError(err)

	lines := finalInvoice.Lines.OrEmpty()
	s.Require().Len(lines, 1)
	line := lines[0]

	s.Require().NotNil(line.TaxConfig, "standard invoice line must have TaxConfig")
	s.Require().NotNil(line.TaxConfig.Behavior, "TaxBehavior must be on standard invoice line")
	s.Equal(productcatalog.ExclusiveTaxBehavior, *line.TaxConfig.Behavior)
	s.Require().NotNil(line.TaxConfig.TaxCodeID, "TaxCodeID must be on standard invoice line")
	s.Equal(tc.ID, *line.TaxConfig.TaxCodeID)
	s.Require().NotNil(line.TaxConfig.Stripe, "Stripe.Code must be backfilled on standard invoice line via TaxCode edge")
	s.Equal(stripeCode, line.TaxConfig.Stripe.Code)
}

// TestFlatFeeBehaviorOnlyTaxConfigPropagatesToGatheringLine verifies that a TaxCodeConfig with only
// Behavior set (no TaxCodeID) propagates correctly to the flat-fee gathering line. Guards against
// regressions that drop Behavior when TaxCodeID is nil.
func (s *TaxCodePersistenceTestSuite) TestFlatFeeBehaviorOnlyTaxConfigPropagatesToGatheringLine() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-flatfee-gathering-behavior-only")

	customInvoicing := s.SetupCustomInvoicing(ns)
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithManualApproval(),
	)
	cust := s.CreateTestCustomer(ns, "test-subject")

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.SetTime(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))

	_, err := s.Charges.Create(ctx, charges.CreateInput{
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
				name:              "flat-fee-gathering-behavior-only",
				managedBy:         billing.ManuallyManagedLine,
				uniqueReferenceID: "flat-fee-gathering-behavior-only",
				taxConfig:         &productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior)},
			}),
		},
	})
	s.NoError(err)

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

	s.Require().NotNil(gatheringLine.TaxConfig, "gathering line TaxConfig must be set")
	s.Require().NotNil(gatheringLine.TaxConfig.Behavior, "TaxBehavior must propagate to gathering line")
	s.Equal(productcatalog.InclusiveTaxBehavior, *gatheringLine.TaxConfig.Behavior)
	s.Nil(gatheringLine.TaxConfig.TaxCodeID, "TaxCodeID must be nil for behavior-only TaxConfig")
	s.Nil(gatheringLine.TaxConfig.Stripe, "Stripe must be nil when no TaxCodeID to resolve")
}

// TestUsageBasedBehaviorOnlyTaxConfigPropagatesToGatheringLine verifies that a TaxCodeConfig with
// only Behavior set (no TaxCodeID) propagates correctly to the usage-based gathering line. Guards
// against regressions that drop Behavior when TaxCodeID is nil.
func (s *TaxCodePersistenceTestSuite) TestUsageBasedBehaviorOnlyTaxConfigPropagatesToGatheringLine() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-taxcode-usagebased-gathering-behavior-only")

	customInvoicing := s.SetupCustomInvoicing(ns)
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithManualApproval(),
	)
	cust := s.CreateTestCustomer(ns, "test-subject")
	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer apiRequestsTotal.Cleanup()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	clock.SetTime(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC))

	_, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:          cust.GetID(),
				currency:          USD,
				servicePeriod:     servicePeriod,
				settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
				price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(1)}),
				featureKey:        apiRequestsTotal.Feature.Key,
				name:              "usage-based-gathering-behavior-only",
				managedBy:         billing.ManuallyManagedLine,
				uniqueReferenceID: "usage-based-gathering-behavior-only",
				taxConfig:         &productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior)},
			}),
		},
	})
	s.NoError(err)

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

	s.Require().NotNil(gatheringLine.TaxConfig, "gathering line TaxConfig must be set")
	s.Require().NotNil(gatheringLine.TaxConfig.Behavior, "TaxBehavior must propagate to gathering line")
	s.Equal(productcatalog.ExclusiveTaxBehavior, *gatheringLine.TaxConfig.Behavior)
	s.Nil(gatheringLine.TaxConfig.TaxCodeID, "TaxCodeID must be nil for behavior-only TaxConfig")
	s.Nil(gatheringLine.TaxConfig.Stripe, "Stripe must be nil when no TaxCodeID to resolve")
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

func (s *TaxCodePersistenceTestSuite) createTestTaxCodeWithStripeMapping(ctx context.Context, ns, key, stripeCode string) taxcode.TaxCode {
	s.T().Helper()
	tc, err := s.TaxCodeService.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       key,
		Name:      "Test Tax Code " + key,
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: stripeCode},
		},
	})
	s.Require().NoError(err, "creating test tax code with stripe mapping must succeed")
	return tc
}
