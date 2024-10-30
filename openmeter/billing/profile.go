package billing

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/api"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type CreateWorkflowConfigInput struct {
	billingentity.WorkflowConfig
}

type CreateProfileInput struct {
	Namespace   string                        `json:"namespace"`
	Name        string                        `json:"name"`
	Description *string                       `json:"description,omitempty"`
	Metadata    map[string]string             `json:"metadata"`
	Supplier    billingentity.SupplierContact `json:"supplier"`
	Default     bool                          `json:"default"`

	WorkflowConfig billingentity.WorkflowConfig `json:"workflowConfig"`
	Apps           CreateProfileAppsInput       `json:"apps"`
}

func (i CreateProfileInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.Name == "" {
		return errors.New("name is required")
	}

	if err := i.Supplier.Validate(); err != nil {
		return fmt.Errorf("invalid supplier: %w", err)
	}

	if err := i.WorkflowConfig.Validate(); err != nil {
		return fmt.Errorf("invalid workflow config: %w", err)
	}

	if err := i.Apps.Validate(); err != nil {
		return fmt.Errorf("invalid apps: %w", err)
	}

	return nil
}

type CreateProfileAppsInput = billingentity.ProfileAppReferences

type ListProfilesResult = pagination.PagedResponse[billingentity.Profile]

type ListProfilesInput struct {
	pagination.Page

	Expand ProfileExpand

	Namespace       string
	IncludeArchived bool
	OrderBy         api.BillingProfileOrderBy
	Order           sortx.Order
}

func (i ListProfilesInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if err := i.Page.Validate(); err != nil {
		return fmt.Errorf("error validating page: %w", err)
	}

	if err := i.Expand.Validate(); err != nil {
		return fmt.Errorf("error validating expand: %w", err)
	}

	return nil
}

type ProfileExpand struct {
	Apps bool
}

var ProfileExpandAll = ProfileExpand{
	Apps: true,
}

func (e ProfileExpand) Validate() error {
	return nil
}

type GetDefaultProfileInput struct {
	Namespace string
}

func (i GetDefaultProfileInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	return nil
}

type genericNamespaceID struct {
	Namespace string
	ID        string
}

func (i genericNamespaceID) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.ID == "" {
		return errors.New("id is required")
	}

	return nil
}

type GetProfileInput struct {
	Profile models.NamespacedID
	Expand  ProfileExpand
}

func (i GetProfileInput) Validate() error {
	if i.Profile.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.Profile.ID == "" {
		return errors.New("id is required")
	}

	return nil
}

type DeleteProfileInput genericNamespaceID

func (i DeleteProfileInput) Validate() error {
	return genericNamespaceID(i).Validate()
}

type UpdateProfileInput billingentity.BaseProfile

func (i UpdateProfileInput) Validate() error {
	if i.ID == "" {
		return errors.New("id is required")
	}

	if i.AppReferences != nil {
		return errors.New("apps cannot be updated")
	}

	return billingentity.BaseProfile(i).Validate()
}

type UpdateProfileAdapterInput struct {
	TargetState      billingentity.BaseProfile
	WorkflowConfigID string
}

func (i UpdateProfileAdapterInput) Validate() error {
	if err := i.TargetState.Validate(); err != nil {
		return fmt.Errorf("error validating target state profile: %w", err)
	}

	if i.TargetState.ID == "" {
		return fmt.Errorf("id is required")
	}

	if i.TargetState.UpdatedAt.IsZero() {
		return fmt.Errorf("updated at is required")
	}

	if i.WorkflowConfigID == "" {
		return fmt.Errorf("workflow config id is required")
	}

	return nil
}
