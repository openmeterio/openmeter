package service

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type taxCodeDefaultingChargeType struct {
	name  string
	price *productcatalog.Price
}

func taxCodeDefaultingChargeTypes() []taxCodeDefaultingChargeType {
	return []taxCodeDefaultingChargeType{
		{
			name: "flat fee",
			price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(10),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
		},
		{
			name: "usage based",
			price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(1),
			}),
		},
	}
}

func (s *TaxCodePersistenceTestSuite) TestChargeTaxCodeDefaultPriority() {
	for _, chargeType := range taxCodeDefaultingChargeTypes() {
		s.Run(chargeType.name, func() {
			for _, testCase := range []struct {
				name               string
				withChargeTaxCode  bool
				withProfileTaxCode bool
				chargeBehavior     *productcatalog.TaxBehavior
				wantTaxCode        string
			}{
				{
					name:               "explicit charge code wins over profile and organization",
					withChargeTaxCode:  true,
					withProfileTaxCode: true,
					chargeBehavior:     lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					wantTaxCode:        "charge",
				},
				{
					name:               "profile code wins over organization",
					withProfileTaxCode: true,
					wantTaxCode:        "profile",
				},
				{
					name:        "organization code is used when profile has no code",
					wantTaxCode: "organization",
				},
				{
					name:               "behavior-only charge inherits profile code",
					withProfileTaxCode: true,
					chargeBehavior:     lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					wantTaxCode:        "profile",
				},
				{
					name:           "behavior-only charge inherits organization code",
					chargeBehavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					wantTaxCode:    "organization",
				},
			} {
				s.Run(testCase.name, func() {
					// given: distinct charge, profile, and organization tax identities
					ctx := s.T().Context()
					ns := s.GetUniqueNamespace("tax-code-default-priority")
					defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
					sandboxApp := s.InstallSandboxApp(s.T(), ns)
					profile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
					cust := s.CreateTestCustomer(ns, "test-subject")
					profileTaxCode := s.createTestTaxCodeWithStripeMapping(ctx, ns, "profile-tax", "txcd_10000000")
					chargeTaxCode := s.createTestTaxCodeWithStripeMapping(ctx, ns, "charge-tax", "txcd_20000000")
					if testCase.withProfileTaxCode {
						s.SeedProfileDefaultTaxConfigViaAdapter(ctx, profile.ProfileID(), &productcatalog.TaxConfig{
							Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
							Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
						})
					}

					var featureKey string
					if chargeType.price.Type() != productcatalog.FlatPriceType {
						usageFeature := s.SetupApiRequestsTotalFeature(ctx, ns)
						defer usageFeature.Cleanup()
						featureKey = usageFeature.Feature.Key
					}

					chargeTaxConfig := productcatalog.TaxCodeConfig{Behavior: testCase.chargeBehavior}
					if testCase.withChargeTaxCode {
						chargeTaxConfig.TaxCodeID = chargeTaxCode.ID
					}

					servicePeriod := timeutil.ClosedPeriod{
						From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
						To:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
					}
					clock.FreezeTime(servicePeriod.From.Add(-time.Hour))
					defer clock.UnFreeze()

					// when: a charge is created through the production defaulting and persistence path
					created, err := s.Charges.Create(ctx, charges.CreateInput{
						Namespace: ns,
						Intents: charges.NewCreateChargeIntents(s.createMockChargeIntent(createMockChargeIntentInput{
							customer:       cust.GetID(),
							currency:       USD,
							servicePeriod:  servicePeriod,
							settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
							price:          chargeType.price,
							featureKey:     featureKey,
							name:           "tax default priority",
							managedBy:      billing.ManuallyManagedLine,
							taxConfig:      chargeTaxConfig,
						})),
					})
					s.Require().NoError(err)
					s.Require().Len(created, 1)
					chargeID, err := created[0].GetChargeID()
					s.Require().NoError(err)

					// then: the selected identity survives reload and explicit behavior is preserved
					expectedTaxCodeIDs := map[string]string{
						"charge":       chargeTaxCode.ID,
						"profile":      profileTaxCode.ID,
						"organization": defaults.InvoicingTaxCodeID,
					}
					s.requirePersistedChargeTaxCodeConfig(chargeID, productcatalog.TaxCodeConfig{
						TaxCodeID: expectedTaxCodeIDs[testCase.wantTaxCode],
						Behavior:  testCase.chargeBehavior,
					})
				})
			}
		})
	}
}

func (s *TaxCodePersistenceTestSuite) TestChargeTaxCodeDefaultsAreCustomerScopedInBatch() {
	for _, chargeType := range taxCodeDefaultingChargeTypes() {
		s.Run(chargeType.name, func() {
			// given: two customers with different effective billing-profile tax codes
			ctx := s.T().Context()
			ns := s.GetUniqueNamespace("tax-code-default-batch")
			s.ProvisionDefaultTaxCodes(ctx, ns)
			sandboxApp := s.InstallSandboxApp(s.T(), ns)
			firstProfile := s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
			firstTaxCode := s.createTestTaxCodeWithStripeMapping(ctx, ns, "first-profile-tax", "txcd_10000000")
			secondTaxCode := s.createTestTaxCodeWithStripeMapping(ctx, ns, "second-profile-tax", "txcd_20000000")
			s.SeedProfileDefaultTaxConfigViaAdapter(ctx, firstProfile.ProfileID(), &productcatalog.TaxConfig{
				TaxCodeID: lo.ToPtr(firstTaxCode.ID),
			})
			secondProfileInput := billingtest.MinimalCreateProfileInputTemplate(sandboxApp.GetID())
			secondProfileInput.Namespace = ns
			secondProfileInput.Default = false
			secondProfileInput.Name = "Second customer profile"
			secondProfile, err := s.BillingService.CreateProfile(ctx, secondProfileInput)
			s.Require().NoError(err)
			s.SeedProfileDefaultTaxConfigViaAdapter(ctx, secondProfile.ProfileID(), &productcatalog.TaxConfig{
				TaxCodeID: lo.ToPtr(secondTaxCode.ID),
			})
			firstCustomer := s.CreateTestCustomer(ns, "first-subject")
			secondCustomer := s.CreateTestCustomer(ns, "second-subject")
			_, err = s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
				Namespace:  ns,
				CustomerID: secondCustomer.ID,
				ProfileID:  secondProfile.ID,
			})
			s.Require().NoError(err)

			var featureKey string
			if chargeType.price.Type() != productcatalog.FlatPriceType {
				usageFeature := s.SetupApiRequestsTotalFeature(ctx, ns)
				defer usageFeature.Cleanup()
				featureKey = usageFeature.Feature.Key
			}

			servicePeriod := timeutil.ClosedPeriod{
				From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			}
			clock.FreezeTime(servicePeriod.From.Add(-time.Hour))
			defer clock.UnFreeze()

			intents := charges.ChargeIntents{}
			for _, cust := range []*customer.Customer{firstCustomer, secondCustomer, firstCustomer} {
				intents = append(intents, s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price:          chargeType.price,
					featureKey:     featureKey,
					name:           "customer-scoped tax default",
					managedBy:      billing.ManuallyManagedLine,
				}))
			}

			// when: the customers' charges are interleaved in a single create request
			created, err := s.Charges.Create(ctx, charges.CreateInput{
				Namespace: ns,
				Intents:   charges.NewCreateChargeIntents(intents...),
			})
			s.Require().NoError(err)
			s.Require().Len(created, 3)

			// then: repeated customer lookups cannot reuse the other customer's default
			for idx, expectedTaxCodeID := range []string{firstTaxCode.ID, secondTaxCode.ID, firstTaxCode.ID} {
				chargeID, err := created[idx].GetChargeID()
				s.Require().NoError(err)
				s.requirePersistedChargeTaxCodeConfig(chargeID, productcatalog.TaxCodeConfig{TaxCodeID: expectedTaxCodeID})
			}
		})
	}
}

func (s *TaxCodePersistenceTestSuite) TestChargeTaxCodeDefaultsHandleMissingProfileAndLookupFailures() {
	for _, chargeType := range taxCodeDefaultingChargeTypes() {
		s.Run(chargeType.name, func() {
			// given: organization defaults exist, but no billing profile has been provisioned
			ctx := s.T().Context()
			ns := s.GetUniqueNamespace("tax-code-default-no-profile")
			s.ProvisionDefaultTaxCodes(ctx, ns)
			cust := s.CreateTestCustomer(ns, "test-subject")
			intent := s.createMockChargeIntent(createMockChargeIntentInput{
				customer: cust.GetID(),
				currency: USD,
				servicePeriod: timeutil.ClosedPeriod{
					From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				},
				price:          chargeType.price,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				name:           "tax default lookup",
				managedBy:      billing.ManuallyManagedLine,
			})

			s.Run("missing billing profile is propagated", func() {
				// when: defaulting encounters the typed absence of a billing profile
				resolved, err := s.Charges.applyDefaultTaxCodes(ctx, ns, charges.NewCreateChargeIntents(intent))
				s.Require().Error(err)

				// then: the missing profile is reported and no fallback result is returned
				s.ErrorIs(err, billing.ErrDefaultProfileNotFound)
				s.Empty(resolved)
			})

			s.Run("missing customer lookup is propagated", func() {
				missingCustomerIntent := s.createMockChargeIntent(createMockChargeIntentInput{
					customer: customer.CustomerID{Namespace: ns, ID: ulid.Make().String()},
					currency: USD,
					servicePeriod: timeutil.ClosedPeriod{
						From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
						To:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
					},
					price:          chargeType.price,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					name:           "missing customer tax lookup",
					managedBy:      billing.ManuallyManagedLine,
				})

				// when: the profile lookup fails because the customer does not exist
				resolved, err := s.Charges.applyDefaultTaxCodes(ctx, ns, charges.NewCreateChargeIntents(missingCustomerIntent))
				s.Require().Error(err)

				// then: the cause is retained and no fallback result is returned
				var notFoundErr *models.GenericNotFoundError
				s.ErrorAs(err, &notFoundErr)
				s.Empty(resolved)
			})
		})
	}
}

func (s *TaxCodePersistenceTestSuite) requirePersistedChargeTaxCodeConfig(chargeID meta.ChargeID, expected productcatalog.TaxCodeConfig) {
	s.T().Helper()
	persistedCharge := s.mustGetChargeByID(chargeID)
	var actual productcatalog.TaxCodeConfig
	switch persistedCharge.Type() {
	case meta.ChargeTypeFlatFee:
		flatFeeCharge, err := persistedCharge.AsFlatFeeCharge()
		s.Require().NoError(err)
		actual = flatFeeCharge.Intent.GetTaxConfig()
	case meta.ChargeTypeUsageBased:
		usageBasedCharge, err := persistedCharge.AsUsageBasedCharge()
		s.Require().NoError(err)
		actual = usageBasedCharge.Intent.GetTaxConfig()
	default:
		s.FailNow("expected a flat-fee or usage-based charge")
	}

	s.Equal(expected.TaxCodeID, actual.TaxCodeID)
	s.Equal(expected.Behavior, actual.Behavior)
}
