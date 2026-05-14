package meta

import (
	"errors"
	"fmt"
	"slices"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	_             Patch = (*PatchDelete)(nil)
	TriggerDelete       = stateless.Trigger("delete")
)

type PatchDelete struct {
	policy PatchDeletePolicy
}

func NewPatchDelete(policy PatchDeletePolicy) PatchDelete {
	var patch PatchDelete
	patch.SetPolicy(policy)
	return patch
}

func (p *PatchDelete) SetPolicy(policy PatchDeletePolicy) {
	p.policy = policy
}

func (p PatchDelete) GetPolicy() PatchDeletePolicy {
	return p.policy
}

func (p PatchDelete) Op() PatchType {
	return PatchTypeDelete
}

func (p PatchDelete) Trigger() stateless.Trigger {
	return TriggerDelete
}

func (p PatchDelete) TriggerParams() any {
	return p.GetPolicy()
}

func (p PatchDelete) Validate() error {
	return p.GetPolicy().Validate()
}

type CreditRefundPolicy string

var _ models.Validator = (*CreditRefundPolicy)(nil)

const (
	// CreditRefundPolicyCorrect will refund the credit to the customer by reversing the credit transactions.
	CreditRefundPolicyCorrect CreditRefundPolicy = "correct"
	// CreditRefundPolicyIgnore will ignore the credit and leave it as is without performing any action.
	CreditRefundPolicyIgnore CreditRefundPolicy = "ignore"
)

func (p CreditRefundPolicy) Values() []CreditRefundPolicy {
	return []CreditRefundPolicy{
		CreditRefundPolicyCorrect,
		CreditRefundPolicyIgnore,
	}
}

func (p CreditRefundPolicy) Validate() error {
	if !slices.Contains(p.Values(), p) {
		return models.NewGenericValidationError(fmt.Errorf("invalid credit refund policy: %s", p))
	}

	return nil
}

type InvoiceRefundPolicy string

var _ models.Validator = (*InvoiceRefundPolicy)(nil)

const (
	// InvoiceRefundPolicyRefund will refund the payment to the customer using the app's refund functionality.
	InvoiceRefundPolicyRefund InvoiceRefundPolicy = "refund"
	// InvoiceRefundPolicyGrantCredits will grant credits to the customer to cover the payment amount.
	InvoiceRefundPolicyGrantCredits InvoiceRefundPolicy = "grant_credits"
	// InvoiceRefundPolicyIgnore will ignore the payment and leave it as is without performing any action. (this can be used
	// to settle the payment manually)
	InvoiceRefundPolicyIgnore InvoiceRefundPolicy = "ignore"
)

func (p InvoiceRefundPolicy) Values() []InvoiceRefundPolicy {
	return []InvoiceRefundPolicy{
		InvoiceRefundPolicyRefund,
		InvoiceRefundPolicyGrantCredits,
		InvoiceRefundPolicyIgnore,
	}
}

func (p InvoiceRefundPolicy) Validate() error {
	if !slices.Contains(p.Values(), p) {
		return models.NewGenericValidationError(fmt.Errorf("invalid invoice refund policy: %s", p))
	}

	return nil
}

var _ models.Validator = (*PatchDeletePolicy)(nil)

type PatchDeletePolicy struct {
	CreditRefundPolicy  CreditRefundPolicy
	InvoiceRefundPolicy InvoiceRefundPolicy
}

func (p PatchDeletePolicy) Validate() error {
	var errs []error

	if err := p.CreditRefundPolicy.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credit refund policy: %w", err))
	}

	if err := p.InvoiceRefundPolicy.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invoice refund policy: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// RefundAsCreditsDeletePolicy is a policy that will refund the usage as credits to the customer. For now this can
// be considered as the default policy for delete patches.
var RefundAsCreditsDeletePolicy PatchDeletePolicy = PatchDeletePolicy{
	CreditRefundPolicy:  CreditRefundPolicyCorrect,
	InvoiceRefundPolicy: InvoiceRefundPolicyGrantCredits,
}
