package testutils

import (
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewTestPlan(t *testing.T, namespace string) plan.CreatePlanInput {
	t.Helper()

	return plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Key:         "pro",
				Name:        "Pro",
				Description: lo.ToPtr("Pro plan v1"),
				Metadata:    models.Metadata{"name": "pro"},
				Currency:    currency.USD,
			},
		},
	}
}
