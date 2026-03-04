package credits

import (
	"context"
	"fmt"
	"math"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
)

var (
	_ charges.FlatFeeHandler        = (*MockLedger)(nil)
	_ charges.CreditPurchaseHandler = (*MockLedger)(nil)
)

type MockLedger struct {
	customerCredits            float64
	customerPromotionalCredits float64

	invoiceAccruals float64
	receivables     float64
}

func newMockLedger() *MockLedger {
	return &MockLedger{}
}

// Flat fee handler methods

func (l *MockLedger) OnFlatFeeAssignedToInvoice(ctx context.Context, input charges.OnFlatFeeAssignedToInvoiceInput) ([]charges.CreditRealizationCreateInput, error) {
	out := []charges.CreditRealizationCreateInput{}

	totalToAllocate := input.PreTaxTotalAmount.InexactFloat64()

	if l.customerPromotionalCredits > 0 {
		promotionalCreditsToAllocate := math.Min(totalToAllocate, l.customerPromotionalCredits)
		out = append(out, charges.CreditRealizationCreateInput{
			ServicePeriod: input.ServicePeriod,
			Amount:        alpacadecimal.NewFromFloat(promotionalCreditsToAllocate),
			LedgerTransaction: charges.LedgerTransactionGroupReference{
				TransactionGroupID: ulid.Make().String(),
			},
		})

		totalToAllocate -= promotionalCreditsToAllocate
		l.customerPromotionalCredits -= promotionalCreditsToAllocate
	}

	if totalToAllocate > 0 {
		creditsToAllocate := math.Min(totalToAllocate, l.customerCredits)

		out = append(out, charges.CreditRealizationCreateInput{
			ServicePeriod: input.ServicePeriod,
			Amount:        alpacadecimal.NewFromFloat(creditsToAllocate),
			LedgerTransaction: charges.LedgerTransactionGroupReference{
				TransactionGroupID: ulid.Make().String(),
			},
		})

		l.customerCredits -= creditsToAllocate
		totalToAllocate -= creditsToAllocate
	}

	return out, nil
}

func (l *MockLedger) OnFlatFeeStandardInvoiceUsageAccrued(ctx context.Context, input charges.OnFlatFeeStandardInvoiceUsageAccruedInput) (charges.LedgerTransactionGroupReference, error) {
	l.invoiceAccruals += input.Totals.Total.InexactFloat64()

	return charges.LedgerTransactionGroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) OnFlatFeePaymentAuthorized(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error) {
	return charges.LedgerTransactionGroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) OnFlatFeePaymentSettled(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error) {
	return charges.LedgerTransactionGroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) OnFlatFeePaymentUncollectible(ctx context.Context, charge charges.FlatFeeCharge) (charges.LedgerTransactionGroupReference, error) {
	return charges.LedgerTransactionGroupReference{}, fmt.Errorf("flat fee payment uncollectible is not implemented")
}

// Credit purchase handler methods

func (l *MockLedger) OnCreditPurchaseInitiated(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.LedgerTransactionGroupReference, error) {
	creditAmount := charge.Intent.CreditAmount.InexactFloat64()

	externalCreditPurchaseSettlement, err := charge.Intent.Settlement.AsExternalCreditPurchaseSettlement()
	if err != nil {
		return charges.LedgerTransactionGroupReference{}, err
	}

	costBasis := externalCreditPurchaseSettlement.CostBasis.InexactFloat64()
	if costBasis == 0 {
		return charges.LedgerTransactionGroupReference{}, fmt.Errorf("cost basis is 0")
	}

	paymentAmount := creditAmount * costBasis
	l.receivables += paymentAmount
	l.customerCredits += creditAmount

	return charges.LedgerTransactionGroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) OnCreditPurchasePaymentAuthorized(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.LedgerTransactionGroupReference, error) {
	return charges.LedgerTransactionGroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) OnCreditPurchasePaymentSettled(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.LedgerTransactionGroupReference, error) {
	return charges.LedgerTransactionGroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) OnPromotionalCreditPurchase(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.LedgerTransactionGroupReference, error) {
	l.customerPromotionalCredits += charge.Intent.CreditAmount.InexactFloat64()
	return charges.LedgerTransactionGroupReference{
		TransactionGroupID: ulid.Make().String(),
	}, nil
}

func (l *MockLedger) Reset() {
	l.customerCredits = 0
	l.customerPromotionalCredits = 0
}
