package customerscredits

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/handlers/billingcommon"
	"github.com/openmeterio/openmeter/api/v3/labels"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func toAPIBillingCreditGrant(charge creditpurchase.Charge) (apiv3models.BillingCreditGrant, error) {
	grant := apiv3models.BillingCreditGrant{
		Id:            charge.ID,
		Name:          charge.Intent.Name,
		Description:   charge.Intent.Description,
		Amount:        charge.Intent.CreditAmount.String(),
		Currency:      apiv3models.BillingCurrencyCode(charge.Intent.Currency.GetCode()),
		EffectiveAt:   charge.Intent.EffectiveAt,
		ExpiresAt:     charge.Intent.ExpiresAt,
		FundingMethod: toAPIBillingCreditFundingMethod(charge.Intent.Settlement),
		Status:        toAPIBillingCreditGrantStatus(charge),
		VoidedAt:      charge.State.VoidedAt,
		CreatedAt:     charge.CreatedAt,
		UpdatedAt:     charge.UpdatedAt,
		DeletedAt:     charge.DeletedAt,
		Labels:        labels.FromMetadata(charge.Intent.Metadata),
		Key:           charge.Intent.Key,
	}
	grant.ValidationIssues = billingcommon.ToAPIValidationIssues(charge.ValidationIssues)
	if charge.Intent.Currency.IsCustom() {
		grant.CustomCurrency = &apiv3models.CurrencyCustomReference{Id: charge.Intent.Currency.ID}
	}

	grant.Filters = toAPIBillingCreditGrantFilters(charge.Intent.Filters)

	if charge.Intent.Priority != nil {
		p := int16(*charge.Intent.Priority)
		grant.Priority = &p
	}

	purchase, err := toAPICreditGrantPurchase(charge)
	if err != nil {
		return grant, fmt.Errorf("converting purchase: %w", err)
	}

	grant.Purchase = purchase
	grant.TaxConfig = toAPITaxCodeConfig(charge)

	return grant, nil
}

func toAPIBillingCreditFundingMethod(settlement creditpurchase.Settlement) apiv3models.BillingCreditFundingMethod {
	switch settlement.Type() {
	case creditpurchase.SettlementTypeInvoice:
		return apiv3models.BillingCreditFundingMethodInvoice
	case creditpurchase.SettlementTypeExternal:
		return apiv3models.BillingCreditFundingMethodExternal
	default:
		return apiv3models.BillingCreditFundingMethodNone
	}
}

func toAPIBillingCreditGrantStatus(charge creditpurchase.Charge) apiv3models.BillingCreditGrantStatus {
	if charge.State.VoidedAt != nil {
		return apiv3models.BillingCreditGrantStatusVoided
	}

	if charge.Intent.ExpiresAt != nil && !charge.Intent.ExpiresAt.After(clock.Now()) {
		return apiv3models.BillingCreditGrantStatusExpired
	}

	switch charge.Status {
	case creditpurchase.StatusActive, creditpurchase.StatusFinal:
		return apiv3models.BillingCreditGrantStatusActive
	case creditpurchase.StatusCreated:
		return apiv3models.BillingCreditGrantStatusPending
	case creditpurchase.StatusDeleted:
		return apiv3models.BillingCreditGrantStatusVoided
	default:
		return apiv3models.BillingCreditGrantStatusActive
	}
}

// toAPICreditGrantPurchase builds the purchase block for funded grants (invoice or external).
// Returns nil for promotional grants (funding_method=none). Amount and the resolved
// cost basis are omitted while a dynamic cost basis is still unresolved.
func toAPICreditGrantPurchase(charge creditpurchase.Charge) (*apiv3models.BillingCreditGrantPurchase, error) {
	settlement := charge.Intent.Settlement

	var availabilityPolicy *apiv3models.BillingCreditAvailabilityPolicy
	settlementStatus := apiv3models.BillingCreditPurchasePaymentSettlementStatusPending

	switch settlement.Type() {
	case creditpurchase.SettlementTypeInvoice:
		if charge.Realizations.InvoiceSettlement != nil {
			settlementStatus = toAPIBillingCreditPurchasePaymentSettlementStatus(charge.Realizations.InvoiceSettlement.Status)
		}
	case creditpurchase.SettlementTypeExternal:
		ext, err := settlement.AsExternalSettlement()
		if err != nil {
			return nil, fmt.Errorf("getting external settlement: %w", err)
		}

		policy, err := toAPIBillingCreditAvailabilityPolicy(ext.InitialStatus)
		if err != nil {
			return nil, fmt.Errorf("converting availability policy: %w", err)
		}

		availabilityPolicy = &policy

		if charge.Realizations.ExternalPaymentSettlement != nil {
			settlementStatus = toAPIBillingCreditPurchasePaymentSettlementStatus(charge.Realizations.ExternalPaymentSettlement.Status)
		}
	case creditpurchase.SettlementTypePromotional:
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid settlement type: %s", settlement.Type())
	}

	fiatCurrency, err := charge.Intent.GetSettlementFiatCurrency()
	if err != nil {
		return nil, fmt.Errorf("getting settlement fiat currency: %w", err)
	}

	resolvedCostBasis, err := billingcommon.ToAPIChargeResolvedCostBasis(fiatCurrency, charge.State.ResolvedCostBasis)
	if err != nil {
		return nil, fmt.Errorf("converting resolved cost basis: %w", err)
	}

	purchase := &apiv3models.BillingCreditGrantPurchase{
		Currency:           apiv3models.CurrencyCode(fiatCurrency.GetFiatCode()),
		ResolvedCostBasis:  resolvedCostBasis,
		AvailabilityPolicy: availabilityPolicy,
		SettlementStatus:   &settlementStatus,
	}

	if resolvedCostBasis != nil {
		purchaseAmount, err := charge.GetFiatSettlementAmount()
		if err != nil {
			return nil, fmt.Errorf("getting fiat settlement amount: %w", err)
		}

		purchase.Amount = lo.ToPtr(purchaseAmount.String())

		if charge.Intent.Currency.IsFiat() {
			purchase.PerUnitCostBasis = &resolvedCostBasis.Rate
		}
	}

	return purchase, nil
}

func toAPIBillingCreditPurchasePaymentSettlementStatus(status payment.Status) apiv3models.BillingCreditPurchasePaymentSettlementStatus {
	switch status {
	case payment.StatusAuthorized:
		return apiv3models.BillingCreditPurchasePaymentSettlementStatusAuthorized
	case payment.StatusSettled:
		return apiv3models.BillingCreditPurchasePaymentSettlementStatusSettled
	default:
		return apiv3models.BillingCreditPurchasePaymentSettlementStatusPending
	}
}

func toAPIBillingCreditAvailabilityPolicy(status creditpurchase.InitialPaymentSettlementStatus) (apiv3models.BillingCreditAvailabilityPolicy, error) {
	switch status {
	case creditpurchase.CreatedInitialPaymentSettlementStatus:
		return apiv3models.BillingCreditAvailabilityPolicyOnCreation, nil
	default:
		return "", fmt.Errorf("invalid availability policy: %s", status)
	}
}

func toAPITaxCodeConfig(charge creditpurchase.Charge) *apiv3models.TaxCodeConfig {
	cfg := charge.Intent.TaxConfig
	if lo.IsEmpty(cfg) {
		return nil
	}

	tc := &apiv3models.TaxCodeConfig{}

	if cfg.Behavior != nil {
		behavior := apiv3models.BillingTaxBehavior(*cfg.Behavior)
		tc.Behavior = &behavior
	}

	if cfg.TaxCodeID != "" {
		tc.Code = &apiv3models.TaxCodeReference{Id: cfg.TaxCodeID}
	}

	return tc
}

func toAPIBillingCreditGrantFilters(filters ledger.CreditFilters) *apiv3models.BillingCreditGrantFilters {
	if filters.IsEmpty() {
		return nil
	}

	filters = filters.Normalize()
	result := &apiv3models.BillingCreditGrantFilters{}
	if len(filters.Features) > 0 {
		result.Features = lo.ToPtr(filters.Features)
	}

	if len(filters.Plans) > 0 {
		result.Plans = lo.ToPtr(lo.Map(filters.Plans, func(plan ledger.PlanFilter, _ int) apiv3models.BillingCreditGrantPlanFilter {
			mapped := apiv3models.BillingCreditGrantPlanFilter{Key: plan.Key}
			if v := plan.Version; v != nil {
				mapped.Version = &apiv3models.VersionFilter{}
				if v.Eq != nil {
					mapped.Version.Eq = lo.ToPtr(int32(*v.Eq))
				}

				if v.Gte != nil {
					mapped.Version.Gte = lo.ToPtr(int32(*v.Gte))
				}

				if v.Lte != nil {
					mapped.Version.Lte = lo.ToPtr(int32(*v.Lte))
				}

				if v.In != nil {
					mapped.Version.Oeq = lo.ToPtr(lo.Map(v.In, func(n int, _ int) int32 { return int32(n) }))
				}
			}

			return mapped
		}))
	}

	return result
}

func fromAPIBillingCreditFundingMethod(fm apiv3models.BillingCreditFundingMethod) creditgrant.FundingMethod {
	switch fm {
	case apiv3models.BillingCreditFundingMethodInvoice:
		return creditgrant.FundingMethodInvoice
	case apiv3models.BillingCreditFundingMethodExternal:
		return creditgrant.FundingMethodExternal
	default:
		return creditgrant.FundingMethodNone
	}
}

func fromAPIBillingCreditAvailabilityPolicy(policy apiv3models.BillingCreditAvailabilityPolicy) (creditpurchase.InitialPaymentSettlementStatus, error) {
	switch policy {
	case apiv3models.BillingCreditAvailabilityPolicyOnCreation:
		return creditpurchase.CreatedInitialPaymentSettlementStatus, nil
	default:
		return "", models.NewGenericValidationError(fmt.Errorf("invalid availability policy: %s", policy))
	}
}

func fromAPITaxCodeConfig(tc *apiv3models.CreateTaxCodeConfig) *productcatalog.TaxConfig {
	if tc == nil {
		return nil
	}

	config := &productcatalog.TaxConfig{}

	if tc.Behavior != nil {
		behavior := productcatalog.TaxBehavior(*tc.Behavior)
		config.Behavior = &behavior
	}

	if tc.Code != nil {
		config.TaxCodeID = &tc.Code.Id
	}

	return config
}

func fromAPIBillingCreditGrantFilters(filters *apiv3models.CreateCreditGrantFilters) (*creditgrant.GrantFilters, error) {
	if filters == nil {
		return nil, nil
	}

	result := &creditgrant.GrantFilters{Features: lo.FromPtr(filters.Features)}
	if filters.Plans != nil {
		result.Plans = lo.Map(*filters.Plans, func(plan apiv3models.CreateCreditGrantPlanFilter, _ int) ledger.PlanFilter {
			mapped := ledger.PlanFilter{Key: plan.Key}
			if v := plan.Version; v != nil {
				mapped.Version = &ledger.VersionFilter{}
				if v.Eq != nil {
					mapped.Version.Eq = lo.ToPtr(int(*v.Eq))
				}

				if v.Gte != nil {
					mapped.Version.Gte = lo.ToPtr(int(*v.Gte))
				}

				if v.Lte != nil {
					mapped.Version.Lte = lo.ToPtr(int(*v.Lte))
				}

				if v.Oeq != nil {
					mapped.Version.In = make([]int, len(*v.Oeq))
					for i, n := range *v.Oeq {
						mapped.Version.In[i] = int(n)
					}
				}
			}

			return mapped
		})
	}

	return result, nil
}

func fromAPIBillingCreditGrantStatus(status apiv3models.BillingCreditGrantStatus) (creditgrant.GrantStatus, error) {
	switch status {
	case apiv3models.BillingCreditGrantStatusActive:
		return creditgrant.GrantStatusActive, nil
	case apiv3models.BillingCreditGrantStatusPending:
		return creditgrant.GrantStatusPending, nil
	case apiv3models.BillingCreditGrantStatusVoided:
		return creditgrant.GrantStatusVoided, nil
	case apiv3models.BillingCreditGrantStatusExpired:
		return creditgrant.GrantStatusExpired, nil
	default:
		return "", fmt.Errorf("unsupported credit grant status: %s", status)
	}
}

type billingCreditTransactionCursorPayload struct {
	BookedAt  time.Time `json:"booked_at"`
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func encodeBillingCreditTransactionCursor(cursor ledger.TransactionCursor) (string, error) {
	payload := billingCreditTransactionCursorPayload{
		BookedAt:  cursor.BookedAt,
		CreatedAt: cursor.CreatedAt,
		ID:        cursor.ID.ID,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode cursor payload: %w", err)
	}

	return base64.StdEncoding.EncodeToString(raw), nil
}

func decodeBillingCreditTransactionCursor(cursor string, namespace string) (*ledger.TransactionCursor, error) {
	raw, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	var payload billingCreditTransactionCursorPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode cursor payload: %w", err)
	}

	ledgerCursor := ledger.TransactionCursor{
		BookedAt:  payload.BookedAt,
		CreatedAt: payload.CreatedAt,
		ID: models.NamespacedID{
			Namespace: namespace,
			ID:        payload.ID,
		},
	}

	if err := ledgerCursor.Validate(); err != nil {
		return nil, fmt.Errorf("invalid cursor value: %w", err)
	}

	return &ledgerCursor, nil
}

func fromAPICreateCreditGrantRequest(ns string, customerID apiv3models.ULID, body apiv3models.CreateCreditGrantRequest) (creditgrant.CreateInput, error) {
	amount, err := alpacadecimal.NewFromString(body.Amount)
	if err != nil {
		return creditgrant.CreateInput{}, fmt.Errorf("invalid amount: %w", err)
	}

	req := creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    customerID,
		Name:          body.Name,
		Description:   body.Description,
		Currency:      currencyx.Code(body.Currency),
		Amount:        amount,
		EffectiveAt:   body.EffectiveAt,
		FundingMethod: fromAPIBillingCreditFundingMethod(body.FundingMethod),
		Priority:      body.Priority,
		Labels:        lo.FromPtrOr(body.Labels, apiv3models.Labels{}),
		Key:           body.Key,
	}

	if body.ExpiresAfter != nil {
		expiresAfter, err := datetime.ISODurationString(*body.ExpiresAfter).Parse()
		if err != nil {
			return creditgrant.CreateInput{}, fmt.Errorf("invalid expires_after: %w", err)
		}

		req.ExpiresAfter = &expiresAfter
	}

	if body.Purchase != nil {
		purchase := &creditgrant.PurchaseTerms{
			Currency: currencyx.Code(body.Purchase.Currency),
		}

		if body.Purchase.PerUnitCostBasis != nil {
			costBasis, err := alpacadecimal.NewFromString(*body.Purchase.PerUnitCostBasis)
			if err != nil {
				return creditgrant.CreateInput{}, fmt.Errorf("invalid per_unit_cost_basis: %w", err)
			}

			purchase.PerUnitCostBasis = &costBasis
		}

		purchase.CostBasis, err = fromAPICreateChargeCostBasis(body.Purchase.CostBasis)
		if err != nil {
			return creditgrant.CreateInput{}, fmt.Errorf("invalid cost_basis: %w", err)
		}

		if body.Purchase.AvailabilityPolicy != nil {
			policy, err := fromAPIBillingCreditAvailabilityPolicy(*body.Purchase.AvailabilityPolicy)
			if err != nil {
				return creditgrant.CreateInput{}, err
			}

			purchase.AvailabilityPolicy = &policy
		}

		req.Purchase = purchase
	}

	if body.TaxConfig != nil {
		req.TaxConfig = fromAPITaxCodeConfig(body.TaxConfig)
	}

	filters, err := fromAPIBillingCreditGrantFilters(body.Filters)
	if err != nil {
		return creditgrant.CreateInput{}, fmt.Errorf("invalid filters: %w", err)
	}

	req.Filters = filters

	return req, nil
}

func fromAPICreateChargeCostBasis(in *apiv3models.CreateChargeCostBasis) (*creditpurchase.CostBasis, error) {
	if in == nil {
		return nil, nil
	}

	costBasisType, err := in.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("invalid cost basis: %w", err)
	}

	if costBasisType == string(apiv3models.BillingChargeCostBasisManualTypeManual) {
		manual, err := in.AsCreateChargeCostBasisManual()
		if err != nil {
			return nil, fmt.Errorf("invalid manual cost basis: %w", err)
		}

		rate, err := alpacadecimal.NewFromString(manual.Rate)
		if err != nil {
			return nil, fmt.Errorf("invalid cost basis rate: %w", err)
		}

		if manual.FiatCurrency == nil {
			return new(creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{
				Rate: rate,
			})), nil
		}
	}

	costBasis, err := billingcommon.FromAPIChargeCostBasis(new(apiv3models.BillingChargeCostBasis(*in)))
	if err != nil {
		return nil, err
	}

	return new(creditpurchase.NewCostBasis(*costBasis)), nil
}

func fromAPIVoidCreditGrantRequest(
	ns string,
	customerID apiv3models.ULID,
	creditGrantID apiv3models.ULID,
	body apiv3models.VoidCreditGrantRequest,
) (creditgrant.VoidInput, error) {
	paymentAdjustment, err := fromAPIBillingCreditGrantVoidPaymentAdjustment(body.PaymentAdjustment)
	if err != nil {
		return creditgrant.VoidInput{}, err
	}

	return creditgrant.VoidInput{
		Namespace:         ns,
		CustomerID:        customerID,
		ChargeID:          creditGrantID,
		PaymentAdjustment: paymentAdjustment,
	}, nil
}

func fromAPIUpdateCreditGrantExternalSettlementRequest(
	ns string,
	customerID apiv3models.ULID,
	creditGrantID apiv3models.ULID,
	body apiv3models.UpdateCreditGrantExternalSettlementRequest,
) (creditgrant.UpdateExternalSettlementInput, error) {
	targetStatus, err := fromAPIBillingCreditPurchasePaymentSettlementStatus(body.Status)
	if err != nil {
		return creditgrant.UpdateExternalSettlementInput{}, err
	}

	return creditgrant.UpdateExternalSettlementInput{
		Namespace:    ns,
		CustomerID:   customerID,
		ChargeID:     creditGrantID,
		TargetStatus: targetStatus,
	}, nil
}

func fromAPIBillingCreditGrantVoidPaymentAdjustment(adjustment *apiv3models.BillingCreditGrantVoidPaymentAdjustment) (creditgrant.VoidPaymentAdjustment, error) {
	if adjustment == nil {
		return creditgrant.VoidPaymentAdjustmentNone, nil
	}

	switch *adjustment {
	case apiv3models.BillingCreditGrantVoidPaymentAdjustmentNone:
		return creditgrant.VoidPaymentAdjustmentNone, nil
	default:
		return "", newCreditGrantVoidPaymentAdjustmentInvalid(string(*adjustment))
	}
}

func fromAPIBillingCreditPurchasePaymentSettlementStatus(status apiv3models.BillingCreditPurchasePaymentSettlementStatus) (payment.Status, error) {
	switch status {
	case apiv3models.BillingCreditPurchasePaymentSettlementStatusAuthorized:
		return payment.StatusAuthorized, nil
	case apiv3models.BillingCreditPurchasePaymentSettlementStatusSettled:
		return payment.StatusSettled, nil
	default:
		return "", newCreditGrantExternalSettlementStatusInvalid(string(status))
	}
}

func toAPICreditBalance(currency currencies.CurrencyReference, balance customerbalance.Balance) apiv3models.CreditBalance {
	result := apiv3models.CreditBalance{
		Currency:  apiv3models.BillingCurrencyCode(currency.GetCode()),
		Booked:    balance.Booked().String(),
		Settled:   balance.Booked().String(),
		Spendable: balance.Spendable().String(),
		Live:      balance.Live().String(),
		Pending:   balance.Pending().String(),
	}
	if currency.CustomCurrencyID != nil {
		result.CustomCurrency = &apiv3models.CurrencyCustomReference{Id: *currency.CustomCurrencyID}
	}

	return result
}

func fromAPIBillingCreditTransactionType(filter *apiv3models.BillingCreditTransactionType) *customerbalance.CreditTransactionType {
	if filter == nil {
		return nil
	}

	var txType customerbalance.CreditTransactionType
	switch *filter {
	case apiv3models.BillingCreditTransactionTypeFunded:
		txType = customerbalance.CreditTransactionTypeFunded
	case apiv3models.BillingCreditTransactionTypeConsumed:
		txType = customerbalance.CreditTransactionTypeConsumed
	case apiv3models.BillingCreditTransactionTypeCorrection:
		txType = customerbalance.CreditTransactionTypeCorrection
	case apiv3models.BillingCreditTransactionTypeExpired:
		txType = customerbalance.CreditTransactionTypeExpired
	case apiv3models.BillingCreditTransactionTypeVoided:
		txType = customerbalance.CreditTransactionTypeVoided
	default:
		return nil
	}

	return &txType
}

func toAPIBillingCreditTransactions(items []customerbalance.CreditTransaction) []apiv3models.BillingCreditTransaction {
	out := make([]apiv3models.BillingCreditTransaction, 0, len(items))

	for _, item := range items {
		out = append(out, toAPIBillingCreditTransaction(item))
	}

	return out
}

func toAPIBillingCreditTransaction(tx customerbalance.CreditTransaction) apiv3models.BillingCreditTransaction {
	currency := tx.CurrencyReference()
	apiTx := apiv3models.BillingCreditTransaction{
		Id:          tx.ID.ID,
		CreatedAt:   tx.CreatedAt,
		BookedAt:    tx.BookedAt,
		Type:        toAPIBillingCreditTransactionType(tx.Type),
		Currency:    apiv3models.BillingCurrencyCode(tx.Currency),
		Amount:      tx.Amount.String(),
		Name:        tx.Name,
		Description: tx.Description,
		AvailableBalance: struct {
			After  apiv3models.Numeric `json:"after"`
			Before apiv3models.Numeric `json:"before"`
		}{
			Before: tx.Balance.Before.String(),
			After:  tx.Balance.After.String(),
		},
	}
	if currency.CustomCurrencyID != nil {
		apiTx.CustomCurrency = &apiv3models.CurrencyCustomReference{Id: *currency.CustomCurrencyID}
	}

	labels := creditTransactionLabels(tx.Annotations)
	if tx.Type == customerbalance.CreditTransactionTypeFunded && tx.GrantVoided {
		labels["voided"] = "true"
	}

	if len(labels) > 0 {
		apiLabels := apiv3models.Labels(labels)
		apiTx.Labels = &apiLabels
	}

	return apiTx
}

func toAPIBillingCreditTransactionType(txType customerbalance.CreditTransactionType) apiv3models.BillingCreditTransactionType {
	switch txType {
	case customerbalance.CreditTransactionTypeFunded:
		return apiv3models.BillingCreditTransactionTypeFunded
	case customerbalance.CreditTransactionTypeCorrection:
		return apiv3models.BillingCreditTransactionTypeCorrection
	case customerbalance.CreditTransactionTypeExpired:
		return apiv3models.BillingCreditTransactionTypeExpired
	case customerbalance.CreditTransactionTypeVoided:
		return apiv3models.BillingCreditTransactionTypeVoided
	default:
		return apiv3models.BillingCreditTransactionTypeConsumed
	}
}

func creditTransactionLabels(annotations models.Annotations) map[string]string {
	labels := make(map[string]string)

	setLabel := func(key, annotationKey string) {
		value := stringAnnotation(annotations, annotationKey)
		if value != "" {
			labels[key] = value
		}
	}

	setLabel("charge_id", ledger.AnnotationChargeID)
	setLabel("subscription_id", ledger.AnnotationSubscriptionID)
	setLabel("subscription_phase_id", ledger.AnnotationSubscriptionPhaseID)
	setLabel("subscription_item_id", ledger.AnnotationSubscriptionItemID)
	setLabel("feature_id", ledger.AnnotationFeatureID)

	return labels
}

func stringAnnotation(annotations models.Annotations, key string) string {
	raw, ok := annotations[key]
	if !ok {
		return ""
	}

	value, ok := raw.(string)
	if !ok {
		return ""
	}

	return value
}
