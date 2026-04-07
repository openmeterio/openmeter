package customerscredits

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func convertCreditGrant(charge creditpurchase.Charge) api.BillingCreditGrant {
	grant := api.BillingCreditGrant{
		Id:            charge.ID,
		Name:          charge.Intent.Name,
		Description:   charge.Intent.Description,
		Amount:        charge.Intent.CreditAmount.String(),
		Currency:      api.BillingCurrencyCode(charge.Intent.Currency),
		FundingMethod: convertFundingMethod(charge.Intent.Settlement),
		Status:        convertGrantStatus(charge),
		CreatedAt:     lo.ToPtr(charge.CreatedAt),
		UpdatedAt:     lo.ToPtr(charge.UpdatedAt),
		DeletedAt:     charge.DeletedAt,
		Labels:        ConvertMetadataToLabels(&charge.Intent.Metadata),
	}

	if charge.Intent.Priority != nil {
		p := int16(*charge.Intent.Priority)
		grant.Priority = &p
	}

	grant.Purchase = convertPurchase(charge)
	grant.TaxConfig = convertTaxConfig(charge)

	return grant
}

func convertFundingMethod(settlement creditpurchase.Settlement) api.BillingCreditFundingMethod {
	switch settlement.Type() {
	case creditpurchase.SettlementTypeInvoice:
		return api.BillingCreditFundingMethodInvoice
	case creditpurchase.SettlementTypeExternal:
		return api.BillingCreditFundingMethodExternal
	default:
		return api.BillingCreditFundingMethodNone
	}
}

func convertGrantStatus(charge creditpurchase.Charge) api.BillingCreditGrantStatus {
	switch charge.Status {
	case "active":
		return api.BillingCreditGrantStatusActive
	case "created":
		return api.BillingCreditGrantStatusPending
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

// convertPurchase builds the purchase block for funded grants (invoice or external).
// Returns nil for promotional grants (funding_method=none).
func convertPurchase(charge creditpurchase.Charge) *creditGrantPurchase {
	settlement := charge.Intent.Settlement

	switch settlement.Type() {
	case creditpurchase.SettlementTypeInvoice:
		inv, err := settlement.AsInvoiceSettlement()
		if err != nil {
			return nil
		}

		costBasis := inv.CostBasis.String()
		purchaseAmount := charge.Intent.CreditAmount.Mul(inv.CostBasis).String()
		settlementStatus := api.BillingCreditPurchasePaymentSettlementStatusPending

		if charge.State.InvoiceSettlement != nil {
			settlementStatus = convertPaymentStatus(charge.State.InvoiceSettlement.Status)
		}

		return &creditGrantPurchase{
			Amount:           purchaseAmount,
			Currency:         api.CurrencyCode(inv.Currency),
			PerUnitCostBasis: &costBasis,
			SettlementStatus: &settlementStatus,
		}

	case creditpurchase.SettlementTypeExternal:
		ext, err := settlement.AsExternalSettlement()
		if err != nil {
			return nil
		}

		costBasis := ext.CostBasis.String()
		purchaseAmount := charge.Intent.CreditAmount.Mul(ext.CostBasis).String()
		availPolicy, err := convertAvailabilityPolicy(ext.InitialStatus)
		if err != nil {
			return nil
		}
		settlementStatus := api.BillingCreditPurchasePaymentSettlementStatusPending

		if charge.State.ExternalPaymentSettlement != nil {
			settlementStatus = convertPaymentStatus(charge.State.ExternalPaymentSettlement.Status)
		}

		return &creditGrantPurchase{
			Amount:             purchaseAmount,
			Currency:           api.CurrencyCode(ext.Currency),
			PerUnitCostBasis:   &costBasis,
			AvailabilityPolicy: &availPolicy,
			SettlementStatus:   &settlementStatus,
		}

	default:
		return nil
	}
}

func convertPaymentStatus(status payment.Status) api.BillingCreditPurchasePaymentSettlementStatus {
	switch status {
	case payment.StatusAuthorized:
		return api.BillingCreditPurchasePaymentSettlementStatusAuthorized
	case payment.StatusSettled:
		return api.BillingCreditPurchasePaymentSettlementStatusSettled
	default:
		return api.BillingCreditPurchasePaymentSettlementStatusPending
	}
}

func convertAvailabilityPolicy(status creditpurchase.InitialPaymentSettlementStatus) (api.BillingCreditAvailabilityPolicy, error) {
	switch status {
	case creditpurchase.CreatedInitialPaymentSettlementStatus:
		return api.BillingCreditAvailabilityPolicyOnCreation, nil
	default:
		return "", fmt.Errorf("invalid availability policy: %s", status)
	}
}

func convertTaxConfig(charge creditpurchase.Charge) *api.BillingCreditGrantTaxConfig {
	if charge.Intent.TaxConfig == nil {
		return nil
	}

	tc := &api.BillingCreditGrantTaxConfig{}

	if charge.Intent.TaxConfig.Behavior != nil {
		behavior := api.BillingTaxBehavior(*charge.Intent.TaxConfig.Behavior)
		tc.Behavior = &behavior
	}

	return tc
}

// ConvertMetadataToLabels converts models.Metadata to api.Labels.
// Always returns an initialized map (never nil) so JSON serializes to {} instead of null.
func ConvertMetadataToLabels(source *models.Metadata) *api.Labels {
	labels := make(api.Labels)
	if source != nil {
		for k, v := range *source {
			labels[k] = v
		}
	}
	return &labels
}

func convertAPIFundingMethod(fm api.BillingCreditFundingMethod) creditgrant.FundingMethod {
	switch fm {
	case api.BillingCreditFundingMethodInvoice:
		return creditgrant.FundingMethodInvoice
	case api.BillingCreditFundingMethodExternal:
		return creditgrant.FundingMethodExternal
	default:
		return creditgrant.FundingMethodNone
	}
}

func convertAPIAvailabilityPolicy(policy api.BillingCreditAvailabilityPolicy) (creditpurchase.InitialPaymentSettlementStatus, error) {
	switch policy {
	case api.BillingCreditAvailabilityPolicyOnCreation:
		return creditpurchase.CreatedInitialPaymentSettlementStatus, nil
	default:
		return "", models.NewGenericValidationError(fmt.Errorf("invalid availability policy: %s", policy))
	}
}

func convertAPITaxConfig(tc *api.BillingCreditGrantTaxConfig) *productcatalog.TaxConfig {
	if tc == nil {
		return nil
	}

	config := &productcatalog.TaxConfig{}

	if tc.Behavior != nil {
		behavior := productcatalog.TaxBehavior(*tc.Behavior)
		config.Behavior = &behavior
	}

	return config
}

func convertAPIStatusToChargeStatus(status api.BillingCreditGrantStatus) meta.ChargeStatus {
	switch status {
	case api.BillingCreditGrantStatusActive:
		return meta.ChargeStatusActive
	case api.BillingCreditGrantStatusPending:
		return meta.ChargeStatusCreated
	default:
		return meta.ChargeStatus(status)
	}
}

func convertAPICreateCreditGrantRequest(ns string, customerID api.ULID, body api.CreateCreditGrantRequest) (creditgrant.CreateInput, error) {
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
		FundingMethod: convertAPIFundingMethod(body.FundingMethod),
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
			policy, err := convertAPIAvailabilityPolicy(*body.Purchase.AvailabilityPolicy)
			if err != nil {
				return creditgrant.CreateInput{}, err
			}
			purchase.AvailabilityPolicy = &policy
		}

		req.Purchase = purchase
	}

	if body.TaxConfig != nil {
		req.TaxConfig = convertAPITaxConfig(body.TaxConfig)
	}

	if body.Filters != nil && body.Filters.Features != nil && len(*body.Filters.Features) > 0 {
		return creditgrant.CreateInput{}, fmt.Errorf("feature filters are not yet supported for credit grants")
	}

	return req, nil
}

func convertBalance(currency currencyx.Code, balance ledger.Balance) api.CreditBalance {
	// Temporary mapping while the v3 credit-balance schema still predates the
	// customerbalance service's settled/live-pending semantics.
	return api.CreditBalance{
		Currency:  api.BillingCurrencyCode(currency),
		Available: balance.Settled().String(),
		Pending:   balance.Pending().String(),
	}
}
