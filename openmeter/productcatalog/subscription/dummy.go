package subscription

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
	modelref "github.com/openmeterio/openmeter/pkg/models/ref"
)

// Plan is a dummy representation that can be used internally
type Plan struct {
	models.NamespacedModel
	models.ManagedModel
	models.VersionedModel
	ID string `json:"id,omitempty"`
}

type PlanRepo interface {
	GetVersion(ctx context.Context, planRef modelref.VersionedKeyRef) (Plan, error)
	GetContentTemplates(ctx context.Context, planRef modelref.VersionedKeyRef) ([]PlanContent, error)
}

// Content is a dummy representation that can be used internally
type PlanContent struct{}

func PlanContentToContentCreateInput(c PlanContent) ContentCreateInput {
	panic("implement me")
}
