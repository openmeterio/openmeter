package svix

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/notification/v4"
)

func TestResolveResendBaseline(t *testing.T) {
	type testCase struct {
		generation uint64
		data       v4.ProviderData
		baseline   []string
		wantError  string
	}

	tests := map[string]testCase{
		"previous generation observation": {
			generation: 2,
			data: v4.ProviderData{
				providerDataDispatchGeneration: "1",
				providerDataObservedAttempts:   `["b","a"]`,
			},
			baseline: []string{"a", "b"},
		},
		"same generation preserves baseline": {
			generation: 2,
			data: v4.ProviderData{
				providerDataDispatchGeneration: "2",
				providerDataObservedAttempts:   `["a","b"]`,
				providerDataResendBaseline:     `["a"]`,
			},
			baseline: []string{"a"},
		},
		"next generation uses observation not old baseline": {
			generation: 3,
			data: v4.ProviderData{
				providerDataDispatchGeneration: "2",
				providerDataObservedAttempts:   `["a","b"]`,
				providerDataResendBaseline:     `["a"]`,
			},
			baseline: []string{"a", "b"},
		},
		"explicit empty observation": {
			generation: 2,
			data: v4.ProviderData{
				providerDataDispatchGeneration: "1",
				providerDataObservedAttempts:   `[]`,
			},
			baseline: []string{},
		},
		"explicit empty baseline": {
			generation: 2,
			data: v4.ProviderData{
				providerDataDispatchGeneration: "2",
				providerDataResendBaseline:     `[]`,
			},
			baseline: []string{},
		},
		"zero generation": {
			wantError: "invalid resend generation",
		},
		"initial generation": {
			generation: 1,
			wantError:  "invalid resend generation",
		},
		"missing generation": {
			generation: 2,
			wantError:  "invalid observation generation",
		},
		"stale generation": {
			generation: 3,
			data:       v4.ProviderData{providerDataDispatchGeneration: "1"},
			wantError:  "invalid observation generation",
		},
		"future generation": {
			generation: 2,
			data:       v4.ProviderData{providerDataDispatchGeneration: "3"},
			wantError:  "invalid observation generation",
		},
		"missing observation": {
			generation: 2,
			data:       v4.ProviderData{providerDataDispatchGeneration: "1"},
			wantError:  "invalid resend baseline",
		},
		"missing baseline does not fall back to observation": {
			generation: 2,
			data: v4.ProviderData{
				providerDataDispatchGeneration: "2",
				providerDataObservedAttempts:   `["a","b"]`,
			},
			wantError: "invalid resend baseline",
		},
	}

	for name, encoded := range map[string]string{
		"null":          `null`,
		"object":        `{}`,
		"malformed":     `[`,
		"mixed types":   `["a",1]`,
		"null element":  `[null]`,
		"blank ID":      `[" "]`,
		"empty ID":      `[""]`,
		"duplicate IDs": `["a","a"]`,
	} {
		tests[name] = testCase{
			generation: 2,
			data: v4.ProviderData{
				providerDataDispatchGeneration: "1",
				providerDataObservedAttempts:   encoded,
			},
			wantError: "invalid resend baseline",
		}
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			original := maps.Clone(test.data)

			baseline, err := resolveResendBaseline(v4.GetEventDestination{
				DispatchGeneration:              test.generation,
				EventDeliveryStatusProviderData: test.data,
			})

			require.Equal(t, original, test.data)

			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				require.Nil(t, baseline)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.baseline, baseline)
		})
	}
}
