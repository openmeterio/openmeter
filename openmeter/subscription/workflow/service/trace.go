package service

import (
	"context"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
)

func setSpanAttrs(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}

	span.SetAttributes(attrs...)
}

func addSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}

	span.AddEvent(name, trace.WithAttributes(attrs...))
}

func addSubscriptionSpecAttrs(attrs []attribute.KeyValue, prefix string, spec subscription.SubscriptionSpec) []attribute.KeyValue {
	phaseKeys := make([]string, 0, len(spec.Phases))
	itemKeySet := make(map[string]struct{})
	itemVersions := 0
	for k, phase := range spec.Phases {
		phaseKeys = append(phaseKeys, k)
		for ik, items := range phase.ItemsByKey {
			itemKeySet[ik] = struct{}{}
			itemVersions += len(items)
		}
	}
	slices.Sort(phaseKeys)

	itemKeys := make([]string, 0, len(itemKeySet))
	for k := range itemKeySet {
		itemKeys = append(itemKeys, k)
	}
	slices.Sort(itemKeys)

	attrs = append(attrs,
		attribute.String(prefix+".customer_id", spec.CustomerId),
		attribute.Int(prefix+".phases.count", len(spec.Phases)),
		attribute.StringSlice(prefix+".phase_keys", phaseKeys),
		attribute.Int(prefix+".item_keys.count", len(itemKeySet)),
		attribute.StringSlice(prefix+".item_keys", itemKeys),
		attribute.Int(prefix+".item_versions.count", itemVersions),
		attribute.Bool(prefix+".has_billables", spec.HasBillables()),
		attribute.Bool(prefix+".has_metered_billables", spec.HasMeteredBillables()),
		attribute.Bool(prefix+".has_entitlements", spec.HasEntitlements()),
	)

	if spec.Plan != nil {
		attrs = append(attrs,
			attribute.String(prefix+".plan.id", spec.Plan.Id),
			attribute.String(prefix+".plan.key", spec.Plan.Key),
			attribute.Int(prefix+".plan.version", spec.Plan.Version),
		)
	}

	return attrs
}

func addSubscriptionViewAttrs(attrs []attribute.KeyValue, prefix string, view subscription.SubscriptionView) []attribute.KeyValue {
	phaseKeys := make([]string, 0, len(view.Phases))
	itemKeySet := make(map[string]struct{})
	itemVersions := 0
	for _, phase := range view.Phases {
		phaseKeys = append(phaseKeys, phase.SubscriptionPhase.Key)
		for ik, items := range phase.ItemsByKey {
			itemKeySet[ik] = struct{}{}
			itemVersions += len(items)
		}
	}
	slices.Sort(phaseKeys)

	itemKeys := make([]string, 0, len(itemKeySet))
	for k := range itemKeySet {
		itemKeys = append(itemKeys, k)
	}
	slices.Sort(itemKeys)

	return append(attrs,
		attribute.String(prefix+".subscription_id", view.Subscription.ID),
		attribute.String(prefix+".customer_id", view.Subscription.CustomerId),
		attribute.Int(prefix+".phases.count", len(view.Phases)),
		attribute.StringSlice(prefix+".phase_keys", phaseKeys),
		attribute.Int(prefix+".item_keys.count", len(itemKeySet)),
		attribute.StringSlice(prefix+".item_keys", itemKeys),
		attribute.Int(prefix+".item_versions.count", itemVersions),
	)
}

func addSubscriptionAddonsAttrs(attrs []attribute.KeyValue, prefix string, addons []subscriptionaddon.SubscriptionAddon) []attribute.KeyValue {
	instances := 0
	subAddonIDs := make([]string, 0, len(addons))
	addonIDs := make([]string, 0, len(addons))
	addonKeys := make([]string, 0, len(addons))
	for _, add := range addons {
		instances += len(add.GetInstances())
		subAddonIDs = append(subAddonIDs, add.ID)
		addonIDs = append(addonIDs, add.Addon.ID)
		addonKeys = append(addonKeys, add.Addon.Key)
	}

	return append(attrs,
		attribute.Int(prefix+".addons.count", len(addons)),
		attribute.Int(prefix+".instances.count", instances),
		attribute.StringSlice(prefix+".subscription_addon_ids", subAddonIDs),
		attribute.StringSlice(prefix+".addon_ids", addonIDs),
		attribute.StringSlice(prefix+".addon_keys", addonKeys),
	)
}
