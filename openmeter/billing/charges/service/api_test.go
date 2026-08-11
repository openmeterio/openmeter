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
