package routingrules

import "github.com/openmeterio/openmeter/openmeter/ledger"

var DefaultValidator = Validator{
	Rules: []RoutingRule{
		RequireUniqueSubAccountsRule{},
		AllowedAccountSetsRule{
			Sets: [][]ledger.AccountType{
				{ledger.AccountTypeCustomerFBO, ledger.AccountTypeCustomerReceivable},
				{ledger.AccountTypeCustomerReceivable},
				{ledger.AccountTypeCustomerFBO},
				{ledger.AccountTypeCustomerAccrued},
				{ledger.AccountTypeCustomerFBO, ledger.AccountTypeCustomerAccrued},
				{ledger.AccountTypeCustomerReceivable, ledger.AccountTypeCustomerAccrued},
				{ledger.AccountTypeCustomerReceivable, ledger.AccountTypeWash},
				{ledger.AccountTypeCustomerAccrued, ledger.AccountTypeEarnings},
				{ledger.AccountTypeCustomerFBO, ledger.AccountTypeBrokerage},
				{ledger.AccountTypeCustomerFBO, ledger.AccountTypeBreakage},
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
		RequireFlowDirectionRule{
			From: ledger.AccountTypeCustomerFBO,
			To:   ledger.AccountTypeBreakage,
		},
		RequireAccountAuthorizationStatusRule{
			WhenHasAccountTypes: []ledger.AccountType{
				ledger.AccountTypeWash,
				ledger.AccountTypeCustomerReceivable,
			},
			AccountType: ledger.AccountTypeCustomerReceivable,
			Expected:    ledger.TransactionAuthorizationStatusAuthorized,
		},
		RequireReceivableAuthorizationStageRule{},
		RequireFBOCostBasisTranslationRule{},
		RequireAccruedCostBasisTranslationRule{},
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
				RouteFieldCostBasis,
			},
		},
		RequireSameRouteRule{
			Left:  ledger.AccountTypeCustomerReceivable,
			Right: ledger.AccountTypeCustomerAccrued,
			Fields: []RouteField{
				RouteFieldCurrency,
				RouteFieldCostBasis,
			},
		},
		RequireSameRouteRule{
			Left:  ledger.AccountTypeCustomerAccrued,
			Right: ledger.AccountTypeEarnings,
			Fields: []RouteField{
				RouteFieldCurrency,
				RouteFieldCostBasis,
			},
		},
		RequireSameRouteRule{
			Left:  ledger.AccountTypeCustomerReceivable,
			Right: ledger.AccountTypeWash,
			Fields: []RouteField{
				RouteFieldCurrency,
				RouteFieldCostBasis,
			},
		},
		RequireSameRouteRule{
			Left:  ledger.AccountTypeCustomerFBO,
			Right: ledger.AccountTypeBreakage,
			Fields: []RouteField{
				RouteFieldCurrency,
				RouteFieldCostBasis,
			},
		},
	},
}
