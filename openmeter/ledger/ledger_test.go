package ledger_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/stretchr/testify/require"
)

// This package aims to test if the primitive API is convenient to work with

type exampleEntryInput struct {
	account ledger.Address
	amount  alpacadecimal.Decimal
	typ     ledger.EntryType
}

func (e exampleEntryInput) Type() ledger.EntryType {
	return e.typ
}

func (e exampleEntryInput) Account() ledger.Address {
	return e.account
}

func (e exampleEntryInput) Amount() alpacadecimal.Decimal {
	return e.amount
}

var _ ledger.LedgerEntryInput = exampleEntryInput{}

func TestTwoAccountTransaction(t *testing.T) {
	t.Skipf("This is just to assert the types, it would fail on unimplemented")

	var l ledger.Ledger
	var a1, a2 ledger.Account

	// Let's create a TX between two accounts
	tx, err := l.SetUpTransaction(t.Context(), time.Now(), []ledger.LedgerEntryInput{
		exampleEntryInput{
			account: a1.Address(),
			amount:  alpacadecimal.NewFromInt(100),
			typ:     ledger.EntryTypeCredit,
		},
		exampleEntryInput{
			account: a2.Address(),
			amount:  alpacadecimal.NewFromInt(100),
			typ:     ledger.EntryTypeDebit,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, tx)

	err = l.CommitGroup(t.Context(), tx.AsGroup(nil))
	require.NoError(t, err)
}

func TestMultiAccountTransaction(t *testing.T) {
	t.Skipf("This is just to assert the types, it would fail on unimplemented")

	var l ledger.Ledger
	var a1, a2, a3 ledger.Account

	// Let's create a TX between multiple
	tx, err := l.SetUpTransaction(t.Context(), time.Now(), []ledger.LedgerEntryInput{
		exampleEntryInput{
			account: a1.Address(),
			amount:  alpacadecimal.NewFromInt(100),
			typ:     ledger.EntryTypeCredit,
		},
		exampleEntryInput{
			account: a2.Address(),
			amount:  alpacadecimal.NewFromInt(50),
			typ:     ledger.EntryTypeDebit,
		},
		exampleEntryInput{
			account: a3.Address(),
			amount:  alpacadecimal.NewFromInt(49),
			typ:     ledger.EntryTypeDebit,
		},
	})

	require.Nil(t, tx)

	// Just an example on checking errors... 99 - 100 <> 0
	found := false
	if issues, err := models.AsValidationIssues(err); err != nil {
		for _, issue := range issues {
			if issue.Code() == ledger.ErrCodeInvalidTransactionTotal {
				// Just
				found = true
			}
		}
	}
	require.True(t, found, "expected validation issue not found, got %v", err)
}

func TestGetAccountBalance(t *testing.T) {
	t.Skipf("This is just to assert the types, it would fail on unimplemented")

	var l ledger.Ledger
	var addr ledger.Address

	acc, err := l.GetAccount(t.Context(), addr)
	require.NoError(t, err)

	balance, err := acc.GetBalance(t.Context())
	require.NoError(t, err)
	require.NotNil(t, balance)
}
