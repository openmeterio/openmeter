package testutils

import (
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewTestAddon(t *testing.T, namespace string) addon.CreateAddonInput {
	t.Helper()

	return addon.CreateAddonInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Addon: productcatalog.Addon{
			AddonMeta: productcatalog.AddonMeta{
				Key:          "addon1",
				Name:         "Addon v1",
				Description:  lo.ToPtr("Addon v1"),
				Metadata:     models.Metadata{"name": "addon1"},
				Annotations:  models.Annotations{"key": "value"},
				Currency:     currency.USD,
				InstanceType: productcatalog.AddonInstanceTypeSingle,
			},
			RateCards: nil,
		},
	}
}
