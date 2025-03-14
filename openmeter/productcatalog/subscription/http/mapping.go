package httpdriver

import (
	"fmt"
	"slices"
	"time"

	"github.com/invopop/gobl/currency"
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
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
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

		var sa isodate.Period
		if apiP.Phase.StartAfter != nil {
			saStr := isodate.String(*apiP.Phase.StartAfter)
			sa, err = saStr.Parse()
			if err != nil {
				return nil, fmt.Errorf("failed to parse start after: %w", err)
			}
		}

		var dur *isodate.Period

		if apiP.Phase.Duration != nil {
			dS := isodate.String(*apiP.Phase.Duration)
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

		durStr := isodate.String(apiP.ExtendBy)
		d, err := durStr.Parse()
		if err != nil {
			return nil, fmt.Errorf("failed to parse duration: %w", err)
		}

		p := patch.PatchStretchPhase{
			PhaseKey: apiP.PhaseKey,
			Duration: d,
		}

		return p, nil
	case string(api.EditSubscriptionUnscheduleEditOpUnscheduleEdit):
		p := patch.PatchUnscheduleEdit{}
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
		Alignment: &api.Alignment{
			BillablesMustAlign: &sub.BillablesMustAlign,
		},
		IsCustom: sub.IsCustom(),
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

func MapAPITimingToTiming(apiTiming api.SubscriptionTiming) (subscription.Timing, error) {
	var res subscription.Timing

	t, err := apiTiming.AsSubscriptionTiming1()
	if err != nil {
		e, err := apiTiming.AsSubscriptionTimingEnum()
		if err != nil {
			return res, fmt.Errorf("failed to cast to SubscriptionChangeTiming: %w", err)
		}

		res.Enum = lo.ToPtr(subscription.TimingEnum(e))
	} else {
		res.Custom = &t
	}

	return res, nil
}

// We map the items as follows:
// - for the current phase, the API will only return the active item for each key
// - for past phases, the API will return the last item for each key
// - for future phases, the API will return the first version
func MapSubscriptionPhaseToAPI(subView subscription.SubscriptionView, phaseView subscription.SubscriptionPhaseView) (api.SubscriptionPhaseExpanded, error) {
	var endOfPhase *time.Time
	if dur, err := subView.Spec.GetPhaseCadence(phaseView.SubscriptionPhase.Key); err == nil {
		endOfPhase = dur.ActiveTo
	}

	now := clock.Now()
	currPhase, currExists := subView.Spec.GetCurrentPhaseAt(now)

	flatItems := lo.Flatten(lo.Values(phaseView.ItemsByKey))
	apiItems := make([]api.SubscriptionItem, 0, len(flatItems))

	apiItemTimelines := make(map[string][]api.SubscriptionItem)

	var relativePhaseTime string

	if currExists && currPhase.PhaseKey == phaseView.SubscriptionPhase.Key {
		relativePhaseTime = "current"
	} else if phaseView.SubscriptionPhase.ActiveFrom.After(now) {
		relativePhaseTime = "future"
	} else {
		relativePhaseTime = "past"
	}

	for key, items := range phaseView.ItemsByKey {
		// Let's add the items to the timeline
		timeline, err := slicesx.MapWithErr(items, func(item subscription.SubscriptionItemView) (api.SubscriptionItem, error) {
			apiItem, err := MapSubscriptionItemToAPI(item)
			if err != nil {
				return api.SubscriptionItem{}, err
			}

			return apiItem, nil
		})
		if err != nil {
			return api.SubscriptionPhaseExpanded{}, err
		}

		apiItemTimelines[key] = timeline

		// Then let's add the items to the flat list
		switch relativePhaseTime {
		// If this is the current phase
		case "current":
			// Let's find if there's a current item, if so add that to the output
			curr := slices.IndexFunc(items, func(i subscription.SubscriptionItemView) bool {
				return i.SubscriptionItem.IsActiveAt(now)
			})

			if curr != -1 {
				apiItem, err := MapSubscriptionItemToAPI(items[curr])
				if err != nil {
					return api.SubscriptionPhaseExpanded{}, err
				}

				apiItems = append(apiItems, apiItem)
			}

			continue
		// If this is a future phase lets add the first item
		case "future":
			if len(items) > 0 {
				apiItem, err := MapSubscriptionItemToAPI(items[0])
				if err != nil {
					return api.SubscriptionPhaseExpanded{}, err
				}

				apiItems = append(apiItems, apiItem)
			}

			continue
		// If this is a past phase
		case "past":
			// Let's find the last item
			if len(items) > 0 {
				apiItem, err := MapSubscriptionItemToAPI(items[len(items)-1])
				if err != nil {
					return api.SubscriptionPhaseExpanded{}, err
				}

				apiItems = append(apiItems, apiItem)
			}

			continue
		default:
			return api.SubscriptionPhaseExpanded{}, fmt.Errorf("no logical branch enetered: %s", relativePhaseTime)
		}
	}

	return api.SubscriptionPhaseExpanded{
		ActiveFrom:    phaseView.SubscriptionPhase.ActiveFrom,
		ActiveTo:      endOfPhase,
		CreatedAt:     phaseView.SubscriptionPhase.CreatedAt,
		UpdatedAt:     phaseView.SubscriptionPhase.UpdatedAt,
		DeletedAt:     phaseView.SubscriptionPhase.DeletedAt,
		Description:   phaseView.SubscriptionPhase.Description,
		Discounts:     nil, // TODO: add discounts
		Id:            phaseView.SubscriptionPhase.ID,
		Items:         apiItems,
		ItemTimelines: apiItemTimelines,
		Key:           phaseView.SubscriptionPhase.Key,
		Metadata:      &phaseView.SubscriptionPhase.Metadata,
		Name:          phaseView.SubscriptionPhase.Name,
	}, nil
}

func MapSubscriptionViewToAPI(view subscription.SubscriptionView) (api.SubscriptionExpanded, error) {
	apiSub := MapSubscriptionToAPI(view.Subscription)
	alg := api.SubscriptionAlignment{
		BillablesMustAlign: apiSub.Alignment.BillablesMustAlign,
	}

	if view.Subscription.BillablesMustAlign {
		if currPhase, ok := view.Spec.GetCurrentPhaseAt(clock.Now()); ok && currPhase.HasBillables() {
			period, err := view.Spec.GetAlignedBillingPeriodAt(currPhase.PhaseKey, clock.Now())
			if err != nil {
				// GetAlignedBillingPeriodAt cannot be calculated for all aligned subscriptions.
				if _, ok := lo.ErrorsAs[subscription.NoBillingPeriodError](err); !ok {
					return api.SubscriptionExpanded{}, err
				}
			}

			if err == nil {
				alg.CurrentAlignedBillingPeriod = &api.Period{
					From: period.From,
					To:   period.To,
				}
			}
		}
	}

	base := api.SubscriptionExpanded{
		ActiveFrom:  apiSub.ActiveFrom,
		ActiveTo:    apiSub.ActiveTo,
		CreatedAt:   apiSub.CreatedAt,
		Currency:    apiSub.Currency,
		CustomerId:  apiSub.CustomerId,
		DeletedAt:   apiSub.DeletedAt,
		Description: apiSub.Description,
		Id:          apiSub.Id,
		Metadata:    apiSub.Metadata,
		Name:        apiSub.Name,
		Phases:      nil,
		Plan:        apiSub.Plan,
		UpdatedAt:   apiSub.UpdatedAt,
		Status:      apiSub.Status,
		Alignment:   &alg,
		IsCustom:    apiSub.IsCustom,
	}

	phases := make([]api.SubscriptionPhaseExpanded, 0, len(view.Phases))
	for _, phase := range view.Phases {
		phaseAPI, err := MapSubscriptionPhaseToAPI(view, phase)
		if err != nil {
			return base, err
		}

		phases = append(phases, phaseAPI)
	}

	base.Phases = phases

	return base, nil
}

func CustomPlanToCreatePlanRequest(a api.CustomPlanInput, namespace string) (planhttpdriver.CreatePlanRequest, error) {
	var err error

	req := planhttpdriver.CreatePlanRequest{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:        a.Name,
				Description: a.Description,
				Metadata:    lo.FromPtrOr(a.Metadata, nil),
			},
			Phases: nil,
		},
	}

	if a.Alignment != nil && a.Alignment.BillablesMustAlign != nil {
		req.Plan.PlanMeta.Alignment = productcatalog.Alignment{
			BillablesMustAlign: *a.Alignment.BillablesMustAlign,
		}
	}

	req.Currency = currency.Code(a.Currency)
	if err = req.Currency.Validate(); err != nil {
		return req, fmt.Errorf("invalid CurrencyCode: %w", err)
	}

	if len(a.Phases) > 0 {
		req.Phases = make([]productcatalog.Phase, 0, len(a.Phases))

		for _, phase := range a.Phases {
			planPhase, err := planhttpdriver.AsPlanPhase(phase)
			if err != nil {
				return req, fmt.Errorf("failed to cast PlanPhase: %w", err)
			}

			req.Phases = append(req.Phases, planPhase)
		}
	}

	return req, nil
}
