package creditpurchase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CreditPurchaseHandler is the interface for handling credit purchase charges.
// It is used to handle the different types of credit purchase charges (promotional, external, invoice).
//
// Promotional credit purchases are handled by the OnPromotionalCreditPurchase method only.
//
// Cost basis > 0 credit purchases are handled by the OnCreditPurchaseInitiated method, which is the initial call.
// Happy path:
// - OnCreditPurchaseInitiated is called
// - OnCreditPurchasePaymentAuthorized is called
// - OnCreditPurchasePaymentSettled is called
//
// Failed payment can occur either after the OnCreditPurchaseInitiated or after the OnCreditPurchasePaymentAuthorized call.

type Handler interface {
	// Promotional credit handler methods (cost basis == 0)
	// ----------------------------------------------------

	// OnPromotionalCreditPurchase is called when a promotional credit purchase is created (e.g. costbasis is 0)
	// For promotional credit purchases we don't call any of the payment handler methods.
	OnPromotionalCreditPurchase(ctx context.Context, input CreditGrantInput) (CreditGrantResult, error)

	// Credit purchase handler methods (cost basis > 0)
	// ------------------------------------------------

	// OnCreditPurchaseInitiated is called when a credit purchase is initiated that is going to be settled by
	// a payment (either external or a standard invoice)
	// Initial call
	OnCreditPurchaseInitiated(ctx context.Context, input CreditGrantInput) (CreditGrantResult, error)

	// OnCreditPurchasePaymentAuthorized is called when a credit purchase payment is authorized for a credit
	// purchase.
	OnCreditPurchasePaymentAuthorized(ctx context.Context, input PaymentEventInput) (ledgertransaction.GroupReference, error)

	// OnCreditPurchasePaymentSettled is called when a credit purchase payment is settled for a credit
	// purchase.
	OnCreditPurchasePaymentSettled(ctx context.Context, input PaymentEventInput) (ledgertransaction.GroupReference, error)
}

type PaymentEventInput struct {
	Charge     Charge                `json:"charge"`
	EventAt    time.Time             `json:"eventAt"`
	FiatAmount alpacadecimal.Decimal `json:"fiatAmount"`
}

func (i PaymentEventInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if i.EventAt.IsZero() {
		errs = append(errs, fmt.Errorf("event at is required"))
	}

	if !i.FiatAmount.IsPositive() {
		errs = append(errs, fmt.Errorf("fiat amount must be positive"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// CreditGrantInput supplies legacy advance occurrences. The ledger merges them
// with origin-tracked occurrences in immutable collection-time and ID order.
// Nil and empty AdvanceLineages skip legacy accrued backfill only.
// Remaining eligible receivable may still be attributed without accrued backfill.
type CreditGrantInput struct {
	Charge          Charge
	AdvanceLineages []legacylineage.Lineage
}

func (i CreditGrantInput) Validate() error {
	var errs []error
	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// GetCostBasis returns zero for fiat promotions and nil for custom-currency
// promotions. Payment-backed grants require the persisted resolved cost basis.
func (i CreditGrantInput) GetCostBasis() (*alpacadecimal.Decimal, error) {
	charge := i.Charge

	if charge.Intent.Settlement.Type() == SettlementTypePromotional {
		if charge.Intent.Currency.IsCustom() {
			return nil, nil
		}

		return lo.ToPtr(alpacadecimal.Zero), nil
	}

	if charge.State.ResolvedCostBasis == nil {
		return nil, models.NewGenericPreConditionFailedError(
			fmt.Errorf("credit purchase charge[%s] cost basis is unresolved", charge.ID),
		)
	}

	return &charge.State.ResolvedCostBasis.CostBasis, nil
}

// GetCostBasisCurrency supplies the fiat denomination for paid custom credits.
// Fiat credits use their own currency; promotional custom credits have no cost basis.
func (i CreditGrantInput) GetCostBasisCurrency() (*currencyx.Code, error) {
	charge := i.Charge

	if !charge.Intent.Currency.IsCustom() || charge.Intent.Settlement.Type() == SettlementTypePromotional {
		return nil, nil
	}

	fiatCurrency, err := charge.Intent.GetSettlementFiatCurrency()
	if err != nil {
		return nil, fmt.Errorf("get settlement fiat currency: %w", err)
	}

	return lo.ToPtr(currencyx.Code(fiatCurrency.GetFiatCode())), nil
}

var _ models.Validator = CreditGrantInput{}

type CreditGrantResult struct {
	ledgertransaction.GroupReference
	// BackfillAllocations updates deprecated lineage state for legacy histories only.
	// It excludes origin-tracked backfill and receivable-only attribution.
	BackfillAllocations []legacylineage.AdvanceBackfillAllocation
}
