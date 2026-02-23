package taxcode

import (
	"errors"
	"regexp"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
)

var TaxCodeStripeRegexp = regexp.MustCompile(`^txcd_\d{8}$`)

// TaxCodeAppMapping represents a mapping of an app type to a tax code.
type TaxCodeAppMapping struct {
	AppType app.AppType `json:"app_type"`
	TaxCode string      `json:"tax_code"`
}

func (t TaxCodeAppMapping) Validate() error {
	var errs []error

	if err := t.AppType.Validate(); err != nil {
		errs = append(errs, err)
	}

	if t.TaxCode == "" {
		errs = append(errs, ErrTaxCodeEmpty)
	}

	switch t.AppType {
	case app.AppTypeStripe:
		if !TaxCodeStripeRegexp.MatchString(t.TaxCode) {
			errs = append(errs, ErrTaxCodeStripeInvalid)
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// TaxCodeAppMappings is a list of TaxCodeAppMapping.
type TaxCodeAppMappings []TaxCodeAppMapping

func (t TaxCodeAppMappings) Validate() error {
	var errs []error

	appTypes := lo.UniqBy(t, func(t TaxCodeAppMapping) app.AppType {
		return t.AppType
	})

	if len(appTypes) != len(t) {
		errs = append(errs, ErrAppTypesMustBeUnique)
	}

	for _, t := range t {
		if err := t.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// TaxCode represents a tax code with mappings to app types.
type TaxCode struct {
	models.NamespacedID
	models.ManagedModel

	// Key is the unique key for TaxCode.
	Key string `json:"key"`

	// Name is the display name for TaxCode.
	Name string `json:"name"`

	// Description is the description for TaxCode.
	Description *string `json:"description,omitempty"`

	// AppMappings is the mapping of app types to tax codes.
	AppMappings TaxCodeAppMappings `json:"app_mappings"`

	// Metadata
	Metadata models.Metadata `json:"metadata,omitempty"`
}

func (t TaxCode) Validate() error {
	var errs []error

	if err := t.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := t.ManagedModel.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := t.AppMappings.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
