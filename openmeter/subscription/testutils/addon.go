package subscriptiontestutils

import (
	"context"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/models"
)

func BuildAddonForTesting(t *testing.T, period productcatalog.EffectivePeriod, typ productcatalog.AddonInstanceType, rcs ...productcatalog.RateCard) addon.CreateAddonInput {
	t.Helper()

	inp := addon.CreateAddonInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: ExampleNamespace,
		},
		Addon: productcatalog.Addon{
			AddonMeta: productcatalog.AddonMeta{
				Name:            "Test Addon",
				Description:     lo.ToPtr("Test Addon Description"),
				EffectivePeriod: period,
				Key:             "test-addon",
				Version:         1,
				Currency:        currency.USD,
				InstanceType:    typ,
				Metadata: models.NewMetadata(map[string]string{
					"test": "test",
				}),
			},
			RateCards: rcs,
		},
	}

	return inp
}

var ExampleAddonRateCard1 = productcatalog.FlatFeeRateCard{
	RateCardMeta: productcatalog.RateCardMeta{
		Name:        "Test Addon Rate Card 1",
		Description: lo.ToPtr("Test Addon Rate Card 1 Description"),
		Key:         ExampleFeatureKey,
		FeatureKey:  lo.ToPtr(ExampleFeatureKey),
	},
	BillingCadence: &ISOMonth,
}

var ExampleAddonRateCard2 = productcatalog.FlatFeeRateCard{
	RateCardMeta: productcatalog.RateCardMeta{
		Name:        "Test Addon Rate Card 2",
		Description: lo.ToPtr("Test Addon Rate Card 2 Description"),
		Key:         "addon-rc-key-2",
	},
	BillingCadence: &ISOMonth,
}

// Adds a bool entitlement and a 100 flat price without feature
var ExampleAddonRateCard3 = productcatalog.FlatFeeRateCard{
	RateCardMeta: productcatalog.RateCardMeta{
		Name:                "Test Addon Rate Card 3",
		Description:         lo.ToPtr("Test Addon Rate Card 3 Description"),
		Key:                 "addon-rc-key-3",
		EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}),
		Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(100),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		}),
	},
	BillingCadence: &ISOMonth,
}

// Adds a 100 flat price to ExampleFeatureKey2
var ExampleAddonRateCard4 = productcatalog.FlatFeeRateCard{
	RateCardMeta: productcatalog.RateCardMeta{
		Name:        "Test Addon Rate Card 4",
		Description: lo.ToPtr("Test Addon Rate Card 4 Description"),
		Key:         ExampleFeatureKey2,
		FeatureKey:  lo.ToPtr(ExampleFeatureKey2),
		Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(100),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		}),
	},
	BillingCadence: &ISOMonth,
}

// Adds 50 usage to ExampleFeatureKey
var ExampleAddonRateCard5 = productcatalog.FlatFeeRateCard{
	RateCardMeta: productcatalog.RateCardMeta{
		Name:        "Test Addon Rate Card 5",
		Description: lo.ToPtr("Test Addon Rate Card 5 Description"),
		Key:         ExampleFeatureKey,
		FeatureKey:  lo.ToPtr(ExampleFeatureKey),
		EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
			IssueAfterReset: lo.ToPtr(50.0),
			UsagePeriod:     ISOMonth,
		}),
	},
	BillingCadence: &ISOMonth,
}

// Adds a boolean entitlement to ExampleFeatureKey
var ExampleAddonRateCard6 = productcatalog.FlatFeeRateCard{
	RateCardMeta: productcatalog.RateCardMeta{
		Name:                "Test Addon Rate Card 6",
		Description:         lo.ToPtr("Test Addon Rate Card 6 Description"),
		Key:                 ExampleFeatureKey,
		FeatureKey:          lo.ToPtr(ExampleFeatureKey),
		EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}),
	},
	BillingCadence: &ISOMonth,
}

func GetExampleAddonInput(t *testing.T, effectivePeriod productcatalog.EffectivePeriod) addon.CreateAddonInput {
	return BuildAddonForTesting(t, effectivePeriod, productcatalog.AddonInstanceTypeSingle, &ExampleAddonRateCard1)
}

type testAddonService struct {
	addon.Service
}

func NewTestAddonService(svc addon.Service) *testAddonService {
	return &testAddonService{svc}
}

func (s *testAddonService) CreateTestAddon(t *testing.T, addInp addon.CreateAddonInput) addon.Addon {
	t.Helper()

	add, err := s.CreateAddon(context.Background(), addInp)
	if err != nil {
		t.Fatalf("failed to create addon: %v", err)
	}

	add, err = s.PublishAddon(context.Background(), addon.PublishAddonInput{
		NamespacedID:    add.NamespacedID,
		EffectivePeriod: addInp.EffectivePeriod,
	})
	if err != nil {
		t.Fatalf("failed to publish addon: %v", err)
	}

	return *add
}
