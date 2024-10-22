package adapter

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/applieddiscount"
	"github.com/openmeterio/openmeter/openmeter/subscription/price"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

func MapDBSubscription(sub *db.Subscription) (subscription.Subscription, error) {
	if sub == nil {
		return subscription.Subscription{}, fmt.Errorf("unexpected nil subscription")
	}

	return subscription.Subscription{
		ID: sub.ID,
		NamespacedModel: models.NamespacedModel{
			Namespace: sub.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: sub.CreatedAt,
			UpdatedAt: sub.UpdatedAt,
			DeletedAt: sub.DeletedAt,
		},
		CreateSubscriptionInput: subscription.CreateSubscriptionInput{
			Plan: subscription.PlanRef{
				Key:     sub.PlanKey,
				Version: sub.PlanVersion,
			},
			CustomerId: sub.CustomerID,
			Currency:   sub.Currency,
			CadencedModel: models.CadencedModel{
				ActiveFrom: sub.ActiveFrom,
				ActiveTo:   sub.ActiveTo,
			},
		},
	}, nil
}

func MapDBSubscriptionPatch(patch *db.SubscriptionPatch) (subscription.SubscriptionPatch, error) {
	if patch == nil {
		return subscription.SubscriptionPatch{}, fmt.Errorf("unexpected nil subscription patch")
	}

	sp := subscription.SubscriptionPatch{
		NamespacedModel: models.NamespacedModel{
			Namespace: patch.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: patch.CreatedAt,
			UpdatedAt: patch.UpdatedAt,
			DeletedAt: patch.DeletedAt,
		},
		ID:             patch.ID,
		SubscriptionId: patch.SubscriptionID,
		AppliedAt:      patch.AppliedAt,
		BatchIndex:     patch.BatchIndex,
		Operation:      patch.Operation,
		Path:           patch.Path,
	}

	pPath := subscription.PatchPath(sp.Path)
	if err := pPath.Validate(); err != nil {
		return subscription.SubscriptionPatch{}, err
	}

	pOp := subscription.PatchOperation(sp.Operation)
	if err := pOp.Validate(); err != nil {
		return subscription.SubscriptionPatch{}, err
	}

	if pPath.Type() == subscription.PatchPathTypeItem && pOp == subscription.PatchOperationAdd {
		val, err := patch.Edges.ValueAddItemOrErr()
		if err != nil {
			return subscription.SubscriptionPatch{}, err
		}

		// We use the full patch for type hinting
		p := subscription.PatchAddItem{
			CreateInput: subscription.SubscriptionItemSpec{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					PhaseKey:   val.PhaseKey,
					ItemKey:    val.ItemKey,
					FeatureKey: val.FeatureKey,
				},
			},
		}

		// Type is required field for all entitlement types, so we know an entitlement should be defined
		if val.CreateEntitlementEntitlementType != nil {
			p.CreateInput.CreateEntitlementInput = &subscription.CreateSubscriptionEntitlementSpec{
				EntitlementType:         entitlement.EntitlementType(*val.CreateEntitlementEntitlementType),
				IssueAfterReset:         val.CreateEntitlementIssueAfterReset,
				IssueAfterResetPriority: val.CreateEntitlementIssueAfterResetPriority,
				IsSoftLimit:             val.CreateEntitlementIsSoftLimit,
				Config:                  val.CreateEntitlementConfig,
				PreserveOverageAtReset:  val.CreateEntitlementPreserveOverageAtReset,
			}

			if val.CreateEntitlementMeasureUsageFrom != nil {
				m := &entitlement.MeasureUsageFromInput{}
				// We ignore the error
				err = m.FromTime(*val.CreateEntitlementMeasureUsageFrom)
				if err != nil {
					return subscription.SubscriptionPatch{}, fmt.Errorf("failed to map measure usage from: %w", err)
				}
				p.CreateInput.CreateEntitlementInput.MeasureUsageFrom = m
			}

			if val.CreateEntitlementUsagePeriodInterval != nil && val.CreateEntitlementUsagePeriodAnchor != nil {
				p.CreateInput.CreateEntitlementInput.UsagePeriod = &entitlement.UsagePeriod{
					Anchor:   val.CreateEntitlementUsagePeriodAnchor.In(time.UTC),
					Interval: recurrence.RecurrenceInterval(*val.CreateEntitlementUsagePeriodInterval),
				}
			}
		}

		if val.CreatePriceValue != nil {
			p.CreateInput.CreatePriceInput = &price.Spec{
				Value:    *val.CreatePriceValue,
				PhaseKey: val.PhaseKey,
				ItemKey:  val.ItemKey,
			}
		}

		sp.Value = p.Value()
	} else if pPath.Type() == subscription.PatchPathTypePhase && pOp == subscription.PatchOperationAdd {
		val, err := patch.Edges.ValueAddPhaseOrErr()
		if err != nil {
			return subscription.SubscriptionPatch{}, err
		}

		startAfter, err := datex.ISOString(val.StartAfterIso).Parse()
		if err != nil {
			return subscription.SubscriptionPatch{}, fmt.Errorf("failed to parse start after: %w", err)
		}

		// We use the full patch for type hinting
		p := subscription.PatchAddPhase{
			PhaseKey: val.PhaseKey,
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   val.PhaseKey,
					StartAfter: startAfter,
				},
			},
		}

		if val.CreateDiscount {
			p.CreateInput.CreateSubscriptionPhaseCustomerInput.CreateDiscountInput = &applieddiscount.Spec{
				PhaseKey:  val.PhaseKey,
				AppliesTo: val.CreateDiscountAppliesTo,
			}
		}

		sp.Value = p.Value()
	} else if pPath.Type() == subscription.PatchPathTypePhase && pOp == subscription.PatchOperationExtend {
		val, err := patch.Edges.ValueExtendPhaseOrErr()
		if err != nil {
			return subscription.SubscriptionPatch{}, err
		}

		extendDuration, err := datex.ISOString(val.ExtendDuration).Parse()
		if err != nil {
			return subscription.SubscriptionPatch{}, fmt.Errorf("failed to parse extend duration: %w", err)
		}

		// We use the full patch for type hinting
		p := subscription.PatchExtendPhase{
			PhaseKey: val.PhaseKey,
			Duration: extendDuration,
		}

		sp.Value = p.Value()
	}

	return sp, nil
}
