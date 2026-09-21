package chargeadapter

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	chargecreditpurchase "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type creditPurchaseIssuance struct {
	charge            chargecreditpurchase.Charge
	costBasis         *alpacadecimal.Decimal
	costBasisCurrency *currencyx.Code
	backfillPlan      advance.BackfillPlan
}

func (i creditPurchaseIssuance) buildTemplates() ([]transactions.TransactionTemplate, error) {
	charge := i.charge
	featureFilters := charge.Intent.FeatureFilters.Normalize()

	issuableAmount := charge.Intent.CreditAmount.Sub(i.backfillPlan.Amount)
	if issuableAmount.IsNegative() {
		issuableAmount = alpacadecimal.Zero
	}

	templates := i.backfillPlan.Templates

	if issuableAmount.IsPositive() {
		templates = append(templates, transactions.IssueCustomerReceivableTemplate{
			At:                charge.Intent.ServicePeriod.To,
			Amount:            issuableAmount,
			Currency:          charge.Intent.Currency.Reference(),
			CostBasisCurrency: i.costBasisCurrency,
			CostBasis:         i.costBasis,
			Features:          featureFilters,
			SourceChargeID:    &charge.ID,
			CreditPriority:    charge.Intent.Priority,
		})
	}

	switch charge.Intent.Settlement.Type() {
	case chargecreditpurchase.SettlementTypePromotional:
		// Promotional grants settle immediately through wash so the credited FBO balance
		// does not leave an unsettled receivable behind.
		templates = append(templates,
			transactions.AuthorizeCustomerReceivablePaymentTemplate{
				At:             charge.Intent.ServicePeriod.To,
				Amount:         charge.Intent.CreditAmount,
				Currency:       charge.Intent.Currency.Reference(),
				CostBasis:      i.costBasis,
				Features:       featureFilters,
				SourceChargeID: &charge.ID,
			},
			transactions.SettleCustomerReceivableFromPaymentTemplate{
				At:             charge.Intent.ServicePeriod.To,
				Amount:         charge.Intent.CreditAmount,
				Currency:       charge.Intent.Currency.Reference(),
				CostBasis:      i.costBasis,
				Features:       featureFilters,
				SourceChargeID: &charge.ID,
			},
		)
	case chargecreditpurchase.SettlementTypeExternal, chargecreditpurchase.SettlementTypeInvoice:
		// Deferred settlement modes are handled by later lifecycle events.
	default:
		return nil, fmt.Errorf("unsupported settlement type: %s", charge.Intent.Settlement.Type())
	}

	return templates, nil
}

func (i creditPurchaseIssuance) mapBreakageInput() *breakage.PlanIssuanceInput {
	if i.charge.Intent.ExpiresAt == nil {
		return nil
	}

	immediateReleases := make([]breakage.PlanIssuanceImmediateRelease, 0, len(i.backfillPlan.Backfills))

	for _, backfill := range i.backfillPlan.Backfills {
		if !backfill.Amount.IsPositive() {
			continue
		}

		immediateReleases = append(immediateReleases, breakage.PlanIssuanceImmediateRelease{
			Amount:             backfill.Amount,
			SpendChargeID:      backfill.SpendChargeID,
			CollectionOriginID: backfill.CollectionOriginID,
		})
	}

	return &breakage.PlanIssuanceInput{
		CustomerID:        i.charge.GetCustomerID(),
		Amount:            i.charge.Intent.CreditAmount,
		ImmediateReleases: immediateReleases,
		Currency:          i.charge.Intent.Currency.Reference(),
		CostBasisCurrency: i.costBasisCurrency,
		CostBasis:         i.costBasis,
		CreditPriority:    i.charge.Intent.Priority,
		Features:          i.charge.Intent.FeatureFilters.Normalize(),
		ExpiresAt:         *i.charge.Intent.ExpiresAt,
		SourceChargeID:    &i.charge.ID,
	}
}
