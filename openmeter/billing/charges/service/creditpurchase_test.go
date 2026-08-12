package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	creditpurchaseservice "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type CreditPurchaseTestSuite struct {
	BaseSuite
}

func TestCreditPurchase(t *testing.T) {
	suite.Run(t, new(CreditPurchaseTestSuite))
}

func (s *CreditPurchaseTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *CreditPurchaseTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *CreditPurchaseTestSuite) TestPromotionalCreditPurchase() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-promotional-credit-purchase")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	intent := CreateCreditPurchaseIntent(s.T(),
		createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromFloat(100),
			servicePeriod: timeutil.ClosedPeriod{
				From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
				To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
			},
			settlement: creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
		},
	)

	promotionalCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
	s.CreditPurchaseTestHandler.onPromotionalCreditPurchase = promotionalCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
		assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypePromotional)
		assert.Nil(t, charge.Realizations.CreditGrantRealization, "credit grant realization should not be set")
		assert.Nil(t, charge.Realizations.ExternalPaymentSettlement, "external payment settlement should not be set")
	})

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			intent,
		},
	})
	s.NoError(err)
	s.Len(res, 1)
	s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())

	s.Equal(1, promotionalCallback.nrInvocations)
	cpCharge, err := res[0].AsCreditPurchaseCharge()
	s.NoError(err)
	s.NotNil(cpCharge.Realizations.CreditGrantRealization)
	s.Equal(promotionalCallback.id, cpCharge.Realizations.CreditGrantRealization.GroupReference.TransactionGroupID)
	s.Equal(creditpurchase.StatusFinal, cpCharge.Status)

	charge := s.mustGetChargeByID(cpCharge.GetChargeID())
	updatedCPCharge, err := charge.AsCreditPurchaseCharge()
	s.NoError(err)
	s.Equal(promotionalCallback.id, updatedCPCharge.Realizations.CreditGrantRealization.GroupReference.TransactionGroupID)
	s.Equal(creditpurchase.StatusFinal, updatedCPCharge.Status)
}

func (s *CreditPurchaseTestSuite) TestPromotionalCreditPurchaseWithCustomCurrency() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-promotional-credit-purchase-custom-currency")

	var customCurrency currencies.Currency
	var customerID string
	var defaultCreditGrantTaxCodeID string
	var createdCharge creditpurchase.Charge
	var promotionalTransactionGroupID string

	s.Run("#1 setup customer and custom currency", func() {
		// given:
		// - a customer and a persisted custom currency
		defaultCreditGrantTaxCodeID = s.ProvisionDefaultTaxCodes(ctx, ns).CreditGrantTaxCodeID

		cust := s.CreateTestCustomer(ns, "test-subject")
		s.NotEmpty(cust.ID)
		customerID = cust.ID
		customCurrency = s.createTestCustomCurrency(ctx, ns)
	})

	s.Run("#2 create promotional credit purchase", func() {
		// given:
		// - a promotional credit-purchase intent in the custom currency
		// - mocked ledger and lineage callbacks
		// when:
		// - the charge is created through the root charges service
		// then:
		// - the callbacks run once and the charge reaches final with a persisted realization
		servicePeriod := timeutil.ClosedPeriod{
			From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
			To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
		}
		creditPurchaseIntent := creditpurchase.Intent{
			Intent: meta.Intent{
				ManagedBy:  billing.ManuallyManagedLine,
				CustomerID: customerID,
				Currency:   customCurrency,
				TaxConfig: productcatalog.TaxCodeConfig{
					TaxCodeID: defaultCreditGrantTaxCodeID,
				},
			},
			IntentMutableFields: creditpurchase.IntentMutableFields{
				IntentMutableFields: meta.IntentMutableFields{
					Name:              "Custom Currency Credit Purchase",
					ServicePeriod:     servicePeriod,
					BillingPeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
				},
				CreditAmount: alpacadecimal.NewFromFloat(100.1234),
				Settlement:   creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
			},
		}
		intent := charges.NewChargeIntent(creditPurchaseIntent)

		promotionalCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		promotionalTransactionGroupID = promotionalCallback.id
		s.CreditPurchaseTestHandler.onPromotionalCreditPurchase = promotionalCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
			assert.Equal(t, creditpurchase.SettlementTypePromotional, charge.Intent.Settlement.Type())
			assert.True(t, charge.Intent.Currency.IsCustom())
			assert.Equal(t, customCurrency.ID, charge.Intent.Currency.ID)
			assert.Nil(t, charge.Realizations.CreditGrantRealization)
		})

		lineageMock := &mockLineageService{Service: s.LineageService}
		lineageMock.On("BackfillAdvanceLineageSegments", mock.Anything, mock.Anything).
			Return(nil).
			Once()

		creditPurchaseService, err := creditpurchaseservice.New(creditpurchaseservice.Config{
			Adapter:     s.CreditPurchaseAdapter,
			Handler:     s.CreditPurchaseTestHandler,
			Lineage:     lineageMock,
			MetaAdapter: s.MetaAdapter,
			Currencies:  s.CurrencyService,
		})
		s.Require().NoError(err)

		originalCreditPurchaseService := s.Charges.creditPurchaseService
		s.Charges.creditPurchaseService = creditPurchaseService
		defer func() {
			s.Charges.creditPurchaseService = originalCreditPurchaseService
		}()

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents:   charges.ChargeIntents{intent},
		})
		s.Require().NoError(err)
		s.Require().Len(created, 1)
		s.Equal(1, promotionalCallback.nrInvocations)
		lineageMock.AssertExpectations(s.T())

		createdCharge, err = created[0].AsCreditPurchaseCharge()
		s.Require().NoError(err)
		s.Equal(creditpurchase.StatusFinal, createdCharge.Status)
		s.True(createdCharge.Intent.Currency.IsCustom())
		s.Equal(customCurrency.ID, createdCharge.Intent.Currency.ID)
		s.Equal(float64(100.123), createdCharge.Intent.CreditAmount.InexactFloat64())
		s.Require().NotNil(createdCharge.Realizations.CreditGrantRealization)
		s.Equal(promotionalTransactionGroupID, createdCharge.Realizations.CreditGrantRealization.TransactionGroupID)
	})

	s.Run("#3 reload persisted charge", func() {
		// when:
		// - the charge is loaded again from Postgres
		// then:
		// - its final state, custom currency, and realization are preserved
		persisted, err := s.mustGetChargeByID(createdCharge.GetChargeID()).AsCreditPurchaseCharge()
		s.Require().NoError(err)
		s.Equal(creditpurchase.StatusFinal, persisted.Status)
		s.True(persisted.Intent.Currency.IsCustom())
		s.Equal(customCurrency.ID, persisted.Intent.Currency.ID)
		s.Require().NotNil(persisted.Realizations.CreditGrantRealization)
		s.Equal(promotionalTransactionGroupID, persisted.Realizations.CreditGrantRealization.TransactionGroupID)
	})
}

func (s *CreditPurchaseTestSuite) TestCreditPurchaseRejectsFiatCostBasisForCustomCurrency() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-credit-purchase-fiat-cost-basis-custom-currency")
	defaultTaxCodes := s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)
	customCurrency := s.createTestCustomCurrency(ctx, ns)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	for _, tc := range []struct {
		name       string
		settlement creditpurchase.Settlement
	}{
		{
			name: "external",
			settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			}),
		},
		{
			name:       "invoice",
			settlement: creditpurchase.NewInvoiceSettlement(),
		},
	} {
		s.Run(tc.name, func() {
			intent := charges.NewChargeIntent(creditpurchase.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.ManuallyManagedLine,
					CustomerID: cust.ID,
					Currency:   customCurrency,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: defaultTaxCodes.CreditGrantTaxCodeID,
					},
				},
				IntentMutableFields: creditpurchase.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "Custom Currency Credit Purchase",
						ServicePeriod:     servicePeriod,
						BillingPeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
					},
					CreditAmount: alpacadecimal.NewFromFloat(100),
					Settlement:   tc.settlement,
				},
				CostBasis: creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{
					Rate: alpacadecimal.NewFromFloat(0.5),
				}),
			})

			res, err := s.Charges.Create(ctx, charges.CreateInput{
				Namespace: ns,
				Intents: charges.ChargeIntents{
					intent,
				},
			})
			s.ErrorContains(err, "custom currency credit purchase requires a custom currency cost basis")
			s.Empty(res)
		})
	}
}

func (s *CreditPurchaseTestSuite) TestCreditPurchaseRejectsNonPositiveCostBasisBeforeCallbacks() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-credit-purchase-non-positive-cost-basis")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	for _, tc := range []struct {
		name       string
		settlement creditpurchase.Settlement
		costBasis  alpacadecimal.Decimal
	}{
		{
			name:      "external zero",
			costBasis: alpacadecimal.Zero,
			settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			}),
		},
		{
			name:      "external negative",
			costBasis: alpacadecimal.NewFromFloat(-0.5),
			settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			}),
		},
		{
			name:       "invoice zero",
			costBasis:  alpacadecimal.Zero,
			settlement: creditpurchase.NewInvoiceSettlement(),
		},
		{
			name:       "invoice negative",
			costBasis:  alpacadecimal.NewFromFloat(-0.5),
			settlement: creditpurchase.NewInvoiceSettlement(),
		},
	} {
		s.Run(tc.name, func() {
			// given:
			// - a credit-purchase intent with a non-positive external or invoice cost basis
			// when:
			// - charge creation validates the intent
			// then:
			// - it fails before lifecycle callbacks or charge persistence can run
			costBasis := creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{Rate: tc.costBasis})
			intent := charges.NewChargeIntent(creditpurchase.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.ManuallyManagedLine,
					CustomerID: cust.ID,
					Currency:   currenciestestutils.NewFiatCurrency(s.T(), USD),
				},
				IntentMutableFields: creditpurchase.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "Credit Purchase",
						ServicePeriod:     servicePeriod,
						BillingPeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
					},
					CreditAmount: alpacadecimal.NewFromFloat(100),
					Settlement:   tc.settlement,
				},
				CostBasis: costBasis,
			})

			res, err := s.Charges.Create(ctx, charges.CreateInput{
				Namespace: ns,
				Intents: charges.ChargeIntents{
					intent,
				},
			})

			s.Error(err)
			s.True(models.IsGenericValidationError(err))
			s.ErrorContains(err, "rate must be positive")
			s.Empty(res)

			chargesResult, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
				Namespace:   ns,
				CustomerIDs: []string{cust.ID},
				ChargeTypes: []meta.ChargeType{meta.ChargeTypeCreditPurchase},
			})
			s.NoError(err)
			s.Empty(chargesResult.Items)
		})
	}
}

type createCreditPurchaseIntentInput struct {
	customer      customer.CustomerID
	currency      currencyx.Code
	amount        alpacadecimal.Decimal
	servicePeriod timeutil.ClosedPeriod
	settlement    creditpurchase.Settlement
	costBasis     creditpurchase.CostBasis
}

func (i createCreditPurchaseIntentInput) Validate() error {
	if err := i.customer.Validate(); err != nil {
		return fmt.Errorf("customer: %w", err)
	}

	if i.currency == "" {
		return errors.New("currency is required")
	}

	if !i.amount.IsPositive() {
		return errors.New("amount must be positive")
	}

	if err := i.servicePeriod.Validate(); err != nil {
		return fmt.Errorf("service period: %w", err)
	}

	if err := i.settlement.Validate(); err != nil {
		return fmt.Errorf("settlement: %w", err)
	}

	return nil
}

func CreateCreditPurchaseIntent(t *testing.T, input createCreditPurchaseIntentInput) charges.ChargeIntent {
	t.Helper()
	require.NoError(t, input.Validate())

	return charges.NewChargeIntent(creditpurchase.Intent{
		Intent: meta.Intent{
			ManagedBy:  billing.ManuallyManagedLine,
			CustomerID: input.customer.ID,
			Currency:   currenciestestutils.NewFiatCurrency(t, input.currency),
		},
		IntentMutableFields: creditpurchase.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              "Credit Purchase",
				ServicePeriod:     input.servicePeriod,
				BillingPeriod:     input.servicePeriod,
				FullServicePeriod: input.servicePeriod,
			},
			CreditAmount: input.amount,
			Settlement:   input.settlement,
		},
		CostBasis: input.costBasis,
	})
}

func newFiatCreditPurchaseCostBasis(rate alpacadecimal.Decimal) creditpurchase.CostBasis {
	return creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{Rate: rate})
}

func (s *CreditPurchaseTestSuite) TestExternalAuthorizedCreditPurchaseAutoSettled() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-external-authorized-credit-purchase-auto-settled")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	// Let's buy 100.123 USD credits at a $0.50 cost basis. The external
	// payment settles for $50.06 after rounding to USD precision.
	intent := CreateCreditPurchaseIntent(s.T(),
		createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromFloat(100.123),
			servicePeriod: timeutil.ClosedPeriod{
				From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
				To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
			},
			settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				InitialStatus: creditpurchase.SettledInitialPaymentSettlementStatus,
			}),
			costBasis: newFiatCreditPurchaseCostBasis(alpacadecimal.NewFromFloat(0.5)),
		},
	)

	// The initiated callback is invoked before the payment lifecycle starts, so the
	// purchased credits are available while the external payment is still pending.
	initiatedCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
	s.CreditPurchaseTestHandler.onCreditPurchaseInitiated = initiatedCallback.Handler(s.T(), func(t *testing.T, charge creditpurchase.Charge) {
		assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypeExternal)
		assert.Nil(t, charge.Realizations.CreditGrantRealization, "credit grant realization should not be set")
		assert.Nil(t, charge.Realizations.ExternalPaymentSettlement, "external payment settlement should not be set")
		assert.Equal(t, creditpurchase.StatusActiveInitialCreditGrant, charge.Status, "charge status should be initial credit grant")
	})

	// Then the authorized callback should be called in the direct-paid authorization state,
	// with the credit grant realization and no payment settlement.
	authorizedCallback := newCountedLedgerTransactionCallback[creditpurchase.PaymentEventInput]()
	s.CreditPurchaseTestHandler.onCreditPurchasePaymentAuthorized = authorizedCallback.Handler(s.T(), func(t *testing.T, input creditpurchase.PaymentEventInput) {
		charge := input.Charge
		assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypeExternal)
		assert.NotNil(t, charge.Realizations.CreditGrantRealization, "credit grant realization should be set")
		assert.Equal(t, initiatedCallback.id, charge.Realizations.CreditGrantRealization.TransactionGroupID)
		assert.Nil(t, charge.Realizations.ExternalPaymentSettlement)
		assert.Equal(t, creditpurchase.StatusActivePaymentPaidAndAuthorized, charge.Status, "charge status should be paid and authorized")
	})

	// Then the settled callback should be called in the settlement state with a grant
	// realization and a payment settlement.
	settledCallback := newCountedLedgerTransactionCallback[creditpurchase.PaymentEventInput]()
	s.CreditPurchaseTestHandler.onCreditPurchasePaymentSettled = settledCallback.Handler(s.T(), func(t *testing.T, input creditpurchase.PaymentEventInput) {
		charge := input.Charge
		assert.Equal(t, charge.Intent.Settlement.Type(), creditpurchase.SettlementTypeExternal)
		assert.NotNil(t, charge.Realizations.CreditGrantRealization, "credit grant realization should be set")
		assert.Equal(t, initiatedCallback.id, charge.Realizations.CreditGrantRealization.TransactionGroupID)
		assert.NotNil(t, charge.Realizations.ExternalPaymentSettlement, "external payment settlement should be set")

		// Authorized transaction group ID should be set
		assert.Equal(t, authorizedCallback.id, charge.Realizations.ExternalPaymentSettlement.Authorized.TransactionGroupID)
		assert.Equal(t, payment.StatusAuthorized, charge.Realizations.ExternalPaymentSettlement.Status)
		assert.Equal(t, creditpurchase.StatusActivePaymentSettled, charge.Status, "charge status should be payment settled")
	})
	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			intent,
		},
	})
	s.NoError(err)
	s.Len(res, 1)
	s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())

	s.Equal(1, initiatedCallback.nrInvocations)
	s.Equal(1, authorizedCallback.nrInvocations)
	s.Equal(1, settledCallback.nrInvocations)

	dbCharge := s.mustGetChargeByID(lo.Must(res[0].GetChargeID()))

	// Let's validate both the output from the Create and the DB state
	for _, tc := range []struct {
		name   string
		charge charges.Charge
	}{
		{name: "output", charge: res[0]},
		{name: "db", charge: dbCharge},
	} {
		s.Run(tc.name, func() {
			// The charge should have both a grant and a payment settlement.
			creditPurchaseCharge, err := tc.charge.AsCreditPurchaseCharge()
			s.NoError(err)
			s.Require().False(creditPurchaseCharge.Intent.CostBasis.IsEmpty())
			fiatCostBasis, err := creditPurchaseCharge.Intent.CostBasis.AsFiat()
			s.NoError(err)
			s.Equal(float64(0.5), fiatCostBasis.Rate.InexactFloat64())
			s.NotNil(creditPurchaseCharge.Realizations.CreditGrantRealization, "credit grant realization should be set")
			s.Equal(initiatedCallback.id, creditPurchaseCharge.Realizations.CreditGrantRealization.TransactionGroupID)

			// Payment settlement should be set
			s.NotNil(creditPurchaseCharge.Realizations.ExternalPaymentSettlement, "external payment settlement should be set")
			s.Equal(authorizedCallback.id, creditPurchaseCharge.Realizations.ExternalPaymentSettlement.Authorized.TransactionGroupID, "authorized transaction group ID should be set")
			s.Equal(settledCallback.id, creditPurchaseCharge.Realizations.ExternalPaymentSettlement.Settled.TransactionGroupID, "settled transaction group ID should be set")
			s.Equal(float64(50.06), creditPurchaseCharge.Realizations.ExternalPaymentSettlement.FiatAmount.InexactFloat64())

			// The charge should be final
			s.Equal(creditpurchase.StatusFinal, creditPurchaseCharge.Status)
		})
	}
}

func (s *CreditPurchaseTestSuite) TestStandardInvoiceCreditPurchaseDeferred() {
	// This test exercises the deferred invoicing path where InvoiceAt is in the future.
	// In this case, InvoiceSettlement remains nil at Create() time.
	// The gathering line is created but not immediately invoiced.
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-standard-invoice-credit-purchase-deferred")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	clock.FreezeTime(datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime())
	defer clock.ResetTime()

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	customInvoicing := s.SetupCustomInvoicing(ns)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	// Let's buy 100 USD credits for $0.50 each (total cost is $50)
	// Service period is in the FUTURE to exercise the deferred invoicing path
	intent := CreateCreditPurchaseIntent(s.T(),
		createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromFloat(100),
			servicePeriod: timeutil.ClosedPeriod{
				From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
				To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
			},
			settlement: creditpurchase.NewInvoiceSettlement(),
			costBasis:  newFiatCreditPurchaseCostBasis(alpacadecimal.NewFromFloat(0.5)),
		},
	)

	var chargeID meta.ChargeID

	s.Run("initiated", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				intent,
			},
		})
		s.NoError(err)
		s.Len(res, 1)
		s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())

		creditPurchaseCharge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)
		s.Equal(creditpurchase.StatusCreated, creditPurchaseCharge.Status)

		// For deferred invoicing, InvoiceSettlement should be nil at this point
		// because the invoice hasn't been created yet (InvoiceAt is in the future)
		assert.Nil(s.T(), creditPurchaseCharge.Realizations.InvoiceSettlement, "invoice settlement should be nil for deferred invoicing")

		chargeID = creditPurchaseCharge.GetChargeID()
	})

	s.Run("gathering_line_created", func() {
		gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
			Namespace: ns,
			Customers: []string{cust.ID},
			Expand:    billing.GatheringInvoiceExpandAll,
		})
		s.NoError(err)
		s.Len(gatheringInvoices.Items, 1)
		gatheringInvoice := gatheringInvoices.Items[0]

		lines := gatheringInvoice.Lines.OrEmpty()
		s.Len(lines, 1)
		line := lines[0]

		s.Equal(*line.ChargeID, chargeID.ID)
	})

	// The TestStandardInvoiceCreditPurchase test covers the full non-deferred invoicing path.
	// This path only covers parts up to the point where the gathering line is created, as for
	// invoice based triggers the lifecycle is governed by the invoicing line engine.
}
