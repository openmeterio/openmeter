package testutils

import (
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewTestAddon(t *testing.T, namespace string, rateCards ...productcatalog.RateCard) addon.CreateAddonInput {
	t.Helper()

	return addon.CreateAddonInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Addon: productcatalog.Addon{
			AddonMeta: productcatalog.AddonMeta{
				Key:          "test-addon",
				Name:         "Test Addon",
				Description:  lo.ToPtr("Test Addon"),
				Metadata:     models.Metadata{"name": "test-addon"},
				Annotations:  models.Annotations{"name": "test-addon"},
				Currency:     currency.USD,
				InstanceType: productcatalog.AddonInstanceTypeSingle,
			},
			RateCards: rateCards,
		},
	}
}
