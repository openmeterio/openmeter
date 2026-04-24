package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

const cloudEventsSpecVersion = "1.0"

// toAPIMeteringIngestedEvent converts a meterevent.Event to its API wire form.
func toAPIMeteringIngestedEvent(e meterevent.Event) (api.MeteringIngestedEvent, error) {
	event := api.MeteringEvent{
		Id:          e.ID,
		Source:      e.Source,
		Specversion: cloudEventsSpecVersion,
		Subject:     e.Subject,
		Type:        e.Type,
		Time:        nullable.NewNullableWithValue[api.DateTime](e.Time),
	}

	if e.Data != "" {
		var data map[string]any
		if err := json.Unmarshal([]byte(e.Data), &data); err != nil {
			return api.MeteringIngestedEvent{}, fmt.Errorf("parse event data as json: %w", err)
		}
		event.Data = nullable.NewNullableWithValue(data)
		event.Datacontenttype = nullable.NewNullableWithValue[api.MeteringEventDatacontenttype](api.MeteringEventDatacontenttype("application/json"))
	}

	return api.MeteringIngestedEvent{
		Event:            event,
		Customer:         toAPICustomerReference(e.CustomerID),
		IngestedAt:       e.IngestedAt,
		StoredAt:         e.StoredAt,
		ValidationErrors: toAPIMeteringIngestedEventValidationErrors(e.ValidationErrors),
	}, nil
}

func toAPICustomerReference(id *string) *api.CustomerReference {
	if id == nil || *id == "" {
		return nil
	}
	return &api.CustomerReference{Id: *id}
}

func toAPIMeteringIngestedEventValidationErrors(errs []error) *[]api.MeteringIngestedEventValidationError {
	if len(errs) == 0 {
		return nil
	}

	result := lo.FilterMap(errs, func(err error, _ int) (api.MeteringIngestedEventValidationError, bool) {
		if err == nil {
			return api.MeteringIngestedEventValidationError{}, false
		}

		return api.MeteringIngestedEventValidationError{
			Code:    "validation_error",
			Message: err.Error(),
		}, true
	})

	if len(result) == 0 {
		return nil
	}

	return lo.ToPtr(result)
}

// fromAPICustomerIDFilter maps the v3 customer_id filter to the backend filter,
// rejecting every operator that the underlying service cannot evaluate. Only
// `eq` and `oeq` are supported because ListEventsV2Params requires a concrete
// IN set.
func fromAPICustomerIDFilter(ctx context.Context, f *api.ULIDFieldFilter) (*filter.FilterString, error) {
	if f == nil {
		return nil, nil
	}

	if f.Neq != nil || f.Contains != nil || len(f.Ocontains) > 0 || f.Exists != nil {
		err := errors.New("only eq and oeq operators are supported")
		return nil, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "filter[customer_id]",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}

	var values []string
	if f.Eq != nil {
		values = append(values, *f.Eq)
	}
	if len(f.Oeq) > 0 {
		values = append(values, f.Oeq...)
	}

	if len(values) == 0 {
		return nil, nil
	}

	return &filter.FilterString{In: &values}, nil
}

// fromAPIEventSort resolves the public sort query into a backend sort field and direction.
func fromAPIEventSort(ctx context.Context, sort *api.SortQuery) (streaming.EventSortField, sortx.Order, error) {
	if lo.FromPtr(sort) == "" {
		return "", "", nil
	}

	parsed, err := request.ParseSortBy(*sort)
	if err != nil {
		return "", "", apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "sort",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}

	var field streaming.EventSortField
	switch parsed.Field {
	case string(streaming.EventSortFieldTime):
		field = streaming.EventSortFieldTime
	case string(streaming.EventSortFieldIngestedAt):
		field = streaming.EventSortFieldIngestedAt
	case string(streaming.EventSortFieldStoredAt):
		field = streaming.EventSortFieldStoredAt
	default:
		err := fmt.Errorf("unsupported sort field: %q", parsed.Field)
		return "", "", apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "sort",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}

	// If the caller did not supply an explicit asc/desc suffix, default to
	// descending so `sort=time` behaves the same as omitting the parameter
	// (most recent first).
	order := parsed.Order.ToSortxOrder()
	if len(strings.Fields(*sort)) == 1 {
		order = sortx.OrderDesc
	}

	return field, order, nil
}
