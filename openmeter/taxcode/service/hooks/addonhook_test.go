package hooks_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/openmeter/taxcode/service/hooks"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// stubAddonService implements addon.Service for testing the addon hook.
// Only ListAddons has real behavior; all other methods panic.
type stubAddonService struct {
	listResult pagination.Result[addon.Addon]
	listErr    error
	lastInput  addon.ListAddonsInput
}

func (s *stubAddonService) ListAddons(ctx context.Context, params addon.ListAddonsInput) (pagination.Result[addon.Addon], error) {
	s.lastInput = params
	return s.listResult, s.listErr
}

func (s *stubAddonService) CreateAddon(_ context.Context, _ addon.CreateAddonInput) (*addon.Addon, error) {
	panic("not implemented")
}

func (s *stubAddonService) DeleteAddon(_ context.Context, _ addon.DeleteAddonInput) error {
	panic("not implemented")
}

func (s *stubAddonService) GetAddon(_ context.Context, _ addon.GetAddonInput) (*addon.Addon, error) {
	panic("not implemented")
}

func (s *stubAddonService) UpdateAddon(_ context.Context, _ addon.UpdateAddonInput) (*addon.Addon, error) {
	panic("not implemented")
}

func (s *stubAddonService) PublishAddon(_ context.Context, _ addon.PublishAddonInput) (*addon.Addon, error) {
	panic("not implemented")
}

func (s *stubAddonService) ArchiveAddon(_ context.Context, _ addon.ArchiveAddonInput) (*addon.Addon, error) {
	panic("not implemented")
}

func (s *stubAddonService) NextAddon(_ context.Context, _ addon.NextAddonInput) (*addon.Addon, error) {
	panic("not implemented")
}

var _ addon.Service = (*stubAddonService)(nil)

const addonTestTaxCodeID = "01234567890123456789012346"

func TestAddonHook_PreDelete(t *testing.T) {
	tc := &taxcode.TaxCode{
		NamespacedID: models.NamespacedID{
			Namespace: "test-ns",
			ID:        addonTestTaxCodeID,
		},
	}

	t.Run("blocks deletion when an add-on references the tax code", func(t *testing.T) {
		// given: an addon service that returns one matching add-on
		stub := &stubAddonService{
			listResult: pagination.Result[addon.Addon]{
				Items: []addon.Addon{
					{ManagedModel: models.ManagedModel{}, NamespacedID: models.NamespacedID{ID: "addon-abc"}},
				},
				TotalCount: 1,
			},
		}

		hook, err := hooks.NewAddonHook(hooks.AddonHookConfig{AddonService: stub})
		require.NoError(t, err)

		// when: PreDelete is called
		err = hook.PreDelete(t.Context(), tc)

		// then: an error is returned and it is a TaxCodeReferencedByAddon error
		require.Error(t, err)
		require.True(t, taxcode.IsTaxCodeReferencedByAddonError(err),
			"expected TaxCodeReferencedByAddon error, got: %v", err)

		// and: the stub received a ListAddonsInput whose TaxCodes.In contains the tax code id
		require.NotNil(t, stub.lastInput.TaxCodes)
		require.NotNil(t, stub.lastInput.TaxCodes.In)
		require.Contains(t, *stub.lastInput.TaxCodes.In, addonTestTaxCodeID)
	})

	t.Run("allows deletion when no add-on references the tax code", func(t *testing.T) {
		// given: an addon service that returns no matching add-ons
		stub := &stubAddonService{
			listResult: pagination.Result[addon.Addon]{
				Items:      []addon.Addon{},
				TotalCount: 0,
			},
		}

		hook, err := hooks.NewAddonHook(hooks.AddonHookConfig{AddonService: stub})
		require.NoError(t, err)

		// when: PreDelete is called
		err = hook.PreDelete(t.Context(), tc)

		// then: no error is returned
		require.NoError(t, err)
	})
}
