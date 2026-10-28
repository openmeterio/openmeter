package customerscredits

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/handlers/billingcommon"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/api/v3/request"
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
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func toAPIBillingCreditGrant(charge creditpurchase.Charge) (api.BillingCreditGrant, error) {
	grant := api.BillingCreditGrant{
		Id:            charge.ID,
		Customer:      api.CustomerReference{Id: charge.Intent.CustomerID},
		Name:          charge.Intent.Name,
		Description:   charge.Intent.Description,
		Amount:        charge.Intent.CreditAmount.String(),
		Currency:      api.BillingCurrencyCode(charge.Intent.Currency.GetCode()),
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
		grant.CustomCurrency = &api.CurrencyCustomReference{Id: charge.Intent.Currency.ID}
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
	if charge.State.VoidedAt != nil {
		return api.BillingCreditGrantStatusVoided
	}

	if charge.Intent.ExpiresAt != nil && !charge.Intent.ExpiresAt.After(clock.Now()) {
		return api.BillingCreditGrantStatusExpired
	}

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

// toAPICreditGrantPurchase builds the purchase block for funded grants (invoice or external).
// Returns nil for promotional grants (funding_method=none). Amount and the resolved
// cost basis are omitted while a dynamic cost basis is still unresolved.
func toAPICreditGrantPurchase(charge creditpurchase.Charge) (*api.BillingCreditGrantPurchase, error) {
	settlement := charge.Intent.Settlement

	var availabilityPolicy *api.BillingCreditAvailabilityPolicy
	settlementStatus := api.BillingCreditPurchasePaymentSettlementStatusPending

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

	purchase := &api.BillingCreditGrantPurchase{
		Currency:           api.CurrencyCode(fiatCurrency.GetFiatCode()),
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

func toAPITaxCodeConfig(charge creditpurchase.Charge) *api.TaxCodeConfig {
	cfg := charge.Intent.TaxConfig
	if lo.IsEmpty(cfg) {
		return nil
	}

	tc := &api.TaxCodeConfig{}

	if cfg.Behavior != nil {
		behavior := api.BillingTaxBehavior(*cfg.Behavior)
		tc.Behavior = &behavior
	}

	if cfg.TaxCodeID != "" {
		tc.Code = &api.TaxCodeReference{Id: cfg.TaxCodeID}
	}

	return tc
}

func toAPIBillingCreditGrantFilters(filters ledger.CreditFilters) *api.BillingCreditGrantFilters {
	if filters.IsEmpty() {
		return nil
	}

	filters = filters.Normalize()
	result := &api.BillingCreditGrantFilters{}
	if len(filters.Features) > 0 {
		result.Features = lo.ToPtr(filters.Features)
	}

	if len(filters.Plans) > 0 {
		result.Plans = lo.ToPtr(lo.Map(filters.Plans, func(plan ledger.PlanFilter, _ int) api.BillingCreditGrantPlanFilter {
			mapped := api.BillingCreditGrantPlanFilter{Key: plan.Key}
			if v := plan.Version; v != nil {
				mapped.Version = &api.VersionFilter{}
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

func fromAPITaxCodeConfig(tc *api.CreateTaxCodeConfig) *productcatalog.TaxConfig {
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

func fromAPIBillingCreditGrantFilters(filters *api.CreateCreditGrantFilters) (*creditgrant.GrantFilters, error) {
	if filters == nil {
		return nil, nil
	}

	result := &creditgrant.GrantFilters{Features: lo.FromPtr(filters.Features)}
	if filters.Plans != nil {
		result.Plans = lo.Map(*filters.Plans, func(plan api.CreateCreditGrantPlanFilter, _ int) ledger.PlanFilter {
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

func fromAPIBillingCreditGrantStatus(status api.BillingCreditGrantStatus) (creditgrant.GrantStatus, error) {
	switch status {
	case api.BillingCreditGrantStatusActive:
		return creditgrant.GrantStatusActive, nil
	case api.BillingCreditGrantStatusPending:
		return creditgrant.GrantStatusPending, nil
	case api.BillingCreditGrantStatusVoided:
		return creditgrant.GrantStatusVoided, nil
	case api.BillingCreditGrantStatusExpired:
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
		EffectiveAt:   body.EffectiveAt,
		FundingMethod: fromAPIBillingCreditFundingMethod(body.FundingMethod),
		Priority:      body.Priority,
		Labels:        lo.FromPtrOr(body.Labels, api.Labels{}),
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

func fromAPICreateChargeCostBasis(in *api.CreateChargeCostBasis) (*creditpurchase.CostBasis, error) {
	if in == nil {
		return nil, nil
	}

	costBasisType, err := in.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("invalid cost basis: %w", err)
	}

	if costBasisType == string(api.BillingChargeCostBasisManualTypeManual) {
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

	costBasis, err := billingcommon.FromAPIChargeCostBasis(new(api.BillingChargeCostBasis(*in)))
	if err != nil {
		return nil, err
	}

	return new(creditpurchase.NewCostBasis(*costBasis)), nil
}

func fromAPIVoidCreditGrantRequest(
	ns string,
	customerID api.ULID,
	creditGrantID api.ULID,
	body api.VoidCreditGrantRequest,
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

func fromAPIBillingCreditGrantVoidPaymentAdjustment(adjustment *api.BillingCreditGrantVoidPaymentAdjustment) (creditgrant.VoidPaymentAdjustment, error) {
	if adjustment == nil {
		return creditgrant.VoidPaymentAdjustmentNone, nil
	}

	switch *adjustment {
	case api.BillingCreditGrantVoidPaymentAdjustmentNone:
		return creditgrant.VoidPaymentAdjustmentNone, nil
	default:
		return "", newCreditGrantVoidPaymentAdjustmentInvalid(string(*adjustment))
	}
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

func toAPICreditBalance(currency currencies.CurrencyReference, balance customerbalance.Balance) api.CreditBalance {
	result := api.CreditBalance{
		Currency:  api.BillingCurrencyCode(currency.GetCode()),
		Booked:    balance.Booked().String(),
		Settled:   balance.Booked().String(),
		Spendable: balance.Spendable().String(),
		Live:      balance.Live().String(),
		Pending:   balance.Pending().String(),
	}
	if currency.CustomCurrencyID != nil {
		result.CustomCurrency = &api.CurrencyCustomReference{Id: *currency.CustomCurrencyID}
	}

	return result
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
	case api.BillingCreditTransactionTypeExpired:
		txType = customerbalance.CreditTransactionTypeExpired
	case api.BillingCreditTransactionTypeVoided:
		txType = customerbalance.CreditTransactionTypeVoided
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
	currency := tx.CurrencyReference()
	apiTx := api.BillingCreditTransaction{
		Id:          tx.ID.ID,
		CreatedAt:   tx.CreatedAt,
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
	if currency.CustomCurrencyID != nil {
		apiTx.CustomCurrency = &api.CurrencyCustomReference{Id: *currency.CustomCurrencyID}
	}

	labels := creditTransactionLabels(tx.Annotations)
	if tx.Type == customerbalance.CreditTransactionTypeFunded && tx.GrantVoided {
		labels["voided"] = "true"
	}

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
	case customerbalance.CreditTransactionTypeExpired:
		return api.BillingCreditTransactionTypeExpired
	case customerbalance.CreditTransactionTypeVoided:
		return api.BillingCreditTransactionTypeVoided
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

// fromAPIListCreditGrantsParams maps the query parameters shared by the per-customer
// and namespace-wide grant listings. Callers set the namespace and customer scope.
func fromAPIListCreditGrantsParams(ctx context.Context, params api.ListCreditGrantsParams) (ListCreditGrantsRequest, error) {
	apiPage, apiSort, apiFilter := params.Page, params.Sort, params.Filter

	page := pagination.NewPage(1, 20)
	if apiPage != nil {
		page = pagination.NewPage(
			lo.FromPtrOr(apiPage.Number, 1),
			lo.FromPtrOr(apiPage.Size, 20),
		)
	}

	if err := page.Validate(); err != nil {
		return ListCreditGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "page",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}

	req := ListCreditGrantsRequest{
		Page:    page,
		OrderBy: creditpurchase.OrderByCreatedAt,
		Order:   sortx.OrderDesc,
	}

	if apiSort != nil {
		sort, err := request.ParseSortBy(*apiSort)
		if err != nil {
			return ListCreditGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
				{Field: "sort", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
			})
		}

		req.OrderBy = creditpurchase.OrderBy(sort.Field)
		if err := req.OrderBy.Validate(); err != nil {
			return ListCreditGrantsRequest{}, apierrors.NewUnsupportedSortFieldError(ctx, sort.Field, "created_at", "updated_at", "name", "id")
		}

		req.Order = sort.Order.ToSortxOrder()
	}

	if apiFilter == nil {
		return req, nil
	}

	if apiFilter.Status != nil {
		status, err := fromAPIBillingCreditGrantStatus(*apiFilter.Status)
		if err != nil {
			return ListCreditGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
				{
					Field:  "filter[status]",
					Reason: err.Error(),
					Source: apierrors.InvalidParamSourceQuery,
				},
			})
		}

		req.Status = &status
	}

	if apiFilter.Currency != nil {
		currency := currencyx.Code(*apiFilter.Currency)
		req.Currency = &currency
	}

	key, err := filters.FromAPIFilterString(apiFilter.Key)
	if err != nil {
		return ListCreditGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "filter[key]",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}

	req.Key = key

	req.UpdatedAt, err = filters.FromAPIFilterDateTime(apiFilter.UpdatedAt)
	if err != nil {
		return ListCreditGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "filter[updated_at]",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}

	req.FeatureFilters, err = fromAPICreditGrantFeatureFilters(apiFilter.FeatureKey)
	if err != nil {
		return ListCreditGrantsRequest{}, newFeatureKeyFilterBadRequest(ctx, err)
	}

	return req, nil
}

// fromAPICreditBalanceCurrencyFilter maps the balance currency filter. Only positive
// selection is supported: a balance listing cannot exclude a currency it discovers.
func fromAPICreditBalanceCurrencyFilter(ctx context.Context, f *api.StringFieldFilterExact) (customerbalance.CurrencyFilter, error) {
	if f == nil {
		return customerbalance.CurrencyFilter{}, nil
	}

	if f.Neq != nil {
		return customerbalance.CurrencyFilter{}, apierrors.NewBadRequestError(
			ctx,
			errors.New("neq operator is not supported for currency"),
			apierrors.InvalidParameters{{
				Field:  "filter[currency]",
				Reason: "neq operator is not supported",
				Source: apierrors.InvalidParamSourceQuery,
			}},
		)
	}

	codes := make([]currencyx.Code, 0, 1+len(f.Oeq))
	if f.Eq != nil {
		codes = append(codes, currencyx.Code(*f.Eq))
	}

	for _, v := range f.Oeq {
		codes = append(codes, currencyx.Code(v))
	}

	if len(codes) == 0 {
		return customerbalance.CurrencyFilter{}, nil
	}

	return customerbalance.CurrencyFilter{Codes: codes}, nil
}

// fromAPICustomerIDFilter maps a customer ID filter of a namespace-wide credit
// listing to an explicit customer set. An empty result selects every customer.
func fromAPICustomerIDFilter(ctx context.Context, f *api.ULIDFieldFilter) ([]string, error) {
	if f == nil {
		return nil, nil
	}

	badRequest := func(err error) error {
		return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{Field: "filter[customer_id]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
		})
	}

	if f.Neq != nil {
		return nil, badRequest(errors.New("neq operator is not supported for customer_id"))
	}

	ids := slices.Clone(f.Oeq)
	if f.Eq != nil {
		ids = append(ids, *f.Eq)
	}

	// A present filter that selects no customer must not fall through to an empty
	// selection, which would widen the listing to every customer.
	if len(ids) == 0 || lo.Contains(ids, "") {
		return nil, badRequest(errors.New("customer_id must not be empty"))
	}

	return lo.Uniq(ids), nil
}
