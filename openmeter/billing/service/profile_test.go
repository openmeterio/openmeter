package billingservice

import (
	"context"
	"log/slog"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// recordingAdapter is a minimal billing.Adapter test double that captures the
// last ListProfiles call and returns a canned result.
type recordingAdapter struct {
	billing.Adapter // nil; panics if any other method is called

	receivedInput billing.ListProfilesInput
	result        pagination.Result[billing.BaseProfile]
	err           error
}

func (a *recordingAdapter) ListProfiles(_ context.Context, input billing.ListProfilesInput) (pagination.Result[billing.BaseProfile], error) {
	a.receivedInput = input
	return a.result, a.err
}

func newServiceForProfileTest(adapter *recordingAdapter) *Service {
	return &Service{
		adapter: adapter,
		logger:  slog.Default(),
	}
}

func TestListProfiles(t *testing.T) {
	const ns = "test-ns"

	type testCase struct {
		name          string
		input         billing.ListProfilesInput
		wantErr       bool
		assertAdapter func(t *testing.T, got billing.ListProfilesInput)
	}

	cases := []testCase{
		{
			name: "FilterByNameEq",
			input: billing.ListProfilesInput{
				Namespace: ns,
				Name:      &filter.FilterString{Eq: lo.ToPtr("Acme Billing")},
			},
			assertAdapter: func(t *testing.T, got billing.ListProfilesInput) {
				t.Helper()
				require.NotNil(t, got.Name)
				require.Equal(t, lo.ToPtr("Acme Billing"), got.Name.Eq)
			},
		},
		{
			name: "FilterByNameContains",
			input: billing.ListProfilesInput{
				Namespace: ns,
				Name:      &filter.FilterString{Contains: lo.ToPtr("acme")},
			},
			assertAdapter: func(t *testing.T, got billing.ListProfilesInput) {
				t.Helper()
				require.NotNil(t, got.Name)
				require.Equal(t, lo.ToPtr("acme"), got.Name.Contains)
			},
		},
		{
			name: "FilterByNameOeq",
			input: billing.ListProfilesInput{
				Namespace: ns,
				Name:      &filter.FilterString{In: lo.ToPtr([]string{"Acme", "Beta"})},
			},
			assertAdapter: func(t *testing.T, got billing.ListProfilesInput) {
				t.Helper()
				require.NotNil(t, got.Name)
				require.Equal(t, lo.ToPtr([]string{"Acme", "Beta"}), got.Name.In)
			},
		},
		{
			name: "FilterByIDEq",
			input: billing.ListProfilesInput{
				Namespace: ns,
				ID:        &filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr("01HXYZ1234567890ABCDEFGHJK")}},
			},
			assertAdapter: func(t *testing.T, got billing.ListProfilesInput) {
				t.Helper()
				require.NotNil(t, got.ID)
				require.Equal(t, lo.ToPtr("01HXYZ1234567890ABCDEFGHJK"), got.ID.Eq)
			},
		},
		{
			name: "SortByNameAsc",
			input: billing.ListProfilesInput{
				Namespace: ns,
				OrderBy:   "name",
				Order:     sortx.OrderAsc,
			},
			assertAdapter: func(t *testing.T, got billing.ListProfilesInput) {
				t.Helper()
				require.Equal(t, billing.OrderByName, got.OrderBy)
				require.Equal(t, sortx.OrderAsc, got.Order)
			},
		},
		{
			name: "SortByUpdatedAt",
			input: billing.ListProfilesInput{
				Namespace: ns,
				OrderBy:   "updatedAt",
				Order:     sortx.OrderDesc,
			},
			assertAdapter: func(t *testing.T, got billing.ListProfilesInput) {
				t.Helper()
				require.Equal(t, billing.OrderByUpdatedAt, got.OrderBy)
				require.Equal(t, sortx.OrderDesc, got.Order)
			},
		},
		{
			// both Eq and Contains set — validateSingleOperator returns ErrFilterMultipleOperators
			name: "ValidationError_NameMultipleOperators",
			input: billing.ListProfilesInput{
				Namespace: ns,
				Name:      &filter.FilterString{Eq: lo.ToPtr("x"), Contains: lo.ToPtr("y")},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()
			rec := &recordingAdapter{}
			svc := newServiceForProfileTest(rec)

			_, err := svc.ListProfiles(t.Context(), tc.input)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tc.assertAdapter != nil {
				tc.assertAdapter(t, rec.receivedInput)
			}
		})
	}
}
