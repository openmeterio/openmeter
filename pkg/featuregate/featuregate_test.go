package featuregate_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/featuregate"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
)

// stubGate is a controllable Gate implementation for testing.
type stubGate struct {
	result    bool
	err       error
	callCount int
}

func (s *stubGate) EvaluateBool(_, _ string, _ bool) (bool, error) {
	s.callCount++
	return s.result, s.err
}

func TestFlags_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		flags   featuregate.Flags
		wantErr bool
	}{
		{
			name:    "valid key",
			flags:   featuregate.Flags{featuregate.FeatureFlag("om_ff_credits_enabled"): "my-flag"},
			wantErr: false,
		},
		{
			name:    "unknown key",
			flags:   featuregate.Flags{featuregate.FeatureFlag("unknown_key"): "my-flag"},
			wantErr: true,
		},
		{
			name:    "empty flags",
			flags:   featuregate.Flags{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.flags.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFeatureGateChecker_Enabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		gate    featuregate.Gate
		flag    string
		want    bool
		wantErr bool
	}{
		{
			name: "nil gate returns true",
			gate: nil,
			flag: "some-flag",
			want: true,
		},
		{
			name: "empty flag returns true",
			gate: &stubGate{result: false},
			flag: "",
			want: true,
		},
		{
			name: "gate returns true",
			gate: &stubGate{result: true},
			flag: "my-flag",
			want: true,
		},
		{
			name: "gate returns false",
			gate: &stubGate{result: false},
			flag: "my-flag",
			want: false,
		},
		{
			name:    "gate returns error",
			gate:    &stubGate{err: errors.New("gate error")},
			flag:    "my-flag",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checker := featuregate.NewFeatureGateChecker(tc.gate, featuregate.Flags{})
			got, err := checker.Enabled("test-ns", tc.flag)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFeatureGateChecker_Enabled_Caching(t *testing.T) {
	t.Parallel()

	gate := &stubGate{result: true}
	checker := featuregate.NewFeatureGateChecker(gate, featuregate.Flags{})

	// First call — gate is invoked
	got, err := checker.Enabled("test-ns", "my-flag")
	require.NoError(t, err)
	assert.True(t, got)
	assert.Equal(t, 1, gate.callCount)

	// Second call with same ns+flag — served from cache
	got, err = checker.Enabled("test-ns", "my-flag")
	require.NoError(t, err)
	assert.True(t, got)
	assert.Equal(t, 1, gate.callCount, "gate should not be called again on cache hit")
}

func TestFeatureGateChecker_Validate(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver", func(t *testing.T) {
		var checker *featuregate.FeatureGateChecker
		require.Error(t, checker.Validate())
	})

	t.Run("nil gate", func(t *testing.T) {
		checker := featuregate.NewFeatureGateChecker(nil, featuregate.Flags{})
		require.Error(t, checker.Validate())
	})

	t.Run("valid checker", func(t *testing.T) {
		checker := featuregate.NewFeatureGateChecker(
			&stubGate{},
			featuregate.Flags{featuregate.FeatureFlag("om_ff_credits_enabled"): "val"},
		)
		require.NoError(t, checker.Validate())
	})
}

func TestNewMiddleware(t *testing.T) {
	t.Parallel()

	creditsKey := featuregate.FeatureFlag("om_ff_credits_enabled")
	flags := featuregate.Flags{creditsKey: "credits-flag"}

	t.Run("populates context with flag value", func(t *testing.T) {
		gate := &stubGate{result: false}
		checker := featuregate.NewFeatureGateChecker(gate, flags)

		getNS := func(ctx context.Context) (string, bool) { return "test-ns", true }

		var capturedCtx context.Context
		next := operation.Operation[string, string](func(ctx context.Context, _ string) (string, error) {
			capturedCtx = ctx
			return "ok", nil
		})

		mw := featuregate.NewMiddleware[string, string](getNS, checker)
		op := mw(next)

		_, err := op(context.Background(), "req")
		require.NoError(t, err)

		creditEnabled, found := featuregate.ContextResolver().Credits(capturedCtx)
		assert.True(t, found)
		assert.False(t, creditEnabled)
	})

	t.Run("returns 500 when namespace not found", func(t *testing.T) {
		checker := featuregate.NewFeatureGateChecker(&stubGate{result: true}, flags)

		getNS := func(ctx context.Context) (string, bool) { return "", false }

		next := operation.Operation[string, string](func(ctx context.Context, _ string) (string, error) {
			return "ok", nil
		})

		mw := featuregate.NewMiddleware[string, string](getNS, checker)
		op := mw(next)

		_, err := op(context.Background(), "req")
		require.Error(t, err)

		var httpErr commonhttp.ErrorWithHTTPStatusCode
		require.ErrorAs(t, err, &httpErr)
		assert.Equal(t, http.StatusInternalServerError, httpErr.StatusCode)
	})

	t.Run("propagates gate error", func(t *testing.T) {
		gateErr := errors.New("gate unavailable")
		checker := featuregate.NewFeatureGateChecker(&stubGate{err: gateErr}, flags)

		getNS := func(ctx context.Context) (string, bool) { return "test-ns", true }

		called := false
		next := operation.Operation[string, string](func(ctx context.Context, _ string) (string, error) {
			called = true
			return "ok", nil
		})

		mw := featuregate.NewMiddleware[string, string](getNS, checker)
		op := mw(next)

		_, err := op(context.Background(), "req")
		require.Error(t, err)
		assert.False(t, called, "next should not be called when gate errors")
	})
}

func TestContextResolver_Credits(t *testing.T) {
	t.Parallel()

	creditsKey := featuregate.FeatureFlag("om_ff_credits_enabled")

	t.Run("no value in context", func(t *testing.T) {
		val, found := featuregate.ContextResolver().Credits(context.Background())
		assert.False(t, found)
		assert.False(t, val)
	})

	t.Run("context value true", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), creditsKey, true)
		val, found := featuregate.ContextResolver().Credits(ctx)
		assert.True(t, found)
		assert.True(t, val)
	})

	t.Run("context value false", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), creditsKey, false)
		val, found := featuregate.ContextResolver().Credits(ctx)
		assert.True(t, found)
		assert.False(t, val)
	})
}
