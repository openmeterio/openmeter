package account

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Repo interface {
	entutils.TxCreator

	// CreateAccount creates a new account
	CreateAccount(ctx context.Context, input CreateAccountInput) (*AccountData, error)

	// GetAccountByID returns the account by its ID
	GetAccountByID(ctx context.Context, id models.NamespacedID) (*AccountData, error)

	// EnsureSubAccount creates a new sub-account or returns the existing one if the route already exists.
	EnsureSubAccount(ctx context.Context, input CreateSubAccountInput) (*SubAccountData, error)

	// GetSubAccountByID returns the sub-account by its ID
	GetSubAccountByID(ctx context.Context, id models.NamespacedID) (*SubAccountData, error)

	// ListSubAccounts returns sub-accounts for account + route filters
	ListSubAccounts(ctx context.Context, input ListSubAccountsInput) ([]*SubAccountData, error)

	// ListAccounts returns accounts filtered by type within a namespace
	ListAccounts(ctx context.Context, input ListAccountsInput) ([]*AccountData, error)
}
