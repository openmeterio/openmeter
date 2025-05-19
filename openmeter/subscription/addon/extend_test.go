package subscriptionaddon_test

import (
	"fmt"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestValidations(t *testing.T) {
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

		t.Run("Apply", func(t *testing.T) {
			err := rc.Apply(nil, nil)
			require.Error(t, err)
			require.ErrorContains(t, err, "target must not be nil")
		})

		t.Run("Restore", func(t *testing.T) {
			err := rc.Restore(nil, nil)
			require.Error(t, err)
			require.ErrorContains(t, err, "target must not be nil")
		})
	})

	t.Run("Should error if provided RateCard is not a pointer", func(t *testing.T) {
		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: someMeta.Clone(),
		})

		t.Run("Apply", func(t *testing.T) {
			err := rc.Apply(nonPointerRateCard{}, nil)
			require.Error(t, err)
			require.ErrorContains(t, err, "target must be a pointer")
		})

		t.Run("Restore", func(t *testing.T) {
			err := rc.Restore(nonPointerRateCard{}, nil)
			require.Error(t, err)
			require.ErrorContains(t, err, "target must be a pointer")
		})
	})

	t.Run("Should error if provided annotations are nil", func(t *testing.T) {
		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: someMeta.Clone(),
		})

		t.Run("Apply", func(t *testing.T) {
			err := rc.Apply(&productcatalog.FlatFeeRateCard{
				RateCardMeta: someMeta.Clone(),
			}, nil)
			require.Error(t, err)
			require.ErrorContains(t, err, "annotations must not be nil")
		})

		t.Run("Restore", func(t *testing.T) {
			err := rc.Restore(&productcatalog.FlatFeeRateCard{
				RateCardMeta: someMeta.Clone(),
			}, nil)
			require.Error(t, err)
			require.ErrorContains(t, err, "annotations must not be nil")
		})
	})

	t.Run("Should error if target RateCard has different price type", func(t *testing.T) {
		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: someMeta.Clone(),
		})

		meta := someMeta.Clone()
		meta.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(100),
		})

		t.Run("Apply", func(t *testing.T) {
			err := rc.Apply(&productcatalog.UsageBasedRateCard{
				RateCardMeta: meta,
			}, models.Annotations{})
			require.Error(t, err)
			require.ErrorContains(t, err, "price type must match")
		})

		t.Run("Restore", func(t *testing.T) {
			err := rc.Restore(&productcatalog.UsageBasedRateCard{
				RateCardMeta: meta,
			}, models.Annotations{})
			require.Error(t, err)
			require.ErrorContains(t, err, "price type must match")
		})
	})

	t.Run("Should error if target RateCard has different entitlement type", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta,
		})

		meta = someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.StaticEntitlementTemplate{})

		t.Run("Apply", func(t *testing.T) {
			err := rc.Apply(&productcatalog.UsageBasedRateCard{
				RateCardMeta: meta,
			}, models.Annotations{})
			require.Error(t, err)
			require.ErrorContains(t, err, "entitlement template type must match")
		})

		t.Run("Restore", func(t *testing.T) {
			err := rc.Restore(&productcatalog.UsageBasedRateCard{
				RateCardMeta: meta,
			}, models.Annotations{})
			require.Error(t, err)
			require.ErrorContains(t, err, "entitlement template type must match")
		})
	})

	t.Run("Should error if target RateCard has a different key", func(t *testing.T) {
		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: someMeta.Clone(),
		})

		meta := someMeta.Clone()
		meta.Key = "different-key"

		t.Run("Apply", func(t *testing.T) {
			err := rc.Apply(&productcatalog.FlatFeeRateCard{
				RateCardMeta: meta,
			}, models.Annotations{})
			require.Error(t, err)
			require.ErrorContains(t, err, "key must match")
		})

		t.Run("Restore", func(t *testing.T) {
			err := rc.Restore(&productcatalog.FlatFeeRateCard{
				RateCardMeta: meta,
			}, models.Annotations{})
			require.Error(t, err)
			require.ErrorContains(t, err, "key must match")
		})
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

		t.Run("Apply", func(t *testing.T) {
			err := rc.Apply(target, models.Annotations{})

			require.Error(t, err)
			require.ErrorContains(t, err, "price payment term must match")
		})

		t.Run("Restore", func(t *testing.T) {
			err := rc.Restore(target, models.Annotations{})

			require.Error(t, err)
			require.ErrorContains(t, err, "price payment term must match")
		})
	})

	t.Run("Should error on UsageBasedRateCards with different BillingCadence", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.Price = nil

		// The addon would do nothing (which is valid)
		rc := getTestAddonRateCard(&productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta.Clone(),
			BillingCadence: testutils.GetISODuration(t, "P2M"),
		})

		target := &productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta.Clone(),
			BillingCadence: testutils.GetISODuration(t, "P1M"),
		}

		err := rc.Apply(target, models.Annotations{})
		require.Error(t, err)
		require.ErrorContains(t, err, "billing cadence must match")
	})
}

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

		err := rc.Apply(target, models.Annotations{})

		require.NoError(t, err)
		require.Equal(t, targetClone, target)
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

		err := rc.Apply(target, models.Annotations{})

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

		err := rc.Apply(target, models.Annotations{})

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

		err := rc.Apply(target, models.Annotations{})

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

		err := rc.Apply(target, models.Annotations{})

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

		err := rc.Apply(target, models.Annotations{})

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

		err := rc.Apply(target, models.Annotations{})

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

		err := rc.Apply(target, models.Annotations{})

		require.NoError(t, err)
		me, err := target.AsMeta().EntitlementTemplate.AsMetered()
		require.NoError(t, err)
		require.NotNil(t, me.IssueAfterReset)
		require.Equal(t, 100.0, *me.IssueAfterReset)

		err = rc.Apply(target, models.Annotations{})

		require.NoError(t, err)
		me, err = target.AsMeta().EntitlementTemplate.AsMetered()
		require.NoError(t, err)
		require.NotNil(t, me.IssueAfterReset)
		require.Equal(t, 200.0, *me.IssueAfterReset)
	})

	t.Run("Should add Usage Discounts from UsageBasedRateCard to target with UnitPrice", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.Price = nil
		meta.Discounts = productcatalog.Discounts{
			Usage: &productcatalog.UsageDiscount{
				Quantity: alpacadecimal.NewFromInt(100),
			},
		}

		rc := getTestAddonRateCard(&productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta.Clone(),
			BillingCadence: testutils.GetISODuration(t, "P1M"),
		})

		meta.Discounts = productcatalog.Discounts{}
		meta.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(1),
		})

		target := &productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta.Clone(),
			BillingCadence: testutils.GetISODuration(t, "P1M"),
		}

		err := rc.Apply(target, models.Annotations{})

		require.NoError(t, err)
		require.Equal(t, alpacadecimal.NewFromInt(100).InexactFloat64(), target.AsMeta().Discounts.Usage.Quantity.InexactFloat64())
	})
}

func TestExtendRestore(t *testing.T) {
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

		err := rc.Restore(target, models.Annotations{})

		require.NoError(t, err)
		require.Equal(t, targetClone, target)
	})

	t.Run("Should error if target Price is nil but addon has a flat price", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = nil

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		meta.Price = nil

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Restore(target, models.Annotations{})

		require.Error(t, err)
		require.ErrorContains(t, err, "target price is nil, cannot restore price without addon")
	})

	t.Run("Should deduct flat price of addon from target", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = nil
		meta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(10),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: someMeta.Clone(),
		}

		err := rc.Restore(target, models.Annotations{})

		require.NoError(t, err)
		fp, err := target.AsMeta().Price.AsFlat()
		require.NoError(t, err)
		require.Equal(t, alpacadecimal.NewFromInt(90), fp.Amount)
	})

	t.Run("Should allow 0 resulting flat price", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = nil

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Restore(target, models.Annotations{})

		require.NoError(t, err)
		fp, err := target.AsMeta().Price.AsFlat()
		require.NoError(t, err)
		require.Equal(t, alpacadecimal.NewFromInt(0), fp.Amount)
	})

	t.Run("Should error if target has no entitlement but addon has a metered entitlement", func(t *testing.T) {
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

		err := rc.Restore(target, models.Annotations{})

		require.Error(t, err)
		require.ErrorContains(t, err, "target entitlement template is nil, cannot restore entitlement template without addon")
	})

	t.Run("Should error if target has boolean entitlement but no annotation", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Restore(target, models.Annotations{})

		require.Error(t, err)
		require.ErrorContains(t, err, "target doesn't have boolean entitlement count annotation while has a boolean entitlement template")
	})

	t.Run("Should error if target has boolean entitlement but annotation is invalid (negative count)", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		ann := models.Annotations{}
		if _, err := subscription.AnnotationParser.SetBooleanEntitlementCount(ann, -1); err != nil {
			t.Fatalf("failed to set boolean entitlement count: %s", err)
		}

		err := rc.Restore(target, ann)

		require.Error(t, err)
		require.ErrorContains(t, err, "received invalid entitlement count annotation value: -1")
	})

	t.Run("Should error if target has boolean entitlement with annotation value of 0", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		ann := models.Annotations{}
		if _, err := subscription.AnnotationParser.SetBooleanEntitlementCount(ann, 0); err != nil {
			t.Fatalf("failed to set boolean entitlement count: %s", err)
		}

		err := rc.Restore(target, ann)

		require.Error(t, err)
		require.ErrorContains(t, err, "target doesn't have boolean entitlement count annotation while has a boolean entitlement template")
	})

	t.Run("Should remove boolean entitlement if annotation value falls to 0", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		ann := models.Annotations{}
		if _, err := subscription.AnnotationParser.SetBooleanEntitlementCount(ann, 1); err != nil {
			t.Fatalf("failed to set boolean entitlement count: %s", err)
		}

		err := rc.Restore(target, ann)

		require.NoError(t, err)
		require.Nil(t, target.EntitlementTemplate)
	})

	t.Run("Should decrement boolean entitlement count", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		ann := models.Annotations{}
		if _, err := subscription.AnnotationParser.SetBooleanEntitlementCount(ann, 2); err != nil {
			t.Fatalf("failed to set boolean entitlement count: %s", err)
		}

		err := rc.Restore(target, ann)

		require.NoError(t, err)
		require.NotNil(t, target.EntitlementTemplate)
		require.Equal(t, 1, subscription.AnnotationParser.GetBooleanEntitlementCount(ann))
	})

	t.Run("Should deduct issueAfterReset of addon from target", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
			IssueAfterReset: lo.ToPtr(10.0),
			UsagePeriod:     testutils.GetISODuration(t, "P1M"),
		})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
			IssueAfterReset: lo.ToPtr(100.0),
			UsagePeriod:     testutils.GetISODuration(t, "P1M"),
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Restore(target, models.Annotations{})

		require.NoError(t, err)
		me, err := target.AsMeta().EntitlementTemplate.AsMetered()
		require.NoError(t, err)
		require.Equal(t, 90.0, *me.IssueAfterReset)
	})

	t.Run("Should not allow negative issueAfterReset", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
			IssueAfterReset: lo.ToPtr(100.0),
			UsagePeriod:     testutils.GetISODuration(t, "P1M"),
		})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
			IssueAfterReset: lo.ToPtr(50.0),
			UsagePeriod:     testutils.GetISODuration(t, "P1M"),
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Restore(target, models.Annotations{})

		require.Error(t, err)
		require.ErrorContains(t, err, "restoring entitlement template would yield a negative issue after reset: 50 - 100 = -50")
	})

	t.Run("Should allow 0 resulting issueAfterReset", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
			IssueAfterReset: lo.ToPtr(100.0),
			UsagePeriod:     testutils.GetISODuration(t, "P1M"),
		})

		rc := getTestAddonRateCard(&productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		})

		target := &productcatalog.FlatFeeRateCard{
			RateCardMeta: meta.Clone(),
		}

		err := rc.Restore(target, models.Annotations{})

		require.NoError(t, err)
		me, err := target.AsMeta().EntitlementTemplate.AsMetered()
		require.NoError(t, err)
		require.Equal(t, 0.0, *me.IssueAfterReset)
	})

	t.Run("Should error if trying to restore Usage Discount when one is not present on target", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.Price = nil
		meta.Discounts = productcatalog.Discounts{
			Usage: &productcatalog.UsageDiscount{
				Quantity: alpacadecimal.NewFromInt(100),
			},
		}

		rc := getTestAddonRateCard(&productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta.Clone(),
			BillingCadence: testutils.GetISODuration(t, "P1M"),
		})

		meta.Discounts = productcatalog.Discounts{}
		meta.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(1),
		})

		target := &productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta.Clone(),
			BillingCadence: testutils.GetISODuration(t, "P1M"),
		}

		err := rc.Restore(target, models.Annotations{})

		require.Error(t, err)
		require.ErrorContains(t, err, "target doesn't have usage discount while addon has a usage discount")
	})

	t.Run("Should error if trying to restore Usage Discount greater than target's discount", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.Price = nil
		meta.Discounts = productcatalog.Discounts{
			Usage: &productcatalog.UsageDiscount{
				Quantity: alpacadecimal.NewFromInt(100),
			},
		}

		rc := getTestAddonRateCard(&productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta.Clone(),
			BillingCadence: testutils.GetISODuration(t, "P1M"),
		})

		meta.Discounts = productcatalog.Discounts{
			Usage: &productcatalog.UsageDiscount{
				Quantity: alpacadecimal.NewFromInt(50),
			},
		}
		meta.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(1),
		})

		target := &productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta.Clone(),
			BillingCadence: testutils.GetISODuration(t, "P1M"),
		}

		err := rc.Restore(target, models.Annotations{})

		require.Error(t, err)
		require.ErrorContains(t, err, fmt.Sprintf("target has %.0f usage discount which is less than addon's %.0f", 50.0, 100.0))
	})

	t.Run("Should restore Usage Discounts from UsageBasedRateCard to target with UnitPrice", func(t *testing.T) {
		meta := someMeta.Clone()
		meta.Price = nil
		meta.Discounts = productcatalog.Discounts{
			Usage: &productcatalog.UsageDiscount{
				Quantity: alpacadecimal.NewFromInt(100),
			},
		}

		rc := getTestAddonRateCard(&productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta.Clone(),
			BillingCadence: testutils.GetISODuration(t, "P1M"),
		})

		meta.Discounts = productcatalog.Discounts{
			Usage: &productcatalog.UsageDiscount{
				Quantity: alpacadecimal.NewFromInt(150),
			},
		}
		meta.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(1),
		})

		target := &productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta.Clone(),
			BillingCadence: testutils.GetISODuration(t, "P1M"),
		}

		err := rc.Restore(target, models.Annotations{})

		require.NoError(t, err)
		require.Equal(t, alpacadecimal.NewFromInt(50).InexactFloat64(), target.AsMeta().Discounts.Usage.Quantity.InexactFloat64())
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
