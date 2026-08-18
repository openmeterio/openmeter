package chargeadapter_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargecreditpurchase "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargecostbasis "github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// TestOnCreditPurchase_CustomCurrency_FullLifecycle exercises a custom
// currency purchase end to end: covering a pre-existing unknown-cost-basis
// advance (and its matching accrued balance), issuing residual credit into
// FBO, then authorizing and settling payment in the fiat settlement
// currency. This is the chargeadapter-level counterpart to the
// primitives-only coverage in transactions/fx_test.go.
func TestOnCreditPurchase_CustomCurrency_FullLifecycle(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)

	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	customCurrency := customCurrencyValue.GetCode()
	customCurrencyIdentity := customCurrencyValue.Reference()
	settlementCurrency := currencyx.Code("USD")
	costBasis := alpacadecimal.NewFromFloat(0.25)

	// given: a pre-existing custom-currency advance (unknown cost basis) worth
	// 40 ACME, created the way credit_only usage collection creates one when
	// FBO can't cover a spend.
	env.createCustomAdvanceExposure(t, customCurrency, customCurrencyIdentity, alpacadecimal.NewFromInt(40), nil)

	// when: a 100 ACME external-settlement purchase comes in, priced at
	// 0.25 USD per ACME (25 USD total).
	charge := env.newExternalChargeCustomCurrency(t, customCurrencyValue, alpacadecimal.NewFromInt(100), costBasis, settlementCurrency)

	ref, err := env.handler.OnCreditPurchaseInitiated(t.Context(), charge)
	require.NoError(t, err)
	require.NotEmpty(t, ref.TransactionGroupID)

	// then: the advance (unknown cost basis) and its matching accrued balance
	// are fully attributed into the known cost-basis bucket...
	require.True(t, env.sumBalance(t, env.customReceivableSubAccount(t, customCurrency, customCurrencyIdentity, nil, nil)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.customAccruedSubAccount(t, customCurrency, customCurrencyIdentity, nil, nil)).Equal(alpacadecimal.Zero))

	// ...the known cost-basis accrued bucket receives the attributed amount
	// *with* the fiat denomination it was purchased against...
	require.True(t, env.sumBalance(t, env.customAccruedSubAccount(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis)).Equal(alpacadecimal.NewFromInt(40)))

	// ...only the residual (100-40=60) is issued into available FBO...
	require.True(t, env.sumBalance(t, env.customFBOSubAccount(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis)).Equal(alpacadecimal.NewFromInt(60)))

	// ...and the known cost-basis custom receivable holds the full purchase
	// amount pending conversion into the fiat settlement currency.
	customReceivable := env.customReceivableSubAccount(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis)
	require.True(t, env.sumBalance(t, customReceivable).Equal(alpacadecimal.NewFromInt(-100)))
	env.requireAccountSourceBucketAmounts(t, customReceivable.AccountID().ID, map[string]float64{
		charge.ID: -100,
	})

	// when: payment for the purchase is authorized.
	authorizationInput := chargecreditpurchase.PaymentEventInput{
		Charge:     charge,
		EventAt:    charge.CreatedAt.Add(15 * time.Minute),
		FiatAmount: alpacadecimal.NewFromInt(25),
	}
	authRef, err := env.handler.OnCreditPurchasePaymentAuthorized(t.Context(), authorizationInput)
	require.NoError(t, err)
	require.NotEmpty(t, authRef.TransactionGroupID)

	// then: the custom-currency IOU is re-denominated into fiat and clears...
	require.True(t, env.sumBalance(t, env.customReceivableSubAccount(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis)).Equal(alpacadecimal.Zero))

	// ...and the fiat authorized receivable holds the converted amount
	// (100 ACME x 0.25 = 25 USD), not the raw custom credit amount.
	require.True(t, env.sumBalance(t, env.fiatAuthorizedReceivableSubAccount(t, settlementCurrency, costBasis)).Equal(alpacadecimal.NewFromInt(-25)))
	require.True(t, env.sumBalance(t, env.fiatOpenReceivableSubAccount(t, settlementCurrency, costBasis)).Equal(alpacadecimal.Zero))
	env.requireAccountSourceBucketAmounts(t, customReceivable.AccountID().ID, map[string]float64{
		charge.ID: -25,
	})

	// when: payment is settled.
	settlementInput := chargecreditpurchase.PaymentEventInput{
		Charge:     charge,
		EventAt:    charge.CreatedAt.Add(30 * time.Minute),
		FiatAmount: alpacadecimal.NewFromInt(25),
	}
	settleRef, err := env.handler.OnCreditPurchasePaymentSettled(t.Context(), settlementInput)
	require.NoError(t, err)
	require.NotEmpty(t, settleRef.TransactionGroupID)

	// then: the fiat authorized receivable clears against wash for the fiat
	// amount, and the customer's custom-currency FBO balance is untouched by
	// settlement (they already have their 60 ACME of usable credit).
	require.True(t, env.sumBalance(t, env.fiatAuthorizedReceivableSubAccount(t, settlementCurrency, costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.fiatWashSubAccount(t, settlementCurrency, costBasis)).Equal(alpacadecimal.NewFromInt(-25)))
	require.True(t, env.sumBalance(t, env.customFBOSubAccount(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis)).Equal(alpacadecimal.NewFromInt(60)))
	env.requireAccountSourceBucketAmounts(t, customReceivable.AccountID().ID, map[string]float64{})
}

func TestOnCreditPurchase_CustomCurrency_UsesExactFiatAmount(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)
	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	customCurrency := customCurrencyValue.GetCode()
	customCurrencyIdentity := customCurrencyValue.Reference()
	settlementCurrency := currencyx.Code("USD")
	costBasis := alpacadecimal.RequireFromString("1.005")
	fiatAmount := alpacadecimal.RequireFromString("1.01")
	charge := env.newExternalChargeCustomCurrency(t, customCurrencyValue, alpacadecimal.NewFromInt(1), costBasis, settlementCurrency)

	_, err := env.handler.OnCreditPurchaseInitiated(t.Context(), charge)
	require.NoError(t, err)

	_, err = env.handler.OnCreditPurchasePaymentAuthorized(t.Context(), chargecreditpurchase.PaymentEventInput{
		Charge:     charge,
		EventAt:    charge.CreatedAt.Add(15 * time.Minute),
		FiatAmount: fiatAmount,
	})
	require.NoError(t, err)
	require.True(t, env.sumBalance(t, env.customReceivableSubAccount(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.fiatAuthorizedReceivableSubAccount(t, settlementCurrency, costBasis)).Equal(fiatAmount.Neg()))

	_, err = env.handler.OnCreditPurchasePaymentSettled(t.Context(), chargecreditpurchase.PaymentEventInput{
		Charge:     charge,
		EventAt:    charge.CreatedAt.Add(30 * time.Minute),
		FiatAmount: fiatAmount,
	})
	require.NoError(t, err)
	require.True(t, env.sumBalance(t, env.fiatAuthorizedReceivableSubAccount(t, settlementCurrency, costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.fiatWashSubAccount(t, settlementCurrency, costBasis)).Equal(fiatAmount.Neg()))
}

func TestOnCreditPurchaseInitiated_CustomCurrency_BackfillAllocatesFractionalRemainder(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)

	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	customCurrency := customCurrencyValue.GetCode()
	customCurrencyIdentity := customCurrencyValue.Reference()
	settlementCurrency := currencyx.Code("USD")
	costBasis := alpacadecimal.NewFromInt(1)
	spendChargeIDs := []string{
		"01JSPEND00123456789ABCDEFG",
		"01JSPEND10123456789ABCDEFG",
		"01JSPEND20123456789ABCDEFG",
	}

	// given: three equal unknown-cost-basis accrued buckets whose proportional
	// share of a 0.05 ACME backfill is a non-representable 0.0166... ACME.
	for i := range spendChargeIDs {
		env.createCustomAdvanceExposure(t, customCurrency, customCurrencyIdentity, alpacadecimal.NewFromInt(1), &spendChargeIDs[i])
	}

	// when: the purchase is too small to cover the full advance exposure.
	charge := env.newExternalChargeCustomCurrency(t, customCurrencyValue, mustDecimal(t, "0.05"), costBasis, settlementCurrency)
	charge.ID = "01JABCDEF0123456789ABCDEFG"
	_, err := env.handler.OnCreditPurchaseInitiated(t.Context(), charge)
	require.NoError(t, err)

	// then: the largest-remainder allocation preserves both the 0.05 total and
	// each spend bucket, with deterministic tie-breaking by spend charge ID.
	env.requireAccountSourceSpendBucketAmounts(t, env.customAccruedSubAccount(t, customCurrency, customCurrencyIdentity, nil, nil).AccountID().ID, map[string]float64{
		sourceSpendChargeKey(nil, &spendChargeIDs[0]):        0.98,
		sourceSpendChargeKey(nil, &spendChargeIDs[1]):        0.98,
		sourceSpendChargeKey(nil, &spendChargeIDs[2]):        0.99,
		sourceSpendChargeKey(&charge.ID, &spendChargeIDs[0]): 0.02,
		sourceSpendChargeKey(&charge.ID, &spendChargeIDs[1]): 0.02,
		sourceSpendChargeKey(&charge.ID, &spendChargeIDs[2]): 0.01,
	})
}

// TestOnPromotionalCreditPurchase_CustomCurrency locks the promotional path
// for a custom currency: a promotional grant has no cost basis to denominate
// (settlement returns a zero rate, not a real one), so the credited FBO and
// its immediate wash settlement must land in the unknown-cost-basis custom
// bucket (nil cost basis, nil exchange source), never a zero-rate bucket.
// Without threading a nil cost basis through, route validation rejects the
// custom grant outright.
func TestOnPromotionalCreditPurchase_CustomCurrency(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)

	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	customCurrency := customCurrencyValue.GetCode()
	customCurrencyIdentity := customCurrencyValue.Reference()

	charge := env.newPromotionalChargeCustomCurrency(t, customCurrencyValue, alpacadecimal.NewFromInt(100))
	ref, err := env.handler.OnPromotionalCreditPurchase(t.Context(), charge)
	require.NoError(t, err)
	require.NotEmpty(t, ref.TransactionGroupID)

	// then: the full grant is credited to the unknown-cost-basis custom FBO
	// bucket and settled through wash, leaving no open or authorized receivable.
	require.True(t, env.sumBalance(t, env.customFBOSubAccount(t, customCurrency, customCurrencyIdentity, nil, nil)).Equal(alpacadecimal.NewFromInt(100)))
	require.True(t, env.sumBalance(t, env.customReceivableSubAccount(t, customCurrency, customCurrencyIdentity, nil, nil)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.customAuthorizedReceivableSubAccount(t, customCurrency, customCurrencyIdentity, nil, nil)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.customWashSubAccount(t, customCurrency, customCurrencyIdentity, nil, nil)).Equal(alpacadecimal.NewFromInt(-100)))
}

// TestOnPromotionalCreditPurchase_CustomCurrency_CoversAdvance covers the
// Settlement Modes hypothetical #1 case: a promotional custom grant landing on
// a pre-existing unknown-cost-basis advance. A promotional grant carries no
// cost basis, so it cannot re-bucket the advance into a known cost basis (that
// is a paid purchase's job). Instead it issues its full amount to FBO and
// leaves the advance on its unknown-cost-basis route. The advance is not
// double-counted: the customer balance formula (FBO + nil-cost-basis advance
// receivable) nets the untouched −40 advance against the +100 grant, so the
// customer ends up with 60 spendable while the advance stays recorded for a
// later paid purchase to attribute.
func TestOnPromotionalCreditPurchase_CustomCurrency_CoversAdvance(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)

	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	customCurrency := customCurrencyValue.GetCode()
	customCurrencyIdentity := customCurrencyValue.Reference()

	env.createCustomAdvanceExposure(t, customCurrency, customCurrencyIdentity, alpacadecimal.NewFromInt(40), nil)

	charge := env.newPromotionalChargeCustomCurrency(t, customCurrencyValue, alpacadecimal.NewFromInt(100))
	ref, err := env.handler.OnPromotionalCreditPurchase(t.Context(), charge)
	require.NoError(t, err)
	require.NotEmpty(t, ref.TransactionGroupID)

	// then: the full 100 ACME grant is credited to FBO (no attribution, since a
	// promotional grant has no cost basis to attribute into)...
	require.True(t, env.sumBalance(t, env.customFBOSubAccount(t, customCurrency, customCurrencyIdentity, nil, nil)).Equal(alpacadecimal.NewFromInt(100)))

	// ...and the pre-existing advance is left untouched on its unknown-cost-basis
	// route (receivable −40, accrued +40), so the balance formula nets it to 60
	// spendable rather than double-counting it.
	require.True(t, env.sumBalance(t, env.customReceivableSubAccount(t, customCurrency, customCurrencyIdentity, nil, nil)).Equal(alpacadecimal.NewFromInt(-40)))
	require.True(t, env.sumBalance(t, env.customAccruedSubAccount(t, customCurrency, customCurrencyIdentity, nil, nil)).Equal(alpacadecimal.NewFromInt(40)))

	// the promotional grant's own receivable is fully settled through wash, so it
	// leaves no authorized receivable behind.
	require.True(t, env.sumBalance(t, env.customAuthorizedReceivableSubAccount(t, customCurrency, customCurrencyIdentity, nil, nil)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.customWashSubAccount(t, customCurrency, customCurrencyIdentity, nil, nil)).Equal(alpacadecimal.NewFromInt(-100)))
}

// TestOnCreditPurchaseInitiated_CustomCurrency_FeatureFilteredNoAdvance proves
// that an external-settlement custom purchase keeps its feature restriction
// while authorization re-denominates the custom IOU into fiat and settlement
// clears it against wash.
func TestOnCreditPurchaseInitiated_CustomCurrency_FeatureFilteredNoAdvance(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)

	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	customCurrency := customCurrencyValue.GetCode()
	customCurrencyIdentity := customCurrencyValue.Reference()
	settlementCurrency := currencyx.Code("USD")
	costBasis := alpacadecimal.NewFromFloat(0.25)
	featureFilters := chargecreditpurchase.FeatureFilters{"api-calls"}.Normalize()

	charge := env.newExternalChargeCustomCurrency(t, customCurrencyValue, alpacadecimal.NewFromInt(100), costBasis, settlementCurrency)
	charge.Intent.FeatureFilters = featureFilters

	ref, err := env.handler.OnCreditPurchaseInitiated(t.Context(), charge)
	require.NoError(t, err)
	require.NotEmpty(t, ref.TransactionGroupID)

	// then: the full purchase is issued to FBO with its fiat denomination, and
	// the custom receivable holds the full IOU pending conversion.
	require.True(t, env.sumBalance(t, env.customFBOSubAccountWithFeatures(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis, featureFilters)).Equal(alpacadecimal.NewFromInt(100)))
	require.True(t, env.sumBalance(t, env.customReceivableSubAccountWithFeatures(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis, featureFilters)).Equal(alpacadecimal.NewFromInt(-100)))

	authRef, err := env.handler.OnCreditPurchasePaymentAuthorized(t.Context(), chargecreditpurchase.PaymentEventInput{
		Charge:     charge,
		EventAt:    charge.CreatedAt.Add(15 * time.Minute),
		FiatAmount: alpacadecimal.NewFromInt(25),
	})
	require.NoError(t, err)
	require.NotEmpty(t, authRef.TransactionGroupID)

	// then: the custom IOU clears and the fiat authorized receivable holds the
	// converted amount (25 USD), not the raw 100 ACME.
	require.True(t, env.sumBalance(t, env.customReceivableSubAccountWithFeatures(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis, featureFilters)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.fiatOpenReceivableSubAccount(t, settlementCurrency, costBasis, featureFilters...)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.fiatOpenReceivableSubAccount(t, settlementCurrency, costBasis)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.fiatAuthorizedReceivableSubAccount(t, settlementCurrency, costBasis, featureFilters...)).Equal(alpacadecimal.NewFromInt(-25)))

	settleRef, err := env.handler.OnCreditPurchasePaymentSettled(t.Context(), chargecreditpurchase.PaymentEventInput{
		Charge:     charge,
		EventAt:    charge.CreatedAt.Add(30 * time.Minute),
		FiatAmount: alpacadecimal.NewFromInt(25),
	})
	require.NoError(t, err)
	require.NotEmpty(t, settleRef.TransactionGroupID)

	// then: the fiat authorized receivable clears against wash and the custom
	// FBO credit is untouched by settlement.
	require.True(t, env.sumBalance(t, env.fiatAuthorizedReceivableSubAccount(t, settlementCurrency, costBasis, featureFilters...)).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.fiatWashSubAccount(t, settlementCurrency, costBasis)).Equal(alpacadecimal.NewFromInt(-25)))
	require.True(t, env.sumBalance(t, env.customFBOSubAccountWithFeatures(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis, featureFilters)).Equal(alpacadecimal.NewFromInt(100)))
}

func (e *creditPurchaseHandlerTestEnv) newPromotionalChargeCustomCurrency(
	t *testing.T,
	customCurrencyValue currencies.Currency,
	amount alpacadecimal.Decimal,
) chargecreditpurchase.Charge {
	t.Helper()

	charge := e.newExternalChargeCustomCurrency(t, customCurrencyValue, amount, alpacadecimal.Zero, currencyx.Code("USD"))
	charge.ID = "credit-purchase-cc-promo"
	charge.Intent.Name = "Promotional Credit Purchase (custom currency)"
	charge.Intent.Settlement = chargecreditpurchase.NewSettlement(chargecreditpurchase.PromotionalSettlement{})
	charge.Intent.CostBasis = chargecreditpurchase.CostBasis{}
	charge.State.ChargeCostBasisID = nil
	charge.State.ResolvedCostBasis = nil

	return charge
}

// TestOnCreditPurchaseInitiated_CustomCurrency_BackfillsOnlyMatchingFeatureAdvances
// mirrors TestOnCreditPurchaseInitiated_BackfillsOnlyMatchingFeatureAdvances for
// a custom currency: two independent unknown-cost-basis advances scoped to
// different features must not cross-contaminate when a feature-filtered
// purchase attributes only the matching one. This exercises the
// feature-scoped advance-bucket matching (newAdvanceReceivableBuckets) that
// the single-advance lifecycle test never has more than one bucket to
// disambiguate between.
func TestOnCreditPurchaseInitiated_CustomCurrency_BackfillsOnlyMatchingFeatureAdvances(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)

	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	customCurrency := customCurrencyValue.GetCode()
	customCurrencyIdentity := customCurrencyValue.Reference()
	settlementCurrency := currencyx.Code("USD")
	costBasis := mustDecimal(t, "0.5")

	env.createCustomAdvanceExposureWithFeatures(t, customCurrency, customCurrencyIdentity, alpacadecimal.NewFromInt(40), []string{"api-calls"})
	env.createCustomAdvanceExposureWithFeatures(t, customCurrency, customCurrencyIdentity, alpacadecimal.NewFromInt(30), []string{"storage"})

	featureFilters := chargecreditpurchase.FeatureFilters{"api-calls"}
	charge := env.newExternalChargeCustomCurrency(t, customCurrencyValue, alpacadecimal.NewFromInt(100), costBasis, settlementCurrency)
	charge.Intent.FeatureFilters = featureFilters

	ref, err := env.handler.OnCreditPurchaseInitiated(t.Context(), charge)
	require.NoError(t, err)
	require.NotEmpty(t, ref.TransactionGroupID)

	// then: the api-calls advance is fully attributed into the known
	// cost-basis bucket...
	require.True(t, env.sumBalance(t, env.customReceivableSubAccountWithFeatures(t, customCurrency, customCurrencyIdentity, nil, nil, []string{"api-calls"})).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalance(t, env.customAccruedSubAccount(t, customCurrency, customCurrencyIdentity, nil, nil)).Equal(alpacadecimal.NewFromInt(30)))
	require.True(t, env.sumBalance(t, env.customAccruedSubAccount(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis)).Equal(alpacadecimal.NewFromInt(40)))

	// ...the unrelated storage advance is untouched...
	require.True(t, env.sumBalance(t, env.customReceivableSubAccountWithFeatures(t, customCurrency, customCurrencyIdentity, nil, nil, []string{"storage"})).Equal(alpacadecimal.NewFromInt(-30)))

	// ...residual (100-40=60) is issued to FBO scoped to the matching feature,
	// and the known cost-basis receivable holds the full purchase amount.
	require.True(t, env.sumBalance(t, env.customFBOSubAccountWithFeatures(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis, featureFilters.Normalize())).Equal(alpacadecimal.NewFromInt(60)))
	require.True(t, env.sumBalance(t, env.customReceivableSubAccountWithFeatures(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis, featureFilters.Normalize())).Equal(alpacadecimal.NewFromInt(-100)))
}

// TestOnCreditPurchaseInitiated_CustomCurrency_ExpiringCreditReleasesAdvanceCoverage
// mirrors TestOnCreditPurchaseInitiated_ExpiringCreditReleasesAdvanceCoverage
// for a custom currency: an expiring purchase that also backs a pre-existing
// advance must immediately release the attributed slice from breakage (it was
// already consumed before it could ever expire), leaving only the residual,
// still-unconsumed FBO credit as an open breakage plan.
func TestOnCreditPurchaseInitiated_CustomCurrency_ExpiringCreditReleasesAdvanceCoverage(t *testing.T) {
	env := newCreditPurchaseHandlerTestEnv(t)

	customCurrencyValue := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
	customCurrency := customCurrencyValue.GetCode()
	customCurrencyIdentity := customCurrencyValue.Reference()
	settlementCurrency := currencyx.Code("USD")
	costBasis := mustDecimal(t, "0.5")

	env.createCustomAdvanceExposure(t, customCurrency, customCurrencyIdentity, alpacadecimal.NewFromInt(40), nil)

	charge := env.newExternalChargeCustomCurrency(t, customCurrencyValue, alpacadecimal.NewFromInt(100), costBasis, settlementCurrency)
	expiresAt := charge.CreatedAt.Add(time.Hour)
	charge.Intent.ExpiresAt = &expiresAt

	ref, err := env.handler.OnCreditPurchaseInitiated(t.Context(), charge)
	require.NoError(t, err)
	require.NotEmpty(t, ref.TransactionGroupID)
	require.ElementsMatch(t, []string{
		transactions.TemplateCode(transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{}),
		transactions.TemplateCode(transactions.TranslateCustomerAccruedCostBasisTemplate{}),
		transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}),
		transactions.TemplateCode(transactions.PlanCustomerFBOBreakageTemplate{}),
		transactions.TemplateCode(transactions.ReleaseCustomerFBOBreakageTemplate{}),
	}, env.transactionTemplateCodes(t, ref.TransactionGroupID))

	// then: the breakage plan covers the full 100, but 40 of it is immediately
	// released (the slice that covered the advance, already consumed).
	rows := env.breakageRows(t, ref.TransactionGroupID)
	require.Len(t, rows, 2)

	byKind := map[ledger.BreakageKind]alpacadecimal.Decimal{}
	for _, row := range rows {
		byKind[row.Kind] = row.Amount
	}
	require.True(t, byKind[ledger.BreakageKindPlan].Equal(alpacadecimal.NewFromInt(100)))
	require.True(t, byKind[ledger.BreakageKindRelease].Equal(alpacadecimal.NewFromInt(40)))

	// - only the residual (60) is ever at risk of expiring into breakage...
	fbo := env.customFBOSubAccount(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis)
	breakageSubAccount := env.customBreakageSubAccount(t, customCurrency, customCurrencyIdentity, &settlementCurrency, &costBasis)
	require.True(t, env.sumBalanceAsOf(t, fbo, charge.CreatedAt).Equal(alpacadecimal.NewFromInt(60)))
	require.True(t, env.sumBalanceAsOf(t, breakageSubAccount, charge.CreatedAt).Equal(alpacadecimal.Zero))

	// - ...and at expiry, only that residual actually breaks.
	require.True(t, env.sumBalanceAsOf(t, fbo, expiresAt).Equal(alpacadecimal.Zero))
	require.True(t, env.sumBalanceAsOf(t, breakageSubAccount, expiresAt).Equal(alpacadecimal.NewFromInt(60)))
}

func (e *creditPurchaseHandlerTestEnv) customBreakageSubAccount(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference, costBasisCurrency *currencyx.Code, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.BusinessAccounts.BreakageAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:          customCurrency,
		CostBasisCurrency: costBasisCurrency,
		CostBasis:         costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) createCustomAdvanceExposureWithFeatures(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference, amount alpacadecimal.Decimal, features []string) {
	t.Helper()

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService: e.Deps.ResolversService,
			AccountCatalog: e.Deps.AccountService,
			BalanceQuerier: e.Deps.HistoricalLedger,
		},
		transactions.ResolutionScope{
			CustomerID: e.CustomerID,
			Namespace:  e.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:       e.Now(),
			Amount:   amount,
			Currency: customCurrency,
			Features: features,
		},
		transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
			At:       e.Now(),
			Amount:   amount,
			Currency: customCurrency,
			Features: features,
		},
	)
	require.NoError(t, err)

	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}

func (e *creditPurchaseHandlerTestEnv) customFBOSubAccountWithFeatures(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference, costBasisCurrency *currencyx.Code, costBasis *alpacadecimal.Decimal, features []string) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:          customCurrency,
		CostBasisCurrency: costBasisCurrency,
		CostBasis:         costBasis,
		Features:          features,
		CreditPriority:    ledger.DefaultCustomerFBOPriority,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) customReceivableSubAccountWithFeatures(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference, costBasisCurrency *currencyx.Code, costBasis *alpacadecimal.Decimal, features []string) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       customCurrency,
		CostBasisCurrency:              costBasisCurrency,
		CostBasis:                      costBasis,
		Features:                       features,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) newExternalChargeCustomCurrency(
	t *testing.T,
	customCurrencyValue currencies.Currency,
	amount, costBasis alpacadecimal.Decimal,
	settlementCurrency currencyx.Code,
) chargecreditpurchase.Charge {
	t.Helper()

	now := time.Now().UTC()
	servicePeriod := timeutil.ClosedPeriod{
		From: now.Add(-time.Hour),
		To:   now,
	}
	fiatCurrency, err := currencyx.NewFiatCurrency(settlementCurrency)
	require.NoError(t, err)
	return chargecreditpurchase.Charge{
		ChargeBase: chargecreditpurchase.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: e.Namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: now,
					UpdatedAt: now,
				},
				ID: "credit-purchase-charge-cc",
			},
			Intent: chargecreditpurchase.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.SystemManagedLine,
					CustomerID: e.CustomerID.ID,
					Currency:   customCurrencyValue,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: "tax-code-id",
					},
				},
				IntentMutableFields: chargecreditpurchase.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "External Credit Purchase (custom currency)",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					CreditAmount: amount,
					Settlement: chargecreditpurchase.NewSettlement(chargecreditpurchase.ExternalSettlement{
						InitialStatus: chargecreditpurchase.CreatedInitialPaymentSettlementStatus,
					}),
				},
				CostBasis: chargecreditpurchase.NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.ManualIntent{
					FiatCurrency: fiatCurrency,
					Rate:         costBasis,
				})),
			},
			Status: chargecreditpurchase.StatusCreated,
			State: chargecreditpurchase.State{
				ChargeCostBasisID: lo.ToPtr("credit-purchase-cost-basis-cc"),
				ResolvedCostBasis: &chargecostbasis.State{
					CostBasis:  costBasis,
					ResolvedAt: now,
				},
			},
		},
	}
}

// createCustomAdvanceExposure books an unknown-cost-basis custom currency
// advance the same way credit_only usage collection does when FBO can't
// cover a spend: issue receivable + move it straight to accrued.
func (e *creditPurchaseHandlerTestEnv) createCustomAdvanceExposure(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference, amount alpacadecimal.Decimal, spendChargeID *string) {
	t.Helper()

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService: e.Deps.ResolversService,
			AccountCatalog: e.Deps.AccountService,
			BalanceQuerier: e.Deps.HistoricalLedger,
		},
		transactions.ResolutionScope{
			CustomerID: e.CustomerID,
			Namespace:  e.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:            e.Now(),
			Amount:        amount,
			Currency:      customCurrency,
			SpendChargeID: spendChargeID,
		},
		transactions.TransferCustomerFBOAdvanceToAccruedTemplate{
			At:            e.Now(),
			Amount:        amount,
			Currency:      customCurrency,
			SpendChargeID: spendChargeID,
		},
	)
	require.NoError(t, err)

	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}

func (e *creditPurchaseHandlerTestEnv) customFBOSubAccount(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference, costBasisCurrency *currencyx.Code, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{
		Currency:          customCurrency,
		CostBasisCurrency: costBasisCurrency,
		CostBasis:         costBasis,
		CreditPriority:    ledger.DefaultCustomerFBOPriority,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) customReceivableSubAccount(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference, costBasisCurrency *currencyx.Code, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       customCurrency,
		CostBasisCurrency:              costBasisCurrency,
		CostBasis:                      costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) customAccruedSubAccount(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference, costBasisCurrency *currencyx.Code, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.AccruedAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerAccruedRouteParams{
		Currency:          customCurrency,
		CostBasisCurrency: costBasisCurrency,
		CostBasis:         costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) customAuthorizedReceivableSubAccount(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference, costBasisCurrency *currencyx.Code, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       customCurrency,
		CostBasisCurrency:              costBasisCurrency,
		CostBasis:                      costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusAuthorized,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) customWashSubAccount(t *testing.T, currency currencyx.Code, customCurrency currencies.CurrencyReference, costBasisCurrency *currencyx.Code, costBasis *alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.BusinessAccounts.WashAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:          customCurrency,
		CostBasisCurrency: costBasisCurrency,
		CostBasis:         costBasis,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) fiatOpenReceivableSubAccount(t *testing.T, currency currencyx.Code, costBasis alpacadecimal.Decimal, features ...string) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       currencies.NewCurrencyReference(currency),
		CostBasis:                      &costBasis,
		Features:                       features,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) fiatAuthorizedReceivableSubAccount(t *testing.T, currency currencyx.Code, costBasis alpacadecimal.Decimal, features ...string) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.CustomerAccounts.ReceivableAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       currencies.NewCurrencyReference(currency),
		CostBasis:                      &costBasis,
		Features:                       features,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusAuthorized,
	})
	require.NoError(t, err)

	return subAccount
}

func (e *creditPurchaseHandlerTestEnv) fiatWashSubAccount(t *testing.T, currency currencyx.Code, costBasis alpacadecimal.Decimal) ledger.SubAccount {
	t.Helper()

	subAccount, err := e.BusinessAccounts.WashAccount.GetSubAccountForRoute(t.Context(), ledger.BusinessRouteParams{
		Currency:  currencies.NewCurrencyReference(currency),
		CostBasis: &costBasis,
	})
	require.NoError(t, err)

	return subAccount
}
