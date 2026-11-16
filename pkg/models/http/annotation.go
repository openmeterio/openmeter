package http

import (
	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromAnnotations(annotations models.Annotations) *apiv1models.Annotations {
	return lo.ToPtr(apiv1models.Annotations(annotations))
}

func AsAnnotations(annotations *apiv1models.Annotations) models.Annotations {
	if annotations == nil {
		return nil
	}

	return models.Annotations(*annotations)
}
