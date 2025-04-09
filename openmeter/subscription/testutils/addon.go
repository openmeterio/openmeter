package subscriptiontestutils

import (
	"context"
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/models"
)

var ExampleAddonRateCard1 = productcatalog.FlatFeeRateCard{
	RateCardMeta: productcatalog.RateCardMeta{
		Name:        "Test Addon Rate Card 1",
		Description: lo.ToPtr("Test Addon Rate Card 1 Description"),
		Key:         ExampleFeatureKey,
	},
	BillingCadence: &ISOMonth,
}

func GetExampleAddonInput(t *testing.T, effectivePeriod productcatalog.EffectivePeriod) addon.CreateAddonInput {
	t.Helper()

	inp := addon.CreateAddonInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: ExampleNamespace,
		},
		Addon: productcatalog.Addon{
			AddonMeta: productcatalog.AddonMeta{
				Name:            "Test Addon",
				Description:     lo.ToPtr("Test Addon Description"),
				EffectivePeriod: effectivePeriod,
				Key:             "test-addon",
				Version:         1,
				Currency:        currency.USD,
				InstanceType:    productcatalog.AddonInstanceTypeSingle,
				Metadata: models.NewMetadata(map[string]string{
					"test": "test",
				}),
			},
			RateCards: productcatalog.RateCards{
				&ExampleAddonRateCard1,
			},
		},
	}

	return inp
}

type testAddonService struct {
	addon.Service
}

func NewTestAddonService(svc addon.Service) *testAddonService {
	return &testAddonService{svc}
}

func (s *testAddonService) CreateExampleAddon(t *testing.T, period productcatalog.EffectivePeriod) addon.Addon {
	t.Helper()

	add, err := s.CreateAddon(context.Background(), GetExampleAddonInput(t, period))
	if err != nil {
		t.Fatalf("failed to create addon: %v", err)
	}

	add, err = s.PublishAddon(context.Background(), addon.PublishAddonInput{
		NamespacedID:    add.NamespacedID,
		EffectivePeriod: period,
	})
	if err != nil {
		t.Fatalf("failed to publish addon: %v", err)
	}

	return *add
}
