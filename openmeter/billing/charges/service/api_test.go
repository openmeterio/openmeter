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
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
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

func (s *CustomerChargeAPIDeleteTestSuite) TestDeleteCustomerChargeCreatesManualOverride() {
	ctx := s.T().Context()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	var namespace string
	var customerID string
	var chargeIDs []meta.ChargeID

	s.Run("given system-managed flat-fee and usage-based charges", func() {
		// given:
		// - one future flat-fee charge and one future usage-based charge
		// - both base intents are subscription-managed and have no override
		namespace = s.GetUniqueNamespace("charges-service-api-delete")
		s.ProvisionDefaultTaxCodes(ctx, namespace)

		customer := s.CreateTestCustomer(namespace, "api-delete")
		customerID = customer.ID
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

		usageMeter := newTestMeter(namespace, "api-delete-meter")
		require.NoError(s.T(), s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{usageMeter}))
		usageFeature, err := s.FeatureService.CreateFeature(ctx, featurepkg.CreateFeatureInputs{
			Namespace: namespace,
			Name:      "api-delete-feature",
			Key:       "api-delete-feature",
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
					name:              "flat-fee",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-delete-flat-fee",
				}),
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          customer.GetID(),
					currency:          USD,
					servicePeriod:     servicePeriod,
					settlementMode:    productcatalog.CreditOnlySettlementMode,
					price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
					featureKey:        usageFeature.Key,
					name:              "usage-based",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-delete-usage-based",
				}),
			},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 2)

		chargeIDs = make([]meta.ChargeID, 0, len(created))
		for _, createdCharge := range created {
			chargeID, err := createdCharge.GetChargeID()
			require.NoError(s.T(), err)
			chargeIDs = append(chargeIDs, chargeID)
		}
	})

	s.Run("when deleting the charges through the customer charge service", func() {
		// when:
		// - both charges are deleted without compensating payment adjustments
		for _, chargeID := range chargeIDs {
			err := s.Charges.DeleteCustomerCharge(ctx, charges.DeleteCustomerChargeInput{
				Namespace:         namespace,
				CustomerID:        customerID,
				ChargeID:          chargeID.ID,
				PaymentAdjustment: charges.PaymentAdjustmentNone,
			})
			require.NoError(s.T(), err)
		}
	})

	s.Run("then each effective charge is a deleted manual override", func() {
		// then:
		// - each charge is deleted through an override intent
		// - the subscription-managed base intent remains undeleted
		for _, chargeID := range chargeIDs {
			deletedCharge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{ChargeID: chargeID})
			require.NoError(s.T(), err)

			switch deletedCharge.Type() {
			case meta.ChargeTypeFlatFee:
				deletedFlatFee, err := deletedCharge.AsFlatFeeCharge()
				require.NoError(s.T(), err)
				require.Equal(s.T(), flatfee.StatusDeleted, deletedFlatFee.Status)
				require.Nil(s.T(), deletedFlatFee.Intent.GetBaseIntent().IntentDeletedAt)
				override := deletedFlatFee.Intent.GetOverrideLayerMutableFields()
				require.NotNil(s.T(), override)
				require.NotNil(s.T(), override.IntentDeletedAt)
			case meta.ChargeTypeUsageBased:
				deletedUsageBased, err := deletedCharge.AsUsageBasedCharge()
				require.NoError(s.T(), err)
				require.Equal(s.T(), usagebased.StatusDeleted, deletedUsageBased.Status)
				require.Nil(s.T(), deletedUsageBased.Intent.GetBaseIntent().IntentDeletedAt)
				override := deletedUsageBased.Intent.GetOverrideLayerMutableFields()
				require.NotNil(s.T(), override)
				require.NotNil(s.T(), override.IntentDeletedAt)
			default:
				s.T().Fatalf("unexpected deleted charge type: %s", deletedCharge.Type())
			}
		}
	})
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
}
