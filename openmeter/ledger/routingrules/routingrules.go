package routingrules

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type RoutingRule interface {
	Validate(tx TxView) error
}

type Validator struct {
	Rules []RoutingRule
}

var _ ledger.RoutingValidator = (*Validator)(nil)

func (v Validator) ValidateEntries(entries []ledger.EntryInput) error {
	view, err := NewTxView(entries)
	if err != nil {
		return err
	}

	for _, rule := range v.Rules {
		if err := rule.Validate(view); err != nil {
			return err
		}
	}

	return nil
}

type FuncRule func(tx TxView) error

func (f FuncRule) Validate(tx TxView) error {
	return f(tx)
}

type AllowedAccountSetsRule struct {
	Sets [][]ledger.AccountType
}

func (r AllowedAccountSetsRule) Validate(tx TxView) error {
	present := tx.AccountTypes()
	if len(present) == 0 {
		return nil
	}

	for _, allowed := range r.Sets {
		if sameAccountTypeSet(present, allowed) {
			return nil
		}
	}

	return ledger.ErrRoutingRuleViolated.WithAttrs(models.Attributes{
		"reason":        "account_type_combination_not_allowed",
		"account_types": present,
	})
}

type RequireFlowDirectionRule struct {
	From ledger.AccountType
	To   ledger.AccountType
}

func (r RequireFlowDirectionRule) Validate(tx TxView) error {
	if !tx.HasAccountTypes(r.From, r.To) {
		return nil
	}

	for _, entry := range tx.EntriesOf(r.From) {
		if !entry.Amount().IsNegative() {
			return ledger.ErrRoutingRuleViolated.WithAttrs(models.Attributes{
				"reason":       "invalid_flow_direction",
				"account_type": r.From,
				"expected":     "negative",
				"target_type":  r.To,
			})
		}
	}

	for _, entry := range tx.EntriesOf(r.To) {
		if !entry.Amount().IsPositive() {
			return ledger.ErrRoutingRuleViolated.WithAttrs(models.Attributes{
				"reason":       "invalid_flow_direction",
				"account_type": r.To,
				"expected":     "positive",
				"source_type":  r.From,
			})
		}
	}

	return nil
}

type RouteField string

const (
	RouteFieldCurrency       RouteField = "currency"
	RouteFieldTaxCode        RouteField = "tax_code"
	RouteFieldFeatures       RouteField = "features"
	RouteFieldCostBasis      RouteField = "cost_basis"
	RouteFieldCreditPriority RouteField = "credit_priority"
)

type RequireSameRouteRule struct {
	Left   ledger.AccountType
	Right  ledger.AccountType
	Fields []RouteField
}

func (r RequireSameRouteRule) Validate(tx TxView) error {
	if !tx.HasAccountTypes(r.Left, r.Right) {
		return nil
	}

	leftEntries := tx.EntriesOf(r.Left)
	rightEntries := tx.EntriesOf(r.Right)

	for _, left := range leftEntries {
		for _, right := range rightEntries {
			for _, field := range r.Fields {
				same, err := sameRouteField(left.Route(), right.Route(), field)
				if err != nil {
					return err
				}

				if !same {
					return ledger.ErrRoutingRuleViolated.WithAttrs(models.Attributes{
						"reason":     "route_field_mismatch",
						"left_type":  r.Left,
						"right_type": r.Right,
						"field":      field,
					})
				}
			}
		}
	}

	return nil
}

func sameRouteField(left ledger.Route, right ledger.Route, field RouteField) (bool, error) {
	switch field {
	case RouteFieldCurrency:
		return left.Currency == right.Currency, nil
	case RouteFieldTaxCode:
		return optionalStringEqual(left.TaxCode, right.TaxCode), nil
	case RouteFieldFeatures:
		return stringSliceEqual(left.Features, right.Features), nil
	case RouteFieldCostBasis:
		return optionalDecimalEqual(left.CostBasis, right.CostBasis), nil
	case RouteFieldCreditPriority:
		return optionalIntEqual(left.CreditPriority, right.CreditPriority), nil
	default:
		return false, fmt.Errorf("unknown route field: %s", field)
	}
}

func sameAccountTypeSet(left []ledger.AccountType, right []ledger.AccountType) bool {
	if len(left) != len(right) {
		return false
	}

	index := make(map[ledger.AccountType]struct{}, len(left))
	for _, item := range left {
		index[item] = struct{}{}
	}

	for _, item := range right {
		if _, ok := index[item]; !ok {
			return false
		}
	}

	return true
}
