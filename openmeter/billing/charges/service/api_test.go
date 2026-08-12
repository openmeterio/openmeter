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
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestResolveDeletePolicy(t *testing.T) {
	t.Run("none ignores credit and invoice refunds", func(t *testing.T) {
		policy, err := resolveDeletePolicy(charges.PaymentAdjustmentNone)

		require.NoError(t, err)
		require.Equal(t, meta.CreditRefundPolicyIgnore, policy.CreditRefundPolicy)
		require.Equal(t, meta.InvoiceRefundPolicyIgnore, policy.InvoiceRefundPolicy)
	})

	t.Run("unsupported adjustment is rejected", func(t *testing.T) {
		_, err := resolveDeletePolicy(charges.PaymentAdjustment("unsupported"))

		require.ErrorContains(t, err, "unsupported payment adjustment")
	})
}

func TestCustomerChargeAPIDelete(t *testing.T) {
	suite.Run(t, new(CustomerChargeAPIDeleteTestSuite))
}

type CustomerChargeAPIDeleteTestSuite struct {
	BaseSuite
}

func (s *CustomerChargeAPIDeleteTestSuite) TestDeleteCreatesOverrideForSupportedChargeTypes() {
	// given
	// - Future flat-fee and usage-based charges have subscription-owned base intents without API overrides.
	// when:
	// - The customer-facing facade deletes both charges with PaymentAdjustmentNone.
	// then:
	// - Each effective charge is deleted through an override while its base intent remains undeleted.
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

	s.Run("given subscription-managed flat-fee and usage-based charges", func() {
		namespace = s.GetUniqueNamespace("charges-service-api-delete")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		customer := s.CreateTestCustomer(namespace, "api-delete")
		customerID = customer.ID
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
		feature := s.SetupApiRequestsTotalFeature(ctx, namespace)

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
					featureKey:        feature.Feature.Key,
					name:              "usage-based",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-delete-usage-based",
				}),
			},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 2)

		for _, charge := range created {
			chargeID, err := charge.GetChargeID()
			require.NoError(s.T(), err)
			chargeIDs = append(chargeIDs, chargeID)
		}
	})

	s.Run("when deleting both charges without a payment adjustment", func() {
		for _, chargeID := range chargeIDs {
			require.NoError(s.T(), s.Charges.DeleteCustomerCharge(ctx, charges.DeleteCustomerChargeInput{
				Namespace:         namespace,
				CustomerID:        customerID,
				ChargeID:          chargeID.ID,
				PaymentAdjustment: charges.PaymentAdjustmentNone,
			}))
		}
	})

	s.Run("then the effective intents are deleted overrides", func() {
		for _, chargeID := range chargeIDs {
			deletedCharge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{ChargeID: chargeID})
			require.NoError(s.T(), err)

			switch deletedCharge.Type() {
			case meta.ChargeTypeFlatFee:
				charge, err := deletedCharge.AsFlatFeeCharge()
				require.NoError(s.T(), err)
				require.Equal(s.T(), flatfee.StatusDeleted, charge.Status)
				require.Nil(s.T(), charge.Intent.GetBaseIntent().IntentDeletedAt)
				require.NotNil(s.T(), charge.Intent.GetOverrideLayerMutableFields())
				require.NotNil(s.T(), charge.Intent.GetOverrideLayerMutableFields().IntentDeletedAt)
			case meta.ChargeTypeUsageBased:
				charge, err := deletedCharge.AsUsageBasedCharge()
				require.NoError(s.T(), err)
				require.Equal(s.T(), usagebased.StatusDeleted, charge.Status)
				require.Nil(s.T(), charge.Intent.GetBaseIntent().IntentDeletedAt)
				require.NotNil(s.T(), charge.Intent.GetOverrideLayerMutableFields())
				require.NotNil(s.T(), charge.Intent.GetOverrideLayerMutableFields().IntentDeletedAt)
			default:
				s.T().Fatalf("unexpected deleted charge type: %s", deletedCharge.Type())
			}
		}
	})
}

func (s *CustomerChargeAPIDeleteTestSuite) TestDeleteEnforcesCustomerOwnership() {
	// given
	// - A future subscription-managed flat-fee charge and another customer exist in the same namespace.
	// when:
	// - The non-owner customer requests deletion through the facade.
	// then:
	// - The request is rejected and the owner's charge remains unchanged without an override.
	ctx := s.T().Context()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	var namespace string
	var ownerID string
	var otherCustomerID string
	var chargeID meta.ChargeID

	s.Run("given a charge owned by another customer", func() {
		namespace = s.GetUniqueNamespace("charges-service-api-delete-owner")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		owner := s.CreateTestCustomer(namespace, "owner")
		ownerID = owner.ID
		otherCustomerID = s.CreateTestCustomer(namespace, "other").ID

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: namespace,
			Intents: charges.ChargeIntents{s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       owner.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(10),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				name:              "flat-fee",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "api-delete-owner-flat-fee",
			})},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 1)
		chargeID = lo.Must(created[0].GetChargeID())
	})

	s.Run("when another customer deletes the charge", func() {
		err := s.Charges.DeleteCustomerCharge(ctx, charges.DeleteCustomerChargeInput{
			Namespace:         namespace,
			CustomerID:        otherCustomerID,
			ChargeID:          chargeID.ID,
			PaymentAdjustment: charges.PaymentAdjustmentNone,
		})

		require.ErrorContains(s.T(), err, "is not owned by customer")
	})

	s.Run("then the owner charge remains unchanged", func() {
		charge := s.mustGetChargeByID(chargeID)
		flatFee, err := charge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), ownerID, flatFee.Intent.GetCustomerID())
		require.Equal(s.T(), flatfee.StatusCreated, flatFee.Status)
		require.Nil(s.T(), flatFee.Intent.GetOverrideLayerMutableFields())
	})
}

func (s *CustomerChargeAPIDeleteTestSuite) TestDeleteRejectsCreditPurchase() {
	// given
	// - A realized promotional credit purchase exists for the customer.
	// when:
	// - The customer requests its deletion through the charge facade.
	// then:
	// - The unsupported charge type is rejected and its lifecycle and realization remain unchanged.
	ctx := s.T().Context()
	clock.FreezeTime(datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime())
	defer clock.UnFreeze()

	var namespace string
	var customerID string
	var chargeID meta.ChargeID
	var status creditpurchase.Status

	s.Run("given a realized credit-purchase charge", func() {
		namespace = s.GetUniqueNamespace("charges-service-api-delete-credit-purchase")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		customer := s.CreateTestCustomer(namespace, "credit-purchase")
		customerID = customer.ID
		s.CreditPurchaseTestHandler.onPromotionalCreditPurchase = func(_ context.Context, _ creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
			return ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()}, nil
		}

		created := s.grantPromotionalCredits(ctx, customer.GetID(), 10)
		chargeID = lo.Must(created[0].GetChargeID())
		charge, err := created[0].AsCreditPurchaseCharge()
		require.NoError(s.T(), err)
		status = charge.Status
	})

	s.Run("when deleting it through the customer charge facade", func() {
		err := s.Charges.DeleteCustomerCharge(ctx, charges.DeleteCustomerChargeInput{
			Namespace:         namespace,
			CustomerID:        customerID,
			ChargeID:          chargeID.ID,
			PaymentAdjustment: charges.PaymentAdjustmentNone,
		})

		require.ErrorContains(s.T(), err, "unsupported charge type")
	})

	s.Run("then the credit purchase remains unchanged", func() {
		charge := s.mustGetChargeByID(chargeID)
		creditPurchase, err := charge.AsCreditPurchaseCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), status, creditPurchase.Status)
		require.NotNil(s.T(), creditPurchase.Realizations.CreditGrantRealization)
	})
}

func (s *CustomerChargeAPIDeleteTestSuite) TestDeleteWithNoPaymentAdjustmentPreservesFlatFeeCreditRealization() {
	// given
	// - A subscription-managed credit-only flat fee has an allocation and no correction callback.
	// when:
	// - The customer deletes it without compensating realized credits.
	// then:
	// - Deletion succeeds and the existing allocation remains uncorrected on the charge.
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
	var realizationID string

	s.Run("given a realized credit-only flat-fee charge", func() {
		namespace = s.GetUniqueNamespace("charges-service-api-delete-flat-fee-credit")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		customer := s.CreateTestCustomer(namespace, "flat-fee-credit")
		customerID = customer.ID

		s.FlatFeeTestHandler.onAllocateCredits = func(_ context.Context, input flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			return creditrealization.CreateAllocationInputs{{
				ServicePeriod: input.Charge.Intent.GetEffectiveServicePeriod(),
				Amount:        input.PreTaxAmountToAllocate,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
			}}, nil
		}

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: namespace,
			Intents: charges.ChargeIntents{s.createMockChargeIntent(createMockChargeIntentInput{
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
				uniqueReferenceID: "api-delete-flat-fee-credit",
			})},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 1)
		chargeID = lo.Must(created[0].GetChargeID())

		charge, err := created[0].AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.NotNil(s.T(), charge.Realizations.CurrentRun)
		require.Len(s.T(), charge.Realizations.CurrentRun.CreditRealizations, 1)
		realizationID = charge.Realizations.CurrentRun.CreditRealizations[0].ID
	})

	s.Run("when deleting with no payment adjustment", func() {
		require.NoError(s.T(), s.Charges.DeleteCustomerCharge(ctx, charges.DeleteCustomerChargeInput{
			Namespace:         namespace,
			CustomerID:        customerID,
			ChargeID:          chargeID.ID,
			PaymentAdjustment: charges.PaymentAdjustmentNone,
		}))
	})

	s.Run("then the allocation remains uncorrected", func() {
		charge := s.mustGetChargeByID(chargeID)
		flatFee, err := charge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), flatfee.StatusDeleted, flatFee.Status)
		require.NotNil(s.T(), flatFee.Realizations.CurrentRun)
		require.Len(s.T(), flatFee.Realizations.CurrentRun.CreditRealizations, 1)
		require.Equal(s.T(), realizationID, flatFee.Realizations.CurrentRun.CreditRealizations[0].ID)
		require.Equal(s.T(), creditrealization.TypeAllocation, flatFee.Realizations.CurrentRun.CreditRealizations[0].Type)
	})
}

func (s *CustomerChargeAPIDeleteTestSuite) TestDeleteWithNoPaymentAdjustmentPreservesUsageBasedCreditRealization() {
	// given
	// - A subscription-managed credit-only usage charge has an allocation and no correction callback.
	// when:
	// - The customer deletes it without compensating realized credits.
	// then:
	// - Deletion succeeds and the existing allocation remains uncorrected on the charge realization.
	ctx := s.T().Context()
	createAt := datetime.MustParseTimeInLocation(s.T(), "2026-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	var namespace string
	var customerID string
	var chargeID meta.ChargeID
	var realizationID string

	s.Run("given a realized credit-only usage-based charge", func() {
		namespace = s.GetUniqueNamespace("charges-service-api-delete-usage-credit")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		customer := s.CreateTestCustomer(namespace, "usage-credit")
		customerID = customer.ID
		feature := s.SetupApiRequestsTotalFeature(ctx, namespace)
		customInvoicing := s.SetupCustomInvoicing(namespace)
		_ = s.ProvisionBillingProfile(
			ctx,
			namespace,
			customInvoicing.App.GetID(),
			billingtest.WithProgressiveBilling(),
			billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
			billingtest.WithManualApproval(),
		)

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: namespace,
			Intents: charges.ChargeIntents{s.createMockChargeIntent(createMockChargeIntentInput{
				customer:          customer.GetID(),
				currency:          USD,
				servicePeriod:     servicePeriod,
				settlementMode:    productcatalog.CreditOnlySettlementMode,
				price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				featureKey:        feature.Feature.Key,
				name:              "usage-based",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "api-delete-usage-credit",
			})},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 1)
		chargeID = lo.Must(created[0].GetChargeID())

		clock.FreezeTime(servicePeriod.From)
		advanced, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: customer.GetID()})
		require.NoError(s.T(), err)
		require.Len(s.T(), advanced, 1)

		s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued = func(_ context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
			return creditrealization.CreateAllocationInputs{{
				ServicePeriod: input.Charge.Intent.GetEffectiveServicePeriod(),
				Amount:        input.AmountToAllocate,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
			}}, nil
		}
		s.MockStreamingConnector.AddSimpleEvent(
			feature.Feature.Key,
			10,
			datetime.MustParseTimeInLocation(s.T(), "2027-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		clock.FreezeTime(servicePeriod.To.Add(12 * time.Hour))
		advanced, err = s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: customer.GetID()})
		require.NoError(s.T(), err)
		require.Len(s.T(), advanced, 1)

		charge := s.mustGetChargeByID(chargeID)
		usageBased, err := charge.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		require.Len(s.T(), usageBased.Realizations, 1)
		require.Len(s.T(), usageBased.Realizations[0].CreditsAllocated, 1)
		realizationID = usageBased.Realizations[0].CreditsAllocated[0].ID
	})

	s.Run("when deleting with no payment adjustment", func() {
		require.NoError(s.T(), s.Charges.DeleteCustomerCharge(ctx, charges.DeleteCustomerChargeInput{
			Namespace:         namespace,
			CustomerID:        customerID,
			ChargeID:          chargeID.ID,
			PaymentAdjustment: charges.PaymentAdjustmentNone,
		}))
	})

	s.Run("then the allocation remains uncorrected", func() {
		charge := s.mustGetChargeByID(chargeID)
		usageBased, err := charge.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), usagebased.StatusDeleted, usageBased.Status)
		require.Len(s.T(), usageBased.Realizations, 1)
		require.Len(s.T(), usageBased.Realizations[0].CreditsAllocated, 1)
		require.Equal(s.T(), realizationID, usageBased.Realizations[0].CreditsAllocated[0].ID)
		require.Equal(s.T(), creditrealization.TypeAllocation, usageBased.Realizations[0].CreditsAllocated[0].Type)
	})
}

func (s *CustomerChargeAPIDeleteTestSuite) TestDeleteWithNoPaymentAdjustmentRemovesGatheringLines() {
	// given
	// - Active invoice-backed flat-fee and usage-based charges have mutable gathering lines.
	// when:
	// - The customer deletes both charges without compensating economic effects.
	// then:
	// - Normal invoice lifecycle cleanup removes both gathering lines despite PaymentAdjustmentNone.
	ctx := s.T().Context()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	var namespace string
	var customerID string
	var chargeIDs []meta.ChargeID

	s.Run("given active invoice-backed flat-fee and usage-based charges", func() {
		namespace = s.GetUniqueNamespace("charges-service-api-delete-gathering")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		customer := s.CreateTestCustomer(namespace, "gathering")
		customerID = customer.ID
		feature := s.SetupApiRequestsTotalFeature(ctx, namespace)
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), billingtest.WithManualApproval())
		s.FlatFeeTestHandler.onAllocateCredits = func(context.Context, flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			return nil, nil
		}

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
					uniqueReferenceID: "api-delete-gathering-flat-fee",
				}),
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          customer.GetID(),
					currency:          USD,
					servicePeriod:     servicePeriod,
					settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
					price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
					featureKey:        feature.Feature.Key,
					name:              "usage-based",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-delete-gathering-usage-based",
				}),
			},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 2)

		for _, charge := range created {
			chargeID := lo.Must(charge.GetChargeID())
			chargeIDs = append(chargeIDs, chargeID)
			require.Len(s.T(), activeGatheringLinesForCharge(&s.BaseSuite, namespace, customerID, chargeID.ID), 1)
		}
	})

	s.Run("when deleting both charges with no payment adjustment", func() {
		for _, chargeID := range chargeIDs {
			require.NoError(s.T(), s.Charges.DeleteCustomerCharge(ctx, charges.DeleteCustomerChargeInput{
				Namespace:         namespace,
				CustomerID:        customerID,
				ChargeID:          chargeID.ID,
				PaymentAdjustment: charges.PaymentAdjustmentNone,
			}))
		}
	})

	s.Run("then normal invoice lifecycle cleanup removes both gathering lines", func() {
		for _, chargeID := range chargeIDs {
			require.Empty(s.T(), activeGatheringLinesForCharge(&s.BaseSuite, namespace, customerID, chargeID.ID))
		}
	})
}

func (s *CustomerChargeAPIDeleteTestSuite) TestDeleteWithNoPaymentAdjustmentPreservesPaidInvoice() {
	// given
	// - A subscription-managed flat fee has immutable paid invoice and payment history.
	// when:
	// - The customer deletes it without compensating the settled payment.
	// then:
	// - The override deletes the charge, preserves paid history, and records unsupported immutable drift.
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
		namespace = s.GetUniqueNamespace("charges-service-api-delete-paid")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		customer := s.CreateTestCustomer(namespace, "paid")
		customerID = customer.ID
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), billingtest.WithManualApproval())

		s.FlatFeeTestHandler.onAllocateCredits = func(context.Context, flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			return nil, nil
		}
		s.FlatFeeTestHandler.onInvoiceUsageAccrued = newCountedLedgerTransactionCallback[flatfee.OnInvoiceUsageAccruedInput]().Handler(s.T())
		authorizedCallback := newCountedLedgerTransactionCallback[flatfee.OnPaymentAuthorizedInput]()
		paymentAuthorizedTransactionID = authorizedCallback.id
		s.FlatFeeTestHandler.onPaymentAuthorized = authorizedCallback.Handler(s.T())
		settledCallback := newCountedLedgerTransactionCallback[flatfee.OnPaymentSettledInput]()
		paymentSettledTransactionID = settledCallback.id
		s.FlatFeeTestHandler.onPaymentSettled = settledCallback.Handler(s.T())

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: namespace,
			Intents: charges.ChargeIntents{s.createMockChargeIntent(createMockChargeIntentInput{
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
				uniqueReferenceID: "api-delete-paid-flat-fee",
			})},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 1)
		chargeID = lo.Must(created[0].GetChargeID())

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

	s.Run("when deleting with no payment adjustment", func() {
		require.NoError(s.T(), s.Charges.DeleteCustomerCharge(ctx, charges.DeleteCustomerChargeInput{
			Namespace:         namespace,
			CustomerID:        customerID,
			ChargeID:          chargeID.ID,
			PaymentAdjustment: charges.PaymentAdjustmentNone,
		}))
	})

	s.Run("then paid billing history is retained", func() {
		charge := s.mustGetChargeByID(chargeID)
		flatFee, err := charge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), flatfee.StatusDeleted, flatFee.Status)
		require.Nil(s.T(), flatFee.Intent.GetBaseIntent().IntentDeletedAt)
		require.NotNil(s.T(), flatFee.Intent.GetOverrideLayerMutableFields())
		require.NotNil(s.T(), flatFee.Intent.GetOverrideLayerMutableFields().IntentDeletedAt)
		require.Nil(s.T(), flatFee.Realizations.CurrentRun)
		require.Len(s.T(), flatFee.Realizations.PriorRuns, 1)

		payment := flatFee.Realizations.PriorRuns[0].Payment
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

func TestCustomerChargeAPISetOverride(t *testing.T) {
	suite.Run(t, new(CustomerChargeAPISetOverrideTestSuite))
}

type CustomerChargeAPISetOverrideTestSuite struct {
	BaseSuite
}

func (s *CustomerChargeAPISetOverrideTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *CustomerChargeAPISetOverrideTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *CustomerChargeAPISetOverrideTestSuite) TestSetCreatesAndReplacesOverrideForSupportedChargeTypes() {
	// given
	// - Future subscription-managed flat-fee and usage-based charges expose their base mutable snapshots.
	// when:
	// - The customer sets and then replaces a complete override snapshot for each charge.
	// then:
	// - The latest override is effective while each immutable subscription-owned base remains unchanged.
	ctx := s.T().Context()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	var namespace string
	var customerID string
	var flatFeeChargeID meta.ChargeID
	var usageBasedChargeID meta.ChargeID
	var flatFeeBase flatfee.IntentMutableFields
	var usageBasedBase usagebased.IntentMutableFields
	var flatFeeOverride flatfee.IntentMutableFields
	var usageBasedOverride usagebased.IntentMutableFields

	s.Run("given subscription-managed supported charges", func() {
		namespace = s.GetUniqueNamespace("charges-service-api-set")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		customer := s.CreateTestCustomer(namespace, "api-set")
		customerID = customer.ID
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
		feature := s.SetupApiRequestsTotalFeature(ctx, namespace)

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
					uniqueReferenceID: "api-set-flat-fee",
				}),
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          customer.GetID(),
					currency:          USD,
					servicePeriod:     servicePeriod,
					settlementMode:    productcatalog.CreditOnlySettlementMode,
					price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
					featureKey:        feature.Feature.Key,
					name:              "usage-based-base",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-set-usage-based",
				}),
			},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 2)

		for _, charge := range created {
			switch charge.Type() {
			case meta.ChargeTypeFlatFee:
				flatFee, err := charge.AsFlatFeeCharge()
				require.NoError(s.T(), err)
				flatFeeChargeID = flatFee.GetChargeID()
				flatFeeBase = flatFee.Intent.GetBaseIntent().IntentMutableFields.Clone()
			case meta.ChargeTypeUsageBased:
				usageBased, err := charge.AsUsageBasedCharge()
				require.NoError(s.T(), err)
				usageBasedChargeID = usageBased.GetChargeID()
				usageBasedBase = usageBased.Intent.GetBaseIntent().IntentMutableFields.Clone()
			}
		}
	})

	s.Run("when setting complete override snapshots", func() {
		flatFeeOverride = flatFeeBase.Clone()
		flatFeeOverride.Name = "flat-fee-override"
		flatFeeOverride.AmountBeforeProration = alpacadecimal.NewFromInt(20)
		flatFeeOverride.InvoiceAt = servicePeriod.From.Add(24 * time.Hour)

		returned, err := s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   flatFeeChargeID.ID,
			FlatFee:    &flatFeeOverride,
		})
		require.NoError(s.T(), err)
		returnedFlatFee, err := returned.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), flatFeeOverride, returnedFlatFee.Intent.GetEffectiveIntent().IntentMutableFields)

		usageBasedOverride = usageBasedBase.Clone()
		usageBasedOverride.Name = "usage-based-override"
		usageBasedOverride.Price = *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(2)})
		usageBasedOverride.InvoiceAt = servicePeriod.To.Add(24 * time.Hour)

		returned, err = s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   usageBasedChargeID.ID,
			UsageBased: &usageBasedOverride,
		})
		require.NoError(s.T(), err)
		returnedUsageBased, err := returned.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), usageBasedOverride, returnedUsageBased.Intent.GetEffectiveIntent().IntentMutableFields)
	})

	s.Run("when replacing the existing override snapshots", func() {
		flatFeeOverride.Name = "flat-fee-override-replaced"
		flatFeeOverride.AmountBeforeProration = alpacadecimal.NewFromInt(30)
		_, err := s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   flatFeeChargeID.ID,
			FlatFee:    &flatFeeOverride,
		})
		require.NoError(s.T(), err)

		usageBasedOverride.Name = "usage-based-override-replaced"
		usageBasedOverride.Price = *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(3)})
		_, err = s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   usageBasedChargeID.ID,
			UsageBased: &usageBasedOverride,
		})
		require.NoError(s.T(), err)
	})

	s.Run("then only the latest overrides are effective", func() {
		flatFeeCharge := s.mustGetChargeByID(flatFeeChargeID)
		flatFee, err := flatFeeCharge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), flatFeeBase, flatFee.Intent.GetBaseIntent().IntentMutableFields)
		require.Equal(s.T(), flatFeeOverride, *flatFee.Intent.GetOverrideLayerMutableFields())
		require.Equal(s.T(), flatFeeOverride, flatFee.Intent.GetEffectiveIntent().IntentMutableFields)

		usageBasedCharge := s.mustGetChargeByID(usageBasedChargeID)
		usageBased, err := usageBasedCharge.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), usageBasedBase, usageBased.Intent.GetBaseIntent().IntentMutableFields)
		require.Equal(s.T(), usageBasedOverride, *usageBased.Intent.GetOverrideLayerMutableFields())
		require.Equal(s.T(), usageBasedOverride, usageBased.Intent.GetEffectiveIntent().IntentMutableFields)
	})
}

func (s *CustomerChargeAPISetOverrideTestSuite) TestSetDeleteOrdering() {
	// given
	// - Subscription-managed flat-fee and usage-based charges have customer overrides.
	// when:
	// - The customer deletes the charges and then attempts to set another override.
	// then:
	// - Delete marks the existing override as deleted, and Set cannot implicitly restore it.
	ctx := s.T().Context()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	var namespace string
	var customerID string
	var flatFeeChargeID meta.ChargeID
	var usageBasedChargeID meta.ChargeID
	var flatFeeBase flatfee.IntentMutableFields
	var usageBasedBase usagebased.IntentMutableFields
	var flatFeeOverride flatfee.IntentMutableFields
	var usageBasedOverride usagebased.IntentMutableFields
	var flatFeeDeletedAt time.Time
	var usageBasedDeletedAt time.Time

	s.Run("given representative supported charges", func() {
		namespace = s.GetUniqueNamespace("charges-service-api-set-delete-ordering")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		customer := s.CreateTestCustomer(namespace, "api-set-delete-ordering")
		customerID = customer.ID
		feature := s.SetupApiRequestsTotalFeature(ctx, namespace)
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), billingtest.WithManualApproval())
		s.FlatFeeTestHandler.onAllocateCredits = func(context.Context, flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			return nil, nil
		}

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
					name:              "flat-fee-base",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-set-delete-ordering-flat-fee",
				}),
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          customer.GetID(),
					currency:          USD,
					servicePeriod:     servicePeriod,
					settlementMode:    productcatalog.CreditOnlySettlementMode,
					price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
					featureKey:        feature.Feature.Key,
					name:              "usage-based-base",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-set-delete-ordering-usage-based",
				}),
			},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 2)

		for _, charge := range created {
			switch charge.Type() {
			case meta.ChargeTypeFlatFee:
				flatFee, err := charge.AsFlatFeeCharge()
				require.NoError(s.T(), err)
				flatFeeChargeID = flatFee.GetChargeID()
				flatFeeBase = flatFee.Intent.GetBaseIntent().IntentMutableFields.Clone()
			case meta.ChargeTypeUsageBased:
				usageBased, err := charge.AsUsageBasedCharge()
				require.NoError(s.T(), err)
				usageBasedChargeID = usageBased.GetChargeID()
				usageBasedBase = usageBased.Intent.GetBaseIntent().IntentMutableFields.Clone()
			default:
				s.T().Fatalf("unexpected charge type: %s", charge.Type())
			}
		}
	})

	s.Run("when setting customer overrides", func() {
		flatFeeOverride = flatFeeBase.Clone()
		flatFeeOverride.Name = "flat-fee-override"
		flatFeeOverride.AmountBeforeProration = alpacadecimal.NewFromInt(20)
		_, err := s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   flatFeeChargeID.ID,
			FlatFee:    &flatFeeOverride,
		})
		require.NoError(s.T(), err)

		usageBasedOverride = usageBasedBase.Clone()
		usageBasedOverride.Name = "usage-based-override"
		usageBasedOverride.Price = *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(2)})
		_, err = s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   usageBasedChargeID.ID,
			UsageBased: &usageBasedOverride,
		})
		require.NoError(s.T(), err)
	})

	s.Run("when deleting charges with existing overrides", func() {
		for _, chargeID := range []meta.ChargeID{flatFeeChargeID, usageBasedChargeID} {
			require.NoError(s.T(), s.Charges.DeleteCustomerCharge(ctx, charges.DeleteCustomerChargeInput{
				Namespace:         namespace,
				CustomerID:        customerID,
				ChargeID:          chargeID.ID,
				PaymentAdjustment: charges.PaymentAdjustmentNone,
			}))
		}
	})

	s.Run("then deletion reuses the existing overrides", func() {
		flatFeeResult := s.mustGetChargeByID(flatFeeChargeID)
		flatFee, err := flatFeeResult.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), flatfee.StatusDeleted, flatFee.Status)
		require.Equal(s.T(), flatFeeBase, flatFee.Intent.GetBaseIntent().IntentMutableFields)
		actualFlatFeeOverride := flatFee.Intent.GetOverrideLayerMutableFields().Clone()
		require.NotNil(s.T(), actualFlatFeeOverride.IntentDeletedAt)
		flatFeeDeletedAt = *actualFlatFeeOverride.IntentDeletedAt
		actualFlatFeeOverride.IntentDeletedAt = nil
		require.Equal(s.T(), flatFeeOverride, actualFlatFeeOverride)

		usageBasedResult := s.mustGetChargeByID(usageBasedChargeID)
		usageBased, err := usageBasedResult.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), usagebased.StatusDeleted, usageBased.Status)
		require.Equal(s.T(), usageBasedBase, usageBased.Intent.GetBaseIntent().IntentMutableFields)
		actualUsageBasedOverride := usageBased.Intent.GetOverrideLayerMutableFields().Clone()
		require.NotNil(s.T(), actualUsageBasedOverride.IntentDeletedAt)
		usageBasedDeletedAt = *actualUsageBasedOverride.IntentDeletedAt
		actualUsageBasedOverride.IntentDeletedAt = nil
		require.Equal(s.T(), usageBasedOverride, actualUsageBasedOverride)
	})

	s.Run("when setting another override on deleted charges", func() {
		flatFeeRejectedOverride := flatFeeOverride.Clone()
		flatFeeRejectedOverride.Name = "flat-fee-should-not-apply"
		_, err := s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   flatFeeChargeID.ID,
			FlatFee:    &flatFeeRejectedOverride,
		})
		require.ErrorContains(s.T(), err, "cannot set override for flat-fee charge")

		usageBasedRejectedOverride := usageBasedOverride.Clone()
		usageBasedRejectedOverride.Name = "usage-based-should-not-apply"
		_, err = s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   usageBasedChargeID.ID,
			UsageBased: &usageBasedRejectedOverride,
		})
		require.ErrorContains(s.T(), err, "cannot set override for usage-based charge")
	})

	s.Run("then the deleted overrides remain unchanged", func() {
		flatFeeResult := s.mustGetChargeByID(flatFeeChargeID)
		flatFee, err := flatFeeResult.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), flatfee.StatusDeleted, flatFee.Status)
		require.Equal(s.T(), flatFeeBase, flatFee.Intent.GetBaseIntent().IntentMutableFields)
		actualFlatFeeOverride := flatFee.Intent.GetOverrideLayerMutableFields().Clone()
		require.NotNil(s.T(), actualFlatFeeOverride.IntentDeletedAt)
		require.True(s.T(), flatFeeDeletedAt.Equal(*actualFlatFeeOverride.IntentDeletedAt))
		actualFlatFeeOverride.IntentDeletedAt = nil
		require.Equal(s.T(), flatFeeOverride, actualFlatFeeOverride)

		usageBasedResult := s.mustGetChargeByID(usageBasedChargeID)
		usageBased, err := usageBasedResult.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), usagebased.StatusDeleted, usageBased.Status)
		require.Equal(s.T(), usageBasedBase, usageBased.Intent.GetBaseIntent().IntentMutableFields)
		actualUsageBasedOverride := usageBased.Intent.GetOverrideLayerMutableFields().Clone()
		require.NotNil(s.T(), actualUsageBasedOverride.IntentDeletedAt)
		require.True(s.T(), usageBasedDeletedAt.Equal(*actualUsageBasedOverride.IntentDeletedAt))
		actualUsageBasedOverride.IntentDeletedAt = nil
		require.Equal(s.T(), usageBasedOverride, actualUsageBasedOverride)
	})
}

func (s *CustomerChargeAPISetOverrideTestSuite) TestSetValidatesOwnershipPayloadAndChargeType() {
	// given
	// - Supported customer charges and an out-of-scope credit purchase exist without overrides.
	// when:
	// - A non-owner, mismatched payload, or credit-purchase override request is submitted.
	// then:
	// - The facade rejects each request before mutating any charge intent.
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
	var flatFeeChargeID meta.ChargeID
	var usageBasedFields usagebased.IntentMutableFields
	var creditPurchaseChargeID meta.ChargeID

	s.Run("given supported and unsupported customer charges", func() {
		namespace = s.GetUniqueNamespace("charges-service-api-set-validation")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		customer := s.CreateTestCustomer(namespace, "api-set-validation")
		customerID = customer.ID
		otherCustomerID = s.CreateTestCustomer(namespace, "api-set-validation-other").ID
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
		feature := s.SetupApiRequestsTotalFeature(ctx, namespace)

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
					uniqueReferenceID: "api-set-validation-flat-fee",
				}),
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          customer.GetID(),
					currency:          USD,
					servicePeriod:     servicePeriod,
					settlementMode:    productcatalog.CreditOnlySettlementMode,
					price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
					featureKey:        feature.Feature.Key,
					name:              "usage-based",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-set-validation-usage-based",
				}),
			},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 2)

		for _, charge := range created {
			switch charge.Type() {
			case meta.ChargeTypeFlatFee:
				flatFeeChargeID = lo.Must(charge.GetChargeID())
			case meta.ChargeTypeUsageBased:
				usageBased, err := charge.AsUsageBasedCharge()
				require.NoError(s.T(), err)
				usageBasedFields = usageBased.Intent.GetEffectiveIntent().IntentMutableFields.Clone()
			}
		}

		s.CreditPurchaseTestHandler.onPromotionalCreditPurchase = func(_ context.Context, _ creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
			return ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()}, nil
		}
		creditPurchaseChargeID = lo.Must(s.grantPromotionalCredits(ctx, customer.GetID(), 10)[0].GetChargeID())
	})

	s.Run("when a different customer sets an override", func() {
		flatFeeCharge := s.mustGetChargeByID(flatFeeChargeID)
		flatFee, err := flatFeeCharge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		fields := flatFee.Intent.GetEffectiveIntent().IntentMutableFields.Clone()
		fields.Name = "should-not-apply"

		_, err = s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: otherCustomerID,
			ChargeID:   flatFeeChargeID.ID,
			FlatFee:    &fields,
		})
		require.True(s.T(), models.IsGenericNotFoundError(err))
		require.ErrorContains(s.T(), err, "charge not found")
	})

	s.Run("when the payload type does not match the charge", func() {
		_, err := s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   flatFeeChargeID.ID,
			UsageBased: &usageBasedFields,
		})
		require.ErrorContains(s.T(), err, "flat fee override fields are required")
	})

	s.Run("when setting an override on a credit purchase", func() {
		flatFeeCharge := s.mustGetChargeByID(flatFeeChargeID)
		flatFee, err := flatFeeCharge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		fields := flatFee.Intent.GetEffectiveIntent().IntentMutableFields.Clone()

		_, err = s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   creditPurchaseChargeID.ID,
			FlatFee:    &fields,
		})
		require.ErrorContains(s.T(), err, "credit purchase charges is not supported")
	})

	s.Run("then no rejected request created an override", func() {
		flatFeeCharge := s.mustGetChargeByID(flatFeeChargeID)
		flatFee, err := flatFeeCharge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.Nil(s.T(), flatFee.Intent.GetOverrideLayerMutableFields())

		creditPurchaseCharge := s.mustGetChargeByID(creditPurchaseChargeID)
		creditPurchase, err := creditPurchaseCharge.AsCreditPurchaseCharge()
		require.NoError(s.T(), err)
		require.NotNil(s.T(), creditPurchase.Realizations.CreditGrantRealization)
	})
}

func (s *CustomerChargeAPISetOverrideTestSuite) TestSetReconcilesRealizedFlatFeeCredits() {
	// given
	// - A subscription-managed credit-only flat fee has allocated its original amount.
	// when:
	// - The customer sets an override with a larger amount.
	// then:
	// - The base remains unchanged and the current run receives only the additional allocation.
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
	var allocationAmounts []float64

	s.Run("given a realized credit-only flat-fee charge", func() {
		namespace = s.GetUniqueNamespace("charges-service-api-set-flat-fee-credit")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		customer := s.CreateTestCustomer(namespace, "api-set-flat-fee-credit")
		customerID = customer.ID

		s.FlatFeeTestHandler.onAllocateCredits = func(_ context.Context, input flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			allocationAmounts = append(allocationAmounts, input.PreTaxAmountToAllocate.InexactFloat64())
			return creditrealization.CreateAllocationInputs{{
				ServicePeriod: input.ServicePeriod,
				Amount:        input.PreTaxAmountToAllocate,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
			}}, nil
		}

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: namespace,
			Intents: charges.ChargeIntents{s.createMockChargeIntent(createMockChargeIntentInput{
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
				uniqueReferenceID: "api-set-flat-fee-credit",
			})},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 1)
		chargeID = lo.Must(created[0].GetChargeID())
		require.Equal(s.T(), []float64{10}, allocationAmounts)
	})

	s.Run("when increasing the effective amount through an override", func() {
		charge := s.mustGetChargeByID(chargeID)
		flatFee, err := charge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		fields := flatFee.Intent.GetEffectiveIntent().IntentMutableFields.Clone()
		fields.Name = "flat-fee-override"
		fields.AmountBeforeProration = alpacadecimal.NewFromInt(20)

		_, err = s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   chargeID.ID,
			FlatFee:    &fields,
		})
		require.NoError(s.T(), err)
	})

	s.Run("then only the additional credits are allocated", func() {
		charge := s.mustGetChargeByID(chargeID)
		flatFee, err := charge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), float64(10), flatFee.Intent.GetBaseIntent().AmountBeforeProration.InexactFloat64())
		require.Equal(s.T(), float64(20), flatFee.Intent.GetEffectiveIntent().AmountBeforeProration.InexactFloat64())
		require.Equal(s.T(), float64(20), flatFee.State.AmountAfterProration.InexactFloat64())
		require.Equal(s.T(), []float64{10, 10}, allocationAmounts)
		require.NotNil(s.T(), flatFee.Realizations.CurrentRun)
		require.Len(s.T(), flatFee.Realizations.CurrentRun.CreditRealizations, 2)
		require.Equal(s.T(), float64(20), flatFee.Realizations.CurrentRun.CreditRealizations.Sum().InexactFloat64())
	})
}

func (s *CustomerChargeAPISetOverrideTestSuite) TestSetUpdatesInvoiceBackedGatheringLines() {
	// given
	// - Active invoice-backed flat-fee and usage-based charges have mutable gathering lines.
	// when:
	// - The customer changes each charge's complete mutable snapshot before realization.
	// then:
	// - Billing keeps one gathering line per charge and updates it from the effective override.
	ctx := s.T().Context()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	var namespace string
	var customerID string
	var flatFeeChargeID meta.ChargeID
	var usageBasedChargeID meta.ChargeID

	s.Run("given active invoice-backed supported charges", func() {
		namespace = s.GetUniqueNamespace("charges-service-api-set-gathering")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		customer := s.CreateTestCustomer(namespace, "api-set-gathering")
		customerID = customer.ID
		feature := s.SetupApiRequestsTotalFeature(ctx, namespace)
		sandboxApp := s.InstallSandboxApp(s.T(), namespace)
		_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), billingtest.WithManualApproval())
		s.FlatFeeTestHandler.onAllocateCredits = func(context.Context, flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			return nil, nil
		}

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
					name:              "flat-fee-base",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-set-gathering-flat-fee",
				}),
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          customer.GetID(),
					currency:          USD,
					servicePeriod:     servicePeriod,
					settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
					price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
					featureKey:        feature.Feature.Key,
					name:              "usage-based-base",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-set-gathering-usage-based",
				}),
			},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 2)

		for _, charge := range created {
			chargeID := lo.Must(charge.GetChargeID())
			require.Len(s.T(), activeGatheringLinesForCharge(&s.BaseSuite, namespace, customerID, chargeID.ID), 1)
			switch charge.Type() {
			case meta.ChargeTypeFlatFee:
				flatFeeChargeID = chargeID
			case meta.ChargeTypeUsageBased:
				usageBasedChargeID = chargeID
			}
		}
	})

	s.Run("when setting invoice-backed overrides", func() {
		flatFeeCharge := s.mustGetChargeByID(flatFeeChargeID)
		flatFee, err := flatFeeCharge.AsFlatFeeCharge()
		require.NoError(s.T(), err)
		flatFeeFields := flatFee.Intent.GetEffectiveIntent().IntentMutableFields.Clone()
		flatFeeFields.Name = "flat-fee-override"
		flatFeeFields.AmountBeforeProration = alpacadecimal.NewFromInt(20)
		_, err = s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   flatFeeChargeID.ID,
			FlatFee:    &flatFeeFields,
		})
		require.NoError(s.T(), err)

		usageBasedCharge := s.mustGetChargeByID(usageBasedChargeID)
		usageBased, err := usageBasedCharge.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		usageBasedFields := usageBased.Intent.GetEffectiveIntent().IntentMutableFields.Clone()
		usageBasedFields.Name = "usage-based-override"
		usageBasedFields.Price = *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(2)})
		_, err = s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   usageBasedChargeID.ID,
			UsageBased: &usageBasedFields,
		})
		require.NoError(s.T(), err)
	})

	s.Run("then gathering lines reflect the effective overrides", func() {
		flatFeeLines := activeGatheringLinesForCharge(&s.BaseSuite, namespace, customerID, flatFeeChargeID.ID)
		require.Len(s.T(), flatFeeLines, 1)
		require.Equal(s.T(), "flat-fee-override", flatFeeLines[0].Name)
		flatPrice, err := flatFeeLines[0].Price.AsFlat()
		require.NoError(s.T(), err)
		require.Equal(s.T(), float64(20), flatPrice.Amount.InexactFloat64())

		usageBasedLines := activeGatheringLinesForCharge(&s.BaseSuite, namespace, customerID, usageBasedChargeID.ID)
		require.Len(s.T(), usageBasedLines, 1)
		require.Equal(s.T(), "usage-based-override", usageBasedLines[0].Name)
		unitPrice, err := usageBasedLines[0].Price.AsUnit()
		require.NoError(s.T(), err)
		require.Equal(s.T(), float64(2), unitPrice.Amount.InexactFloat64())
	})
}

func (s *CustomerChargeAPISetOverrideTestSuite) TestSetVoidsUsageBasedCreditRealizationHistory() {
	// given
	// - A subscription-managed credit-only usage charge has mutable realization history.
	// when:
	// - The customer replaces its effective period and price with an override.
	// then:
	// - The old run is voided and the active charge is rescheduled from the override without changing its base.
	ctx := s.T().Context()
	createAt := datetime.MustParseTimeInLocation(s.T(), "2026-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	var namespace string
	var customerID string
	var chargeID meta.ChargeID
	var runID usagebased.RealizationRunID
	var overrideFields usagebased.IntentMutableFields
	var correctionCalls int

	s.Run("given a realized credit-only usage-based charge", func() {
		namespace = s.GetUniqueNamespace("charges-service-api-set-usage-credit")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		customer := s.CreateTestCustomer(namespace, "api-set-usage-credit")
		customerID = customer.ID
		feature := s.SetupApiRequestsTotalFeature(ctx, namespace)
		customInvoicing := s.SetupCustomInvoicing(namespace)
		_ = s.ProvisionBillingProfile(
			ctx,
			namespace,
			customInvoicing.App.GetID(),
			billingtest.WithProgressiveBilling(),
			billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
			billingtest.WithManualApproval(),
		)

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: namespace,
			Intents: charges.ChargeIntents{s.createMockChargeIntent(createMockChargeIntentInput{
				customer:          customer.GetID(),
				currency:          USD,
				servicePeriod:     servicePeriod,
				settlementMode:    productcatalog.CreditOnlySettlementMode,
				price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				featureKey:        feature.Feature.Key,
				name:              "usage-based-base",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "api-set-usage-credit",
			})},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 1)
		chargeID = lo.Must(created[0].GetChargeID())

		clock.FreezeTime(servicePeriod.From)
		advanced, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: customer.GetID()})
		require.NoError(s.T(), err)
		require.Len(s.T(), advanced, 1)

		s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued = func(_ context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
			return creditrealization.CreateAllocationInputs{{
				ServicePeriod: input.Charge.Intent.GetEffectiveServicePeriod(),
				Amount:        input.AmountToAllocate,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
			}}, nil
		}
		s.UsageBasedTestHandler.onCreditsOnlyUsageAccruedCorrection = func(_ context.Context, input usagebased.CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
			correctionCalls++
			return lo.Map(input.Corrections, func(item creditrealization.CorrectionRequestItem, _ int) creditrealization.CreateCorrectionInput {
				return creditrealization.CreateCorrectionInput{
					Amount:                item.Amount,
					CorrectsRealizationID: item.Allocation.ID,
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: ulid.Make().String(),
					},
				}
			}), nil
		}
		s.MockStreamingConnector.AddSimpleEvent(
			feature.Feature.Key,
			10,
			datetime.MustParseTimeInLocation(s.T(), "2027-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		clock.FreezeTime(servicePeriod.To.Add(12 * time.Hour))
		advanced, err = s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: customer.GetID()})
		require.NoError(s.T(), err)
		require.Len(s.T(), advanced, 1)

		charge := s.mustGetChargeByID(chargeID)
		usageBased, err := charge.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		require.Len(s.T(), usageBased.Realizations, 1)
		runID = usageBased.Realizations[0].ID
	})

	s.Run("when setting a new period and price override", func() {
		charge := s.mustGetChargeByID(chargeID)
		usageBased, err := charge.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		overrideFields = usageBased.Intent.GetEffectiveIntent().IntentMutableFields.Clone()
		overrideFields.Name = "usage-based-override"
		overrideFields.ServicePeriod.To = servicePeriod.To.AddDate(0, 1, 0)
		overrideFields.FullServicePeriod.To = overrideFields.ServicePeriod.To
		overrideFields.BillingPeriod.To = overrideFields.ServicePeriod.To
		overrideFields.InvoiceAt = overrideFields.ServicePeriod.To
		overrideFields.Price = *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(2)})

		_, err = s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   chargeID.ID,
			UsageBased: &overrideFields,
		})
		require.NoError(s.T(), err)
	})

	s.Run("then old realization history is voided", func() {
		charge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
			ChargeID: chargeID,
			Expands:  meta.Expands{meta.ExpandRealizations, meta.ExpandDeletedRealizations},
		})
		require.NoError(s.T(), err)
		usageBased, err := charge.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), usagebased.StatusActive, usageBased.Status)
		require.Nil(s.T(), usageBased.State.CurrentRealizationRunID)
		require.Equal(s.T(), servicePeriod.To, usageBased.Intent.GetBaseIntent().ServicePeriod.To)
		require.Equal(s.T(), overrideFields, *usageBased.Intent.GetOverrideLayerMutableFields())
		require.True(s.T(), overrideFields.ServicePeriod.To.Equal(*usageBased.State.AdvanceAfter))
		require.Equal(s.T(), 1, correctionCalls)

		run, err := usageBased.Realizations.GetByID(runID.ID)
		require.NoError(s.T(), err)
		require.NotNil(s.T(), run.DeletedAt)
	})
}

func (s *CustomerChargeAPISetOverrideTestSuite) TestSetRejectsUsageBasedInvoiceOverrideAfterRealizationStarts() {
	// given
	// - A credit-then-invoice usage charge has started an invoice realization without an override.
	// when:
	// - The customer attempts to replace its mutable snapshot.
	// then:
	// - The operation fails atomically because historical usage rerating is unsupported.
	ctx := s.T().Context()
	createAt := datetime.MustParseTimeInLocation(s.T(), "2026-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	var namespace string
	var customerID string
	var chargeID meta.ChargeID
	var baseFields usagebased.IntentMutableFields

	s.Run("given an invoice realization has started", func() {
		namespace = s.GetUniqueNamespace("charges-service-api-set-usage-invoice-realized")
		s.ProvisionDefaultTaxCodes(ctx, namespace)
		customer := s.CreateTestCustomer(namespace, "api-set-usage-invoice-realized")
		customerID = customer.ID
		feature := s.SetupApiRequestsTotalFeature(ctx, namespace)
		customInvoicing := s.SetupCustomInvoicing(namespace)
		_ = s.ProvisionBillingProfile(
			ctx,
			namespace,
			customInvoicing.App.GetID(),
			billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
			billingtest.WithManualApproval(),
		)

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: namespace,
			Intents: charges.ChargeIntents{s.createMockChargeIntent(createMockChargeIntentInput{
				customer:          customer.GetID(),
				currency:          USD,
				servicePeriod:     servicePeriod,
				settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
				price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				featureKey:        feature.Feature.Key,
				name:              "usage-based-base",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "api-set-usage-invoice-realized",
			})},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), created, 1)
		chargeID = lo.Must(created[0].GetChargeID())

		s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued = func(context.Context, usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
			return nil, nil
		}
		s.MockStreamingConnector.AddSimpleEvent(
			feature.Feature.Key,
			10,
			datetime.MustParseTimeInLocation(s.T(), "2027-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		clock.FreezeTime(servicePeriod.To)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customer.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), invoices, 1)

		charge := s.mustGetChargeByID(chargeID)
		usageBased, err := charge.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		require.NotNil(s.T(), usageBased.State.CurrentRealizationRunID)
		require.NotEmpty(s.T(), usageBased.Realizations)
		baseFields = usageBased.Intent.GetBaseIntent().IntentMutableFields.Clone()
	})

	s.Run("when setting an override after realization starts", func() {
		fields := baseFields.Clone()
		fields.Name = "should-not-apply"

		_, err := s.Charges.SetCustomerChargeOverride(ctx, charges.SetCustomerChargeOverrideInput{
			Namespace:  namespace,
			CustomerID: customerID,
			ChargeID:   chargeID.ID,
			UsageBased: &fields,
		})
		// TODO: enable this once we have corrections and credit notes implemented.
		require.ErrorContains(s.T(), err, "cannot set override for usage-based charge")
	})

	s.Run("then the charge remains on its base intent", func() {
		charge := s.mustGetChargeByID(chargeID)
		usageBased, err := charge.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), baseFields, usageBased.Intent.GetBaseIntent().IntentMutableFields)
		require.Nil(s.T(), usageBased.Intent.GetOverrideLayerMutableFields())
		require.NotNil(s.T(), usageBased.State.CurrentRealizationRunID)
	})
}
