package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

type (
	CreateProfileRequest  = billing.CreateProfileInput
	CreateProfileResponse = api.BillingProfile
	CreateProfileHandler  httptransport.Handler[CreateProfileRequest, CreateProfileResponse]
)

func (h *handler) CreateProfile() CreateProfileHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateProfileRequest, error) {
			body := api.BillingProfileCreateInput{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateProfileRequest{}, fmt.Errorf("failed to decode request body: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateProfileRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			workflow, err := fromAPIBillingWorkflow(body.Workflow)
			if err != nil {
				return CreateProfileRequest{}, fmt.Errorf("failed to parse workflow: %w", err)
			}

			return CreateProfileRequest{
				Namespace: ns,

				Name:        body.Name,
				Description: body.Description,

				Metadata: lo.FromPtrOr(body.Metadata, map[string]string{}),

				Supplier: apiBillingPartyCreateToSupplierContact(body.Supplier),

				Default: body.Default,

				WorkflowConfig: workflow,
				Apps:           fromAPIBillingProfileCreateAppsInput(body.Apps),
			}, nil
		},
		func(ctx context.Context, request CreateProfileRequest) (CreateProfileResponse, error) {
			profile, err := h.service.CreateProfile(ctx, request)
			if err != nil {
				return CreateProfileResponse{}, fmt.Errorf("failed to create profile: %w", err)
			}

			return profile.ToAPI(), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateProfileResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createProfile"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func apiBillingPartyCreateToSupplierContact(c api.BillingPartyCreate) billing.SupplierContact {
	out := billing.SupplierContact{
		Name: lo.FromPtrOr(c.Name, ""),
	}

	if c.Addresses == nil || len(*c.Addresses) == 0 {
		return out
	}

	// TODO: represent TAX ID

	addr := (*c.Addresses)[0]

	out.Address = models.Address{
		City:        addr.City,
		State:       addr.State,
		PostalCode:  addr.PostalCode,
		Line1:       addr.Line1,
		Line2:       addr.Line2,
		PhoneNumber: addr.PhoneNumber,
	}

	return out
}

func fromAPIBillingProfileCreateAppsInput(i api.BillingProfileCreateAppsInput) billing.CreateProfileAppsInput {
	return billing.CreateProfileAppsInput{
		Tax:       fromAPIBillingAppReference(i.Tax),
		Invoicing: fromAPIBillingAppReference(i.Invoicing),
		Payment:   fromAPIBillingAppReference(i.Payment),
	}
}

func fromAPIBillingAppReference(i api.BillingWorkflowAppReference) billing.AppReference {
	if id, err := i.AsBillingWorkflowAppIDReference(); err == nil {
		return billing.AppReference{
			ID: id.Id,
		}
	}

	typeRef, err := i.AsBillingWorkflowAppTypeReference()
	if err == nil {
		return billing.AppReference{
			Type: appentitybase.AppType(typeRef.AppType),
		}
	}

	return billing.AppReference{}
}

func fromAPIBillingWorkflow(i api.BillingWorkflow) (billing.WorkflowConfig, error) {
	collInterval, err := parseDurationPtr(i.Collection.Interval, billing.DefaultCollectionInterval)
	if err != nil {
		return billing.WorkflowConfig{}, fmt.Errorf("failed to parse collection interval: %w", err)
	}

	draftPeriod, err := parseDurationPtr(i.Invoicing.DraftPeriod, billing.DefaultInvoicingDraftPeriod)
	if err != nil {
		return billing.WorkflowConfig{}, fmt.Errorf("failed to parse draft period: %w", err)
	}

	dueAfter, err := parseDurationPtr(i.Invoicing.DueAfter, billing.DefaultInvoicingDueAfter)
	if err != nil {
		return billing.WorkflowConfig{}, fmt.Errorf("failed to parse due after: %w", err)
	}

	return billing.WorkflowConfig{
		ID:        i.Id,
		CreatedAt: i.CreatedAt,
		UpdatedAt: i.UpdatedAt,
		DeletedAt: i.DeletedAt,

		Collection: billing.CollectionConfig{
			Alignment: lo.FromPtrOr((*billing.AlignmentKind)(i.Collection.Alignment), billing.DefaultCollectionAlignment),
			Interval:  collInterval,
		},

		Invoicing: billing.InvoicingConfig{
			AutoAdvance:    lo.FromPtrOr(i.Invoicing.AutoAdvance, billing.DefaultInvoicingAutoAdvance),
			DraftPeriod:    draftPeriod,
			DueAfter:       dueAfter,
			ItemPerSubject: lo.FromPtrOr(i.Invoicing.ItemPerSubject, billing.DefaultInvoicingItemPerSubject),
			ItemResolution: lo.FromPtrOr((*billing.GranularityResolution)(i.Invoicing.ItemResolution), billing.DefaultInvoicingItemResolution),
		},

		Payment: billing.PaymentConfig{
			CollectionMethod: lo.FromPtrOr((*billing.CollectionMethod)(i.Payment.CollectionMethod), billing.DefaultPaymentCollectionMethod),
		},
	}, nil
}

func parseDurationPtr(d *string, defaultDuration datex.ISOString) (datex.Period, error) {
	if d == nil {
		return defaultDuration.Parse()
	}

	return datex.ISOString(*d).Parse()
}
