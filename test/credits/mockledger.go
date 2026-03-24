package credits

import (
	"context"
	"fmt"
	"math"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

var (
	_ flatfee.Handler        = (*MockLedger)(nil)
	_ creditpurchase.Handler = (*MockLedger)(nil)
	_ usagebased.Handler     = (*MockLedger)(nil)
)

type MockLedger struct {
	usagebased.UnimplementedHandler

	customerCredits            float64
	customerPromotionalCredits float64

	invoiceAccruals float64
	receivables     float64
}

func newMockLedger() *MockLedger {
	return &MockLedger{}
}

// Flat fee handler methods

func (l *MockLedger) OnAssignedToInvoice(ctx context.Context, input flatfee.OnAssignedToInvoiceInput) ([]creditrealization.CreateInput, error) {
	out := []creditrealization.CreateInput{}

	totalToAllocate := input.PreTaxTotalAmount.InexactFloat64()

	if l.customerPromotionalCredits > 0 {
		promotionalCreditsToAllocate := math.Min(totalToAllocate, l.customerPromotionalCredits)
		out = append(out, creditrealization.CreateInput{
			ServicePeriod: input.ServicePeriod,
			Amount:        alpacadecimal.NewFromFloat(promotionalCreditsToAllocate),
			LedgerTransaction: ledgertransaction.GroupReference{
				TransactionGroupID: ulid.Make().String(),
			},
		})

		totalToAllocate -= promotionalCreditsToAllocate
		l.customerPromotionalCredits -= promotionalCreditsToAllocate
	}

	if totalToAllocate > 0 {
		creditsToAllocate := math.Min(totalToAllocate, l.customerCredits)

		out = append(out, creditrealization.CreateInput{
			ServicePeriod: input.ServicePeriod,
			Amount:        alpacadecimal.NewFromFloat(creditsToAllocate),
			LedgerTransaction: ledgertransaction.GroupReference{
				TransactionGroupID: ulid.Make().String(),
			},
		})

		l.customerCredits -= creditsToAllocate
		// totalToAllocate -= creditsToAllocate
	}

	return out, nil
}

func (l *MockLedger) OnInvoiceUsageAccrued(ctx context.Context, input flatfee.OnInvoiceUsageAccruedInput) (ledgertransaction.GroupReference, error) {
	l.invoiceAccruals += input.Totals.Total.InexactFloat64()

	return ledgertransaction.GroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) OnPaymentAuthorized(ctx context.Context, charge flatfee.Charge) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) OnPaymentSettled(ctx context.Context, charge flatfee.Charge) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) OnPaymentUncollectible(ctx context.Context, charge flatfee.Charge) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{}, fmt.Errorf("flat fee payment uncollectible is not implemented")
}

// Credit purchase handler methods

func (l *MockLedger) OnCreditPurchaseInitiated(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	creditAmount := charge.Intent.CreditAmount.InexactFloat64()

	externalSettlement, err := charge.Intent.Settlement.AsExternalSettlement()
	if err != nil {
		return ledgertransaction.GroupReference{}, err
	}

	costBasis := externalSettlement.CostBasis.InexactFloat64()
	if costBasis == 0 {
		return ledgertransaction.GroupReference{}, fmt.Errorf("cost basis is 0")
	}

	paymentAmount := creditAmount * costBasis
	l.receivables += paymentAmount
	l.customerCredits += creditAmount

	return ledgertransaction.GroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) OnCreditPurchasePaymentAuthorized(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) OnCreditPurchasePaymentSettled(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	return ledgertransaction.GroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) OnPromotionalCreditPurchase(ctx context.Context, charge creditpurchase.Charge) (ledgertransaction.GroupReference, error) {
	l.customerPromotionalCredits += charge.Intent.CreditAmount.InexactFloat64()
	return ledgertransaction.GroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) Reset() {
	l.customerCredits = 0
	l.customerPromotionalCredits = 0
}
