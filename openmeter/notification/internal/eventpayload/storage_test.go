package eventpayload

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapJSONFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "model fields and acronyms",
			input: `{"created_at":"snapshot","masked_api_key":"secret","interval_iso":"P1M","lines":[{"feature_id":"f"}]}`,
			want:  `{"createdAt":"snapshot","maskedAPIKey":"secret","intervalISO":"P1M","lines":[{"featureId":"f"}]}`,
		},
		{
			name:  "user keys and opaque annotation objects",
			input: `{"metadata":{"User_Key":"Value"},"annotations":{"nested_value":{"leave_me":true}},"meter_group_by_filters":{"User_Key":"v"},"advanced_meter_group_by_filters":{"User_Key":{"$in":["A"]}}}`,
			want:  `{"metadata":{"User_Key":"Value"},"annotations":{"nested_value":{"leave_me":true}},"meterGroupByFilters":{"User_Key":"v"},"advancedMeterGroupByFilters":{"User_Key":{"$in":["A"]}}}`,
		},
		{
			name:  "exact JSON numbers",
			input: `{"large_number":9007199254740993,"precise_number":0.1234567890123456789}`,
			want:  `{"largeNumber":9007199254740993,"preciseNumber":0.1234567890123456789}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapJSONFields([]byte(tt.input), toCamelCase)
			require.NoError(t, err)

			// Compare canonical bytes too: JSONEq would round numbers through float64.
			require.JSONEq(t, tt.want, string(got))
			if tt.name == "exact JSON numbers" {
				require.Equal(t, tt.want, string(got))
			}
		})
	}
}
