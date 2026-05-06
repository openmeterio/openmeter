package taxcode

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	TaxCodeService
	OrganizationDefaultTaxCodesService
}

type TaxCodeService interface {
	CreateTaxCode(ctx context.Context, input CreateTaxCodeInput) (TaxCode, error)
	UpdateTaxCode(ctx context.Context, input UpdateTaxCodeInput) (TaxCode, error)
	ListTaxCodes(ctx context.Context, input ListTaxCodesInput) (pagination.Result[TaxCode], error)
	GetTaxCode(ctx context.Context, input GetTaxCodeInput) (TaxCode, error)
	GetTaxCodeByAppMapping(ctx context.Context, input GetTaxCodeByAppMappingInput) (TaxCode, error)
	GetOrCreateByAppMapping(ctx context.Context, input GetOrCreateByAppMappingInput) (TaxCode, error)
	DeleteTaxCode(ctx context.Context, input DeleteTaxCodeInput) error
}

type OrganizationDefaultTaxCodesService interface {
	GetOrganizationDefaultTaxCodes(ctx context.Context, input GetOrganizationDefaultTaxCodesInput) (OrganizationDefaultTaxCodes, error)
	UpsertOrganizationDefaultTaxCodes(ctx context.Context, input UpsertOrganizationDefaultTaxCodesInput) (OrganizationDefaultTaxCodes, error)
}

type inputOptions struct {
	AllowAnnotations bool
}

var (
	_ models.Validator = (*CreateTaxCodeInput)(nil)
	_ models.Validator = (*UpdateTaxCodeInput)(nil)
	_ models.Validator = (*ListTaxCodesInput)(nil)
	_ models.Validator = (*GetTaxCodeInput)(nil)
	_ models.Validator = (*GetTaxCodeByAppMappingInput)(nil)
	_ models.Validator = (*GetOrCreateByAppMappingInput)(nil)
	_ models.Validator = (*DeleteTaxCodeInput)(nil)
	_ models.Validator = (*GetOrganizationDefaultTaxCodesInput)(nil)
	_ models.Validator = (*UpsertOrganizationDefaultTaxCodesInput)(nil)
)

type CreateTaxCodeInput struct {
	Namespace   string
	Key         string
	Name        string
	Description *string
	AppMappings TaxCodeAppMappings
	Metadata    models.Metadata
	Annotations models.Annotations
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
	Annotations models.Annotations

	inputOptions
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

type GetTaxCodeByAppMappingInput struct {
	Namespace string
	AppType   app.AppType
	TaxCode   string
}

func (i GetTaxCodeByAppMappingInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, ErrResourceNamespaceEmpty)
	}

	if err := i.AppType.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.TaxCode == "" {
		errs = append(errs, ErrTaxCodeEmpty)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetOrCreateByAppMappingInput struct {
	Namespace string
	AppType   app.AppType
	TaxCode   string
}

func (i GetOrCreateByAppMappingInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, ErrResourceNamespaceEmpty)
	}

	if err := i.AppType.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.TaxCode == "" {
		errs = append(errs, ErrTaxCodeEmpty)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type DeleteTaxCodeInput struct {
	models.NamespacedID

	inputOptions
}

func (i DeleteTaxCodeInput) Validate() error {
	var errs []error
	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetOrganizationDefaultTaxCodesInput struct {
	Namespace string
}

func (i GetOrganizationDefaultTaxCodesInput) Validate() error {
	if i.Namespace == "" {
		return models.NewNillableGenericValidationError(ErrResourceNamespaceEmpty)
	}

	return nil
}

type UpsertOrganizationDefaultTaxCodesInput struct {
	Namespace            string
	InvoicingTaxCodeID   string
	CreditGrantTaxCodeID string
}

func (i UpsertOrganizationDefaultTaxCodesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, ErrResourceNamespaceEmpty)
	}

	if i.InvoicingTaxCodeID == "" {
		errs = append(errs, ErrResourceIDEmpty.WithPathString("invoicing_tax_code_id"))
	}

	if i.CreditGrantTaxCodeID == "" {
		errs = append(errs, ErrResourceIDEmpty.WithPathString("credit_grant_tax_code_id"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
