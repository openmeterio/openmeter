package taxcode

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	CreateTaxCode(ctx context.Context, input CreateTaxCodeInput) (TaxCode, error)
	UpdateTaxCode(ctx context.Context, input UpdateTaxCodeInput) (TaxCode, error)
	ListTaxCodes(ctx context.Context, input ListTaxCodesInput) (pagination.Result[TaxCode], error)
	GetTaxCode(ctx context.Context, input GetTaxCodeInput) (TaxCode, error)
	DeleteTaxCode(ctx context.Context, input DeleteTaxCodeInput) error
}

var (
	_ models.Validator = (*CreateTaxCodeInput)(nil)
	_ models.Validator = (*UpdateTaxCodeInput)(nil)
	_ models.Validator = (*ListTaxCodesInput)(nil)
	_ models.Validator = (*GetTaxCodeInput)(nil)
	_ models.Validator = (*DeleteTaxCodeInput)(nil)
)

type CreateTaxCodeInput struct {
	Namespace   string
	Key         string
	Name        string
	Description *string
	AppMappings TaxCodeAppMappings
	Metadata    models.Metadata
}

func (i CreateTaxCodeInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, ErrResourceNamespaceEmpty)
	}

	if i.Key == "" {
		errs = append(errs, ErrResourceKeyEmpty)
	}

	if i.Name == "" {
		errs = append(errs, ErrResourceNameEmpty)
	}

	if err := i.AppMappings.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type UpdateTaxCodeInput struct {
	models.NamespacedID

	Name        string
	Description *string
	AppMappings TaxCodeAppMappings
	Metadata    models.Metadata
}

func (i UpdateTaxCodeInput) Validate() error {
	var errs []error

	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Name == "" {
		errs = append(errs, ErrResourceNameEmpty)
	}

	if err := i.AppMappings.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ListTaxCodesInput struct {
	Namespace string
	pagination.Page

	IncludeDeleted bool
}

func (i ListTaxCodesInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, ErrResourceNamespaceEmpty)
	}

	if !i.Page.IsZero() {
		if err := i.Page.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetTaxCodeInput struct {
	models.NamespacedID
}

func (i GetTaxCodeInput) Validate() error {
	var errs []error
	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type DeleteTaxCodeInput struct {
	models.NamespacedID
}

func (i DeleteTaxCodeInput) Validate() error {
	var errs []error
	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
