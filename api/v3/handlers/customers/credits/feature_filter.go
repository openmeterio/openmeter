package customerscredits

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/mo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
)

func fromAPICustomerCreditFeatureFilter(f *api.StringFieldFilter) (mo.Option[creditpurchase.FeatureFilters], error) {
	if f == nil {
		return customerbalance.AllFeatureFilter(), nil
	}

	if f.Exists != nil {
		if !*f.Exists {
			return customerbalance.NewUnrestrictedFeatureFilter(), nil
		}

		return customerbalance.AllFeatureFilter(), errors.New("exists=true operator is not supported")
	}

	if op := unsupportedCustomerCreditFeatureKeyOperator(f); op != "" {
		return customerbalance.AllFeatureFilter(), fmt.Errorf("%s operator is not supported", op)
	}

	features := make([]string, 0, 1+len(f.Oeq))
	if f.Eq != nil {
		features = append(features, *f.Eq)
	}
	features = append(features, f.Oeq...)

	if len(features) == 0 {
		return customerbalance.AllFeatureFilter(), nil
	}

	featureFilter := customerbalance.NewFeatureFilter(features)
	if err := customerbalance.ValidateFeatureFilter(featureFilter); err != nil {
		return customerbalance.AllFeatureFilter(), err
	}

	return featureFilter, nil
}

func unsupportedCustomerCreditFeatureKeyOperator(f *api.StringFieldFilter) string {
	switch {
	case f.Neq != nil:
		return "neq"
	case f.Contains != nil:
		return "contains"
	case len(f.Ocontains) > 0:
		return "ocontains"
	case f.Gt != nil:
		return "gt"
	case f.Gte != nil:
		return "gte"
	case f.Lt != nil:
		return "lt"
	case f.Lte != nil:
		return "lte"
	default:
		return ""
	}
}

func newFeatureKeyFilterBadRequest(ctx context.Context, err error) error {
	return apierrors.NewBadRequestError(
		ctx,
		errors.New("invalid feature_key filter"),
		apierrors.InvalidParameters{{
			Field:  "filter[feature_key]",
			Reason: err.Error(),
			Source: apierrors.InvalidParamSourceQuery,
		}},
	)
}
