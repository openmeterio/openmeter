package account

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Repo interface {
	entutils.TxCreator

	// CreateAccount creates a new account
	CreateAccount(ctx context.Context, input CreateAccountInput) (Account, error)

	// GetAccountByID returns the account by its ID
	GetAccountByID(ctx context.Context, id models.NamespacedID) (Account, error)

	// CreateDimension creates a new dimension
	CreateDimension(ctx context.Context, input CreateDimensionInput) (Dimension, error)

	// GetDimensionByID returns the dimension by its ID
	GetDimensionByID(ctx context.Context, id models.NamespacedID) (Dimension, error)
}

type CreateAccountInput struct {
	Namespace   string
	Type        ledger.AccountType
	Annotations models.Annotations
}

type CreateDimensionInput struct {
	Namespace   string
	Annotations models.Annotations
	Key         string
	Value       string // TBD
}
