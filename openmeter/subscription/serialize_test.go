package subscription_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSubscriptionItemSpecSerialize(t *testing.T) {
	sis := subscription.SubscriptionItemSpec{
		CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
			CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{},
			CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
				PhaseKey: "phase-key",
				ItemKey:  "item-key",
				RateCard: &productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:         "rate-card-key",
						Name:        "rate-card-name",
						Description: lo.ToPtr("rate-card-description"),
						FeatureKey:  lo.ToPtr("feature-key"),
						FeatureID:   lo.ToPtr("feature-id"),
						EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
							IssueAfterReset: lo.ToPtr(100.0),
						}),
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromInt(100),
						}),
					},
				},
			},
		},
	}

	// Now let's marshal and unmarshal the subscription item spec
	sisBytes, err := json.MarshalIndent(sis, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal subscription item spec: %v", err)
	}

	bt2 := bytes.Clone(sisBytes)

	var sis2 subscription.SubscriptionItemSpec
	err = json.Unmarshal(bt2, &sis2)
	if err != nil {
		t.Fatalf("failed to unmarshal subscription item spec: %v", err)
	}

	// Now let's compare the two
	if !reflect.DeepEqual(sis, sis2) {
		t.Fatalf("subscription item spec is not equal \n\ninput: %v\n\n serialized: \n%s\n\n unserialized: \n%v\n\n", sis, string(bt2), sis2)
	}
}

func TestSubscriptionItemSerialize(t *testing.T) {
	fixedTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	si := subscription.SubscriptionItem{
		NamespacedID: models.NamespacedID{
			Namespace: "test-namespace",
			ID:        "test-id",
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		},
		MetadataModel: models.MetadataModel{
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		ActiveFromOverrideRelativeToPhaseStart: lo.ToPtr(datetime.NewPeriod(0, 0, 0, 1, 0, 0, 0)),
		ActiveToOverrideRelativeToPhaseStart:   lo.ToPtr(datetime.NewPeriod(0, 0, 0, 2, 0, 0, 0)),
		CadencedModel: models.CadencedModel{
			ActiveFrom: fixedTime,
			ActiveTo:   lo.ToPtr(fixedTime.Add(24 * time.Hour)),
		},
		BillingBehaviorOverride: subscription.BillingBehaviorOverride{
			RestartBillingPeriod: lo.ToPtr(true),
		},
		SubscriptionId: "subscription-id",
		PhaseId:        "phase-id",
		Key:            "item-key",
		RateCard: &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:         "rate-card-key",
				Name:        "rate-card-name",
				Description: lo.ToPtr("rate-card-description"),
				FeatureKey:  lo.ToPtr("feature-key"),
				FeatureID:   lo.ToPtr("feature-id"),
				EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
					IssueAfterReset: lo.ToPtr(100.0),
				}),
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(100),
				}),
			},
			BillingCadence: datetime.NewPeriod(0, 0, 0, 0, 1, 0, 0),
		},
		EntitlementID: lo.ToPtr("entitlement-id"),
		Name:          "item-name",
		Description:   lo.ToPtr("item-description"),
	}

	// Now let's marshal and unmarshal the subscription item
	siBytes, err := json.MarshalIndent(si, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal subscription item: %v", err)
	}

	bt2 := bytes.Clone(siBytes)

	var si2 subscription.SubscriptionItem
	err = json.Unmarshal(bt2, &si2)
	if err != nil {
		t.Fatalf("failed to unmarshal subscription item: %v", err)
	}

	// Now let's compare the two
	if !reflect.DeepEqual(si, si2) {
		t.Fatalf("subscription item is not equal \n\ninput: %v\n\n serialized: \n%s\n\n unserialized: \n%v\n\n", si, string(bt2), si2)
	}
}
