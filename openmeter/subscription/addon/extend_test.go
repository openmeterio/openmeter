package subscriptionaddon_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

func TestExtendApply(t *testing.T) {
	someMeta := productcatalog.RateCardMeta{
		Name:        "Test Addon Rate Card 4",
		Description: lo.ToPtr("Test Addon Rate Card 4 Description"),
		Key:         subscriptiontestutils.ExampleFeatureKey,
		FeatureKey:  lo.ToPtr(subscriptiontestutils.ExampleFeatureKey),
		Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(100),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		}),
	}

	t.Run("Should error if provided RateCard is nil", func(t *testing.T) {
		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: someMeta.Clone(),
		})

		err := rc.Apply(nil)
		require.Error(t, err)
		require.ErrorContains(t, err, "target must not be nil")
	})

	t.Run("Should error if provided RateCard is not a pointer", func(t *testing.T) {
		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: someMeta.Clone(),
		})
		err := rc.Apply(nonPointerRateCard{})
		require.Error(t, err)
		require.ErrorContains(t, err, "target must be a pointer")
	})

	t.Run("Should error if target RateCard has different price type", func(t *testing.T) {
		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: someMeta.Clone(),
		})

		meta := someMeta.Clone()
		meta.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(100),
		})

		err := rc.Apply(&productcatalog.UsageBasedRateCard{
			RateCardMeta: meta,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "target and addon rate card price types do not match")
	})

	t.Run("Should error if target RateCard has different entitlement type", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta,
		})

		meta = someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.StaticEntitlementTemplate{})

		err := rc.Apply(&productcatalog.UsageBasedRateCard{
			RateCardMeta: meta,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "target and addon rate card entitlement types do not match")
	})

	t.Run("Should error if target RateCard has a different key", func(t *testing.T) {
		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: someMeta.Clone(),
		})

		meta := someMeta.Clone()
		meta.Key = "different-key"

		err := rc.Apply(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "target and addon rate card keys do not match")
	})

	t.Run("Should not change target RateCard if addon RateCard has no entitlement & no price", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = nil
		meta.Price = nil

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta,
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: someMeta.Clone(),
		}

		targetClone := target.Clone()

		err := rc.Apply(target)

		require.NoError(t, err)
		require.Equal(t, targetClone, target)
	})

	t.Run("Should error if target Price has different payment term", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(100),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		meta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(50),
			PaymentTerm: productcatalog.InArrearsPaymentTerm,
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Apply(target)

		require.Error(t, err)
		require.ErrorContains(t, err, "target and addon rate card price payment terms do not match")
	})

	t.Run("Should keep FlatPrice of target if addon has no price", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.Price = nil

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		meta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(100),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Apply(target)

		require.NoError(t, err)
		fp, err := target.AsMeta().Price.AsFlat()
		require.NoError(t, err)
		require.Equal(t, alpacadecimal.NewFromInt(100), fp.Amount)
	})

	t.Run("Should update target Price when FlatPrice", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(100),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		meta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(50),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Apply(target)

		require.NoError(t, err)
		fp, err := target.AsMeta().Price.AsFlat()
		require.NoError(t, err)
		require.Equal(t, alpacadecimal.NewFromInt(150), fp.Amount)
	})

	t.Run("Should add addon price to target", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(100),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		meta.Price = nil

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Apply(target)

		require.NoError(t, err)
		fp, err := target.AsMeta().Price.AsFlat()
		require.NoError(t, err)
		require.Equal(t, fp.Amount, alpacadecimal.NewFromInt(100))
	})

	t.Run("Should add addon entitlement to target", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		meta.EntitlementTemplate = nil

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Apply(target)

		require.NoError(t, err)
		require.Equal(t, string(entitlement.EntitlementTypeBoolean), string(target.AsMeta().EntitlementTemplate.Type()))
	})

	t.Run("Should keep BooleanEntitlement of target if addon has no entitlement", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = nil

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Apply(target)

		require.NoError(t, err)
		require.Equal(t, string(entitlement.EntitlementTypeBoolean), string(target.AsMeta().EntitlementTemplate.Type()))
	})

	t.Run("Should set MeteredEntitlement of addon to target", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
			IssueAfterReset: lo.ToPtr(100.0),
			UsagePeriod:     testutils.GetISODuration(t, "P1M"),
		})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		meta.EntitlementTemplate = nil
		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Apply(target)

		require.NoError(t, err)
		me, err := target.AsMeta().EntitlementTemplate.AsMetered()
		require.NoError(t, err)
		require.NotNil(t, me.IsSoftLimit)
		require.Equal(t, 100.0, *me.IssueAfterReset)
	})

	t.Run("Should combine issueAfterReset for metered entitlements treating nil as 0", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
			IssueAfterReset: lo.ToPtr(100.0),
			UsagePeriod:     testutils.GetISODuration(t, "P1M"),
		})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		meta.EntitlementTemplate = nil

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Apply(target)

		require.NoError(t, err)
		me, err := target.AsMeta().EntitlementTemplate.AsMetered()
		require.NoError(t, err)
		require.NotNil(t, me.IssueAfterReset)
		require.Equal(t, 100.0, *me.IssueAfterReset)

		err = rc.Apply(target)

		require.NoError(t, err)
		me, err = target.AsMeta().EntitlementTemplate.AsMetered()
		require.NoError(t, err)
		require.NotNil(t, me.IssueAfterReset)
		require.Equal(t, 200.0, *me.IssueAfterReset)
	})
}

type nonPointerRateCard struct{}

var _ productcatalog.RateCard = nonPointerRateCard{}

func (n nonPointerRateCard) AsMeta() productcatalog.RateCardMeta {
	return productcatalog.RateCardMeta{}
}

func (n nonPointerRateCard) ChangeMeta(func(m productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error)) error {
	return nil
}

func (n nonPointerRateCard) Clone() productcatalog.RateCard {
	return n
}

func (n nonPointerRateCard) Compatible(productcatalog.RateCard) error {
	return nil
}

func (n nonPointerRateCard) Equal(productcatalog.RateCard) bool {
	return false
}

func (n nonPointerRateCard) Validate() error {
	return nil
}

func (n nonPointerRateCard) GetBillingCadence() *isodate.Period {
	return nil
}

func (n nonPointerRateCard) Key() string {
	return subscriptiontestutils.ExampleFeatureKey
}

func (n nonPointerRateCard) Merge(productcatalog.RateCard) error {
	return nil
}

func (n nonPointerRateCard) Type() productcatalog.RateCardType {
	return productcatalog.FlatFeeRateCardType
}

func getTestAddonRateCard(rc productcatalog.RateCard) subscriptionaddon.SubscriptionAddonRateCard {
	return subscriptionaddon.SubscriptionAddonRateCard{
		AddonRateCard: addon.RateCard{
			RateCard: rc,
		},
	}
}
