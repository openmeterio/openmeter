package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	addondiff "github.com/openmeterio/openmeter/openmeter/subscription/addon/diff"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (s *service) AddAddon(ctx context.Context, subscriptionID models.NamespacedID, addonInp subscriptionworkflow.AddAddonWorkflowInput) (subscription.SubscriptionView, subscriptionaddon.SubscriptionAddon, error) {
	var def1 subscription.SubscriptionView
	var def2 subscriptionaddon.SubscriptionAddon

	setSpanAttrs(ctx,
		attribute.String("subscription.namespace", subscriptionID.Namespace),
		attribute.String("subscription.id", subscriptionID.ID),
		attribute.String("workflow.operation", "add_addon"),
		attribute.String("addon.id", addonInp.AddonID),
		attribute.Int("addon.initial_quantity", addonInp.InitialQuantity),
		attribute.Bool("subscription.timing.has_custom", addonInp.Timing.Custom != nil),
		attribute.String("subscription.timing.enum", lo.TernaryF(addonInp.Timing.Enum != nil, func() string {
			return string(*addonInp.Timing.Enum)
		}, func() string {
			return ""
		})),
	)

	if err := addonInp.Validate(); err != nil {
		return def1, def2, models.NewGenericValidationError(err)
	}

	// TODO: maybe we should lock the subscription for this operation
	res, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (purchaseRes, error) {
		var def purchaseRes

		subView, err := s.Service.GetView(ctx, subscriptionID)
		if err != nil {
			return def, fmt.Errorf("failed to get subscription: %w", err)
		}

		subsAdds, err := s.AddonService.List(ctx, subscriptionID.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
			SubscriptionID: subscriptionID.ID,
		})
		if err != nil {
			return def, fmt.Errorf("failed to list subscription addons: %w", err)
		}

		if lo.SomeBy(subsAdds.Items, func(subAdd subscriptionaddon.SubscriptionAddon) bool {
			return subAdd.Addon.ID == addonInp.AddonID
		}) {
			return def, models.NewGenericConflictError(fmt.Errorf("subscription already has that addon purchased"))
		}

		// Let's get a clean spec by restoring the subscription
		spec := subView.AsSpec()

		// Let's try to decode when the subscription should be patched
		if err := addonInp.Timing.ValidateForAction(subscription.SubscriptionActionChangeAddons, &subView); err != nil {
			return def, models.NewGenericValidationError(fmt.Errorf("invalid timing for adding add-on: %w", err))
		}

		editTime, err := addonInp.Timing.ResolveForSpec(spec)
		if err != nil {
			return def, fmt.Errorf("failed to resolve timing: %w", err)
		}
		setSpanAttrs(ctx, attribute.String("subscription.edit_time", editTime.UTC().Format(time.RFC3339Nano)))

		if !subView.Subscription.IsActiveAt(editTime) {
			return def, models.NewGenericValidationError(fmt.Errorf("subscription is not active at the time of adding the addon"))
		}

		diffs, err := asDiffs(subView, subsAdds.Items)
		if err != nil {
			return def, fmt.Errorf("failed to get diffable from addons: %w", err)
		}
		setSpanAttrs(ctx,
			addSubscriptionViewAttrs([]attribute.KeyValue{}, "subscription.view.before", subView)...,
		)
		setSpanAttrs(ctx,
			addSubscriptionAddonsAttrs([]attribute.KeyValue{}, "subscription.addons.before", subsAdds.Items)...,
		)
		setSpanAttrs(ctx, attribute.Int("subscription.addons.before.diffables.count", len(diffs)))

		if len(diffs) != len(subsAdds.Items) {
			return def, fmt.Errorf("failed to get diffable from addons, got %d addons but %d diffs", len(subsAdds.Items), len(diffs))
		}

		// Now let's try to purchase the addon

		subsAdd, err := s.AddonService.Create(ctx, subscriptionID.Namespace, subscriptionaddon.CreateSubscriptionAddonInput{
			MetadataModel:  addonInp.MetadataModel,
			AddonID:        addonInp.AddonID,
			SubscriptionID: subscriptionID.ID,
			InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				ActiveFrom: editTime,
				Quantity:   addonInp.InitialQuantity,
			},
		})
		if err != nil {
			return def, fmt.Errorf("failed to create subscription addon: %w", err)
		}

		if subsAdd == nil {
			return def, errors.New("subscription addon is nil")
		}

		subView, err = s.syncWithAddons(ctx, subView, subsAdds.Items, append(subsAdds.Items, *subsAdd), editTime)
		if err != nil {
			return def, fmt.Errorf("failed to sync with addons: %w", err)
		}

		return purchaseRes{
			sub:    subView,
			subAdd: *subsAdd,
		}, nil
	})
	if err != nil {
		return def1, def2, err
	}

	return res.sub, res.subAdd, nil
}

type purchaseRes struct {
	sub    subscription.SubscriptionView
	subAdd subscriptionaddon.SubscriptionAddon
}

func (s *service) ChangeAddonQuantity(ctx context.Context, subscriptionID models.NamespacedID, changeInp subscriptionworkflow.ChangeAddonQuantityWorkflowInput) (subscription.SubscriptionView, subscriptionaddon.SubscriptionAddon, error) {
	var def1 subscription.SubscriptionView
	var def2 subscriptionaddon.SubscriptionAddon

	setSpanAttrs(ctx,
		attribute.String("subscription.namespace", subscriptionID.Namespace),
		attribute.String("subscription.id", subscriptionID.ID),
		attribute.String("workflow.operation", "change_addon_quantity"),
		attribute.String("subscription_addon.id", changeInp.SubscriptionAddonID.ID),
		attribute.Int("addon.quantity.new", changeInp.Quantity),
		attribute.Bool("subscription.timing.has_custom", changeInp.Timing.Custom != nil),
		attribute.String("subscription.timing.enum", lo.TernaryF(changeInp.Timing.Enum != nil, func() string {
			return string(*changeInp.Timing.Enum)
		}, func() string {
			return ""
		})),
	)

	if subscriptionID.Namespace != changeInp.SubscriptionAddonID.Namespace {
		return def1, def2, models.NewGenericValidationError(fmt.Errorf("subscription and subscription addon are in different namespaces"))
	}

	subsAdd, err := s.AddonService.Get(ctx, changeInp.SubscriptionAddonID)
	if err != nil {
		return def1, def2, fmt.Errorf("failed to get subscription addon: %w", err)
	}

	if subsAdd.SubscriptionID != subscriptionID.ID {
		return def1, def2, models.NewGenericValidationError(fmt.Errorf("subscription addon does not belong to subscription"))
	}

	res, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (purchaseRes, error) {
		var def purchaseRes

		subView, err := s.Service.GetView(ctx, subscriptionID)
		if err != nil {
			return def, err
		}

		subsAddsBefore, err := s.AddonService.List(ctx, subscriptionID.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
			SubscriptionID: subscriptionID.ID,
		})
		if err != nil {
			return def, err
		}

		// Let's try to decode when the subscription should be patched
		if err := changeInp.Timing.ValidateForAction(subscription.SubscriptionActionChangeAddons, &subView); err != nil {
			return def, models.NewGenericValidationError(fmt.Errorf("invalid timing for adding add-on: %w", err))
		}

		editTime, err := changeInp.Timing.ResolveForSpec(subView.AsSpec())
		if err != nil {
			return def, fmt.Errorf("failed to resolve timing: %w", err)
		}
		setSpanAttrs(ctx, attribute.String("subscription.edit_time", editTime.UTC().Format(time.RFC3339Nano)))

		subsAdd, err := s.AddonService.ChangeQuantity(ctx, changeInp.SubscriptionAddonID, subscriptionaddon.CreateSubscriptionAddonQuantityInput{
			Quantity:   changeInp.Quantity,
			ActiveFrom: editTime,
		})
		if err != nil {
			return def, err
		}

		subsAddsAfter, err := s.AddonService.List(ctx, subscriptionID.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
			SubscriptionID: subscriptionID.ID,
		})
		if err != nil {
			return def, err
		}
		setSpanAttrs(ctx,
			addSubscriptionViewAttrs([]attribute.KeyValue{}, "subscription.view.before", subView)...,
		)
		setSpanAttrs(ctx,
			addSubscriptionAddonsAttrs([]attribute.KeyValue{}, "subscription.addons.before", subsAddsBefore.Items)...,
		)
		setSpanAttrs(ctx,
			addSubscriptionAddonsAttrs([]attribute.KeyValue{}, "subscription.addons.after", subsAddsAfter.Items)...,
		)

		subView, err = s.syncWithAddons(ctx, subView, subsAddsBefore.Items, subsAddsAfter.Items, editTime)
		if err != nil {
			return def, fmt.Errorf("failed to sync with addons: %w", err)
		}

		return purchaseRes{
			sub:    subView,
			subAdd: *subsAdd,
		}, nil
	})

	return res.sub, res.subAdd, err
}

func (s *service) syncWithAddons(
	ctx context.Context,
	view subscription.SubscriptionView,
	before []subscriptionaddon.SubscriptionAddon,
	after []subscriptionaddon.SubscriptionAddon,
	currentTime time.Time,
) (subscription.SubscriptionView, error) {
	setSpanAttrs(ctx,
		attribute.String("workflow.operation", "sync_with_addons"),
		attribute.String("subscription.namespace", view.Subscription.Namespace),
		attribute.String("subscription.id", view.Subscription.ID),
		attribute.String("subscription.sync.current_time", currentTime.UTC().Format(time.RFC3339Nano)),
	)
	setSpanAttrs(ctx, addSubscriptionViewAttrs([]attribute.KeyValue{}, "subscription.view.input", view)...)
	setSpanAttrs(ctx, addSubscriptionAddonsAttrs([]attribute.KeyValue{}, "subscription.addons.before", before)...)
	setSpanAttrs(ctx, addSubscriptionAddonsAttrs([]attribute.KeyValue{}, "subscription.addons.after", after)...)
	emitAddonApplyPlanEvents(ctx, "restore", view, before)
	emitAddonApplyPlanEvents(ctx, "apply", view, after)

	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.SubscriptionView, error) {
		var def subscription.SubscriptionView

		// TODO: remove after issue is fixed
		logErrWithArgs := func(mErr error) {
			// Let's json serialize everything
			viewJson, err := json.Marshal(view)
			if err != nil {
				s.Logger.DebugContext(ctx, "failed to marshal subscription view", "error", err)
			}

			beforeJson, err := json.Marshal(before)
			if err != nil {
				s.Logger.DebugContext(ctx, "failed to marshal before addons", "error", err)
			}

			afterJson, err := json.Marshal(after)
			if err != nil {
				s.Logger.DebugContext(ctx, "failed to marshal after addons", "error", err)
			}

			s.Logger.DebugContext(ctx, "failed to restore subscription state without addons",
				"restore_error", mErr,
				"subscription_view", viewJson,
				"before_addons", beforeJson,
				"after_addons", afterJson,
			)
		}

		spec := view.AsSpec()

		restores, err := asDiffs(view, before)
		if err != nil {
			return def, fmt.Errorf("failed to get diffable from addons: %w", err)
		}
		setSpanAttrs(ctx, attribute.Int("subscription.addons.before.diffables.count", len(restores)))

		applies, err := asDiffs(view, after)
		if err != nil {
			return def, fmt.Errorf("failed to get diffable from addons: %w", err)
		}
		setSpanAttrs(ctx, attribute.Int("subscription.addons.after.diffables.count", len(applies)))

		if err := spec.ApplyMany(lo.Map(restores, func(d addondiff.Diffable, _ int) subscription.AppliesToSpec {
			return d.GetRestores()
		}), subscription.ApplyContext{
			CurrentTime: currentTime,
		}); err != nil {
			logErrWithArgs(fmt.Errorf("failed to restore subscription state without addons: %w", err))

			return def, fmt.Errorf("failed to restore subscription state without addons: %w", err)
		}

		if err := spec.ApplyMany(lo.Map(applies, func(d addondiff.Diffable, _ int) subscription.AppliesToSpec {
			return d.GetApplies()
		}), subscription.ApplyContext{
			CurrentTime: currentTime,
		}); err != nil {
			logErrWithArgs(fmt.Errorf("failed to calculate subscription state with addons: %w", err))

			return def, fmt.Errorf("failed to calculate subscription state with addons: %w", err)
		}

		if _, err := s.Service.Update(ctx, view.Subscription.NamespacedID, spec); err != nil {
			logErrWithArgs(fmt.Errorf("failed to update subscription: %w", err))

			return def, fmt.Errorf("failed to update subscription: %w", err)
		}
		updated, err := s.Service.GetView(ctx, view.Subscription.NamespacedID)
		if err != nil {
			return def, err
		}
		setSpanAttrs(ctx, addSubscriptionViewAttrs([]attribute.KeyValue{}, "subscription.view.output", updated)...)
		setSpanAttrs(ctx, addSubscriptionSpecAttrs([]attribute.KeyValue{}, "subscription.spec.output", updated.Spec)...)

		return updated, nil
	})
}

func emitAddonApplyPlanEvents(ctx context.Context, source string, view subscription.SubscriptionView, addons []subscriptionaddon.SubscriptionAddon) {
	order := 0

	for addonOrder, add := range addons {
		affectedByRateCardKey := addondiff.GetAffectedItemIDs(view, add)
		affectedRateCardKeys := lo.Keys(affectedByRateCardKey)
		slices.Sort(affectedRateCardKeys)

		addSpanEvent(ctx, "subscription.addon.apply.plan",
			attribute.String("apply.source", source),
			attribute.Int("apply.order", order),
			attribute.Int("apply.addon_order", addonOrder),
			attribute.String("subscription.namespace", add.Namespace),
			attribute.String("subscription.id", add.SubscriptionID),
			attribute.String("subscription_addon.id", add.ID),
			attribute.String("addon.id", add.Addon.ID),
			attribute.StringSlice("addon.affected_ratecard_keys", affectedRateCardKeys),
		)
		order++

		for _, rateCardKey := range affectedRateCardKeys {
			addSpanEvent(ctx, "subscription.addon.apply.plan",
				attribute.String("apply.source", source),
				attribute.Int("apply.order", order),
				attribute.String("subscription_addon.id", add.ID),
				attribute.String("addon.id", add.Addon.ID),
				attribute.String("addon.ratecard_key", rateCardKey),
				attribute.StringSlice("subscription.item_ids", affectedByRateCardKey[rateCardKey]),
			)
			order++
		}

		instances := add.GetInstances()
		for instanceOrder, inst := range instances {
			rateCardKeys := lo.Map(inst.RateCards, func(rc subscriptionaddon.SubscriptionAddonRateCard, _ int) string {
				return rc.AddonRateCard.Key()
			})

			addSpanEvent(ctx, "subscription.addon.apply.plan",
				attribute.String("apply.source", source),
				attribute.Int("apply.order", order),
				attribute.Int("apply.instance_order", instanceOrder),
				attribute.String("subscription_addon.id", inst.ID),
				attribute.String("addon.id", inst.Addon.ID),
				attribute.Int("addon.quantity", inst.Quantity),
				attribute.String("addon.instance_active_from", inst.ActiveFrom.UTC().Format(time.RFC3339Nano)),
				attribute.String("addon.instance_active_to", lo.TernaryF(inst.ActiveTo != nil, func() string {
					return inst.ActiveTo.UTC().Format(time.RFC3339Nano)
				}, func() string {
					return ""
				})),
				attribute.StringSlice("addon.instance_ratecard_keys", rateCardKeys),
			)
			order++
		}
	}
}

// The sub has addons if it has a non-0 quantity on any of them during its cadence
func hasAddons(view subscription.SubscriptionView, addons []subscriptionaddon.SubscriptionAddon) bool {
	subPer := view.Subscription.CadencedModel.AsPeriod()

	for _, add := range addons {
		for _, addInst := range add.GetInstances() {
			if addInst.Quantity > 0 {
				if addInst.CadencedModel.AsPeriod().Intersection(subPer) != nil {
					return true
				}
			}
		}
	}

	return false
}

func asDiffs(view subscription.SubscriptionView, subsAdds []subscriptionaddon.SubscriptionAddon) ([]addondiff.Diffable, error) {
	diffs, err := slicesx.MapWithErr(subsAdds, func(subAdd subscriptionaddon.SubscriptionAddon) (addondiff.Diffable, error) {
		return addondiff.GetDiffableFromAddon(view, subAdd)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get diffable from addon: %w", err)
	}

	filtered := lo.Filter(diffs, func(d addondiff.Diffable, _ int) bool {
		return d != nil
	})

	return filtered, nil
}
