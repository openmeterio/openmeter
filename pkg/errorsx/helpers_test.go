package errorsx

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api"
	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

func TestIsAPIError(t *testing.T) {
	apiErr := &api.InvalidParamFormatError{
		ParamName: "page",
		Err:       errors.New("invalid syntax"),
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "legacy api error",
			err:  apiErr,
			want: true,
		},
		{
			name: "v3 api error",
			err:  &apiv3.InvalidParamFormatError{ParamName: "page", Err: errors.New("invalid syntax")},
			want: true,
		},
		{
			name: "different generated api error type",
			err:  &api.RequiredParamError{ParamName: "page"},
			want: true,
		},
		{
			name: "wrapped api error",
			err:  fmt.Errorf("wrapped: %w", apiErr),
			want: true,
		},
		{
			name: "joined api error",
			err: errors.Join(
				errors.New("other error"),
				apiErr,
			),
			want: true,
		},
		{
			name: "ordinary error",
			err:  errors.New("other error"),
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, isAPIError(test.err))
		})
	}
}
