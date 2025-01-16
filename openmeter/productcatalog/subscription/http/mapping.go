package httpdriver

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	plandriver "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
	planhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datex"
)

func MapAPISubscriptionEditOperationToPatch(apiPatch api.SubscriptionEditOperation) (subscription.Patch, error) {
	disc, err := apiPatch.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to get discriminator: %w", err)
	}

	switch disc {
	case string(api.EditSubscriptionAddItemOpAddItem):
		apiP, err := apiPatch.AsEditSubscriptionAddItem()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to EditSubscriptionAddItem: %w", err)
		}

		// Let's parse and validate value.
		// Fortunately TypeSpec to OpenAPI generation is utterly logical and consistent, so we have to work with a structurally identical but differently named type.
		planRC, err := plandriver.AsRateCard(apiP.RateCard)
		if err != nil {
			return nil, fmt.Errorf("failed to cast to RateCard: %w", err)
		}

		sPRC := &plansubscription.RateCard{
			PhaseKey: apiP.PhaseKey,
			RateCard: planRC,
		}

		p := patch.PatchAddItem{
			PhaseKey: apiP.PhaseKey,
			ItemKey:  planRC.Key(),
			CreateInput: subscription.SubscriptionItemSpec{
				CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput:     sPRC.ToCreateSubscriptionItemPlanInput(),
					CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{},
				},
			},
		}

		return p, nil
	case string(api.EditSubscriptionRemoveItemOpRemoveItem):
		apiP, err := apiPatch.AsEditSubscriptionRemoveItem()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to EditSubscriptionRemoveItem: %w", err)
		}

		p := patch.PatchRemoveItem{
			PhaseKey: apiP.PhaseKey,
			ItemKey:  apiP.ItemKey,
		}

		return p, nil
	case string(api.EditSubscriptionAddPhaseOpAddPhase):
		apiP, err := apiPatch.AsEditSubscriptionAddPhase()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to EditSubscriptionAddPhase: %w", err)
		}

		var sa datex.Period
		if apiP.Phase.StartAfter != nil {
			saStr := datex.ISOString(*apiP.Phase.StartAfter)
			sa, err = saStr.Parse()
			if err != nil {
				return nil, fmt.Errorf("failed to parse start after: %w", err)
			}
		}

		var dur *datex.Period

		if apiP.Phase.Duration != nil {
			dS := datex.ISOString(*apiP.Phase.Duration)
			d, err := dS.Parse()
			if err != nil {
				return nil, fmt.Errorf("failed to parse duration: %w", err)
			}

			dur = &d
		}

		p := patch.PatchAddPhase{
			PhaseKey: apiP.Phase.Key,
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				Duration: dur,
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:    apiP.Phase.Key,
					StartAfter:  sa,
					Name:        apiP.Phase.Name,
					Description: apiP.Phase.Description,
				},
				CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{},
			},
		}

		return p, nil
	case string(api.EditSubscriptionRemovePhaseOpRemovePhase):
		apiP, err := apiPatch.AsEditSubscriptionRemovePhase()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to EditSubscriptionRemovePhase: %w", err)
		}

		var shift subscription.RemoveSubscriptionPhaseShifting

		if apiP.Shift == api.RemovePhaseShiftingNext {
			shift = subscription.RemoveSubscriptionPhaseShiftNext
		} else if apiP.Shift == api.RemovePhaseShiftingPrev {
			shift = subscription.RemoveSubscriptionPhaseShiftPrev
		} else {
			return nil, fmt.Errorf("unknown shift value: %s", apiP.Shift)
		}

		p := patch.PatchRemovePhase{
			PhaseKey: apiP.PhaseKey,
			RemoveInput: subscription.RemoveSubscriptionPhaseInput{
				Shift: shift,
			},
		}

		return p, nil
	case string(api.EditSubscriptionStretchPhaseOpStretchPhase):
		apiP, err := apiPatch.AsEditSubscriptionStretchPhase()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to EditSubscriptionStretchPhase: %w", err)
		}

		durStr := datex.ISOString(apiP.ExtendBy)
		d, err := durStr.Parse()
		if err != nil {
			return nil, fmt.Errorf("failed to parse duration: %w", err)
		}

		p := patch.PatchStretchPhase{
			PhaseKey: apiP.PhaseKey,
			Duration: d,
		}

		return p, nil
	default:
		return nil, fmt.Errorf("unknown discriminator: %s", disc)
	}
}

func MapSubscriptionToAPI(sub subscription.Subscription) api.Subscription {
	var ref *api.PlanReference

	if sub.PlanRef != nil {
		ref = &api.PlanReference{
			Id:      sub.PlanRef.Id,
			Key:     sub.PlanRef.Key,
			Version: sub.PlanRef.Version,
		}
	}

	return api.Subscription{
		Id:          sub.ID,
		ActiveFrom:  sub.ActiveFrom,
		ActiveTo:    sub.ActiveTo,
		CustomerId:  sub.CustomerId,
		Currency:    string(sub.Currency),
		Description: sub.Description,
		Name:        sub.Name,
		Status:      api.SubscriptionStatus(sub.GetStatusAt(clock.Now())),
		Plan:        ref,
		Metadata:    &sub.Metadata,
		CreatedAt:   sub.CreatedAt,
		UpdatedAt:   sub.UpdatedAt,
		DeletedAt:   sub.DeletedAt,
	}
}

func MapSubscriptionItemToAPI(item subscription.SubscriptionItemView) (api.SubscriptionItem, error) {
	var included *api.SubscriptionItemIncluded

	// TODO: add feature to view

	if item.Entitlement != nil {
		apiEnt, err := entitlementdriver.Parser.ToAPIGeneric(&item.Entitlement.Entitlement)
		if err != nil {
			return api.SubscriptionItem{}, err
		}

		included = &api.SubscriptionItemIncluded{
			Entitlement: apiEnt,
		}
	}

	var tx *api.TaxConfig

	if item.SubscriptionItem.RateCard.TaxConfig != nil {
		txv := planhttpdriver.FromTaxConfig(*item.SubscriptionItem.RateCard.TaxConfig)
		tx = &txv
	}

	var pr api.SubscriptionItem_Price

	if item.SubscriptionItem.RateCard.Price != nil {
		prc, err := MapPriceToAPI(*item.SubscriptionItem.RateCard.Price)
		if err != nil {
			return api.SubscriptionItem{}, err
		}

		pr = prc
	}

	return api.SubscriptionItem{
		ActiveFrom:     item.SubscriptionItem.ActiveFrom,
		ActiveTo:       item.SubscriptionItem.ActiveTo,
		BillingCadence: (*string)(item.SubscriptionItem.RateCard.BillingCadence.ISOStringPtrOrNil()),
		CreatedAt:      item.SubscriptionItem.CreatedAt,
		DeletedAt:      item.SubscriptionItem.DeletedAt,
		Description:    item.SubscriptionItem.Description,
		Id:             item.SubscriptionItem.ID,
		Included:       included,
		Key:            item.SubscriptionItem.Key,
		FeatureKey:     item.SubscriptionItem.RateCard.FeatureKey,
		Metadata:       &item.SubscriptionItem.Metadata,
		Name:           item.SubscriptionItem.Name,
		Price:          pr,
		TaxConfig:      tx,
		UpdatedAt:      item.SubscriptionItem.UpdatedAt,
	}, nil
}

func MapPriceToAPI(price productcatalog.Price) (api.SubscriptionItem_Price, error) {
	var res api.SubscriptionItem_Price

	switch price.Type() {
	case productcatalog.FlatPriceType:
		flatPrice, err := price.AsFlat()
		if err != nil {
			return res, fmt.Errorf("failed to cast FlatPrice: %w", err)
		}

		err = res.FromSubscriptionItemPrice0(api.FlatPriceWithPaymentTerm{
			Amount:      flatPrice.Amount.String(),
			PaymentTerm: lo.ToPtr(planhttpdriver.FromPaymentTerm(flatPrice.PaymentTerm)),
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
		})
		if err != nil {
			return res, fmt.Errorf("failed to cast FlatPrice: %w", err)
		}
	case productcatalog.UnitPriceType:
		unitPrice, err := price.AsUnit()
		if err != nil {
			return res, fmt.Errorf("failed to cast UnitPrice: %w", err)
		}

		var minimumAmount *string
		if unitPrice.MinimumAmount != nil {
			minimumAmount = lo.ToPtr(unitPrice.MinimumAmount.String())
		}

		var maximumAmount *string
		if unitPrice.MaximumAmount != nil {
			maximumAmount = lo.ToPtr(unitPrice.MaximumAmount.String())
		}

		err = res.FromSubscriptionItemPrice1(api.UnitPriceWithCommitments{
			Amount:        unitPrice.Amount.String(),
			MinimumAmount: minimumAmount,
			MaximumAmount: maximumAmount,
			Type:          api.UnitPriceWithCommitmentsTypeUnit,
		})
		if err != nil {
			return res, fmt.Errorf("failed to cast UnitPrice: %w", err)
		}
	case productcatalog.TieredPriceType:
		tieredPrice, err := price.AsTiered()
		if err != nil {
			return res, fmt.Errorf("failed to cast TieredPrice: %w", err)
		}

		var minimumAmount *string
		if tieredPrice.MinimumAmount != nil {
			minimumAmount = lo.ToPtr(tieredPrice.MinimumAmount.String())
		}

		var maximumAmount *string
		if tieredPrice.MaximumAmount != nil {
			maximumAmount = lo.ToPtr(tieredPrice.MaximumAmount.String())
		}

		err = res.FromSubscriptionItemPrice2(api.TieredPriceWithCommitments{
			Type:          api.TieredPriceWithCommitmentsTypeTiered,
			Mode:          api.TieredPriceMode(tieredPrice.Mode),
			MinimumAmount: minimumAmount,
			MaximumAmount: maximumAmount,
			Tiers: lo.Map(tieredPrice.Tiers, func(t productcatalog.PriceTier, _ int) api.PriceTier {
				var upToAmount *string
				if t.UpToAmount != nil {
					upToAmount = lo.ToPtr(t.UpToAmount.String())
				}

				var unitPrice *api.UnitPrice
				if t.UnitPrice != nil {
					unitPrice = &api.UnitPrice{
						Type:   api.UnitPriceTypeUnit,
						Amount: t.UnitPrice.Amount.String(),
					}
				}

				var flatPrice *api.FlatPrice
				if t.FlatPrice != nil {
					flatPrice = &api.FlatPrice{
						Type:   api.FlatPriceTypeFlat,
						Amount: t.FlatPrice.Amount.String(),
					}
				}

				return api.PriceTier{
					UpToAmount: upToAmount,
					UnitPrice:  unitPrice,
					FlatPrice:  flatPrice,
				}
			}),
		})
		if err != nil {
			return res, fmt.Errorf("failed to cast TieredPrice: %w", err)
		}
	default:
		return res, fmt.Errorf("unknown price type: %s", price.Type())
	}

	return res, nil
}

func MapSubscriptionPhaseToAPI(phaseView subscription.SubscriptionPhaseView, endOfPhase *time.Time) (api.SubscriptionPhaseExpanded, error) {
	flatItems := lo.Flatten(lo.Values(phaseView.ItemsByKey))
	items := make([]api.SubscriptionItem, 0, len(flatItems))

	for _, item := range flatItems {
		apiItem, err := MapSubscriptionItemToAPI(item)
		if err != nil {
			return api.SubscriptionPhaseExpanded{}, err
		}

		items = append(items, apiItem)
	}

	return api.SubscriptionPhaseExpanded{
		ActiveFrom:  phaseView.SubscriptionPhase.ActiveFrom,
		ActiveTo:    endOfPhase,
		CreatedAt:   phaseView.SubscriptionPhase.CreatedAt,
		UpdatedAt:   phaseView.SubscriptionPhase.UpdatedAt,
		DeletedAt:   phaseView.SubscriptionPhase.DeletedAt,
		Description: phaseView.SubscriptionPhase.Description,
		Discounts:   nil, // TODO: add discounts
		Id:          phaseView.SubscriptionPhase.ID,
		// TODO: maybe API should also use ItemsByKey?
		Items:    items,
		Key:      phaseView.SubscriptionPhase.Key,
		Metadata: &phaseView.SubscriptionPhase.Metadata,
		Name:     phaseView.SubscriptionPhase.Name,
	}, nil
}

func MapAPISubscriptionToAPIExpanded(sub api.Subscription) api.SubscriptionExpanded {
	return api.SubscriptionExpanded{
		ActiveFrom:  sub.ActiveFrom,
		ActiveTo:    sub.ActiveTo,
		CreatedAt:   sub.CreatedAt,
		Currency:    sub.Currency,
		CustomerId:  sub.CustomerId,
		DeletedAt:   sub.DeletedAt,
		Description: sub.Description,
		Id:          sub.Id,
		Metadata:    sub.Metadata,
		Name:        sub.Name,
		Phases:      nil,
		Plan:        sub.Plan,
		UpdatedAt:   sub.UpdatedAt,
		Status:      sub.Status,
	}
}

func MapSubscriptionViewToAPI(view subscription.SubscriptionView) (api.SubscriptionExpanded, error) {
	base := MapAPISubscriptionToAPIExpanded(MapSubscriptionToAPI(view.Subscription))

	phases := make([]api.SubscriptionPhaseExpanded, 0, len(view.Phases))
	for _, phase := range view.Phases {
		var endOfPhase *time.Time
		if dur, err := view.Spec.GetPhaseCadence(phase.SubscriptionPhase.Key); err == nil {
			endOfPhase = dur.ActiveTo
		}

		phaseAPI, err := MapSubscriptionPhaseToAPI(phase, endOfPhase)
		if err != nil {
			return base, err
		}

		phases = append(phases, phaseAPI)
	}

	base.Phases = phases

	return base, nil
}
