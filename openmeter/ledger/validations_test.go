package ledger_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestValidateTransactionGroupInputWith(t *testing.T) {
	address := mustPostingAddress(t, currencyx.Code("USD"))
	at := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	id := ulid.Make().String()

	newTransaction := func(id string, entryIDs ...string) *testutils.AnyTransactionInput {
		input := &testutils.AnyTransactionInput{
			IDValue:       id,
			BookedAtValue: at,
			EntryInputsValues: []*testutils.AnyEntryInput{
				{
					Address:     address,
					AmountValue: alpacadecimal.NewFromInt(10),
				},
				{
					Address:     address,
					AmountValue: alpacadecimal.NewFromInt(-10),
				},
			},
		}

		for idx, id := range entryIDs {
			input.EntryInputsValues[idx].IDValue = id
		}

		return input
	}

	unbalanced := newTransaction("")
	unbalanced.EntryInputsValues[1].AmountValue = alpacadecimal.NewFromInt(-9)

	for _, tc := range []struct {
		name             string
		input            ledger.TransactionGroupInput
		expectedError    error
		expectedMessages []string
	}{
		{
			name:  "accepts omitted IDs across multiple transactions",
			input: transactions.GroupInputs("ns-test", nil, newTransaction(""), newTransaction("")),
		},
		{
			name: "accepts the same ID across different entity types",
			input: ledger.WithGroupID(
				transactions.GroupInputs("ns-test", nil, newTransaction(id, id, ulid.Make().String())),
				id,
			),
		},
		{
			name:  "accepts mixed supplied and omitted IDs",
			input: transactions.GroupInputs("ns-test", nil, newTransaction(id, ulid.Make().String()), newTransaction("")),
		},
		{
			name:             "missing group",
			expectedMessages: []string{"transaction group is required"},
		},
		{
			name:             "missing namespace",
			input:            transactions.GroupInputs("", nil, newTransaction("")),
			expectedMessages: []string{"namespace is required"},
		},
		{
			name:          "empty group",
			input:         transactions.GroupInputs("ns-test", nil),
			expectedError: ledger.ErrTransactionGroupEmpty,
		},
		{
			name: "invalid group ID",
			input: ledger.WithGroupID(
				transactions.GroupInputs("ns-test", nil, newTransaction("")),
				"invalid",
			),
			expectedMessages: []string{"group ID"},
		},
		{
			name:             "invalid transaction ID",
			input:            transactions.GroupInputs("ns-test", nil, newTransaction("invalid")),
			expectedMessages: []string{"transactions[0]:", "transaction ID"},
		},
		{
			name:          "invalid entry ID",
			input:         transactions.GroupInputs("ns-test", nil, newTransaction("", "invalid")),
			expectedError: ledger.ErrEntryInvalid,
		},
		{
			name:             "duplicate transaction IDs",
			input:            transactions.GroupInputs("ns-test", nil, newTransaction(id), newTransaction(id)),
			expectedMessages: []string{"transactions[1]: duplicate transaction ID"},
		},
		{
			name:             "duplicate entry IDs within a transaction",
			input:            transactions.GroupInputs("ns-test", nil, newTransaction("", id, id)),
			expectedMessages: []string{"transactions[0].entries[1]: duplicate entry ID"},
		},
		{
			name:             "duplicate entry IDs across transactions",
			input:            transactions.GroupInputs("ns-test", nil, newTransaction("", id), newTransaction("", id)),
			expectedMessages: []string{"transactions[1].entries[0]: duplicate entry ID"},
		},
		{
			name:          "missing transaction",
			input:         transactions.GroupInputs("ns-test", nil, nil),
			expectedError: ledger.ErrTransactionInputRequired,
		},
		{
			name: "missing entry",
			input: transactions.GroupInputs("ns-test", nil,
				ledger.WithEntryInputs(newTransaction(""), nil),
			),
			expectedError: ledger.ErrEntryInvalid,
		},
		{
			name:          "unbalanced transaction",
			input:         transactions.GroupInputs("ns-test", nil, unbalanced),
			expectedError: ledger.ErrInvalidTransactionTotal,
		},
		{
			name: "reports independent scope and posting errors together",
			input: ledger.WithGroupID(
				transactions.GroupInputs("", nil, newTransaction("invalid"), newTransaction("", "invalid")),
				"invalid",
			),
			expectedMessages: []string{
				"namespace is required",
				"group ID",
				"transactions[0]:",
				"transaction ID",
				"transactions[1]: ledger entry is invalid",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ledger.ValidateTransactionGroupInputWith(t.Context(), tc.input, nil)
			if tc.expectedError == nil && len(tc.expectedMessages) == 0 {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
			}

			for _, message := range tc.expectedMessages {
				require.ErrorContains(t, err, message)
			}
		})
	}

	t.Run("validates routing for every transaction", func(t *testing.T) {
		// given: a group with two independently balanced transactions
		first, second := newTransaction(""), newTransaction("")
		group := transactions.GroupInputs("ns-test", nil, first, second)
		validator := &groupValidationRoutingValidator{}

		// when: the group is validated with an injected routing policy
		err := ledger.ValidateTransactionGroupInputWith(t.Context(), group, validator)
		require.NoError(t, err)

		// then: routing sees each transaction's complete postings
		require.Equal(t, [][]ledger.EntryInput{first.EntryInputs(), second.EntryInputs()}, validator.entries)
	})

	t.Run("propagates routing rejection", func(t *testing.T) {
		// given: the routing policy rejects a balanced posting
		group := transactions.GroupInputs("ns-test", nil, newTransaction(""))
		routingError := errors.New("routing rejected")
		validator := &groupValidationRoutingValidator{err: routingError}

		// when: the group is validated
		err := ledger.ValidateTransactionGroupInputWith(t.Context(), group, validator)

		// then: the policy's error is retained with transaction context
		require.ErrorIs(t, err, routingError)
		require.ErrorContains(t, err, "transactions[0]")
	})
}

type groupValidationRoutingValidator struct {
	entries [][]ledger.EntryInput
	err     error
}

var _ ledger.RoutingValidator = (*groupValidationRoutingValidator)(nil)

func (v *groupValidationRoutingValidator) ValidateEntries(entries []ledger.EntryInput) error {
	v.entries = append(v.entries, entries)

	return v.err
}

func TestValidateTransactionInputCurrencyAccounting(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		currency currencies.CurrencyReference
		amount   string
		wantErr  bool
	}{
		{
			name:     "accepts fiat precision",
			currency: currencies.NewCurrencyReference("USD"),
			amount:   "10.01",
		},
		{
			name:     "rejects excess fiat precision",
			currency: currencies.NewCurrencyReference("USD"),
			amount:   "10.001",
			wantErr:  true,
		},
		{
			name:     "accepts custom precision matching the route's declared precision",
			currency: mustCustomCurrencyReference(t, "CREDITS", 3),
			amount:   "10.001",
		},
		{
			name:     "rejects custom precision exceeding the route's declared precision",
			currency: mustCustomCurrencyReference(t, "CREDITS", 2),
			amount:   "10.001",
			wantErr:  true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			amount := mustDecimal(t, testCase.amount)
			address := mustPostingAddressWithCurrencyReference(t, testCase.currency)
			txInput := &testutils.AnyTransactionInput{
				BookedAtValue: time.Now(),
				EntryInputsValues: []*testutils.AnyEntryInput{
					{
						Address:     address,
						AmountValue: amount,
					},
					{
						Address:     address,
						AmountValue: amount.Neg(),
					},
				},
			}

			err := ledger.ValidateTransactionInput(t.Context(), txInput)
			if testCase.wantErr {
				require.ErrorIs(t, err, ledger.ErrTransactionAmountInvalid)

				return
			}

			require.NoError(t, err)
		})
	}

	t.Run("rejects a globally balanced transaction that is unbalanced by currency", func(t *testing.T) {
		amount := mustDecimal(t, "25")
		txInput := &testutils.AnyTransactionInput{
			BookedAtValue: time.Now(),
			EntryInputsValues: []*testutils.AnyEntryInput{
				{
					Address:     mustPostingAddress(t, currencyx.Code("USD")),
					AmountValue: amount,
				},
				{
					Address:     mustPostingAddressWithCurrencyReference(t, mustCustomCurrencyReference(t, "ACME", 2)),
					AmountValue: amount.Neg(),
				},
			},
		}

		err := ledger.ValidateTransactionInput(t.Context(), txInput)
		require.Error(t, err)
		require.ErrorIs(t, err, ledger.ErrInvalidTransactionTotal)
	})
}

func TestListTransactionsInputValidateRouteFilter(t *testing.T) {
	costBasis := alpacadecimal.NewFromFloat(0.7)
	costBasisCurrency := currencyx.Code("USD")
	taxCode := "vat"
	taxBehavior := ledger.TaxBehaviorInclusive
	creditPriority := 1
	authStatus := ledger.TransactionAuthorizationStatusAuthorized

	tests := []struct {
		name    string
		route   ledger.RouteFilter
		wantErr bool
	}{
		{
			name: "currency route filter is supported",
			route: ledger.RouteFilter{
				Currency: currencies.NewCurrencyReference(currencyx.Code("USD")),
			},
		},
		{
			name: "exchange source route filter is supported",
			route: ledger.RouteFilter{
				CostBasisCurrency: mo.Some(&costBasisCurrency),
			},
		},
		{
			name: "source-less route filter is supported",
			route: ledger.RouteFilter{
				CostBasisCurrency: mo.Some[*currencyx.Code](nil),
			},
		},
		{
			name: "exact features route filter is supported",
			route: ledger.RouteFilter{
				Features: mo.Some([]string{"feature-a"}),
			},
		},
		{
			name: "match feature route filter is supported",
			route: ledger.RouteFilter{
				MatchFeature: "feature-a",
			},
		},
		{
			name: "cost basis route filter is rejected",
			route: ledger.RouteFilter{
				CostBasis: mo.Some(&costBasis),
			},
			wantErr: true,
		},
		{
			name: "tax code route filter is rejected",
			route: ledger.RouteFilter{
				TaxCode: mo.Some(&taxCode),
			},
			wantErr: true,
		},
		{
			name: "tax behavior route filter is rejected",
			route: ledger.RouteFilter{
				TaxBehavior: mo.Some(&taxBehavior),
			},
			wantErr: true,
		},
		{
			name: "credit priority route filter is rejected",
			route: ledger.RouteFilter{
				CreditPriority: &creditPriority,
			},
			wantErr: true,
		},
		{
			name: "transaction authorization route filter is rejected",
			route: ledger.RouteFilter{
				TransactionAuthorizationStatus: &authStatus,
			},
			wantErr: true,
		},
		{
			name: "exact features and match feature cannot be combined",
			route: ledger.RouteFilter{
				Features:     mo.Some([]string{"feature-a"}),
				MatchFeature: "feature-a",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ledger.ListTransactionsInput{
				Namespace: "ns-test",
				Limit:     1,
				EntryFilter: ledger.TransactionEntryFilter{
					Route: tt.route,
				},
				ReturnOnlyMatchingEntries: true,
			}.Validate()

			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, ledger.ErrListTransactionsInputInvalid)

				return
			}

			require.NoError(t, err)
		})
	}
}

func mustCustomCurrencyReference(t *testing.T, code currencyx.Code, precision int) currencies.CurrencyReference {
	t.Helper()

	reference, err := currencies.ParseCurrencyReference([]byte(fmt.Sprintf("custom|v1|%s|custom-currency-id|%d", code, precision)))
	require.NoError(t, err)

	return reference
}

func mustPostingAddress(t *testing.T, currency currencyx.Code) ledger.PostingAddress {
	t.Helper()

	return mustPostingAddressWithCurrencyReference(t, currencies.NewCurrencyReference(currency))
}

func mustPostingAddressWithCurrencyReference(t *testing.T, currency currencies.CurrencyReference) ledger.PostingAddress {
	t.Helper()

	route := ledger.Route{Currency: currency}
	key, err := ledger.BuildRoutingKey(route)
	require.NoError(t, err)

	address, err := ledgeraccount.NewAddressFromData(ledgeraccount.AddressData{
		SubAccountID: "sub_" + string(currency.Code),
		AccountType:  ledger.AccountTypeCustomerFBO,
		Route:        route,
		RouteID:      "route_" + string(currency.Code),
		RoutingKey:   key,
	})
	require.NoError(t, err)

	return address
}

func mustDecimal(t *testing.T, raw string) alpacadecimal.Decimal {
	t.Helper()

	value, err := alpacadecimal.NewFromString(raw)
	require.NoError(t, err)

	return value
}

func TestOriginIdentityVersionAndPersistenceContract(t *testing.T) {
	origin := "01J00000000000000000000001"
	spend := "01J00000000000000000000002"
	source := "01J00000000000000000000003"
	parts := ledger.EntryIdentityParts{
		Provenance: ledger.Provenance{
			CollectionOriginID: &origin,
			SpendChargeID:      &spend,
			SourceChargeID:     &source,
		},
		CorrectionSource: lo.ToPtr("entry:original"),
	}
	key, version := parts.Text()
	require.Equal(t, ledger.EntryIdentityVersion3, version)

	parsedVersion, parsed, err := key.Parse()
	require.NoError(t, err)
	require.Equal(t, version, parsedVersion)
	require.Equal(t, parts, parsed)

	entry := validationEntryInput{
		identityKey:        string(key),
		schemaVersion:      ledger.EntrySchemaVersionOrigin,
		collectionOriginID: &origin,
		spendChargeID:      &spend,
		sourceChargeID:     &source,
	}
	require.NoError(t, ledger.ValidateEntryIdentityKey(entry))

	for _, test := range []struct {
		name    string
		mutate  func(*validationEntryInput)
		message string
	}{
		{"origin column dropped", func(e *validationEntryInput) { e.collectionOriginID = nil }, "origin"},
		{"origin column mismatched", func(e *validationEntryInput) { e.collectionOriginID = lo.ToPtr("01J00000000000000000000004") }, "does not match"},
		{"new identity in old schema", func(e *validationEntryInput) { e.schemaVersion = ledger.EntrySchemaVersionCurrent }, "origin"},
		{"empty origin", func(e *validationEntryInput) { e.collectionOriginID = lo.ToPtr("") }, "origin"},
		{"invalid origin", func(e *validationEntryInput) { e.collectionOriginID = lo.ToPtr("not-an-origin") }, "origin"},
	} {
		t.Run(test.name, func(t *testing.T) {
			copy := entry
			test.mutate(&copy)
			require.ErrorContains(t, ledger.ValidateEntryIdentityKey(copy), test.message)
		})
	}

	_, _, err = ledger.EntryIdentityKeyText("entry-identity:v3:too|few").Parse()
	require.Error(t, err)
}

func TestOriginProvenanceCannotLeakBetweenBalancedPairs(t *testing.T) {
	origin := "01J00000000000000000000001"
	spend := "01J00000000000000000000002"
	source := "01J00000000000000000000003"
	route := ledger.Route{Currency: currencies.NewCurrencyReference(currencyx.Code("USD"))}
	negativeEntry := validationEntryInput{
		amount:             alpacadecimal.NewFromInt(-10),
		collectionOriginID: &origin,
		spendChargeID:      &spend,
		sourceChargeID:     &source,
		address:            testEntryIdentityAddress(t, ledger.AccountTypeCustomerAccrued, "accrued", route),
	}
	positiveEntry := negativeEntry
	positiveEntry.amount = negativeEntry.amount.Neg()
	positiveEntry.address = testEntryIdentityAddress(t, ledger.AccountTypeEarnings, "earnings", route)
	require.NoError(t, ledger.ValidateOriginProvenance([]ledger.EntryInput{negativeEntry, positiveEntry}))

	for _, test := range []struct {
		name    string
		mutate  func(*validationEntryInput)
		message string
	}{
		{"dropped origin", func(e *validationEntryInput) { e.collectionOriginID = nil }, "balance independently"},
		{"different origin", func(e *validationEntryInput) { e.collectionOriginID = lo.ToPtr("01J00000000000000000000004") }, "balance independently"},
		{"dropped spend", func(e *validationEntryInput) { e.spendChargeID = nil }, "spend_charge_id"},
		{"different spend", func(e *validationEntryInput) { e.spendChargeID = lo.ToPtr("another-spend") }, "preserve spend"},
		{"different purchase", func(e *validationEntryInput) { e.sourceChargeID = lo.ToPtr("another-source") }, "preserve source"},
	} {
		t.Run(test.name, func(t *testing.T) {
			copy := positiveEntry
			test.mutate(&copy)
			require.ErrorContains(t, ledger.ValidateOriginProvenance([]ledger.EntryInput{negativeEntry, copy}), test.message)
		})
	}

	// Attribution can change source within accrued without moving value to another origin.
	translated := positiveEntry
	translated.address = negativeEntry.address
	translated.sourceChargeID = lo.ToPtr("another-source")
	require.ErrorContains(t, ledger.ValidateOriginProvenance([]ledger.EntryInput{negativeEntry, translated}), "attribute unknown")

	unknownSourceEntry := negativeEntry
	unknownSourceEntry.sourceChargeID = nil
	require.NoError(t, ledger.ValidateOriginProvenance([]ledger.EntryInput{unknownSourceEntry, translated}))

	// One recognition can contain distinct backing sources of the same advance.
	secondNegativeEntry, secondPositiveEntry := negativeEntry, positiveEntry
	secondNegativeEntry.sourceChargeID, secondPositiveEntry.sourceChargeID = translated.sourceChargeID, translated.sourceChargeID
	require.NoError(t, ledger.ValidateOriginProvenance([]ledger.EntryInput{negativeEntry, positiveEntry, secondNegativeEntry, secondPositiveEntry}))
}
