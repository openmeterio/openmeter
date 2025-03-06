package httpdriver

import (
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ListCustomerOverridesRequest  = billing.ListCustomerOverridesInput
	ListCustomerOverridesResponse = api.BillingProfileCustomerOverridePaginatedResponse
	ListCustomerOverridesParams   = api.BillingProfileListCustomerOverridesParams
	ListCustomerOverridesHandler  httptransport.HandlerWithArgs[ListCustomerOverridesParams, ListCustomerOverridesResponse, ListCustomerOverridesParams]
)
