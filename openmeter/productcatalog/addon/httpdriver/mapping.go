package httpdriver

import (
	"fmt"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromAddon(a addon.Addon) (api.Addon, error) {
	validationIssues, _ := a.AsProductCatalogAddon().ValidationErrors()

	resp := api.Addon{
		CreatedAt:        a.CreatedAt,
		Currency:         a.Currency.String(),
		DeletedAt:        a.DeletedAt,
		Description:      a.Description,
		InstanceType:     api.AddonInstanceType(a.InstanceType),
		EffectiveFrom:    a.EffectiveFrom,
		EffectiveTo:      a.EffectiveTo,
		Id:               a.ID,
		Key:              a.Key,
		Metadata:         http.FromMetadata(a.Metadata),
		Annotations:      http.FromAnnotations(a.Annotations),
		Name:             a.Name,
		UpdatedAt:        a.UpdatedAt,
		Version:          a.Version,
		ValidationErrors: http.FromValidationErrors(validationIssues),
	}

	resp.RateCards = make([]api.RateCard, 0, len(a.RateCards))
	for _, rateCard := range a.RateCards.AsProductCatalogRateCards() {
		rc, err := http.FromRateCard(rateCard)
		if err != nil {
			return resp, fmt.Errorf("failed to cast ratecard: %w", err)
		}

		resp.RateCards = append(resp.RateCards, rc)
	}

	switch a.Status() {
	case productcatalog.AddonStatusDraft:
		resp.Status = api.AddonStatusDraft
	case productcatalog.AddonStatusActive:
		resp.Status = api.AddonStatusActive
	case productcatalog.AddonStatusArchived:
		resp.Status = api.AddonStatusArchived
	default:
		return resp, fmt.Errorf("invalid add-on status: %s", a.Status())
	}

	return resp, nil
}

func AsCreateAddonRequest(a api.AddonCreate, namespace string) (CreateAddonRequest, error) {
	var err error

	req := CreateAddonRequest{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Addon: productcatalog.Addon{
			AddonMeta: productcatalog.AddonMeta{
				Key:          a.Key,
				Name:         a.Name,
				Description:  a.Description,
				InstanceType: productcatalog.AddonInstanceType(a.InstanceType),
				Metadata:     lo.FromPtrOr(a.Metadata, nil),
			},
			RateCards: nil,
		},
	}

	req.Currency = currency.Code(a.Currency)
	if err = req.Currency.Validate(); err != nil {
		return req, fmt.Errorf("invalid CurrencyCode: %w", err)
	}

	req.RateCards, err = http.AsRateCards(a.RateCards)
	if err != nil {
		return req, err
	}

	return req, nil
}

func AsUpdateAddonRequest(a api.AddonReplaceUpdate, namespace string, addonID string) (UpdateAddonRequest, error) {
	req := UpdateAddonRequest{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
			ID:        addonID,
		},
		Name:         lo.ToPtr(a.Name),
		Description:  a.Description,
		InstanceType: lo.ToPtr(productcatalog.AddonInstanceType(a.InstanceType)),
		Metadata:     (*models.Metadata)(a.Metadata),
	}

	rateCards, err := http.AsRateCards(a.RateCards)
	if err != nil {
		return req, err
	}

	req.RateCards = &rateCards

	return req, nil
}
