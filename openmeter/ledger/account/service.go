package account

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	CreateAccount(ctx context.Context, input CreateAccountInput) (*Account, error)
	CreateSubAccount(ctx context.Context, input CreateSubAccountInput) (*SubAccount, error)
	CreateDimension(ctx context.Context, input CreateDimensionInput) (*DimensionData, error)

	GetAccountByID(ctx context.Context, id models.NamespacedID) (*Account, error)
	GetSubAccountByID(ctx context.Context, id models.NamespacedID) (*SubAccount, error)
	GetDimensionByID(ctx context.Context, id models.NamespacedID) (*DimensionData, error)
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
	Dimensions  SubAccountDimensionInput
}

func (c CreateSubAccountInput) Validate() error {
	if c.AccountID == "" {
		return models.NewGenericValidationError(errors.New("account id is required"))
	}

	if c.Namespace == "" {
		return models.NewGenericValidationError(errors.New("namespace is required"))
	}

	if err := c.Dimensions.Validate(); err != nil {
		return err
	}

	return nil
}

type SubAccountDimensionInput struct {
	CurrencyDimensionID       string
	TaxCodeDimensionID        *string
	FeaturesDimensionID       *string
	CreditPriorityDimensionID *string
}

func (d SubAccountDimensionInput) Validate() error {
	if d.CurrencyDimensionID == "" {
		return models.NewGenericValidationError(errors.New("currency dimension id is required"))
	}

	return nil
}

func (d SubAccountDimensionInput) ValidateForAccountType(accountType ledger.AccountType) error {
	if err := d.Validate(); err != nil {
		return err
	}

	switch accountType {
	case ledger.AccountTypeCustomerFBO, ledger.AccountTypeCustomerReceivable, ledger.AccountTypeCustomerBreakage:
		if d.TaxCodeDimensionID == nil {
			return models.NewGenericValidationError(errors.New("tax code dimension id is required"))
		}

		if d.FeaturesDimensionID == nil {
			return models.NewGenericValidationError(errors.New("features dimension id is required"))
		}

		if d.CreditPriorityDimensionID == nil {
			return models.NewGenericValidationError(errors.New("credit priority dimension id is required"))
		}
	}

	return nil
}

type CreateDimensionInput struct {
	Namespace    string
	Annotations  models.Annotations
	Key          string
	Value        string
	DisplayValue string
}

func (c CreateDimensionInput) Validate() error {
	if c.Namespace == "" {
		return models.NewGenericValidationError(errors.New("namespace is required"))
	}

	if c.Key == "" {
		return models.NewGenericValidationError(errors.New("key is required"))
	}

	if c.Value == "" {
		return models.NewGenericValidationError(errors.New("value is required"))
	}

	if c.DisplayValue == "" {
		return models.NewGenericValidationError(errors.New("display value is required"))
	}

	return nil
}
