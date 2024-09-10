package httpdriver

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	plandriver "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
	planhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionplan "github.com/openmeterio/openmeter/openmeter/subscription/adapters/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	"github.com/openmeterio/openmeter/pkg/datex"
)

func MapAPISubscriptionItemPatchToPatch(apiPatch api.SubscriptionItemPatch) (subscription.Patch, error) {
	// Let's map explicitly

	typ, err := apiPatch.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to get discriminator: %w", err)
	}

	switch typ {
	case string(subscription.PatchOperationAdd):
		apiP, err := apiPatch.AsSubscriptionEditAddSubscriptionItemPatchKey()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to SubscriptionEditAddSubscriptionItemPatchKey: %w", err)
		}

		// Let's parse and validate path and op

		path := subscription.PatchPath(apiP.Path)
		if err := path.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate path: %w", err)
		}

		if path.Type() != subscription.PatchPathTypeItem {
			return nil, fmt.Errorf("invalid path %s, wrong type: %s", path, path.Type())
		}

		op := subscription.PatchOperation(apiP.Op)
		if err := op.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate operation: %w", err)
		}

		// Let's parse and validate value
		planRC, err := plandriver.AsRateCard(apiP.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to cast to RateCard: %w", err)
		}

		sPRC := &subscriptionplan.SubscriptionPlanRateCard{
			PhaseKey: path.PhaseKey(),
			RateCard: planRC,
		}

		return patch.PatchAddItem{
			PhaseKey: path.PhaseKey(),
			ItemKey:  path.ItemKey(),
			CreateInput: subscription.SubscriptionItemSpec{
				CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput: sPRC.ToCreateSubscriptionItemPlanInput(),
				},
			},
		}, nil
	case string(subscription.PatchOperationRemove):
		apiP, err := apiPatch.AsSubscriptionEditRemoveSubscriptionItemPatchKey()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to SubscriptionEditRemoveSubscriptionItemPatchKey: %w", err)
		}

		// Let's parse and validate path and op

		path := subscription.PatchPath(apiP.Path)
		if err := path.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate path: %w", err)
		}

		if path.Type() != subscription.PatchPathTypeItem {
			return nil, fmt.Errorf("invalid path %s, wrong type: %s", path, path.Type())
		}

		op := subscription.PatchOperation(apiP.Op)
		if err := op.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate operation: %w", err)
		}

		return patch.PatchRemoveItem{
			PhaseKey: path.PhaseKey(),
			ItemKey:  path.ItemKey(),
		}, nil
	default:
		return nil, fmt.Errorf("unknown patch operation: %s", typ)
	}
}

func MapAPISubscriptionPatchToPatch(apiPatch api.SubscriptionPatch) (subscription.Patch, error) {
	// As there isn't a discriminator between SubscriptionItemPatch and SubscriptionPhasePatch, calling `As` with either will succeed. Because of this, we hace to manually decide based on the path content.
	bytes, err := apiPatch.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SubscriptionPatch: %w", err)
	}

	hasPath := struct {
		Path string `json:"path"`
	}{}

	if err := json.Unmarshal(bytes, &hasPath); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SubscriptionPatch: %w", err)
	}

	path := subscription.PatchPath(hasPath.Path)
	if err := path.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate path: %w", err)
	}

	if path.Type() == subscription.PatchPathTypeItem {
		// FIXME: for some reason it gets a different name after generating form TypeSpec, let's just re-serialize it for now, but fix this discrepancy later
		itemP, err := apiPatch.AsSubscriptionItemPatchUpdateItem()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to SubscriptionItemPatchUpdateItem: %w", err)
		}
		bytes, err := itemP.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal SubscriptionItemPatchUpdateItem: %w", err)
		}

		p := api.SubscriptionItemPatch{}
		err = p.UnmarshalJSON(bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal SubscriptionItemPatchUpdateItem: %w", err)
		}

		return MapAPISubscriptionItemPatchToPatch(p)
	} else if path.Type() == subscription.PatchPathTypePhase {
		phaseP, err := apiPatch.AsSubscriptionPhasePatch()
		if err != nil {
			return nil, fmt.Errorf("failed to cast to SubscriptionPhasePatch: %w", err)
		}

		typ, err := phaseP.Discriminator()
		if err != nil {
			return nil, fmt.Errorf("failed to get discriminator: %w", err)
		}

		switch typ {
		case string(subscription.PatchOperationAdd):
			apiP, err := phaseP.AsSubscriptionEditAddSubscriptionPhasePatchKey()
			if err != nil {
				return nil, fmt.Errorf("failed to cast to SubscriptionEditAddSubscriptionPhasePatchKey: %w", err)
			}

			// Let's parse and validate path and op

			path := subscription.PatchPath(apiP.Path)
			if err := path.Validate(); err != nil {
				return nil, fmt.Errorf("failed to validate path: %w", err)
			}

			if path.Type() != subscription.PatchPathTypePhase {
				return nil, fmt.Errorf("invalid path %s, wrong type: %s", path, path.Type())
			}

			op := subscription.PatchOperation(apiP.Op)
			if err := op.Validate(); err != nil {
				return nil, fmt.Errorf("failed to validate operation: %w", err)
			}

			// Let's parse and validate value
			durStr := datex.ISOString(apiP.Value.Duration)
			dur, err := durStr.Parse()
			if err != nil {
				return nil, fmt.Errorf("failed to parse duration: %w", err)
			}

			saStr := datex.ISOString("P0M")

			if apiP.Value.StartAfter != nil {
				saStr = datex.ISOString(*apiP.Value.StartAfter)
			}

			sa, err := saStr.Parse()
			if err != nil {
				return nil, fmt.Errorf("failed to parse start after: %w", err)
			}

			return patch.PatchAddPhase{
				PhaseKey: path.PhaseKey(),
				CreateInput: subscription.CreateSubscriptionPhaseInput{
					Duration: &dur,
					CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
						PhaseKey:    path.PhaseKey(),
						StartAfter:  sa,
						Name:        apiP.Value.Name,
						Description: apiP.Value.Description,
					},
					CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{},
				},
			}, nil
		case string(subscription.PatchOperationRemove):
			apiP, err := phaseP.AsSubscriptionEditRemoveWithValueSubscriptionPhasePatchKey()
			if err != nil {
				return nil, fmt.Errorf("failed to cast to SubscriptionEditRemoveWithValueSubscriptionPhasePatchKey: %w", err)
			}

			// Let's parse and validate path and op
			path := subscription.PatchPath(apiP.Path)
			if err := path.Validate(); err != nil {
				return nil, fmt.Errorf("failed to validate path: %w", err)
			}

			if path.Type() != subscription.PatchPathTypePhase {
				return nil, fmt.Errorf("invalid path %s, wrong type: %s", path, path.Type())
			}

			op := subscription.PatchOperation(apiP.Op)
			if err := op.Validate(); err != nil {
				return nil, fmt.Errorf("failed to validate operation: %w", err)
			}

			// Let's parse and validate value

			var shift subscription.RemoveSubscriptionPhaseShifting

			if apiP.Value.Shift == api.RemovePhaseShiftingNext {
				shift = subscription.RemoveSubscriptionPhaseShiftNext
			} else if apiP.Value.Shift == api.RemovePhaseShiftingPrev {
				shift = subscription.RemoveSubscriptionPhaseShiftPrev
			} else {
				return nil, fmt.Errorf("unknown shift value: %s", apiP.Value.Shift)
			}

			// Let's parse and validate value
			return patch.PatchRemovePhase{
				PhaseKey: path.PhaseKey(),
				RemoveInput: subscription.RemoveSubscriptionPhaseInput{
					Shift: shift,
				},
			}, nil

		case string(subscription.PatchOperationStretch):
			apiP, err := phaseP.AsSubscriptionEditStretchSubscriptionPhasePatchKey()
			if err != nil {
				return nil, fmt.Errorf("failed to cast to SubscriptionEditStretchSubscriptionPhasePatchKey: %w", err)
			}

			// Let's parse and validate path and op
			path := subscription.PatchPath(apiP.Path)
			if err := path.Validate(); err != nil {
				return nil, fmt.Errorf("failed to validate path: %w", err)
			}

			if path.Type() != subscription.PatchPathTypePhase {
				return nil, fmt.Errorf("invalid path %s, wrong type: %s", path, path.Type())
			}

			op := subscription.PatchOperation(apiP.Op)
			if err := op.Validate(); err != nil {
				return nil, fmt.Errorf("failed to validate operation: %w", err)
			}

			// Let's parse and validate value
			durStr := datex.ISOString(apiP.Value.ExtendBy)
			dur, err := durStr.Parse()
			if err != nil {
				return nil, fmt.Errorf("failed to parse duration: %w", err)
			}

			return patch.PatchStretchPhase{
				PhaseKey: path.PhaseKey(),
				Duration: dur,
			}, nil
		default:
			return nil, fmt.Errorf("unknown patch operation: %s", typ)
		}
	} else {
		return nil, fmt.Errorf("invalid path %s, wrong type: %s", path, path.Type())
	}
}

func MapSubscriptionToAPI(sub subscription.Subscription) api.Subscription {
	return api.Subscription{
		Id:          sub.ID,
		ActiveFrom:  sub.ActiveFrom,
		ActiveTo:    sub.ActiveTo,
		CustomerId:  sub.CustomerId,
		Currency:    string(sub.Currency),
		Description: sub.Description,
		Name:        sub.Name,
		Plan: api.PlanReference{
			Key:     sub.Plan.Key,
			Version: sub.Plan.Version,
		},
		Metadata:  &sub.Metadata,
		CreatedAt: sub.CreatedAt,
		UpdatedAt: sub.UpdatedAt,
		DeletedAt: sub.DeletedAt,
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
		ActiveFrom:      item.SubscriptionItem.ActiveFrom,
		ActiveTo:        item.SubscriptionItem.ActiveTo,
		BillingCandence: (*string)(item.SubscriptionItem.RateCard.BillingCadence.ISOStringPtrOrNil()),
		CreatedAt:       item.SubscriptionItem.CreatedAt,
		DeletedAt:       item.SubscriptionItem.DeletedAt,
		Description:     item.SubscriptionItem.Description,
		Id:              item.SubscriptionItem.ID,
		Included:        included,
		Key:             item.SubscriptionItem.Key,
		Metadata:        &item.SubscriptionItem.Metadata,
		Name:            item.SubscriptionItem.Name,
		Price:           pr,
		TaxConfig:       tx,
		UpdatedAt:       item.SubscriptionItem.UpdatedAt,
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
				var upToAmount *float64
				if t.UpToAmount != nil {
					a, _ := t.UpToAmount.Float64()
					upToAmount = lo.ToPtr(a)
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
