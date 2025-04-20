package testutils

import (
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewTestPlan(t *testing.T, namespace string, phases ...productcatalog.Phase) plan.CreatePlanInput {
	t.Helper()

	return plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Key:         "test",
				Name:        "Test",
				Description: lo.ToPtr("Test plan"),
				Metadata:    models.Metadata{"name": "test"},
				Currency:    currency.USD,
			},
			Phases: phases,
		},
	}
}
