package streaming

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestQueryParamsValidate(t *testing.T) {
	queryWindowSizeMinute := meter.WindowSizeMinute

	tests := []struct {
		name                string
		paramFrom           string
		paramTo             string
		paramWindowTimeZone string
		paramWindowSize     *meter.WindowSize
		want                error
	}{
		{
			name:            "should fail when from and to are equal",
			paramFrom:       "2023-01-01T00:00:00Z",
			paramTo:         "2023-01-01T00:00:00Z",
			paramWindowSize: &queryWindowSizeMinute,
			want:            models.NewGenericValidationError(fmt.Errorf("from and to cannot be equal")),
		},
		{
			name:            "should fail when from is before to",
			paramFrom:       "2023-01-02T00:00:00Z",
			paramTo:         "2023-01-01T00:00:00Z",
			paramWindowSize: &queryWindowSizeMinute,
			want:            models.NewGenericValidationError(fmt.Errorf("from must be before to")),
		},
	}

	for _, tt := range tests {
		tt := tt
		paramWindowSize := "none"
		if tt.paramWindowSize != nil {
			paramWindowSize = string(*tt.paramWindowSize)
		}
		name := fmt.Sprintf("%s/%s/%s", paramWindowSize, tt.name)
		t.Run(name, func(t *testing.T) {
			from, err := time.Parse(time.RFC3339, tt.paramFrom)
			if err != nil {
				t.Fatal(fmt.Errorf("failed to parse from: %w", err))
				return
			}
			to, err := time.Parse(time.RFC3339, tt.paramTo)
			if err != nil {
				t.Fatal(fmt.Errorf("failed to parse to: %w", err))
				return
			}

			p := QueryParams{
				From:       &from,
				To:         &to,
				WindowSize: tt.paramWindowSize,
			}

			got := p.Validate()
			if tt.want == nil {
				assert.NoError(t, got)
			} else {
				assert.EqualError(t, got, tt.want.Error())
			}
		})
	}
}
