package apiconvert

import (
	"fmt"

	"github.com/samber/lo"

	apimodels "github.com/openmeterio/openmeter/api/models"
	appconvert "github.com/openmeterio/openmeter/openmeter/app/apiconvert"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

func ToAPIBillingProfileApps(a *billing.ProfileApps) (*apimodels.BillingProfileAppsOrReference, error) {
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

	apps := apimodels.BillingProfileApps{
		Tax:       tax,
		Invoicing: invoicing,
		Payment:   payment,
	}

	out := apimodels.BillingProfileAppsOrReference{}

	if err := out.FromBillingProfileApps(apps); err != nil {
		return nil, fmt.Errorf("failed to convert apps to API: %w", err)
	}

	return &out, nil
}

func ToAPIBillingProfileAppReferences(a *billing.ProfileAppReferences) (*apimodels.BillingProfileAppsOrReference, error) {
	if a == nil {
		return nil, nil
	}

	apps := apimodels.BillingProfileAppReferences{
		Tax: apimodels.AppReference{
			Id: a.Tax.ID,
		},
		Invoicing: apimodels.AppReference{
			Id: a.Invoicing.ID,
		},
		Payment: apimodels.AppReference{
			Id: a.Payment.ID,
		},
	}

	out := apimodels.BillingProfileAppsOrReference{}

	if err := out.FromBillingProfileAppReferences(apps); err != nil {
		return nil, fmt.Errorf("failed to convert apps to API: %w", err)
	}

	return &out, nil
}

func ToAPIBillingParty(c billing.SupplierContact) apimodels.BillingParty {
	a := c.Address

	out := apimodels.BillingParty{
		Id:   lo.EmptyableToPtr(c.ID),
		Name: lo.EmptyableToPtr(c.Name),
		Addresses: lo.ToPtr([]apimodels.Address{
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
		out.TaxId = &apimodels.BillingPartyTaxIdentity{
			Code: c.TaxCode,
		}
	}

	return out
}

func mapAlignmentToAPI(c billing.CollectionConfig) (*apimodels.BillingWorkflowCollectionAlignment, error) {
	apiAlignment := &apimodels.BillingWorkflowCollectionAlignment{}

	switch c.Alignment {
	case billing.AlignmentKindSubscription:
		if err := apiAlignment.FromBillingWorkflowCollectionAlignmentSubscription(
			apimodels.BillingWorkflowCollectionAlignmentSubscription{
				Type: apimodels.BillingWorkflowCollectionAlignmentSubscriptionType(c.Alignment),
			},
		); err != nil {
			return &apimodels.BillingWorkflowCollectionAlignment{}, fmt.Errorf("failed to map alignment to API: %w", err)
		}
	case billing.AlignmentKindAnchored:
		if c.AnchoredAlignmentDetail == nil {
			return &apimodels.BillingWorkflowCollectionAlignment{}, fmt.Errorf("anchored alignment detail is not set")
		}

		interval := apimodels.RecurringPeriodInterval{}
		if err := interval.FromRecurringPeriodInterval0(c.AnchoredAlignmentDetail.Interval.ISOString().String()); err != nil {
			return &apimodels.BillingWorkflowCollectionAlignment{}, fmt.Errorf("failed to map interval to API: %w", err)
		}

		if err := apiAlignment.FromBillingWorkflowCollectionAlignmentAnchored(
			apimodels.BillingWorkflowCollectionAlignmentAnchored{
				Type: apimodels.BillingWorkflowCollectionAlignmentAnchoredType(c.Alignment),
				RecurringPeriod: apimodels.RecurringPeriodV2{
					Anchor:   c.AnchoredAlignmentDetail.Anchor,
					Interval: interval,
				},
			},
		); err != nil {
			return &apimodels.BillingWorkflowCollectionAlignment{}, fmt.Errorf("failed to map alignment to API: %w", err)
		}
	default:
		return &apimodels.BillingWorkflowCollectionAlignment{}, fmt.Errorf("invalid alignment: %s", c.Alignment)
	}

	return apiAlignment, nil
}

func ToAPIBillingWorkflow(c billing.WorkflowConfig) (apimodels.BillingWorkflow, error) {
	apiAlignment, err := mapAlignmentToAPI(c.Collection)
	if err != nil {
		return apimodels.BillingWorkflow{}, fmt.Errorf("failed to map alignment to API: %w", err)
	}

	return apimodels.BillingWorkflow{
		Collection: &apimodels.BillingWorkflowCollectionSettings{
			Alignment: apiAlignment,
			Interval:  lo.EmptyableToPtr(c.Collection.Interval.String()),
		},

		Invoicing: &apimodels.BillingWorkflowInvoicingSettings{
			AutoAdvance:                  lo.ToPtr(c.Invoicing.AutoAdvance),
			DraftPeriod:                  lo.EmptyableToPtr(c.Invoicing.DraftPeriod.String()),
			DueAfter:                     lo.EmptyableToPtr(c.Invoicing.DueAfter.String()),
			ProgressiveBilling:           lo.ToPtr(c.Invoicing.ProgressiveBilling),
			SubscriptionEndProrationMode: lo.ToPtr(apimodels.BillingWorkflowInvoicingSubscriptionEndProrationMode(c.Invoicing.SubscriptionEndProrationMode)),
			DefaultTaxConfig:             ToAPITaxConfig(c.Invoicing.DefaultTaxConfig),
		},

		Payment: &apimodels.BillingWorkflowPaymentSettings{
			CollectionMethod: (*apimodels.CollectionMethod)(lo.EmptyableToPtr(string(c.Payment.CollectionMethod))),
		},

		Tax: &apimodels.BillingWorkflowTaxSettings{
			Enabled:  lo.ToPtr(c.Tax.Enabled),
			Enforced: lo.ToPtr(c.Tax.Enforced),
		},
	}, nil
}
