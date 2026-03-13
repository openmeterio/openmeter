package account

import (
	"context"
	"errors"
	"fmt"
	"strconv"

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

	ListSubAccounts(ctx context.Context, input ListSubAccountsInput) ([]*SubAccount, error)
	ListAccounts(ctx context.Context, input ListAccountsInput) ([]*Account, error)

	GetDimensionByKeyAndValue(ctx context.Context, namespace string, key ledger.DimensionKey, value string) (*DimensionData, error)
}

type ListAccountsInput struct {
	Namespace    string
	AccountTypes []ledger.AccountType
}

// TODO: we could do a better API than this :)
type ListSubAccountsInput struct {
	Namespace string
	AccountID string

	// Currency is always enforced.
	// CreditPriority is additionally supported for customer_fbo sub-account lookup.
	// DEFERRED: tax/feature filters are accepted for forward compatibility and ignored.
	Dimensions ledger.QueryDimensions
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
	CurrencyDimensionID string
	// CreditPriorityDimensionID is meaningful / allowed only for customer_fbo.
	// DEFERRED: tax/feature are accepted for forward compatibility and currently inactive.
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

	if accountType == ledger.AccountTypeCustomerFBO {
		if d.CreditPriorityDimensionID == nil {
			return models.NewGenericValidationError(errors.New("credit priority dimension id is required for customer_fbo"))
		}
	}

	if accountType != ledger.AccountTypeCustomerFBO && d.CreditPriorityDimensionID != nil {
		return models.NewGenericValidationError(fmt.Errorf("credit priority dimension is only allowed for customer_fbo accounts"))
	}

	return nil
}

func DefaultCustomerFBOPriorityDimensionValue() string {
	return strconv.Itoa(ledger.DefaultCustomerFBOPriority)
}

type CreateDimensionInput struct {
	Namespace   string
	Annotations models.Annotations
	// Dimensions are externally owned (tax/currency/feature systems).
	// Ledger stores local dimension rows for routing and referential integrity.
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
