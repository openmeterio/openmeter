package customerscredits

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/samber/lo"
	"github.com/samber/mo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
)

func fromAPICustomerCreditPlanFilter(key *api.StringFieldFilter, version *api.NumericFieldFilter) (mo.Option[*ledger.PlanFilter], error) {
	var selected mo.Option[*ledger.PlanFilter]

	if key != nil {
		if op := unsupportedCustomerCreditKeyOperator(key); op != "" {
			return selected, fmt.Errorf("plan_key: %s operator is not supported", op)
		}

		if key.Exists != nil {
			if *key.Exists || key.Eq != nil || len(key.Oeq) > 0 {
				return selected, errors.New("plan_key: only exists=false without other operators is supported")
			}
			selected = mo.Some[*ledger.PlanFilter](nil)
		} else {
			keys := slices.Clone(key.Oeq)
			if key.Eq != nil {
				keys = append(keys, *key.Eq)
			}
			keys = lo.Uniq(keys)
			if len(keys) > 1 {
				return selected, errors.New("plan_key: exactly one plan key is supported")
			}
			if len(keys) == 1 {
				selected = mo.Some(&ledger.PlanFilter{Key: keys[0]})
			}
		}
	}

	if version != nil {
		if version.Neq != nil || len(version.Oeq) > 0 || version.Gt != nil || version.Gte != nil || version.Lt != nil || version.Lte != nil || version.Eq == nil {
			return selected, errors.New("plan_version: only eq is supported")
		}

		value := *version.Eq
		if math.IsNaN(value) || value < 1 || value > math.MaxInt32 || math.Trunc(value) != value {
			return selected, errors.New("plan_version: must be a positive 32-bit integer")
		}

		plan := selected.OrEmpty()
		if plan == nil || plan.Key == "" {
			return selected, errors.New("plan_version: requires a concrete plan_key")
		}

		plan.Version = &ledger.VersionFilter{Eq: lo.ToPtr(int(value))}
	}

	if err := customerbalance.ValidatePlanFilter(selected); err != nil {
		return selected, err
	}

	return selected, nil
}

func newPlanFilterBadRequest(ctx context.Context, err error) error {
	return apierrors.NewBadRequestError(ctx, errors.New("invalid plan filter"), apierrors.InvalidParameters{{
		Field: "filter", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery,
	}})
}
