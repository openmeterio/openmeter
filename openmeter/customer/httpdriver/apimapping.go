package httpdriver

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	subscriptionhttp "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/http"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func MapCustomerCreate(body api.CustomerCreate) customer.CustomerMutate {
	var metadata *models.Metadata

	if body.Metadata != nil {
		metadata = lo.ToPtr(models.NewMetadata(*body.Metadata))
	}

	mut := customer.CustomerMutate{
		Key:            body.Key,
		Name:           body.Name,
		Description:    body.Description,
		PrimaryEmail:   body.PrimaryEmail,
		BillingAddress: MapAddress(body.BillingAddress),
		Currency:       mapCurrency(body.Currency),
		Metadata:       metadata,
	}

	if body.UsageAttribution != nil {
		mut.UsageAttribution = &customer.CustomerUsageAttribution{
			SubjectKeys: body.UsageAttribution.SubjectKeys,
		}
	}

	return mut
}

func MapCustomerReplaceUpdate(body api.CustomerReplaceUpdate) customer.CustomerMutate {
	var metadata *models.Metadata

	if body.Metadata != nil {
		metadata = lo.ToPtr(models.NewMetadata(*body.Metadata))
	}

	mut := customer.CustomerMutate{
		Key:            body.Key,
		Name:           body.Name,
		Description:    body.Description,
		PrimaryEmail:   body.PrimaryEmail,
		BillingAddress: MapAddress(body.BillingAddress),
		Currency:       mapCurrency(body.Currency),
		Metadata:       metadata,
	}

	if body.UsageAttribution != nil {
		mut.UsageAttribution = &customer.CustomerUsageAttribution{
			SubjectKeys: body.UsageAttribution.SubjectKeys,
		}
	}

	return mut
}

func mapCurrency(apiCurrency *string) *currencyx.Code {
	if apiCurrency == nil {
		return nil
	}

	return lo.ToPtr(currencyx.Code(*apiCurrency))
}

func MapAddress(apiAddress *api.Address) *models.Address {
	if apiAddress == nil {
		return nil
	}

	address := models.Address{
		City:        apiAddress.City,
		State:       apiAddress.State,
		PostalCode:  apiAddress.PostalCode,
		Line1:       apiAddress.Line1,
		Line2:       apiAddress.Line2,
		PhoneNumber: apiAddress.PhoneNumber,
	}

	if apiAddress.Country != nil {
		address.Country = lo.ToPtr(models.CountryCode(*apiAddress.Country))
	}

	return &address
}

func FromMetadata(metadata models.Metadata) *api.Metadata {
	if len(metadata) == 0 {
		return nil
	}

	result := make(api.Metadata)
	if len(metadata) > 0 {
		for k, v := range metadata {
			result[k] = v
		}
	}

	return &result
}

func FromAnnotations(annotations models.Annotations) *api.Annotations {
	if len(annotations) == 0 {
		return nil
	}

	result := make(api.Annotations)
	if len(annotations) > 0 {
		for k, v := range annotations {
			result[k] = v
		}
	}

	return &result
}

// CustomerToAPI converts a Customer to an API Customer
func CustomerToAPI(c customer.Customer, subscriptions []subscription.Subscription, expand customer.Expands) (api.Customer, error) {
	// Map the customer to the API Customer
	apiCustomer := api.Customer{
		Id:           c.ManagedResource.ID,
		Key:          c.Key,
		Name:         c.Name,
		PrimaryEmail: c.PrimaryEmail,
		Description:  c.Description,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
		DeletedAt:    c.DeletedAt,
		Metadata:     FromMetadata(lo.FromPtr(c.Metadata)),
		Annotations:  FromAnnotations(lo.FromPtr(c.Annotation)),
	}

	if c.UsageAttribution != nil {
		apiCustomer.UsageAttribution = &api.CustomerUsageAttribution{SubjectKeys: c.UsageAttribution.SubjectKeys}
	}

	if c.BillingAddress != nil {
		address := api.Address{
			City:        c.BillingAddress.City,
			State:       c.BillingAddress.State,
			PostalCode:  c.BillingAddress.PostalCode,
			Line1:       c.BillingAddress.Line1,
			Line2:       c.BillingAddress.Line2,
			PhoneNumber: c.BillingAddress.PhoneNumber,
		}

		if c.BillingAddress.Country != nil {
			address.Country = lo.ToPtr(string(*c.BillingAddress.Country))
		}

		apiCustomer.BillingAddress = &address
	}

	if c.Currency != nil {
		apiCustomer.Currency = lo.ToPtr(string(*c.Currency))
	}

	// Map the subscriptions to the API Subscriptions
	if len(subscriptions) > 0 {
		// Let's find the active one
		// FIXME: this will only work with single subscription per customer
		apiCustomer.CurrentSubscriptionId = lo.ToPtr(subscriptions[0].ID)

		// Map the subscriptions to the API Subscriptions if the expand is set
		if lo.Contains(expand, customer.ExpandSubscriptions) {
			apiCustomer.Subscriptions = lo.ToPtr(lo.Map(subscriptions, func(s subscription.Subscription, _ int) api.Subscription {
				return subscriptionhttp.MapSubscriptionToAPI(s)
			}))
		}
	}

	return apiCustomer, nil
}

func MapAccessToAPI(access entitlement.Access) (api.CustomerAccess, error) {
	entitlements := make(map[string]api.EntitlementValue)

	for fKey, v := range access.Entitlements {
		apiVal, err := entitlementdriver.MapEntitlementValueToAPI(v.Value)
		if err != nil {
			return api.CustomerAccess{}, err
		}

		entitlements[fKey] = apiVal
	}

	return api.CustomerAccess{
		Entitlements: entitlements,
	}, nil
}
