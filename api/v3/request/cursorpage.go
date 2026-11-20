package request

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
)

// CursorPage is a parsed page[size], page[after] and page[before] query. The
// cursor type C is endpoint specific: each list decides what its opaque cursor
// encodes.
type CursorPage[C any] struct {
	Size   int
	After  *C
	Before *C
}

// ParseCursorPage parses a cursor page query. The request validator already
// enforces the page[size] bounds declared in the spec, so only the defaulting,
// the cursor decoding and the after/before exclusivity are checked here.
//
// Endpoints with their own page type, such as one with a smaller size limit,
// convert it: the generated page types share the same fields, for example
// (*apiv3models.CursorPaginationQueryPage)(params.Page).
func ParseCursorPage[C any](
	ctx context.Context,
	page *apiv3models.CursorPaginationQueryPage,
	defaultSize int,
	decode func(cursor string) (*C, error),
) (CursorPage[C], error) {
	out := CursorPage[C]{Size: defaultSize}

	if page == nil {
		return out, nil
	}

	if page.Size != nil {
		out.Size = *page.Size
	}

	if page.After != nil && page.Before != nil {
		return CursorPage[C]{}, apierrors.NewBadRequestError(ctx,
			errors.New("page[after] and page[before] are mutually exclusive"),
			apierrors.InvalidParameters{{
				Field:  "page[after]",
				Reason: "cannot be combined with page[before]",
				Source: apierrors.InvalidParamSourceQuery,
			}},
		)
	}

	var err error

	if out.After, err = decodeCursorParam(ctx, "page[after]", page.After, decode); err != nil {
		return CursorPage[C]{}, err
	}

	if out.Before, err = decodeCursorParam(ctx, "page[before]", page.Before, decode); err != nil {
		return CursorPage[C]{}, err
	}

	return out, nil
}

func decodeCursorParam[C any](ctx context.Context, field string, raw *string, decode func(string) (*C, error)) (*C, error) {
	if raw == nil {
		return nil, nil
	}

	cursor, err := decode(*raw)
	if err != nil {
		return nil, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{{
			Field:  field,
			Reason: err.Error(),
			Source: apierrors.InvalidParamSourceQuery,
		}})
	}

	return cursor, nil
}
