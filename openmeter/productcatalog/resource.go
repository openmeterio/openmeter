package productcatalog

import (
	"errors"
	"strings"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Resource struct {
	Parent     *Resource          `json:"parent,omitempty"`
	Key        string             `json:"key"`
	Kind       string             `json:"kind"`
	Attributes models.Annotations `json:"attributes,omitempty"`
}

func (r Resource) AsPath() string {
	var parts []string

	if r.Parent != nil {
		parts = append(parts, r.Parent.AsPath())
	}

	parts = append(parts, []string{r.Kind, r.Key}...)

	return strings.Join(parts, "/")
}

// Validate validates the Resource.
func (r Resource) Validate() error {
	var errs []error

	if r.Key == "" {
		errs = append(errs, errors.New("missing Key"))
	}

	if r.Kind == "" {
		errs = append(errs, errors.New("missing Kind"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
