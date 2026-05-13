package transactions

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type breakageTemplateBase struct {
	At              time.Time
	Amount          alpacadecimal.Decimal
	FBOAddress      ledger.PostingAddress
	BreakageAddress ledger.PostingAddress
}

func (t breakageTemplateBase) validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if t.FBOAddress == nil {
		return fmt.Errorf("FBO address is required")
	}

	if t.FBOAddress.AccountType() != ledger.AccountTypeCustomerFBO {
		return fmt.Errorf("FBO address account type must be customer_fbo")
	}

	if t.BreakageAddress == nil {
		return fmt.Errorf("breakage address is required")
	}

	if t.BreakageAddress.AccountType() != ledger.AccountTypeBreakage {
		return fmt.Errorf("breakage address account type must be breakage")
	}

	if t.FBOAddress.Route().Route().Currency != t.BreakageAddress.Route().Route().Currency {
		return fmt.Errorf("FBO and breakage currency must match")
	}

	return nil
}

func (t breakageTemplateBase) resolve(fboAmount, breakageAmount alpacadecimal.Decimal) ledger.TransactionInput {
	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			{
				address: t.FBOAddress,
				amount:  fboAmount,
			},
			{
				address: t.BreakageAddress,
				amount:  breakageAmount,
			},
		},
	}
}

type PlanCustomerFBOBreakageTemplate struct {
	At              time.Time
	Amount          alpacadecimal.Decimal
	FBOAddress      ledger.PostingAddress
	BreakageAddress ledger.PostingAddress
}

func (t PlanCustomerFBOBreakageTemplate) Validate() error {
	return t.base().validate()
}

func (t PlanCustomerFBOBreakageTemplate) typeGuard() guard {
	return true
}

func (t PlanCustomerFBOBreakageTemplate) code() TransactionTemplateCode {
	return TemplateCodePlanCustomerFBOBreakage
}

var _ CustomerTransactionTemplate = (PlanCustomerFBOBreakageTemplate{})

func (t PlanCustomerFBOBreakageTemplate) resolve(context.Context, customer.CustomerID, ResolverDependencies) (ledger.TransactionInput, error) {
	return t.base().resolve(t.Amount.Neg(), t.Amount), nil
}

func (t PlanCustomerFBOBreakageTemplate) correct(CorrectionInput) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(string(t.code()))
}

func (t PlanCustomerFBOBreakageTemplate) base() breakageTemplateBase {
	return breakageTemplateBase(t)
}

type ReleaseCustomerFBOBreakageTemplate struct {
	At              time.Time
	Amount          alpacadecimal.Decimal
	FBOAddress      ledger.PostingAddress
	BreakageAddress ledger.PostingAddress
}

func (t ReleaseCustomerFBOBreakageTemplate) Validate() error {
	return t.base().validate()
}

func (t ReleaseCustomerFBOBreakageTemplate) typeGuard() guard {
	return true
}

func (t ReleaseCustomerFBOBreakageTemplate) code() TransactionTemplateCode {
	return TemplateCodeReleaseCustomerFBOBreakage
}

var _ CustomerTransactionTemplate = (ReleaseCustomerFBOBreakageTemplate{})

func (t ReleaseCustomerFBOBreakageTemplate) resolve(context.Context, customer.CustomerID, ResolverDependencies) (ledger.TransactionInput, error) {
	return t.base().resolve(t.Amount, t.Amount.Neg()), nil
}

func (t ReleaseCustomerFBOBreakageTemplate) correct(CorrectionInput) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(string(t.code()))
}

func (t ReleaseCustomerFBOBreakageTemplate) base() breakageTemplateBase {
	return breakageTemplateBase(t)
}

type ReopenCustomerFBOBreakageTemplate struct {
	At              time.Time
	Amount          alpacadecimal.Decimal
	FBOAddress      ledger.PostingAddress
	BreakageAddress ledger.PostingAddress
}

func (t ReopenCustomerFBOBreakageTemplate) Validate() error {
	return t.base().validate()
}

func (t ReopenCustomerFBOBreakageTemplate) typeGuard() guard {
	return true
}

func (t ReopenCustomerFBOBreakageTemplate) code() TransactionTemplateCode {
	return TemplateCodeReopenCustomerFBOBreakage
}

var _ CustomerTransactionTemplate = (ReopenCustomerFBOBreakageTemplate{})

func (t ReopenCustomerFBOBreakageTemplate) resolve(context.Context, customer.CustomerID, ResolverDependencies) (ledger.TransactionInput, error) {
	return t.base().resolve(t.Amount.Neg(), t.Amount), nil
}

func (t ReopenCustomerFBOBreakageTemplate) correct(CorrectionInput) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(string(t.code()))
}

func (t ReopenCustomerFBOBreakageTemplate) base() breakageTemplateBase {
	return breakageTemplateBase(t)
}
