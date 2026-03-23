package routingrules_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/routingrules"
	transactionstestutils "github.com/openmeterio/openmeter/openmeter/ledger/transactions/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestDefaultValidator_AllowsFBOToAccrued(t *testing.T) {
	validator := routingrules.DefaultValidator

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerFBO, "sub-fbo", ledger.Route{
				Currency: currencyx.Code("USD"),
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerAccrued, "sub-accrued", ledger.Route{
				Currency: currencyx.Code("USD"),
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.NoError(t, err)
}

func TestDefaultValidator_RejectsForbiddenAccountCombination(t *testing.T) {
	validator := routingrules.DefaultValidator

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerFBO, "sub-fbo", ledger.Route{
				Currency: currencyx.Code("USD"),
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeEarnings, "sub-earnings", ledger.Route{
				Currency: currencyx.Code("USD"),
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "ledger routing rule violated")
}

func TestDefaultValidator_RejectsMismatchedReceivableAndFBORoute(t *testing.T) {
	validator := routingrules.DefaultValidator

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerReceivable, "sub-rec", ledger.Route{
				Currency: currencyx.Code("USD"),
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerFBO, "sub-fbo", ledger.Route{
				Currency: currencyx.Code("EUR"),
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "ledger routing rule violated")
}

func addressForRoute(t *testing.T, accountType ledger.AccountType, subAccountID string, route ledger.Route) ledger.PostingAddress {
	t.Helper()

	key, err := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, route)
	require.NoError(t, err)

	addr, err := ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{
		SubAccountID: subAccountID,
		AccountType:  accountType,
		Route:        route,
		RouteID:      "route-" + subAccountID + "-" + time.Now().UTC().Format("150405.000000000"),
		RoutingKey:   key,
	})
	require.NoError(t, err)

	return addr
}
