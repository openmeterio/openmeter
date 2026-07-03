package hooks

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	AddonHook     = models.ServiceHook[taxcode.TaxCode]
	NoopAddonHook = models.NoopServiceHook[taxcode.TaxCode]
)

type AddonHookConfig struct {
	AddonService addon.Service
}

func (e AddonHookConfig) Validate() error {
	if e.AddonService == nil {
		return fmt.Errorf("addon service is required")
	}

	return nil
}

var _ models.ServiceHook[taxcode.TaxCode] = (*addonHook)(nil)

type addonHook struct {
	NoopAddonHook

	addonService addon.Service
}

func NewAddonHook(config AddonHookConfig) (AddonHook, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid addon hook config: %w", err)
	}

	return &addonHook{
		addonService: config.AddonService,
	}, nil
}

func (e *addonHook) PreDelete(ctx context.Context, tc *taxcode.TaxCode) error {
	affectedAddons, err := e.addonService.ListAddons(ctx, addon.ListAddonsInput{
		Namespaces: []string{tc.Namespace},
		Status: []productcatalog.AddonStatus{
			productcatalog.AddonStatusActive,
			productcatalog.AddonStatusDraft,
			productcatalog.AddonStatusArchived,
			productcatalog.AddonStatusInvalid,
		},
		TaxCodes: &filter.FilterString{
			In: &[]string{
				tc.ID,
			},
		},
		Page: pagination.Page{
			PageSize:   5,
			PageNumber: 1,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list add-ons: %w", err)
	}

	if len(affectedAddons.Items) > 0 {
		addonIDs := lo.Map(affectedAddons.Items, func(item addon.Addon, _ int) string {
			return item.ID
		})
		return taxcode.NewTaxCodeReferencedByAddonError(tc.ID, addonIDs)
	}

	return nil
}
