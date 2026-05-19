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

func TestDefaultValidator_AllowsAccruedToFBO(t *testing.T) {
	validator := routingrules.DefaultValidator

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerAccrued, "sub-accrued", ledger.Route{
				Currency: currencyx.Code("USD"),
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerFBO, "sub-fbo", ledger.Route{
				Currency: currencyx.Code("USD"),
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.NoError(t, err)
}

func TestDefaultValidator_AllowsFBOToReceivableReverse(t *testing.T) {
	validator := routingrules.DefaultValidator
	openStatus := ledger.TransactionAuthorizationStatusOpen

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerFBO, "sub-fbo", ledger.Route{
				Currency: currencyx.Code("USD"),
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerReceivable, "sub-rec-open", ledger.Route{
				Currency:                       currencyx.Code("USD"),
				TransactionAuthorizationStatus: &openStatus,
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

func TestDefaultValidator_RejectsDuplicateSubAccountEntries(t *testing.T) {
	validator := routingrules.DefaultValidator
	costBasis := alpacadecimal.NewFromInt(1)
	accruedAddress := addressForRoute(t, ledger.AccountTypeCustomerAccrued, "sub-dup", ledger.Route{
		Currency:  currencyx.Code("USD"),
		CostBasis: &costBasis,
	})
	earningsAddress := addressForRoute(t, ledger.AccountTypeEarnings, "sub-dup", ledger.Route{
		Currency:  currencyx.Code("USD"),
		CostBasis: &costBasis,
	})

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address:     accruedAddress,
			AmountValue: alpacadecimal.NewFromInt(-20),
		},
		&transactionstestutils.AnyEntryInput{
			Address:     earningsAddress,
			AmountValue: alpacadecimal.NewFromInt(20),
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "ledger routing rule violated")
}

func TestDefaultValidator_AllowsDuplicateSubAccountEntriesWithUniqueIdentities(t *testing.T) {
	validator := routingrules.DefaultValidator
	costBasis := alpacadecimal.NewFromInt(1)
	accruedAddress := addressForRoute(t, ledger.AccountTypeCustomerAccrued, "sub-source", ledger.Route{
		Currency:  currencyx.Code("USD"),
		CostBasis: &costBasis,
	})
	earningsAddress := addressForRoute(t, ledger.AccountTypeEarnings, "sub-earnings", ledger.Route{
		Currency:  currencyx.Code("USD"),
		CostBasis: &costBasis,
	})

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address:          accruedAddress,
			AmountValue:      alpacadecimal.NewFromInt(-20),
			IdentityKeyValue: "source:0",
		},
		&transactionstestutils.AnyEntryInput{
			Address:          accruedAddress,
			AmountValue:      alpacadecimal.NewFromInt(-10),
			IdentityKeyValue: "source:1",
		},
		&transactionstestutils.AnyEntryInput{
			Address:     earningsAddress,
			AmountValue: alpacadecimal.NewFromInt(30),
		},
	})

	require.NoError(t, err)
}

func TestDefaultValidator_RejectsTaxBehaviorOutsideFBO(t *testing.T) {
	validator := routingrules.DefaultValidator
	taxBehavior := ledger.TaxBehaviorExclusive

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerAccrued, "sub-accrued-tax-behavior", ledger.Route{
				Currency:    currencyx.Code("USD"),
				TaxBehavior: &taxBehavior,
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

func TestDefaultValidator_AllowsFBOCostBasisTranslationBothDirections(t *testing.T) {
	validator := routingrules.DefaultValidator
	priority := ledger.DefaultCustomerFBOPriority
	costBasis := alpacadecimal.NewFromInt(1)

	unknownFBO := addressForRoute(t, ledger.AccountTypeCustomerFBO, "sub-fbo-unknown", ledger.Route{
		Currency:       currencyx.Code("USD"),
		CreditPriority: &priority,
	})
	knownFBO := addressForRoute(t, ledger.AccountTypeCustomerFBO, "sub-fbo-known", ledger.Route{
		Currency:       currencyx.Code("USD"),
		CostBasis:      &costBasis,
		CreditPriority: &priority,
	})

	t.Run("attribute", func(t *testing.T) {
		err := validator.ValidateEntries([]ledger.EntryInput{
			&transactionstestutils.AnyEntryInput{
				Address:     unknownFBO,
				AmountValue: alpacadecimal.NewFromInt(-50),
			},
			&transactionstestutils.AnyEntryInput{
				Address:     knownFBO,
				AmountValue: alpacadecimal.NewFromInt(50),
			},
		})

		require.NoError(t, err)
	})

	t.Run("correction", func(t *testing.T) {
		err := validator.ValidateEntries([]ledger.EntryInput{
			&transactionstestutils.AnyEntryInput{
				Address:     knownFBO,
				AmountValue: alpacadecimal.NewFromInt(-50),
			},
			&transactionstestutils.AnyEntryInput{
				Address:     unknownFBO,
				AmountValue: alpacadecimal.NewFromInt(50),
			},
		})

		require.NoError(t, err)
	})
}

func TestDefaultValidator_RejectsFBOCostBasisTranslationWithMismatchedPriority(t *testing.T) {
	validator := routingrules.DefaultValidator
	unknownPriority := ledger.DefaultCustomerFBOPriority
	knownPriority := ledger.DefaultCustomerFBOPriority + 1
	costBasis := alpacadecimal.NewFromInt(1)

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerFBO, "sub-fbo-unknown", ledger.Route{
				Currency:       currencyx.Code("USD"),
				CreditPriority: &unknownPriority,
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerFBO, "sub-fbo-known", ledger.Route{
				Currency:       currencyx.Code("USD"),
				CostBasis:      &costBasis,
				CreditPriority: &knownPriority,
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "ledger routing rule violated")
}

func TestDefaultValidator_RejectsFBOCostBasisTranslationWithMismatchedTaxBehavior(t *testing.T) {
	validator := routingrules.DefaultValidator
	priority := ledger.DefaultCustomerFBOPriority
	costBasis := alpacadecimal.NewFromInt(1)
	inclusive := ledger.TaxBehaviorInclusive
	exclusive := ledger.TaxBehaviorExclusive

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerFBO, "sub-fbo-unknown", ledger.Route{
				Currency:       currencyx.Code("USD"),
				CreditPriority: &priority,
				TaxBehavior:    &inclusive,
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerFBO, "sub-fbo-known", ledger.Route{
				Currency:       currencyx.Code("USD"),
				CostBasis:      &costBasis,
				CreditPriority: &priority,
				TaxBehavior:    &exclusive,
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "ledger routing rule violated")
}

func TestDefaultValidator_AllowsReceivableAuthorizationTransition(t *testing.T) {
	validator := routingrules.DefaultValidator
	openStatus := ledger.TransactionAuthorizationStatusOpen
	status := ledger.TransactionAuthorizationStatusAuthorized

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerReceivable, "sub-rec-authorized", ledger.Route{
				Currency:                       currencyx.Code("USD"),
				TransactionAuthorizationStatus: &status,
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerReceivable, "sub-rec-open", ledger.Route{
				Currency:                       currencyx.Code("USD"),
				TransactionAuthorizationStatus: &openStatus,
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.NoError(t, err)
}

func TestDefaultValidator_RejectsReceivableAuthorizationTransitionWithWrongDirection(t *testing.T) {
	validator := routingrules.DefaultValidator
	openStatus := ledger.TransactionAuthorizationStatusOpen
	status := ledger.TransactionAuthorizationStatusAuthorized

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerReceivable, "sub-rec-open", ledger.Route{
				Currency:                       currencyx.Code("USD"),
				TransactionAuthorizationStatus: &openStatus,
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerReceivable, "sub-rec-authorized", ledger.Route{
				Currency:                       currencyx.Code("USD"),
				TransactionAuthorizationStatus: &status,
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "ledger routing rule violated")
}

func TestDefaultValidator_AllowsWashToAuthorizedReceivable(t *testing.T) {
	validator := routingrules.DefaultValidator
	status := ledger.TransactionAuthorizationStatusAuthorized

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeWash, "sub-wash", ledger.Route{
				Currency: currencyx.Code("USD"),
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerReceivable, "sub-rec-authorized", ledger.Route{
				Currency:                       currencyx.Code("USD"),
				TransactionAuthorizationStatus: &status,
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.NoError(t, err)
}

func TestDefaultValidator_RejectsFBOToAccruedTaxCodeMismatch(t *testing.T) {
	validator := routingrules.DefaultValidator
	taxA := "tax_A"
	taxB := "tax_B"

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerFBO, "sub-fbo-taxA", ledger.Route{
				Currency: currencyx.Code("USD"),
				TaxCode:  &taxA,
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerAccrued, "sub-accrued-taxB", ledger.Route{
				Currency: currencyx.Code("USD"),
				TaxCode:  &taxB,
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "ledger routing rule violated")
}

func TestDefaultValidator_RejectsReceivableToAccruedTaxCodeMismatch(t *testing.T) {
	validator := routingrules.DefaultValidator
	openStatus := ledger.TransactionAuthorizationStatusOpen
	taxA := "tax_A"
	taxB := "tax_B"

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerReceivable, "sub-rec-taxA", ledger.Route{
				Currency:                       currencyx.Code("USD"),
				TaxCode:                        &taxA,
				TransactionAuthorizationStatus: &openStatus,
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerAccrued, "sub-accrued-taxB", ledger.Route{
				Currency: currencyx.Code("USD"),
				TaxCode:  &taxB,
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "ledger routing rule violated")
}

func TestDefaultValidator_RejectsAccruedToEarningsTaxCodeMismatch(t *testing.T) {
	validator := routingrules.DefaultValidator
	taxA := "tax_A"
	taxB := "tax_B"

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerAccrued, "sub-accrued-taxA", ledger.Route{
				Currency: currencyx.Code("USD"),
				TaxCode:  &taxA,
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeEarnings, "sub-earn-taxB", ledger.Route{
				Currency: currencyx.Code("USD"),
				TaxCode:  &taxB,
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "ledger routing rule violated")
}

func TestDefaultValidator_AllowsFBOToAccruedMatchingTaxCode(t *testing.T) {
	validator := routingrules.DefaultValidator
	tax := "tax_A"

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerFBO, "sub-fbo-tax", ledger.Route{
				Currency: currencyx.Code("USD"),
				TaxCode:  &tax,
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerAccrued, "sub-accrued-tax", ledger.Route{
				Currency: currencyx.Code("USD"),
				TaxCode:  &tax,
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.NoError(t, err)
}

func TestDefaultValidator_AllowsReceivableToAccruedMatchingTaxCode(t *testing.T) {
	validator := routingrules.DefaultValidator
	openStatus := ledger.TransactionAuthorizationStatusOpen
	tax := "tax_A"

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerReceivable, "sub-rec-tax", ledger.Route{
				Currency:                       currencyx.Code("USD"),
				TaxCode:                        &tax,
				TransactionAuthorizationStatus: &openStatus,
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerAccrued, "sub-accrued-tax", ledger.Route{
				Currency: currencyx.Code("USD"),
				TaxCode:  &tax,
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.NoError(t, err)
}

func TestDefaultValidator_AllowsAccruedToEarningsMatchingTaxCode(t *testing.T) {
	validator := routingrules.DefaultValidator
	tax := "tax_A"

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerAccrued, "sub-accrued-tax", ledger.Route{
				Currency: currencyx.Code("USD"),
				TaxCode:  &tax,
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeEarnings, "sub-earn-tax", ledger.Route{
				Currency: currencyx.Code("USD"),
				TaxCode:  &tax,
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.NoError(t, err)
}

func TestDefaultValidator_AllowsAccruedToEarningsNilTaxCodeBothSides(t *testing.T) {
	validator := routingrules.DefaultValidator

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerAccrued, "sub-accrued-nil", ledger.Route{
				Currency: currencyx.Code("USD"),
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeEarnings, "sub-earn-nil", ledger.Route{
				Currency: currencyx.Code("USD"),
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.NoError(t, err)
}

func TestDefaultValidator_RejectsWashToOpenReceivable(t *testing.T) {
	validator := routingrules.DefaultValidator
	status := ledger.TransactionAuthorizationStatusOpen

	err := validator.ValidateEntries([]ledger.EntryInput{
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeWash, "sub-wash", ledger.Route{
				Currency: currencyx.Code("USD"),
			}),
			AmountValue: alpacadecimal.NewFromInt(-50),
		},
		&transactionstestutils.AnyEntryInput{
			Address: addressForRoute(t, ledger.AccountTypeCustomerReceivable, "sub-rec-open", ledger.Route{
				Currency:                       currencyx.Code("USD"),
				TransactionAuthorizationStatus: &status,
			}),
			AmountValue: alpacadecimal.NewFromInt(50),
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "ledger routing rule violated")
}

func addressForRoute(t *testing.T, accountType ledger.AccountType, subAccountID string, route ledger.Route) ledger.PostingAddress {
	t.Helper()

	key, err := ledger.BuildRoutingKey(route)
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
