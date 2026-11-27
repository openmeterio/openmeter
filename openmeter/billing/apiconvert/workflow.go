package apiconvert

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	appconvert "github.com/openmeterio/openmeter/openmeter/app/apiconvert"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

func ToAPIBillingProfileApps(a *billing.ProfileApps) (*apiv1models.BillingProfileAppsOrReference, error) {
	if a == nil {
		return nil, nil
	}

	tax, err := appconvert.ToAPIApp(a.Tax)
	if err != nil {
		return nil, fmt.Errorf("cannot map tax app: %w", err)
	}

	invoicing, err := appconvert.ToAPIApp(a.Invoicing)
	if err != nil {
		return nil, fmt.Errorf("cannot map invoicing app: %w", err)
	}

	payment, err := appconvert.ToAPIApp(a.Payment)
	if err != nil {
		return nil, fmt.Errorf("cannot map payment app: %w", err)
	}

	apps := apiv1models.BillingProfileApps{
		Tax:       tax,
		Invoicing: invoicing,
		Payment:   payment,
	}

	out := apiv1models.BillingProfileAppsOrReference{}

	if err := out.FromBillingProfileApps(apps); err != nil {
		return nil, fmt.Errorf("failed to convert apps to API: %w", err)
	}

	return &out, nil
}

func ToAPIBillingProfileAppReferences(a *billing.ProfileAppReferences) (*apiv1models.BillingProfileAppsOrReference, error) {
	if a == nil {
		return nil, nil
	}

	apps := apiv1models.BillingProfileAppReferences{
		Tax: apiv1models.AppReference{
			Id: a.Tax.ID,
		},
		Invoicing: apiv1models.AppReference{
			Id: a.Invoicing.ID,
		},
		Payment: apiv1models.AppReference{
			Id: a.Payment.ID,
		},
	}

	out := apiv1models.BillingProfileAppsOrReference{}

	if err := out.FromBillingProfileAppReferences(apps); err != nil {
		return nil, fmt.Errorf("failed to convert apps to API: %w", err)
	}

	return &out, nil
}

func ToAPIBillingParty(c billing.SupplierContact) apiv1models.BillingParty {
	a := c.Address

	out := apiv1models.BillingParty{
		Id:   lo.EmptyableToPtr(c.ID),
		Name: lo.EmptyableToPtr(c.Name),
		Addresses: lo.ToPtr([]apiv1models.Address{
			{
				Country:     (*string)(a.Country),
				PostalCode:  a.PostalCode,
				State:       a.State,
				City:        a.City,
				Line1:       a.Line1,
				Line2:       a.Line2,
				PhoneNumber: a.PhoneNumber,
			},
		}),
	}

	if c.TaxCode != nil {
		out.TaxId = &apiv1models.BillingPartyTaxIdentity{
			Code: c.TaxCode,
		}
	}

	return out
}

func mapAlignmentToAPI(c billing.CollectionConfig) (*apiv1models.BillingWorkflowCollectionAlignment, error) {
	apiAlignment := &apiv1models.BillingWorkflowCollectionAlignment{}

	switch c.Alignment {
	case billing.AlignmentKindSubscription:
		if err := apiAlignment.FromBillingWorkflowCollectionAlignmentSubscription(
			apiv1models.BillingWorkflowCollectionAlignmentSubscription{
				Type: apiv1models.BillingWorkflowCollectionAlignmentSubscriptionType(c.Alignment),
			},
		); err != nil {
			return &apiv1models.BillingWorkflowCollectionAlignment{}, fmt.Errorf("failed to map alignment to API: %w", err)
		}
	case billing.AlignmentKindAnchored:
		if c.AnchoredAlignmentDetail == nil {
			return &apiv1models.BillingWorkflowCollectionAlignment{}, fmt.Errorf("anchored alignment detail is not set")
		}

		interval := apiv1models.RecurringPeriodInterval{}
		if err := interval.FromRecurringPeriodInterval0(c.AnchoredAlignmentDetail.Interval.ISOString().String()); err != nil {
			return &apiv1models.BillingWorkflowCollectionAlignment{}, fmt.Errorf("failed to map interval to API: %w", err)
		}

		if err := apiAlignment.FromBillingWorkflowCollectionAlignmentAnchored(
			apiv1models.BillingWorkflowCollectionAlignmentAnchored{
				Type: apiv1models.BillingWorkflowCollectionAlignmentAnchoredType(c.Alignment),
				RecurringPeriod: apiv1models.RecurringPeriodV2{
					Anchor:   c.AnchoredAlignmentDetail.Anchor,
					Interval: interval,
				},
			},
		); err != nil {
			return &apiv1models.BillingWorkflowCollectionAlignment{}, fmt.Errorf("failed to map alignment to API: %w", err)
		}
	default:
		return &apiv1models.BillingWorkflowCollectionAlignment{}, fmt.Errorf("invalid alignment: %s", c.Alignment)
	}

	return apiAlignment, nil
}

// ToAPIBillingWorkflow maps a workflow config. taxCodes must hold its default Tax Code (see
// ResolveWorkflowTaxCodes) for the default's stripe.code to show.
func ToAPIBillingWorkflow(c billing.WorkflowConfig, taxCodes map[string]*taxcode.TaxCode) (apiv1models.BillingWorkflow, error) {
	apiAlignment, err := mapAlignmentToAPI(c.Collection)
	if err != nil {
		return apiv1models.BillingWorkflow{}, fmt.Errorf("failed to map alignment to API: %w", err)
	}

	defaultTaxConfig, err := toAPIDefaultTaxConfig(c.Invoicing.DefaultTaxConfig, taxCodes)
	if err != nil {
		return apiv1models.BillingWorkflow{}, fmt.Errorf("failed to map default tax config to API: %w", err)
	}

	return apiv1models.BillingWorkflow{
		Collection: &apiv1models.BillingWorkflowCollectionSettings{
			Alignment: apiAlignment,
			Interval:  lo.EmptyableToPtr(c.Collection.Interval.String()),
		},

		Invoicing: &apiv1models.BillingWorkflowInvoicingSettings{
			AutoAdvance:                  lo.ToPtr(c.Invoicing.AutoAdvance),
			DraftPeriod:                  lo.EmptyableToPtr(c.Invoicing.DraftPeriod.String()),
			DueAfter:                     lo.EmptyableToPtr(c.Invoicing.DueAfter.String()),
			ProgressiveBilling:           lo.ToPtr(c.Invoicing.ProgressiveBilling),
			SubscriptionEndProrationMode: lo.ToPtr(apiv1models.BillingWorkflowInvoicingSubscriptionEndProrationMode(c.Invoicing.SubscriptionEndProrationMode)),
			DefaultTaxConfig:             defaultTaxConfig,
		},

		Payment: &apiv1models.BillingWorkflowPaymentSettings{
			CollectionMethod: (*apiv1models.CollectionMethod)(lo.EmptyableToPtr(string(c.Payment.CollectionMethod))),
		},

		Tax: &apiv1models.BillingWorkflowTaxSettings{
			Enabled:  lo.ToPtr(c.Tax.Enabled),
			Enforced: lo.ToPtr(c.Tax.Enforced),
		},
	}, nil
}

// ResolveWorkflowTaxCodes looks up, in one batch, the Tax Codes referenced by the workflows'
// default tax configs, so v1 responses can show their stripe.code.
func ResolveWorkflowTaxCodes(ctx context.Context, resolver productcatalog.TaxCodeResolver, namespace string, workflows ...billing.WorkflowConfig) (map[string]*taxcode.TaxCode, error) {
	ids := lo.Uniq(lo.FilterMap(workflows, func(wf billing.WorkflowConfig, _ int) (string, bool) {
		id := wf.Invoicing.DefaultTaxCodeID()
		return id, id != ""
	}))

	return resolveTaxCodesByIDs(ctx, resolver, namespace, ids)
}

// ResolveInvoiceTaxCodesInput carries what one v1 invoice response renders: the workflows'
// default tax configs and the lines' own tax configs.
type ResolveInvoiceTaxCodesInput struct {
	Namespace string
	// Workflows are the workflows whose default tax config references render on the response.
	Workflows []billing.WorkflowConfig
	// StandardLines and GatheringLines are the lines rendered on the response. Child
	// (detailed) lines inherit their parent's config and need no entry of their own.
	StandardLines  []*billing.StandardLine
	GatheringLines []billing.GatheringLine
}

func (i ResolveInvoiceTaxCodesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ResolveInvoiceTaxCodes looks up, in one batch, every Tax Code a v1 invoice response renders:
// the workflows' default tax configs' references plus the lines' own tax config references.
// stripe.code comes from the referenced Tax Code's current Stripe mapping, so a line whose
// stored stripe.code no longer matches renders the current one, and the whole page shares one
// lookup. A referenced ID the namespace can't resolve fails rendering (a 500), consistent with
// the default.
func ResolveInvoiceTaxCodes(ctx context.Context, resolver productcatalog.TaxCodeResolver, input ResolveInvoiceTaxCodesInput) (productcatalog.TaxCodesByID, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	ids := lo.Uniq(slices.Concat(
		lo.FilterMap(input.Workflows, func(wf billing.WorkflowConfig, _ int) (string, bool) {
			id := wf.Invoicing.DefaultTaxCodeID()
			return id, id != ""
		}),
		lo.FilterMap(input.StandardLines, func(line *billing.StandardLine, _ int) (string, bool) {
			id := lo.FromPtr(line.TaxConfig).TaxCodeID
			return id, id != ""
		}),
		lo.FilterMap(input.GatheringLines, func(line billing.GatheringLine, _ int) (string, bool) {
			id := lo.FromPtr(line.TaxConfig).TaxCodeID
			return id, id != ""
		}),
	))

	return resolveTaxCodesByIDs(ctx, resolver, input.Namespace, ids)
}

func resolveTaxCodesByIDs(ctx context.Context, resolver productcatalog.TaxCodeResolver, namespace string, ids []string) (productcatalog.TaxCodesByID, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	if resolver == nil {
		return nil, errors.New("tax code resolver is required to render tax configs")
	}

	taxCodes, err := resolver.BatchResolve(ctx, namespace, ids...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tax codes: %w", err)
	}

	return taxCodes, nil
}

// toAPIDefaultTaxConfig renders a workflow's default tax config in the v1 shape; stripe.code
// comes from the referenced Tax Code's current mapping.
func toAPIDefaultTaxConfig(cfg *productcatalog.TaxCodeConfig, taxCodes map[string]*taxcode.TaxCode) (*apiv1models.TaxConfig, error) {
	if cfg == nil {
		return nil, nil
	}

	taxConfig, err := productcataloghttp.ToAPITaxConfig(*cfg, taxCodes)
	if err != nil {
		return nil, err
	}

	return &taxConfig, nil
}
