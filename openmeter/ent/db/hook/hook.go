// Code generated by ent, DO NOT EDIT.

package hook

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
)

// The AddonFunc type is an adapter to allow the use of ordinary
// function as Addon mutator.
type AddonFunc func(context.Context, *db.AddonMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f AddonFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.AddonMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.AddonMutation", m)
}

// The AddonRateCardFunc type is an adapter to allow the use of ordinary
// function as AddonRateCard mutator.
type AddonRateCardFunc func(context.Context, *db.AddonRateCardMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f AddonRateCardFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.AddonRateCardMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.AddonRateCardMutation", m)
}

// The AppFunc type is an adapter to allow the use of ordinary
// function as App mutator.
type AppFunc func(context.Context, *db.AppMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f AppFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.AppMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.AppMutation", m)
}

// The AppCustomerFunc type is an adapter to allow the use of ordinary
// function as AppCustomer mutator.
type AppCustomerFunc func(context.Context, *db.AppCustomerMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f AppCustomerFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.AppCustomerMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.AppCustomerMutation", m)
}

// The AppStripeFunc type is an adapter to allow the use of ordinary
// function as AppStripe mutator.
type AppStripeFunc func(context.Context, *db.AppStripeMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f AppStripeFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.AppStripeMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.AppStripeMutation", m)
}

// The AppStripeCustomerFunc type is an adapter to allow the use of ordinary
// function as AppStripeCustomer mutator.
type AppStripeCustomerFunc func(context.Context, *db.AppStripeCustomerMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f AppStripeCustomerFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.AppStripeCustomerMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.AppStripeCustomerMutation", m)
}

// The BalanceSnapshotFunc type is an adapter to allow the use of ordinary
// function as BalanceSnapshot mutator.
type BalanceSnapshotFunc func(context.Context, *db.BalanceSnapshotMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f BalanceSnapshotFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.BalanceSnapshotMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.BalanceSnapshotMutation", m)
}

// The BillingCustomerLockFunc type is an adapter to allow the use of ordinary
// function as BillingCustomerLock mutator.
type BillingCustomerLockFunc func(context.Context, *db.BillingCustomerLockMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f BillingCustomerLockFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.BillingCustomerLockMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.BillingCustomerLockMutation", m)
}

// The BillingCustomerOverrideFunc type is an adapter to allow the use of ordinary
// function as BillingCustomerOverride mutator.
type BillingCustomerOverrideFunc func(context.Context, *db.BillingCustomerOverrideMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f BillingCustomerOverrideFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.BillingCustomerOverrideMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.BillingCustomerOverrideMutation", m)
}

// The BillingInvoiceFunc type is an adapter to allow the use of ordinary
// function as BillingInvoice mutator.
type BillingInvoiceFunc func(context.Context, *db.BillingInvoiceMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f BillingInvoiceFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.BillingInvoiceMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.BillingInvoiceMutation", m)
}

// The BillingInvoiceFlatFeeLineConfigFunc type is an adapter to allow the use of ordinary
// function as BillingInvoiceFlatFeeLineConfig mutator.
type BillingInvoiceFlatFeeLineConfigFunc func(context.Context, *db.BillingInvoiceFlatFeeLineConfigMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f BillingInvoiceFlatFeeLineConfigFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.BillingInvoiceFlatFeeLineConfigMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.BillingInvoiceFlatFeeLineConfigMutation", m)
}

// The BillingInvoiceLineFunc type is an adapter to allow the use of ordinary
// function as BillingInvoiceLine mutator.
type BillingInvoiceLineFunc func(context.Context, *db.BillingInvoiceLineMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f BillingInvoiceLineFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.BillingInvoiceLineMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.BillingInvoiceLineMutation", m)
}

// The BillingInvoiceLineDiscountFunc type is an adapter to allow the use of ordinary
// function as BillingInvoiceLineDiscount mutator.
type BillingInvoiceLineDiscountFunc func(context.Context, *db.BillingInvoiceLineDiscountMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f BillingInvoiceLineDiscountFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.BillingInvoiceLineDiscountMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.BillingInvoiceLineDiscountMutation", m)
}

// The BillingInvoiceLineUsageDiscountFunc type is an adapter to allow the use of ordinary
// function as BillingInvoiceLineUsageDiscount mutator.
type BillingInvoiceLineUsageDiscountFunc func(context.Context, *db.BillingInvoiceLineUsageDiscountMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f BillingInvoiceLineUsageDiscountFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.BillingInvoiceLineUsageDiscountMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.BillingInvoiceLineUsageDiscountMutation", m)
}

// The BillingInvoiceUsageBasedLineConfigFunc type is an adapter to allow the use of ordinary
// function as BillingInvoiceUsageBasedLineConfig mutator.
type BillingInvoiceUsageBasedLineConfigFunc func(context.Context, *db.BillingInvoiceUsageBasedLineConfigMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f BillingInvoiceUsageBasedLineConfigFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.BillingInvoiceUsageBasedLineConfigMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.BillingInvoiceUsageBasedLineConfigMutation", m)
}

// The BillingInvoiceValidationIssueFunc type is an adapter to allow the use of ordinary
// function as BillingInvoiceValidationIssue mutator.
type BillingInvoiceValidationIssueFunc func(context.Context, *db.BillingInvoiceValidationIssueMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f BillingInvoiceValidationIssueFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.BillingInvoiceValidationIssueMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.BillingInvoiceValidationIssueMutation", m)
}

// The BillingProfileFunc type is an adapter to allow the use of ordinary
// function as BillingProfile mutator.
type BillingProfileFunc func(context.Context, *db.BillingProfileMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f BillingProfileFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.BillingProfileMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.BillingProfileMutation", m)
}

// The BillingSequenceNumbersFunc type is an adapter to allow the use of ordinary
// function as BillingSequenceNumbers mutator.
type BillingSequenceNumbersFunc func(context.Context, *db.BillingSequenceNumbersMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f BillingSequenceNumbersFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.BillingSequenceNumbersMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.BillingSequenceNumbersMutation", m)
}

// The BillingWorkflowConfigFunc type is an adapter to allow the use of ordinary
// function as BillingWorkflowConfig mutator.
type BillingWorkflowConfigFunc func(context.Context, *db.BillingWorkflowConfigMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f BillingWorkflowConfigFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.BillingWorkflowConfigMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.BillingWorkflowConfigMutation", m)
}

// The CustomerFunc type is an adapter to allow the use of ordinary
// function as Customer mutator.
type CustomerFunc func(context.Context, *db.CustomerMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f CustomerFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.CustomerMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.CustomerMutation", m)
}

// The CustomerSubjectsFunc type is an adapter to allow the use of ordinary
// function as CustomerSubjects mutator.
type CustomerSubjectsFunc func(context.Context, *db.CustomerSubjectsMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f CustomerSubjectsFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.CustomerSubjectsMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.CustomerSubjectsMutation", m)
}

// The EntitlementFunc type is an adapter to allow the use of ordinary
// function as Entitlement mutator.
type EntitlementFunc func(context.Context, *db.EntitlementMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f EntitlementFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.EntitlementMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.EntitlementMutation", m)
}

// The FeatureFunc type is an adapter to allow the use of ordinary
// function as Feature mutator.
type FeatureFunc func(context.Context, *db.FeatureMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f FeatureFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.FeatureMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.FeatureMutation", m)
}

// The GrantFunc type is an adapter to allow the use of ordinary
// function as Grant mutator.
type GrantFunc func(context.Context, *db.GrantMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f GrantFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.GrantMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.GrantMutation", m)
}

// The MeterFunc type is an adapter to allow the use of ordinary
// function as Meter mutator.
type MeterFunc func(context.Context, *db.MeterMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f MeterFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.MeterMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.MeterMutation", m)
}

// The NotificationChannelFunc type is an adapter to allow the use of ordinary
// function as NotificationChannel mutator.
type NotificationChannelFunc func(context.Context, *db.NotificationChannelMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f NotificationChannelFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.NotificationChannelMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.NotificationChannelMutation", m)
}

// The NotificationEventFunc type is an adapter to allow the use of ordinary
// function as NotificationEvent mutator.
type NotificationEventFunc func(context.Context, *db.NotificationEventMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f NotificationEventFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.NotificationEventMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.NotificationEventMutation", m)
}

// The NotificationEventDeliveryStatusFunc type is an adapter to allow the use of ordinary
// function as NotificationEventDeliveryStatus mutator.
type NotificationEventDeliveryStatusFunc func(context.Context, *db.NotificationEventDeliveryStatusMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f NotificationEventDeliveryStatusFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.NotificationEventDeliveryStatusMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.NotificationEventDeliveryStatusMutation", m)
}

// The NotificationRuleFunc type is an adapter to allow the use of ordinary
// function as NotificationRule mutator.
type NotificationRuleFunc func(context.Context, *db.NotificationRuleMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f NotificationRuleFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.NotificationRuleMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.NotificationRuleMutation", m)
}

// The PlanFunc type is an adapter to allow the use of ordinary
// function as Plan mutator.
type PlanFunc func(context.Context, *db.PlanMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f PlanFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.PlanMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.PlanMutation", m)
}

// The PlanPhaseFunc type is an adapter to allow the use of ordinary
// function as PlanPhase mutator.
type PlanPhaseFunc func(context.Context, *db.PlanPhaseMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f PlanPhaseFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.PlanPhaseMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.PlanPhaseMutation", m)
}

// The PlanRateCardFunc type is an adapter to allow the use of ordinary
// function as PlanRateCard mutator.
type PlanRateCardFunc func(context.Context, *db.PlanRateCardMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f PlanRateCardFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.PlanRateCardMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.PlanRateCardMutation", m)
}

// The SubscriptionFunc type is an adapter to allow the use of ordinary
// function as Subscription mutator.
type SubscriptionFunc func(context.Context, *db.SubscriptionMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f SubscriptionFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.SubscriptionMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.SubscriptionMutation", m)
}

// The SubscriptionAddonFunc type is an adapter to allow the use of ordinary
// function as SubscriptionAddon mutator.
type SubscriptionAddonFunc func(context.Context, *db.SubscriptionAddonMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f SubscriptionAddonFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.SubscriptionAddonMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.SubscriptionAddonMutation", m)
}

// The SubscriptionAddonQuantityFunc type is an adapter to allow the use of ordinary
// function as SubscriptionAddonQuantity mutator.
type SubscriptionAddonQuantityFunc func(context.Context, *db.SubscriptionAddonQuantityMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f SubscriptionAddonQuantityFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.SubscriptionAddonQuantityMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.SubscriptionAddonQuantityMutation", m)
}

// The SubscriptionAddonRateCardFunc type is an adapter to allow the use of ordinary
// function as SubscriptionAddonRateCard mutator.
type SubscriptionAddonRateCardFunc func(context.Context, *db.SubscriptionAddonRateCardMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f SubscriptionAddonRateCardFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.SubscriptionAddonRateCardMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.SubscriptionAddonRateCardMutation", m)
}

// The SubscriptionAddonRateCardItemLinkFunc type is an adapter to allow the use of ordinary
// function as SubscriptionAddonRateCardItemLink mutator.
type SubscriptionAddonRateCardItemLinkFunc func(context.Context, *db.SubscriptionAddonRateCardItemLinkMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f SubscriptionAddonRateCardItemLinkFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.SubscriptionAddonRateCardItemLinkMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.SubscriptionAddonRateCardItemLinkMutation", m)
}

// The SubscriptionItemFunc type is an adapter to allow the use of ordinary
// function as SubscriptionItem mutator.
type SubscriptionItemFunc func(context.Context, *db.SubscriptionItemMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f SubscriptionItemFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.SubscriptionItemMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.SubscriptionItemMutation", m)
}

// The SubscriptionPhaseFunc type is an adapter to allow the use of ordinary
// function as SubscriptionPhase mutator.
type SubscriptionPhaseFunc func(context.Context, *db.SubscriptionPhaseMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f SubscriptionPhaseFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.SubscriptionPhaseMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.SubscriptionPhaseMutation", m)
}

// The UsageResetFunc type is an adapter to allow the use of ordinary
// function as UsageReset mutator.
type UsageResetFunc func(context.Context, *db.UsageResetMutation) (db.Value, error)

// Mutate calls f(ctx, m).
func (f UsageResetFunc) Mutate(ctx context.Context, m db.Mutation) (db.Value, error) {
	if mv, ok := m.(*db.UsageResetMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *db.UsageResetMutation", m)
}

// Condition is a hook condition function.
type Condition func(context.Context, db.Mutation) bool

// And groups conditions with the AND operator.
func And(first, second Condition, rest ...Condition) Condition {
	return func(ctx context.Context, m db.Mutation) bool {
		if !first(ctx, m) || !second(ctx, m) {
			return false
		}
		for _, cond := range rest {
			if !cond(ctx, m) {
				return false
			}
		}
		return true
	}
}

// Or groups conditions with the OR operator.
func Or(first, second Condition, rest ...Condition) Condition {
	return func(ctx context.Context, m db.Mutation) bool {
		if first(ctx, m) || second(ctx, m) {
			return true
		}
		for _, cond := range rest {
			if cond(ctx, m) {
				return true
			}
		}
		return false
	}
}

// Not negates a given condition.
func Not(cond Condition) Condition {
	return func(ctx context.Context, m db.Mutation) bool {
		return !cond(ctx, m)
	}
}

// HasOp is a condition testing mutation operation.
func HasOp(op db.Op) Condition {
	return func(_ context.Context, m db.Mutation) bool {
		return m.Op().Is(op)
	}
}

// HasAddedFields is a condition validating `.AddedField` on fields.
func HasAddedFields(field string, fields ...string) Condition {
	return func(_ context.Context, m db.Mutation) bool {
		if _, exists := m.AddedField(field); !exists {
			return false
		}
		for _, field := range fields {
			if _, exists := m.AddedField(field); !exists {
				return false
			}
		}
		return true
	}
}

// HasClearedFields is a condition validating `.FieldCleared` on fields.
func HasClearedFields(field string, fields ...string) Condition {
	return func(_ context.Context, m db.Mutation) bool {
		if exists := m.FieldCleared(field); !exists {
			return false
		}
		for _, field := range fields {
			if exists := m.FieldCleared(field); !exists {
				return false
			}
		}
		return true
	}
}

// HasFields is a condition validating `.Field` on fields.
func HasFields(field string, fields ...string) Condition {
	return func(_ context.Context, m db.Mutation) bool {
		if _, exists := m.Field(field); !exists {
			return false
		}
		for _, field := range fields {
			if _, exists := m.Field(field); !exists {
				return false
			}
		}
		return true
	}
}

// If executes the given hook under condition.
//
//	hook.If(ComputeAverage, And(HasFields(...), HasAddedFields(...)))
func If(hk db.Hook, cond Condition) db.Hook {
	return func(next db.Mutator) db.Mutator {
		return db.MutateFunc(func(ctx context.Context, m db.Mutation) (db.Value, error) {
			if cond(ctx, m) {
				return hk(next).Mutate(ctx, m)
			}
			return next.Mutate(ctx, m)
		})
	}
}

// On executes the given hook only for the given operation.
//
//	hook.On(Log, db.Delete|db.Create)
func On(hk db.Hook, op db.Op) db.Hook {
	return If(hk, HasOp(op))
}

// Unless skips the given hook only for the given operation.
//
//	hook.Unless(Log, db.Update|db.UpdateOne)
func Unless(hk db.Hook, op db.Op) db.Hook {
	return If(hk, Not(HasOp(op)))
}

// FixedError is a hook returning a fixed error.
func FixedError(err error) db.Hook {
	return func(db.Mutator) db.Mutator {
		return db.MutateFunc(func(context.Context, db.Mutation) (db.Value, error) {
			return nil, err
		})
	}
}

// Reject returns a hook that rejects all operations that match op.
//
//	func (T) Hooks() []db.Hook {
//		return []db.Hook{
//			Reject(db.Delete|db.Update),
//		}
//	}
func Reject(op db.Op) db.Hook {
	hk := FixedError(fmt.Errorf("%s operation is not allowed", op))
	return On(hk, op)
}

// Chain acts as a list of hooks and is effectively immutable.
// Once created, it will always hold the same set of hooks in the same order.
type Chain struct {
	hooks []db.Hook
}

// NewChain creates a new chain of hooks.
func NewChain(hooks ...db.Hook) Chain {
	return Chain{append([]db.Hook(nil), hooks...)}
}

// Hook chains the list of hooks and returns the final hook.
func (c Chain) Hook() db.Hook {
	return func(mutator db.Mutator) db.Mutator {
		for i := len(c.hooks) - 1; i >= 0; i-- {
			mutator = c.hooks[i](mutator)
		}
		return mutator
	}
}

// Append extends a chain, adding the specified hook
// as the last ones in the mutation flow.
func (c Chain) Append(hooks ...db.Hook) Chain {
	newHooks := make([]db.Hook, 0, len(c.hooks)+len(hooks))
	newHooks = append(newHooks, c.hooks...)
	newHooks = append(newHooks, hooks...)
	return Chain{newHooks}
}

// Extend extends a chain, adding the specified chain
// as the last ones in the mutation flow.
func (c Chain) Extend(chain Chain) Chain {
	return c.Append(chain.hooks...)
}
