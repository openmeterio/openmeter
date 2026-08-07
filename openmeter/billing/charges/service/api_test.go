package service

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	featurepkg "github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestCustomerChargeAPIDelete(t *testing.T) {
	suite.Run(t, new(CustomerChargeAPIDeleteTestSuite))
}

type CustomerChargeAPIDeleteTestSuite struct {
	BaseSuite
}

func (s *CustomerChargeAPIDeleteTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *CustomerChargeAPIDeleteTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *CustomerChargeAPIDeleteTestSuite) TestDeleteCustomerChargeMapsPaymentAdjustmentNoneToIgnoredCreditRefund() {
	defer s.FlatFeeTestHandler.Reset()

	ctx := s.T().Context()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	var namespace string
	var customerID string
	var chargeID meta.ChargeID

	s.Run("given a realized credit-only flat-fee charge", func() {
		// given:
		// - a subscription-managed flat-fee charge with an allocated credit realization
		// - no credit-correction handler, so a correct policy would fail deletion
		namespace = s.GetUniqueNamespace("charges-service-api-delete-policy")
		s.ProvisionDefaultTaxCodes(ctx, namespace)

		customer := s.CreateTestCustomer(namespace, "api-delete-policy")
		customerID = customer.ID
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), billingtest.WithManualApproval())

		s.FlatFeeTestHandler.onAllocateCredits = func(_ context.Context, input flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			return creditrealization.CreateAllocationInputs{
				{
					ServicePeriod: input.Charge.Intent.GetEffectiveServicePeriod(),
					Amount:        input.PreTaxAmountToAllocate,
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: ulid.Make().String(),
					},
				},
			}, nil
		}

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: namespace,
			Intents: charges.ChargeIntents{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       customer.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditOnlySettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(10),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					name:              "flat-fee",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-delete-policy-flat-fee",
				}),
			},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 1)

		chargeID, err = created[0].GetChargeID()
		require.NoError(s.T(), err)

		createdFlatFee, err := created[0].AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.NotNil(s.T(), createdFlatFee.Realizations.CurrentRun)
		require.Len(s.T(), createdFlatFee.Realizations.CurrentRun.CreditRealizations, 1)
	})

	s.Run("when deleting with no payment adjustment", func() {
		// when:
		// - the caller deletes the charge without compensating prior credit effects
		err := s.Charges.DeleteCustomerCharge(ctx, charges.DeleteCustomerChargeInput{
			Namespace:         namespace,
			CustomerID:        customerID,
			ChargeID:          chargeID.ID,
			PaymentAdjustment: charges.PaymentAdjustmentNone,
		})
		require.NoError(s.T(), err)
	})

	s.Run("then deletion preserves the credit realization", func() {
		// then:
		// - the charge is deleted without creating a credit correction
		// - the original allocation remains attached to the current run
		deletedCharge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
			ChargeID: chargeID,
			Expands:  meta.Expands{meta.ExpandRealizations},
		})
		require.NoError(s.T(), err)

		deletedFlatFee, err := deletedCharge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), flatfee.StatusDeleted, deletedFlatFee.Status)
		require.NotNil(s.T(), deletedFlatFee.Realizations.CurrentRun)
		require.Len(s.T(), deletedFlatFee.Realizations.CurrentRun.CreditRealizations, 1)
	})
}

func (s *CustomerChargeAPIDeleteTestSuite) TestDeleteCustomerChargePreservesPaidInvoiceWhenPaymentAdjustmentIsNone() {
	ctx := s.T().Context()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	var namespace string
	var customerID string
	var chargeID meta.ChargeID
	var invoiceID billing.InvoiceID
	var lineID billing.LineID
	var paymentAuthorizedTransactionID string
	var paymentSettledTransactionID string

	s.Run("given a paid invoice for a subscription-managed flat-fee charge", func() {
		// given:
		// - an invoice-backed flat-fee charge has been invoiced and paid
		// - its payment and invoice are immutable billing history
		namespace = s.GetUniqueNamespace("charges-service-api-delete-ignore-invoice")
		s.ProvisionDefaultTaxCodes(ctx, namespace)

		customer := s.CreateTestCustomer(namespace, "api-delete-ignore-invoice")
		customerID = customer.ID
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), billingtest.WithManualApproval())

		s.FlatFeeTestHandler.onAllocateCredits = func(context.Context, flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			return nil, nil
		}
		s.FlatFeeTestHandler.onInvoiceUsageAccrued = newCountedLedgerTransactionCallback[flatfee.OnInvoiceUsageAccruedInput]().Handler(s.T())
		paymentAuthorizedCallback := newCountedLedgerTransactionCallback[flatfee.OnPaymentAuthorizedInput]()
		paymentAuthorizedTransactionID = paymentAuthorizedCallback.id
		s.FlatFeeTestHandler.onPaymentAuthorized = paymentAuthorizedCallback.Handler(s.T())
		paymentSettledCallback := newCountedLedgerTransactionCallback[flatfee.OnPaymentSettledInput]()
		paymentSettledTransactionID = paymentSettledCallback.id
		s.FlatFeeTestHandler.onPaymentSettled = paymentSettledCallback.Handler(s.T())

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: namespace,
			Intents: charges.ChargeIntents{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       customer.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(10),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					name:              "flat-fee",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-delete-ignore-invoice-flat-fee",
				}),
			},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 1)

		chargeID, err = created[0].GetChargeID()
		require.NoError(s.T(), err)

		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customer.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.From),
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), invoices, 1)
		require.Len(s.T(), invoices[0].Lines.OrEmpty(), 1)

		invoice, err := s.BillingService.ApproveInvoice(ctx, invoices[0].GetInvoiceID())
		require.NoError(s.T(), err)
		require.Equal(s.T(), billing.StandardInvoiceStatusPaid, invoice.Status)
		require.Len(s.T(), invoice.Lines.OrEmpty(), 1)

		invoiceID = invoice.GetInvoiceID()
		lineID = invoice.Lines.OrEmpty()[0].GetLineID()
	})

	s.Run("when deleting the charge with no payment adjustment", func() {
		// when:
		// - the caller deletes the charge without compensating its invoice payment
		err := s.Charges.DeleteCustomerCharge(ctx, charges.DeleteCustomerChargeInput{
			Namespace:         namespace,
			CustomerID:        customerID,
			ChargeID:          chargeID.ID,
			PaymentAdjustment: charges.PaymentAdjustmentNone,
		})
		require.NoError(s.T(), err)
	})

	s.Run("then the paid invoice and its payment history remain unchanged", func() {
		// then:
		// - the effective charge is deleted through a manual override
		// - the paid invoice and its line remain intact
		// - immutable invoice drift is recorded for later reconciliation
		deletedCharge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
			ChargeID: chargeID,
			Expands:  meta.Expands{meta.ExpandRealizations},
		})
		require.NoError(s.T(), err)
		deletedFlatFee, err := deletedCharge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), flatfee.StatusDeleted, deletedFlatFee.Status)
		require.Nil(s.T(), deletedFlatFee.Intent.GetBaseIntent().IntentDeletedAt)
		override := deletedFlatFee.Intent.GetOverrideLayerMutableFields()
		require.NotNil(s.T(), override)
		require.NotNil(s.T(), override.IntentDeletedAt)
		require.Nil(s.T(), deletedFlatFee.Realizations.CurrentRun)
		require.Len(s.T(), deletedFlatFee.Realizations.PriorRuns, 1)
		payment := deletedFlatFee.Realizations.PriorRuns[0].Payment
		require.NotNil(s.T(), payment)
		require.NotNil(s.T(), payment.Authorized)
		require.Equal(s.T(), paymentAuthorizedTransactionID, payment.Authorized.TransactionGroupID)
		require.NotNil(s.T(), payment.Settled)
		require.Equal(s.T(), paymentSettledTransactionID, payment.Settled.TransactionGroupID)

		invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoiceID,
			Expand:  billing.StandardInvoiceExpandAll,
		})
		require.NoError(s.T(), err)
		require.Equal(s.T(), billing.StandardInvoiceStatusPaid, invoice.Status)
		require.Nil(s.T(), invoice.DeletedAt)
		require.Len(s.T(), invoice.Lines.OrEmpty(), 1)
		require.Equal(s.T(), lineID, invoice.Lines.OrEmpty()[0].GetLineID())
		require.Nil(s.T(), invoice.Lines.OrEmpty()[0].DeletedAt)
		require.Len(s.T(), invoice.ValidationIssues, 1)
		require.Equal(s.T(), billing.ImmutableInvoiceHandlingNotSupportedErrorCode, invoice.ValidationIssues[0].Code)
		require.Equal(s.T(), billing.ComponentName("charges.invoiceupdater"), invoice.ValidationIssues[0].Component)
	})

	s.Run("when clearing the deletion override", func() {
		// when:
		// - the caller restores the base after deletion left payment handling out of band
		_, err := s.Charges.ClearCustomerChargeOverride(ctx, charges.ClearCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   chargeID.ID,
		})
		require.NoError(s.T(), err)
	})

	s.Run("then prior invoice history does not block restored billing", func() {
		// then:
		// - the old paid realization remains prior history
		// - the base intent is effective and starts a new gathering lifecycle
		clearedCharge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
			ChargeID: chargeID,
			Expands:  meta.Expands{meta.ExpandRealizations},
		})
		require.NoError(s.T(), err)
		clearedFlatFee, err := clearedCharge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), flatfee.StatusActive, clearedFlatFee.Status)
		require.Nil(s.T(), clearedFlatFee.Intent.GetOverrideLayerMutableFields())
		require.Nil(s.T(), clearedFlatFee.Realizations.CurrentRun)
		require.Len(s.T(), clearedFlatFee.Realizations.PriorRuns, 1)

		payment := clearedFlatFee.Realizations.PriorRuns[0].Payment
		require.NotNil(s.T(), payment)
		require.NotNil(s.T(), payment.Authorized)
		require.Equal(s.T(), paymentAuthorizedTransactionID, payment.Authorized.TransactionGroupID)
		require.NotNil(s.T(), payment.Settled)
		require.Equal(s.T(), paymentSettledTransactionID, payment.Settled.TransactionGroupID)

		invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoiceID,
			Expand:  billing.StandardInvoiceExpandAll,
		})
		require.NoError(s.T(), err)
		require.Equal(s.T(), billing.StandardInvoiceStatusPaid, invoice.Status)
		require.Len(s.T(), invoice.Lines.OrEmpty(), 1)
		require.Equal(s.T(), lineID, invoice.Lines.OrEmpty()[0].GetLineID())
	})
}

func TestCustomerChargeAPIClearOverride(t *testing.T) {
	suite.Run(t, new(CustomerChargeAPIClearOverrideTestSuite))
}

type CustomerChargeAPIClearOverrideTestSuite struct {
	BaseSuite
}

func (s *CustomerChargeAPIClearOverrideTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *CustomerChargeAPIClearOverrideTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *CustomerChargeAPIClearOverrideTestSuite) TestClearCustomerChargeOverrideValidatesOwnershipAndIsIdempotent() {
	ctx := s.T().Context()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	var namespace string
	var customerID string
	var otherCustomerID string
	chargeIDs := map[meta.ChargeType]meta.ChargeID{}

	s.Run("given system-managed flat-fee and usage-based charges", func() {
		// given:
		// - future subscription-managed charges have no override layer
		// - their base mutable intents are the current effective intents
		// when:
		// - the test provisions both supported charge types
		// then:
		// - their IDs are available for façade validation checks
		namespace = s.GetUniqueNamespace("charges-service-api-clear-override")
		s.ProvisionDefaultTaxCodes(ctx, namespace)

		customer := s.CreateTestCustomer(namespace, "api-clear-override")
		customerID = customer.ID
		otherCustomerID = s.CreateTestCustomer(namespace, "api-clear-override-other").ID
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

		usageMeter := newTestMeter(namespace, "api-clear-override-meter")
		require.NoError(s.T(), s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{usageMeter}))
		usageFeature, err := s.FeatureService.CreateFeature(ctx, featurepkg.CreateFeatureInputs{
			Namespace: namespace,
			Name:      "api-clear-override-feature",
			Key:       "api-clear-override-feature",
			MeterID:   lo.ToPtr(usageMeter.ID),
		})
		require.NoError(s.T(), err)

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: namespace,
			Intents: charges.ChargeIntents{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       customer.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditOnlySettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(10),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					name:              "flat-fee-base",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-clear-override-flat-fee",
				}),
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          customer.GetID(),
					currency:          USD,
					servicePeriod:     servicePeriod,
					settlementMode:    productcatalog.CreditOnlySettlementMode,
					price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
					featureKey:        usageFeature.Key,
					name:              "usage-based-base",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-clear-override-usage-based",
				}),
			},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 2)

		for _, createdCharge := range created {
			chargeID, err := createdCharge.GetChargeID()
			require.NoError(s.T(), err)
			chargeIDs[createdCharge.Type()] = chargeID
		}
	})

	s.Run("given each charge has a customer override", func() {
		// given:
		// - both supported charge types have a customer override as clear-operation setup
		// when:
		// - the façade sets one override for each charge
		// then:
		// - subsequent clear calls have override layers to remove
		flatFeeCharge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{ChargeID: chargeIDs[meta.ChargeTypeFlatFee]})
		require.NoError(s.T(), err)
		flatFee, err := flatFeeCharge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		flatFeeFields := flatFee.Intent.GetEffectiveIntent().IntentMutableFields.Clone()
		flatFeeFields.Name = "flat-fee-override"
		_, err = s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   flatFee.ID,
			FlatFee:    &flatFeeFields,
		})
		require.NoError(s.T(), err)

		usageBasedCharge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{ChargeID: chargeIDs[meta.ChargeTypeUsageBased]})
		require.NoError(s.T(), err)
		usageBased, err := usageBasedCharge.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		usageBasedFields := usageBased.Intent.GetEffectiveIntent().IntentMutableFields.Clone()
		usageBasedFields.Name = "usage-based-override"
		_, err = s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   usageBased.ID,
			UsageBased: &usageBasedFields,
		})
		require.NoError(s.T(), err)
	})

	s.Run("when the payload type does not match the charge", func() {
		// given:
		// - a flat-fee charge and an otherwise valid usage-based override payload
		// when:
		// - the caller sets the usage-based payload on the flat-fee charge
		// then:
		// - the facade rejects the mismatched charge type before applying a patch
		flatFeeCharge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{ChargeID: chargeIDs[meta.ChargeTypeFlatFee]})
		require.NoError(s.T(), err)
		flatFee, err := flatFeeCharge.AsFlatFeeCharge()
		require.NoError(s.T(), err)

		usageBasedCharge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{ChargeID: chargeIDs[meta.ChargeTypeUsageBased]})
		require.NoError(s.T(), err)
		usageBased, err := usageBasedCharge.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		usageBasedFields := usageBased.Intent.GetEffectiveIntent().IntentMutableFields.Clone()
		_, err = s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   flatFee.ID,
			UsageBased: &usageBasedFields,
		})
		require.ErrorContains(s.T(), err, "flat fee override fields are required")
	})

	s.Run("when a different customer clears an override", func() {
		// given:
		// - the charge belongs to the original customer
		// when:
		// - a different customer attempts to clear its override
		// then:
		// - ownership validation rejects the request before changing the charge
		_, err := s.Charges.ClearCustomerChargeOverride(ctx, charges.ClearCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: otherCustomerID,
			ChargeID:   chargeIDs[meta.ChargeTypeFlatFee].ID,
		})
		require.ErrorContains(s.T(), err, "is not owned by customer")
	})

	s.Run("given the setup overrides are cleared", func() {
		// given:
		// - the ownership check left both customer overrides intact
		// when:
		// - the customer clears them once as idempotency-test setup
		// then:
		// - the following clear call starts without an override layer
		for _, chargeType := range []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased} {
			_, err := s.Charges.ClearCustomerChargeOverride(ctx, charges.ClearCustomerChargeOverrideInput{
				Namespace:  namespace,
				CustomerID: customerID,
				ChargeID:   chargeIDs[chargeType].ID,
			})
			require.NoError(s.T(), err)
		}
	})

	s.Run("when clearing the already absent overrides", func() {
		// given:
		// - both charges already expose their base intent
		// when:
		// - clear is repeated
		// then:
		// - the operation is an idempotent no-op
		for _, chargeType := range []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased} {
			cleared, err := s.Charges.ClearCustomerChargeOverride(ctx, charges.ClearCustomerChargeOverrideInput{
				Namespace:  namespace,
				CustomerID: customerID,
				ChargeID:   chargeIDs[chargeType].ID,
			})
			require.NoError(s.T(), err)
			require.Equal(s.T(), chargeType, cleared.Type())
		}
	})
}

func (s *CustomerChargeAPIClearOverrideTestSuite) TestOverrideOperationsRejectCreditPurchaseCharges() {
	defer s.CreditPurchaseTestHandler.Reset()

	ctx := s.T().Context()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	var namespace string
	var customerID string
	var chargeID string

	s.Run("given a customer credit-purchase charge", func() {
		// given:
		// - a credit purchase belongs to the customer
		// when:
		// - the test creates the credit-purchase charge
		// then:
		// - its ID is available for unsupported-operation checks
		namespace = s.GetUniqueNamespace("charges-service-api-override-credit-purchase")
		s.ProvisionDefaultTaxCodes(ctx, namespace)

		customer := s.CreateTestCustomer(namespace, "api-override-credit-purchase")
		customerID = customer.ID
		promotionalCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		s.CreditPurchaseTestHandler.onPromotionalCreditPurchase = promotionalCallback.Handler(s.T())
		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: namespace,
			Intents: charges.ChargeIntents{
				CreateCreditPurchaseIntent(s.T(), createCreditPurchaseIntentInput{
					customer:      customer.GetID(),
					currency:      USD,
					amount:        alpacadecimal.NewFromInt(10),
					servicePeriod: servicePeriod,
					settlement:    creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
				}),
			},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 1)
		chargeID = created[0].GetID()
	})

	s.Run("when setting or clearing an override", func() {
		// given:
		// - the customer owns a credit-purchase charge
		// when:
		// - the facade receives either override operation for the credit purchase
		// then:
		// - both operations reject the unsupported charge type
		_, err := s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   chargeID,
			FlatFee: &flatfee.IntentMutableFields{
				IntentMutableFields: meta.IntentMutableFields{
					Name:              "unused override",
					ServicePeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
					BillingPeriod:     servicePeriod,
				},
				InvoiceAt:             servicePeriod.From,
				PaymentTerm:           productcatalog.InAdvancePaymentTerm,
				AmountBeforeProration: alpacadecimal.NewFromInt(10),
			},
		})
		require.ErrorContains(s.T(), err, "credit purchase charges is not supported")

		_, err = s.Charges.ClearCustomerChargeOverride(ctx, charges.ClearCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   chargeID,
		})
		require.ErrorContains(s.T(), err, "credit purchase charges is not supported")
	})
}
