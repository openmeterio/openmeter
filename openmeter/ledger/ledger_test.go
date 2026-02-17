package ledger_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

// This package aims to test if the primitive API is convenient to work with

type exampleEntryInput struct {
	account ledger.PostingAddress
	amount  alpacadecimal.Decimal
}

func (e exampleEntryInput) PostingAddress() ledger.PostingAddress {
	return e.account
}

func (e exampleEntryInput) Amount() alpacadecimal.Decimal {
	return e.amount
}

var _ ledger.EntryInput = exampleEntryInput{}

func TestTwoAccountTransaction(t *testing.T) {
	t.Skipf("This is just to assert the types, it would fail on unimplemented")

	var l ledger.Ledger
	var a1, a2 ledger.SubAccount

	// Let's create a TX between two accounts
	tx, err := l.SetUpTransactionInput(t.Context(), time.Now(), []ledger.EntryInput{
		exampleEntryInput{
			account: a1.Address(),
			amount:  alpacadecimal.NewFromInt(-100),
		},
		exampleEntryInput{
			account: a2.Address(),
			amount:  alpacadecimal.NewFromInt(100),
		},
	})

	require.NoError(t, err)
	require.NotNil(t, tx)

	_, err = l.CommitGroup(t.Context(), tx.AsGroupInput(nil))
	require.NoError(t, err)
}

func TestMultiAccountTransaction(t *testing.T) {
	t.Skipf("This is just to assert the types, it would fail on unimplemented")

	var l ledger.Ledger
	var a1, a2, a3 ledger.SubAccount

	// Let's create a TX between multiple
	tx, err := l.SetUpTransactionInput(t.Context(), time.Now(), []ledger.EntryInput{
		exampleEntryInput{
			account: a1.Address(),
			amount:  alpacadecimal.NewFromInt(-100),
		},
		exampleEntryInput{
			account: a2.Address(),
			amount:  alpacadecimal.NewFromInt(50),
		},
		exampleEntryInput{
			account: a3.Address(),
			amount:  alpacadecimal.NewFromInt(49),
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

	var acc ledger.Account

	balance, err := acc.GetBalance(t.Context(), ledger.QueryDimensions{
		CurrencyID: "01KHNVYVZ6FBKD6QKCGRX6S4Z4",
	})
	require.NoError(t, err)
	require.NotNil(t, balance)
}
