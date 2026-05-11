package noop

import (
	"context"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

const noopCurrency = currencyx.Code("USD")

type balance struct{}

func (balance) Settled() alpacadecimal.Decimal {
	return alpacadecimal.Zero
}

func (balance) Pending() alpacadecimal.Decimal {
	return alpacadecimal.Zero
}

type subAccount struct {
	accountID   models.NamespacedID
	accountType ledger.AccountType
	route       ledger.Route
}

func (s subAccount) Address() ledger.PostingAddress {
	return postingAddress{accountType: s.accountType}
}

func (s subAccount) Route() ledger.Route {
	return s.route
}

func (s subAccount) AccountID() models.NamespacedID {
	return s.accountID
}

func (subAccount) GetBalance(context.Context) (ledger.Balance, error) {
	return balance{}, nil
}

type postingAddress struct {
	accountType ledger.AccountType
}

func (p postingAddress) Equal(other ledger.PostingAddress) bool {
	return other != nil && p.AccountType() == other.AccountType()
}

func (postingAddress) SubAccountID() string {
	return ""
}

func (p postingAddress) AccountType() ledger.AccountType {
	return p.accountType
}

func (postingAddress) Route() ledger.SubAccountRoute {
	return ledger.SubAccountRoute{}
}

type Ledger struct{}

var (
	_ ledger.Ledger         = Ledger{}
	_ ledger.BalanceQuerier = Ledger{}
)

func (Ledger) CommitGroup(context.Context, ledger.TransactionGroupInput) (ledger.TransactionGroup, error) {
	return nil, nil
}

func (Ledger) GetTransactionGroup(context.Context, models.NamespacedID) (ledger.TransactionGroup, error) {
	return nil, nil
}

func (Ledger) ListTransactions(context.Context, ledger.ListTransactionsInput) (ledger.ListTransactionsResult, error) {
	return ledger.ListTransactionsResult{}, nil
}

func (Ledger) GetAccountBalance(context.Context, ledger.Account, ledger.RouteFilter, ledger.BalanceQuery) (ledger.Balance, error) {
	return balance{}, nil
}

func (Ledger) GetSubAccountBalance(context.Context, ledger.SubAccount, ledger.BalanceQuery) (ledger.Balance, error) {
	return balance{}, nil
}

type AccountResolver struct{}

var _ ledger.AccountResolver = AccountResolver{}

func (AccountResolver) CreateCustomerAccounts(context.Context, customer.CustomerID) (ledger.CustomerAccounts, error) {
	return ledger.CustomerAccounts{
		FBOAccount:        customerFBOAccount{customerAccount: customerAccount{accountType: ledger.AccountTypeCustomerFBO}},
		ReceivableAccount: customerReceivableAccount{customerAccount: customerAccount{accountType: ledger.AccountTypeCustomerReceivable}},
		AccruedAccount:    customerAccruedAccount{customerAccount: customerAccount{accountType: ledger.AccountTypeCustomerAccrued}},
	}, nil
}

func (AccountResolver) GetCustomerAccounts(context.Context, customer.CustomerID) (ledger.CustomerAccounts, error) {
	return ledger.CustomerAccounts{
		FBOAccount:        customerFBOAccount{customerAccount: customerAccount{accountType: ledger.AccountTypeCustomerFBO}},
		ReceivableAccount: customerReceivableAccount{customerAccount: customerAccount{accountType: ledger.AccountTypeCustomerReceivable}},
		AccruedAccount:    customerAccruedAccount{customerAccount: customerAccount{accountType: ledger.AccountTypeCustomerAccrued}},
	}, nil
}

func (AccountResolver) EnsureBusinessAccounts(context.Context, string) (ledger.BusinessAccounts, error) {
	return ledger.BusinessAccounts{
		WashAccount:      businessAccount{accountType: ledger.AccountTypeWash},
		EarningsAccount:  businessAccount{accountType: ledger.AccountTypeEarnings},
		BrokerageAccount: businessAccount{accountType: ledger.AccountTypeBrokerage},
	}, nil
}

func (AccountResolver) GetBusinessAccounts(context.Context, string) (ledger.BusinessAccounts, error) {
	return ledger.BusinessAccounts{
		WashAccount:      businessAccount{accountType: ledger.AccountTypeWash},
		EarningsAccount:  businessAccount{accountType: ledger.AccountTypeEarnings},
		BrokerageAccount: businessAccount{accountType: ledger.AccountTypeBrokerage},
	}, nil
}

type AccountService struct{}

var _ ledgeraccount.Service = AccountService{}

func (AccountService) CreateAccount(_ context.Context, input ledger.CreateAccountInput) (ledger.Account, error) {
	return newAccount(input.Namespace, input.Type, normalizeID(string(input.Type), "noop-account")), nil
}

func (AccountService) EnsureSubAccount(_ context.Context, input ledger.CreateSubAccountInput) (ledger.SubAccount, error) {
	return newSubAccount(input.Namespace, input.AccountID, input.Route), nil
}

func (AccountService) GetAccountByID(_ context.Context, id models.NamespacedID) (ledger.Account, error) {
	return newAccount(id.Namespace, ledger.AccountTypeCustomerAccrued, id.ID), nil
}

func (AccountService) GetSubAccountByID(_ context.Context, id models.NamespacedID) (ledger.SubAccount, error) {
	return newSubAccount(id.Namespace, id.ID, ledger.Route{Currency: noopCurrency}), nil
}

func (AccountService) ListSubAccounts(context.Context, ledger.ListSubAccountsInput) ([]ledger.SubAccount, error) {
	return []ledger.SubAccount{}, nil
}

func (AccountService) ListAccounts(context.Context, ledger.ListAccountsInput) ([]ledger.Account, error) {
	return []ledger.Account{}, nil
}

func (AccountService) LockAccountsForPosting(context.Context, []ledger.Account) error {
	return nil
}

type customerAccount struct {
	accountType ledger.AccountType
}

func (customerAccount) GetBalance(context.Context, ledger.RouteFilter, *ledger.TransactionCursor) (ledger.Balance, error) {
	return balance{}, nil
}

func (customerAccount) Lock(context.Context) error {
	return nil
}

func (c customerAccount) ID() models.NamespacedID {
	return models.NamespacedID{Namespace: "noop", ID: string(c.accountType)}
}

func (c customerAccount) Type() ledger.AccountType {
	return c.accountType
}

type customerFBOAccount struct {
	customerAccount
}

func (customerFBOAccount) GetSubAccountForRoute(context.Context, ledger.CustomerFBORouteParams) (ledger.SubAccount, error) {
	return subAccount{accountType: ledger.AccountTypeCustomerFBO}, nil
}

type customerReceivableAccount struct {
	customerAccount
}

func (customerReceivableAccount) GetSubAccountForRoute(context.Context, ledger.CustomerReceivableRouteParams) (ledger.SubAccount, error) {
	return subAccount{accountType: ledger.AccountTypeCustomerReceivable}, nil
}

type customerAccruedAccount struct {
	customerAccount
}

func (customerAccruedAccount) GetSubAccountForRoute(context.Context, ledger.CustomerAccruedRouteParams) (ledger.SubAccount, error) {
	return subAccount{accountType: ledger.AccountTypeCustomerAccrued}, nil
}

type businessAccount struct {
	accountType ledger.AccountType
}

func (businessAccount) GetBalance(context.Context, ledger.RouteFilter, *ledger.TransactionCursor) (ledger.Balance, error) {
	return balance{}, nil
}

func (b businessAccount) ID() models.NamespacedID {
	return models.NamespacedID{Namespace: "noop", ID: string(b.accountType)}
}

func (b businessAccount) Type() ledger.AccountType {
	return b.accountType
}

func (b businessAccount) GetSubAccountForRoute(context.Context, ledger.BusinessRouteParams) (ledger.SubAccount, error) {
	return subAccount{accountType: b.accountType}, nil
}

type NamespaceHandler struct{}

var _ namespace.Handler = NamespaceHandler{}

func (NamespaceHandler) CreateNamespace(context.Context, string) error {
	return nil
}

func (NamespaceHandler) DeleteNamespace(context.Context, string) error {
	return nil
}

func newAccount(namespace string, accountType ledger.AccountType, id string) ledger.Account {
	if accountType == "" {
		accountType = ledger.AccountTypeCustomerAccrued
	}

	account, err := ledgeraccount.NewAccountFromData(ledgeraccount.AccountData{
		ID: models.NamespacedID{
			Namespace: normalizeNamespace(namespace),
			ID:        normalizeID(id, "noop-account"),
		},
		AccountType: accountType,
	}, ledgeraccount.AccountLiveServices{
		SubAccountService: AccountService{},
	})
	if err != nil {
		return customerAccruedAccount{customerAccount: customerAccount{accountType: ledger.AccountTypeCustomerAccrued}}
	}

	return account
}

func newSubAccount(namespace, accountID string, route ledger.Route) *ledgeraccount.SubAccount {
	normalizedRoute := route
	if normalizedRoute.Currency == "" {
		normalizedRoute.Currency = noopCurrency
	}

	accountType := accountTypeForRoute(normalizedRoute)
	routingKey, err := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, normalizedRoute)
	if err != nil {
		return &ledgeraccount.SubAccount{}
	}

	subAccount, err := ledgeraccount.NewSubAccountFromData(ledgeraccount.SubAccountData{
		ID:          "noop-sub-account",
		Namespace:   normalizeNamespace(namespace),
		AccountID:   normalizeID(accountID, "noop-account"),
		AccountType: accountType,
		Route:       normalizedRoute,
		RouteMeta: ledgeraccount.SubAccountRouteData{
			ID:                "noop-route",
			RoutingKeyVersion: ledger.RoutingKeyVersionV1,
			RoutingKey:        routingKey.Value(),
		},
		CreatedAt: time.Unix(0, 0),
	})
	if err != nil {
		return &ledgeraccount.SubAccount{}
	}

	return subAccount
}

func normalizeNamespace(namespace string) string {
	return normalizeID(namespace, "noop")
}

func normalizeID(id, fallback string) string {
	if id == "" {
		return fallback
	}

	return id
}

func accountTypeForRoute(route ledger.Route) ledger.AccountType {
	switch {
	case route.TransactionAuthorizationStatus != nil:
		return ledger.AccountTypeCustomerReceivable
	case route.CreditPriority != nil:
		return ledger.AccountTypeCustomerFBO
	default:
		return ledger.AccountTypeCustomerAccrued
	}
}
