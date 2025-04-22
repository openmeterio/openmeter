package adapter

import (
	"errors"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	addonadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromPlanAddonRow(a entdb.PlanAddon) (*planaddon.PlanAddon, error) {
	planAddon := &planaddon.PlanAddon{
		NamespacedID: models.NamespacedID{
			Namespace: a.Namespace,
			ID:        a.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
			DeletedAt: a.DeletedAt,
		},
		PlanAddonMeta: productcatalog.PlanAddonMeta{
			Metadata:    a.Metadata,
			Annotations: a.Annotations,
			PlanAddonConfig: productcatalog.PlanAddonConfig{
				FromPlanPhase: a.FromPlanPhase,
				MaxQuantity:   a.MaxQuantity,
			},
		},
	}

	// Set Plan
	planAddon.Plan = plan.Plan{
		NamespacedID: models.NamespacedID{
			Namespace: a.Namespace,
			ID:        a.PlanID,
		},
	}

	if a.Edges.Plan != nil {
		p, err := planadapter.FromPlanRow(*a.Edges.Plan)
		if err != nil {
			return nil, fmt.Errorf("failed to cast plan: %w", err)
		}

		if p == nil {
			return nil, errors.New("failed to cast plan: plan is nil")
		}

		planAddon.Plan = *p
	}

	// Set Addon
	planAddon.Addon = addon.Addon{
		NamespacedID: models.NamespacedID{
			Namespace: a.Namespace,
			ID:        a.AddonID,
		},
	}

	if a.Edges.Addon != nil {
		aa, err := addonadapter.FromAddonRow(*a.Edges.Addon)
		if err != nil {
			return nil, fmt.Errorf("failed to cast add-on: %w", err)
		}

		if aa == nil {
			return nil, errors.New("failed to cast add-on: add-on is nil")
		}

		planAddon.Addon = *aa
	}

	return planAddon, nil
}
