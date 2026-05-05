package customerscredits

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func toAPIBillingCreditGrant(charge creditpurchase.Charge) (api.BillingCreditGrant, error) {
	grant := api.BillingCreditGrant{
		Id:            charge.ID,
		Name:          charge.Intent.Name,
		Description:   charge.Intent.Description,
		Amount:        charge.Intent.CreditAmount.String(),
		Currency:      api.BillingCurrencyCode(charge.Intent.Currency),
		FundingMethod: toAPIBillingCreditFundingMethod(charge.Intent.Settlement),
		Status:        toAPIBillingCreditGrantStatus(charge),
		CreatedAt:     lo.ToPtr(charge.CreatedAt),
		UpdatedAt:     lo.ToPtr(charge.UpdatedAt),
		DeletedAt:     charge.DeletedAt,
		Labels:        labels.FromMetadata(charge.Intent.Metadata),
	}

	if charge.Intent.Priority != nil {
		p := int16(*charge.Intent.Priority)
		grant.Priority = &p
	}

	purchase, err := toAPICreditGrantPurchase(charge)
	if err != nil {
		return grant, fmt.Errorf("converting purchase: %w", err)
	}
	grant.Purchase = purchase
	grant.TaxConfig = toAPIBillingCreditGrantTaxConfig(charge)

	return grant, nil
}

func toAPIBillingCreditFundingMethod(settlement creditpurchase.Settlement) api.BillingCreditFundingMethod {
	switch settlement.Type() {
	case creditpurchase.SettlementTypeInvoice:
		return api.BillingCreditFundingMethodInvoice
	case creditpurchase.SettlementTypeExternal:
		return api.BillingCreditFundingMethodExternal
	default:
		return api.BillingCreditFundingMethodNone
	}
}

func toAPIBillingCreditGrantStatus(charge creditpurchase.Charge) api.BillingCreditGrantStatus {
	switch charge.Status {
	case creditpurchase.StatusActive, creditpurchase.StatusFinal:
		return api.BillingCreditGrantStatusActive
	case creditpurchase.StatusCreated:
		return api.BillingCreditGrantStatusPending
	case creditpurchase.StatusDeleted:
		return api.BillingCreditGrantStatusVoided
	default:
		return api.BillingCreditGrantStatusActive
	}
}

type creditGrantPurchase = struct {
	Amount             api.Numeric                                       `json:"amount"`
	AvailabilityPolicy *api.BillingCreditAvailabilityPolicy              `json:"availability_policy,omitempty"`
	Currency           api.CurrencyCode                                  `json:"currency"`
	PerUnitCostBasis   *api.Numeric                                      `json:"per_unit_cost_basis,omitempty"`
	SettlementStatus   *api.BillingCreditPurchasePaymentSettlementStatus `json:"settlement_status,omitempty"`
}

// toAPICreditGrantPurchase builds the purchase block for funded grants (invoice or external).
// Returns nil for promotional grants (funding_method=none).
func toAPICreditGrantPurchase(charge creditpurchase.Charge) (*creditGrantPurchase, error) {
	settlement := charge.Intent.Settlement

	switch settlement.Type() {
	case creditpurchase.SettlementTypeInvoice:
		inv, err := settlement.AsInvoiceSettlement()
		if err != nil {
			return nil, fmt.Errorf("getting invoice settlement: %w", err)
		}

		costBasis := inv.CostBasis.String()
		currencyCalculator, err := charge.Intent.Currency.Calculator()
		if err != nil {
			return nil, fmt.Errorf("getting currency calculator: %w", err)
		}
		purchaseAmount := currencyCalculator.RoundToPrecision(charge.Intent.CreditAmount.Mul(inv.CostBasis))
		settlementStatus := api.BillingCreditPurchasePaymentSettlementStatusPending

		if charge.Realizations.InvoiceSettlement != nil {
			settlementStatus = toAPIBillingCreditPurchasePaymentSettlementStatus(charge.Realizations.InvoiceSettlement.Status)
		}

		return &creditGrantPurchase{
			Amount:           purchaseAmount.String(),
			Currency:         api.CurrencyCode(inv.Currency),
			PerUnitCostBasis: &costBasis,
			SettlementStatus: &settlementStatus,
		}, nil

	case creditpurchase.SettlementTypeExternal:
		ext, err := settlement.AsExternalSettlement()
		if err != nil {
			return nil, fmt.Errorf("getting external settlement: %w", err)
		}

		costBasis := ext.CostBasis.String()
		currencyCalculator, err := charge.Intent.Currency.Calculator()
		if err != nil {
			return nil, fmt.Errorf("getting currency calculator: %w", err)
		}
		purchaseAmount := currencyCalculator.RoundToPrecision(charge.Intent.CreditAmount.Mul(ext.CostBasis))
		availPolicy, err := toAPIBillingCreditAvailabilityPolicy(ext.InitialStatus)
		if err != nil {
			return nil, fmt.Errorf("converting availability policy: %w", err)
		}
		settlementStatus := api.BillingCreditPurchasePaymentSettlementStatusPending

		if charge.Realizations.ExternalPaymentSettlement != nil {
			settlementStatus = toAPIBillingCreditPurchasePaymentSettlementStatus(charge.Realizations.ExternalPaymentSettlement.Status)
		}

		return &creditGrantPurchase{
			Amount:             purchaseAmount.String(),
			Currency:           api.CurrencyCode(ext.Currency),
			PerUnitCostBasis:   &costBasis,
			AvailabilityPolicy: &availPolicy,
			SettlementStatus:   &settlementStatus,
		}, nil

	case creditpurchase.SettlementTypePromotional:
		return nil, nil

	default:
		return nil, fmt.Errorf("invalid settlement type: %s", settlement.Type())
	}
}

func toAPIBillingCreditPurchasePaymentSettlementStatus(status payment.Status) api.BillingCreditPurchasePaymentSettlementStatus {
	switch status {
	case payment.StatusAuthorized:
		return api.BillingCreditPurchasePaymentSettlementStatusAuthorized
	case payment.StatusSettled:
		return api.BillingCreditPurchasePaymentSettlementStatusSettled
	default:
		return api.BillingCreditPurchasePaymentSettlementStatusPending
	}
}

func toAPIBillingCreditAvailabilityPolicy(status creditpurchase.InitialPaymentSettlementStatus) (api.BillingCreditAvailabilityPolicy, error) {
	switch status {
	case creditpurchase.CreatedInitialPaymentSettlementStatus:
		return api.BillingCreditAvailabilityPolicyOnCreation, nil
	default:
		return "", fmt.Errorf("invalid availability policy: %s", status)
	}
}

func toAPIBillingCreditGrantTaxConfig(charge creditpurchase.Charge) *api.BillingCreditGrantTaxConfig {
	if charge.Intent.TaxConfig == nil {
		return nil
	}

	tc := &api.BillingCreditGrantTaxConfig{}

	if charge.Intent.TaxConfig.Behavior != nil {
		behavior := api.BillingTaxBehavior(*charge.Intent.TaxConfig.Behavior)
		tc.Behavior = &behavior
	}

	if charge.Intent.TaxConfig.TaxCodeID != nil {
		tc.TaxCode = &api.BillingTaxCodeReference{Id: *charge.Intent.TaxConfig.TaxCodeID}
	}

	return tc
}

func fromAPIBillingCreditFundingMethod(fm api.BillingCreditFundingMethod) creditgrant.FundingMethod {
	switch fm {
	case api.BillingCreditFundingMethodInvoice:
		return creditgrant.FundingMethodInvoice
	case api.BillingCreditFundingMethodExternal:
		return creditgrant.FundingMethodExternal
	default:
		return creditgrant.FundingMethodNone
	}
}

func fromAPIBillingCreditAvailabilityPolicy(policy api.BillingCreditAvailabilityPolicy) (creditpurchase.InitialPaymentSettlementStatus, error) {
	switch policy {
	case api.BillingCreditAvailabilityPolicyOnCreation:
		return creditpurchase.CreatedInitialPaymentSettlementStatus, nil
	default:
		return "", models.NewGenericValidationError(fmt.Errorf("invalid availability policy: %s", policy))
	}
}

func fromAPIBillingCreditGrantTaxConfig(tc *api.BillingCreditGrantTaxConfig) *productcatalog.TaxConfig {
	if tc == nil {
		return nil
	}

	config := &productcatalog.TaxConfig{}

	if tc.Behavior != nil {
		behavior := productcatalog.TaxBehavior(*tc.Behavior)
		config.Behavior = &behavior
	}

	if tc.TaxCode != nil {
		config.TaxCodeID = &tc.TaxCode.Id
	}

	return config
}

func fromAPIBillingCreditGrantStatus(status api.BillingCreditGrantStatus) (meta.ChargeStatus, error) {
	switch status {
	case api.BillingCreditGrantStatusActive:
		return meta.ChargeStatusActive, nil
	case api.BillingCreditGrantStatusPending:
		return meta.ChargeStatusCreated, nil
	case api.BillingCreditGrantStatusVoided:
		return meta.ChargeStatusDeleted, nil
	case api.BillingCreditGrantStatusExpired:
		// Expired maps to final (terminal state, no further actions).
		return meta.ChargeStatusFinal, nil
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

func fromAPICreateCreditGrantRequest(ns string, customerID api.ULID, body api.CreateCreditGrantRequest) (creditgrant.CreateInput, error) {
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
		FundingMethod: fromAPIBillingCreditFundingMethod(body.FundingMethod),
		Priority:      body.Priority,
		Labels:        lo.FromPtrOr(body.Labels, api.Labels{}),
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
		req.TaxConfig = fromAPIBillingCreditGrantTaxConfig(body.TaxConfig)
	}

	if body.Filters != nil && body.Filters.Features != nil && len(*body.Filters.Features) > 0 {
		return creditgrant.CreateInput{}, fmt.Errorf("feature filters are not yet supported for credit grants")
	}

	return req, nil
}

func fromAPIUpdateCreditGrantExternalSettlementRequest(
	ns string,
	customerID api.ULID,
	creditGrantID api.ULID,
	body api.UpdateCreditGrantExternalSettlementRequest,
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

func fromAPIBillingCreditPurchasePaymentSettlementStatus(status api.BillingCreditPurchasePaymentSettlementStatus) (payment.Status, error) {
	switch status {
	case api.BillingCreditPurchasePaymentSettlementStatusAuthorized:
		return payment.StatusAuthorized, nil
	case api.BillingCreditPurchasePaymentSettlementStatusSettled:
		return payment.StatusSettled, nil
	default:
		return "", newCreditGrantExternalSettlementStatusInvalid(string(status))
	}
}

func toAPICreditBalance(currency currencyx.Code, balance ledger.Balance) api.CreditBalance {
	// Temporary mapping while the v3 credit-balance schema still predates the
	// customerbalance service's settled/live-pending semantics.
	return api.CreditBalance{
		Currency:  api.BillingCurrencyCode(currency),
		Available: balance.Settled().String(),
		Pending:   balance.Pending().String(),
	}
}

func fromAPIBillingCreditTransactionType(filter *api.BillingCreditTransactionType) *customerbalance.CreditTransactionType {
	if filter == nil {
		return nil
	}

	var txType customerbalance.CreditTransactionType
	switch *filter {
	case api.BillingCreditTransactionTypeFunded:
		txType = customerbalance.CreditTransactionTypeFunded
	case api.BillingCreditTransactionTypeConsumed:
		txType = customerbalance.CreditTransactionTypeConsumed
	default:
		return nil
	}

	return &txType
}

func toAPIBillingCreditTransactions(items []customerbalance.CreditTransaction) []api.BillingCreditTransaction {
	out := make([]api.BillingCreditTransaction, 0, len(items))

	for _, item := range items {
		out = append(out, toAPIBillingCreditTransaction(item))
	}

	return out
}

func toAPIBillingCreditTransaction(tx customerbalance.CreditTransaction) api.BillingCreditTransaction {
	apiTx := api.BillingCreditTransaction{
		Id:          tx.ID.ID,
		CreatedAt:   &tx.CreatedAt,
		BookedAt:    tx.BookedAt,
		Type:        toAPIBillingCreditTransactionType(tx.Type),
		Currency:    api.BillingCurrencyCode(tx.Currency),
		Amount:      tx.Amount.String(),
		Name:        tx.Name,
		Description: tx.Description,
		AvailableBalance: struct {
			After  api.Numeric `json:"after"`
			Before api.Numeric `json:"before"`
		}{
			Before: tx.Balance.Before.String(),
			After:  tx.Balance.After.String(),
		},
	}

	labels := creditTransactionLabels(tx.Annotations)
	if len(labels) > 0 {
		apiLabels := api.Labels(labels)
		apiTx.Labels = &apiLabels
	}

	return apiTx
}

func toAPIBillingCreditTransactionType(txType customerbalance.CreditTransactionType) api.BillingCreditTransactionType {
	switch txType {
	case customerbalance.CreditTransactionTypeFunded:
		return api.BillingCreditTransactionTypeFunded
	default:
		return api.BillingCreditTransactionTypeConsumed
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
