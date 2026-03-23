package routingrules

import "github.com/openmeterio/openmeter/openmeter/ledger"

var DefaultValidator = Validator{
	Rules: []RoutingRule{
		AllowedAccountSetsRule{
			Sets: [][]ledger.AccountType{
				{ledger.AccountTypeCustomerFBO, ledger.AccountTypeCustomerReceivable},
				{ledger.AccountTypeCustomerFBO, ledger.AccountTypeCustomerAccrued},
				{ledger.AccountTypeCustomerReceivable, ledger.AccountTypeCustomerAccrued},
				{ledger.AccountTypeCustomerReceivable, ledger.AccountTypeWash},
				{ledger.AccountTypeCustomerAccrued, ledger.AccountTypeEarnings},
				{ledger.AccountTypeCustomerFBO, ledger.AccountTypeBrokerage},
			},
		},
		RequireFlowDirectionRule{
			From: ledger.AccountTypeCustomerFBO,
			To:   ledger.AccountTypeCustomerAccrued,
		},
		RequireFlowDirectionRule{
			From: ledger.AccountTypeCustomerReceivable,
			To:   ledger.AccountTypeCustomerAccrued,
		},
		RequireFlowDirectionRule{
			From: ledger.AccountTypeCustomerAccrued,
			To:   ledger.AccountTypeEarnings,
		},
		RequireFlowDirectionRule{
			From: ledger.AccountTypeWash,
			To:   ledger.AccountTypeCustomerReceivable,
		},
		RequireSameRouteRule{
			Left:  ledger.AccountTypeCustomerFBO,
			Right: ledger.AccountTypeCustomerReceivable,
			Fields: []RouteField{
				RouteFieldCurrency,
				RouteFieldTaxCode,
				RouteFieldFeatures,
				RouteFieldCostBasis,
			},
		},
		RequireSameRouteRule{
			Left:  ledger.AccountTypeCustomerFBO,
			Right: ledger.AccountTypeCustomerAccrued,
			Fields: []RouteField{
				RouteFieldCurrency,
			},
		},
		RequireSameRouteRule{
			Left:  ledger.AccountTypeCustomerReceivable,
			Right: ledger.AccountTypeCustomerAccrued,
			Fields: []RouteField{
				RouteFieldCurrency,
			},
		},
		RequireSameRouteRule{
			Left:  ledger.AccountTypeCustomerAccrued,
			Right: ledger.AccountTypeEarnings,
			Fields: []RouteField{
				RouteFieldCurrency,
			},
		},
		RequireSameRouteRule{
			Left:  ledger.AccountTypeCustomerReceivable,
			Right: ledger.AccountTypeWash,
			Fields: []RouteField{
				RouteFieldCurrency,
			},
		},
	},
}
