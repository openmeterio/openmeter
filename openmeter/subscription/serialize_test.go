package subscription_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

func TestSubscriptionSerialize(t *testing.T) {
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
