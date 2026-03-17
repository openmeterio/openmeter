package account

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	CreateAccount(ctx context.Context, input CreateAccountInput) (*Account, error)
	EnsureSubAccount(ctx context.Context, input CreateSubAccountInput) (*SubAccount, error)

	GetAccountByID(ctx context.Context, id models.NamespacedID) (*Account, error)
	GetSubAccountByID(ctx context.Context, id models.NamespacedID) (*SubAccount, error)

	ListSubAccounts(ctx context.Context, input ListSubAccountsInput) ([]*SubAccount, error)
	ListAccounts(ctx context.Context, input ListAccountsInput) ([]*Account, error)
}

type ListAccountsInput struct {
	Namespace    string
	AccountTypes []ledger.AccountType
}

type ListSubAccountsInput struct {
	Namespace string
	AccountID string

	Route ledger.RouteFilter
}

type CreateAccountInput struct {
	Namespace   string
	Type        ledger.AccountType
	Annotations models.Annotations
}

func (c CreateAccountInput) Validate() error {
	if err := c.Type.Validate(); err != nil {
		return err
	}

	return nil
}

type CreateSubAccountInput struct {
	Namespace   string
	AccountID   string
	Annotations models.Annotations
	Route       ledger.Route
}

func (c CreateSubAccountInput) Validate() error {
	if c.AccountID == "" {
		return models.NewGenericValidationError(errors.New("account id is required"))
	}

	if c.Namespace == "" {
		return models.NewGenericValidationError(errors.New("namespace is required"))
	}

	if err := c.Route.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	return nil
}
