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

	// CreateSubAccount creates a new sub-account
	CreateSubAccount(ctx context.Context, input CreateSubAccountInput) (*SubAccountData, error)

	// GetSubAccountByID returns the sub-account by its ID
	GetSubAccountByID(ctx context.Context, id models.NamespacedID) (*SubAccountData, error)

	// CreateDimension creates a new dimension
	CreateDimension(ctx context.Context, input CreateDimensionInput) (*DimensionData, error)

	// GetDimensionByID returns the dimension by its ID
	GetDimensionByID(ctx context.Context, id models.NamespacedID) (*DimensionData, error)
}
