package billingprofiles

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/handlers/billingerrors"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListBillingProfiles() ListBillingProfilesHandler
	CreateBillingProfile() CreateBillingProfileHandler
	GetBillingProfile() GetBillingProfileHandler
	UpdateBillingProfile() UpdateBillingProfileHandler
	DeleteBillingProfile() DeleteBillingProfileHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          billing.Service
	taxCodeResolver  productcatalog.TransactionalTaxCodeResolver
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service billing.Service,
	taxCodeResolver productcatalog.TransactionalTaxCodeResolver,
	options ...httptransport.HandlerOption,
) Handler {
	sharedOptions := make([]httptransport.HandlerOption, 0, len(options)+1)
	sharedOptions = append(sharedOptions, httptransport.WithErrorEncoder(billingerrors.ErrorEncoder()))
	sharedOptions = append(sharedOptions, options...)

	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		taxCodeResolver:  taxCodeResolver,
		options:          sharedOptions,
	}
}

// taxCodesForProfiles looks up, in one batch, the Tax Codes the profiles' default tax configs
// reference, so responses can show their deprecated Stripe code.
func (h *handler) taxCodesForProfiles(ctx context.Context, namespace string, profiles ...billing.Profile) (TaxCodes, error) {
	ids := lo.Uniq(lo.FilterMap(profiles, func(p billing.Profile, _ int) (string, bool) {
		id := p.WorkflowConfig.Invoicing.DefaultTaxCodeID()
		return id, id != ""
	}))

	if len(ids) == 0 {
		return nil, nil
	}

	taxCodes, err := h.taxCodeResolver.BatchResolve(ctx, namespace, ids...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tax codes: %w", err)
	}

	return taxCodes, nil
}
