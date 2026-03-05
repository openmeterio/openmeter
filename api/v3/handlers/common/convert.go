package common

import (
	"maps"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ConvertMetadataToLabels converts models.Metadata to api.Labels.
// Always returns an initialized map (never nil) so JSON serializes to {} instead of null.
func ConvertMetadataToLabels(source models.Metadata) *api.Labels {
	if len(source) == 0 {
		return &api.Labels{}
	}
	return lo.ToPtr((api.Labels)(maps.Clone(source)))
}
