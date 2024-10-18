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

		extendDuration, err := datex.ISOString(val.ExtendDurationIso).Parse()
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

// patchCreator is a helper struct to create the different patch types based on value
//
// patchGetter should return the exact patch based on the known batch index
type patchCreator struct {
	patch       func(s *db.SubscriptionPatchCreate)
	addItem     func(s *db.SubscriptionPatchValueAddItemCreate, patchGetter func(batchIndex int) *db.SubscriptionPatch)
	addPhase    func(s *db.SubscriptionPatchValueAddPhaseCreate, patchGetter func(batchIndex int) *db.SubscriptionPatch)
	extendPhase func(s *db.SubscriptionPatchValueExtendPhaseCreate, patchGetter func(batchIndex int) *db.SubscriptionPatch)
}

// mapPatchesToCreates maps the subscription patches to the different create types
//
// This method is extracted so an error can be returned if there's a mapping error (as CreateBulk doesn't support errors).
// As a side-effect all these value references come from this closure.
func mapPatchesToCreates(subscriptionID models.NamespacedID, patches []subscription.CreateSubscriptionPatchInput) ([]patchCreator, error) {
	creates := make([]patchCreator, 0, len(patches))
	for i := range patches {
		patchCreator := patchCreator{
			patch: func(s *db.SubscriptionPatchCreate) {
				s.SetSubscriptionID(subscriptionID.ID).
					SetNamespace(subscriptionID.Namespace).
					SetAppliedAt(patches[i].AppliedAt).
					SetBatchIndex(patches[i].BatchIndex).
					SetOperation(string(patches[i].Op())).
					SetPath(string(patches[i].Path()))
			},
		}

		if patches[i].Op() == subscription.PatchOperationAdd && patches[i].Path().Type() == subscription.PatchPathTypeItem {
			p, ok := patches[i].Patch.(subscription.PatchAddItem)
			if !ok {
				return nil, fmt.Errorf("unexpected patch type %T based on Op and Path should have been %T", patches[i].Patch, subscription.PatchAddItem{})
			}

			patchCreator.addItem = func(s *db.SubscriptionPatchValueAddItemCreate, patchGetter func(batchIndex int) *db.SubscriptionPatch) {
				val := p.Value()
				dbPatch := patchGetter(patches[i].BatchIndex)

				s.SetNamespace(dbPatch.Namespace).
					SetPhaseKey(val.PhaseKey).
					SetItemKey(val.ItemKey).
					SetSubscriptionPatchID(dbPatch.ID)

				if val.FeatureKey != nil {
					s.SetFeatureKey(*val.FeatureKey)
				}

				if v := val.CreateEntitlementInput; v != nil {
					s.SetCreateEntitlementEntitlementType(string(v.EntitlementType))

					if v := val.CreateEntitlementInput.MeasureUsageFrom; v != nil {
						s.SetCreateEntitlementMeasureUsageFrom(v.Get())
					}
					if v := val.CreateEntitlementInput.IssueAfterReset; v != nil {
						s.SetCreateEntitlementIssueAfterReset(*v)
					}
					if v := val.CreateEntitlementInput.IssueAfterResetPriority; v != nil {
						s.SetCreateEntitlementIssueAfterResetPriority(*v)
					}
					if v := val.CreateEntitlementInput.IsSoftLimit; v != nil {
						s.SetCreateEntitlementIsSoftLimit(*v)
					}
					if v := val.CreateEntitlementInput.PreserveOverageAtReset; v != nil {
						s.SetCreateEntitlementPreserveOverageAtReset(*v)
					}
					if v := val.CreateEntitlementInput.Config; v != nil {
						s.SetCreateEntitlementConfig(v)
					}
					if v := val.CreateEntitlementInput.UsagePeriod; v != nil {
						s.SetCreateEntitlementUsagePeriodAnchor(v.Anchor)
						s.SetCreateEntitlementUsagePeriodInterval(string(v.Interval))
					}
				}

				if v := val.CreatePriceInput; v != nil {
					s.SetCreatePriceValue(v.Value)
				}
			}

		} else if patches[i].Op() == subscription.PatchOperationAdd && patches[i].Path().Type() == subscription.PatchPathTypePhase {
			p, ok := patches[i].Patch.(subscription.PatchAddPhase)
			if !ok {
				return nil, fmt.Errorf("unexpected patch type %T based on Op and Path should have been %T", patches[i].Patch, subscription.PatchAddPhase{})
			}

			patchCreator.addPhase = func(s *db.SubscriptionPatchValueAddPhaseCreate, patchGetter func(batchIndex int) *db.SubscriptionPatch) {
				val := p.Value()
				dbPatch := patchGetter(patches[i].BatchIndex)

				s.SetNamespace(dbPatch.Namespace).
					SetSubscriptionPatchID(dbPatch.ID).
					SetPhaseKey(val.PhaseKey).
					SetStartAfterIso(val.StartAfter.String())

				if val.CreateDiscountInput != nil {
					s.SetCreateDiscount(true)
					// TODO: add discount,
				} else {
					s.SetCreateDiscount(false)
				}
			}
		} else if patches[i].Op() == subscription.PatchOperationExtend && patches[i].Path().Type() == subscription.PatchPathTypePhase {
			p, ok := patches[i].Patch.(subscription.PatchExtendPhase)
			if !ok {
				return nil, fmt.Errorf("unexpected patch type %T based on Op and Path should have been %T", patches[i].Patch, subscription.PatchExtendPhase{})
			}

			patchCreator.extendPhase = func(s *db.SubscriptionPatchValueExtendPhaseCreate, patchGetter func(batchIndex int) *db.SubscriptionPatch) {
				val := p.Value()
				dbPatch := patchGetter(patches[i].BatchIndex)

				s.SetNamespace(dbPatch.Namespace).
					SetSubscriptionPatchID(dbPatch.ID).
					SetPhaseKey(p.PhaseKey).
					SetExtendDurationIso(val.String())
			}
		}

		creates = append(creates, patchCreator)
	}
	return creates, nil
}
