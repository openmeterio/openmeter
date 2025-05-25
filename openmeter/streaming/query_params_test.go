package streaming

import (
	"fmt"
	"testing"
	"time"

	"github.com/samber/lo"
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
		name := fmt.Sprintf("%s/%s", paramWindowSize, tt.name)
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

func TestQueryParamsHash(t *testing.T) {
	tests := []struct {
		name  string
		query QueryParams
		want  string
	}{
		{
			name: "should hash with from and to",
			query: QueryParams{
				From: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				To:   lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
			want: "4345920ea060935f",
		},
		{
			name: "should hash with only from",
			query: QueryParams{
				From: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
			want: "4345920ea060935f",
		},
		{
			name: "should hash with from and non aligned window size",
			query: QueryParams{
				From:       lo.ToPtr(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)),
				WindowSize: lo.ToPtr(meter.WindowSizeDay),
			},
			want: "4fdb33bbe00aba51",
		},
		{
			name: "should hash with from and aligned window size",
			query: QueryParams{
				From:       lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				WindowSize: lo.ToPtr(meter.WindowSizeDay),
			},
			want: "8214c60c3d26b5de",
		},
		{
			name: "should hash with subject filter",
			query: QueryParams{
				From:          lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				FilterSubject: []string{"subject1", "subject2"},
			},
			want: "d25970cdd89ed8ff",
		},
		{
			name: "should hash with subject filter in different order",
			query: QueryParams{
				From:          lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				FilterSubject: []string{"subject2", "subject1"},
			},
			want: "d25970cdd89ed8ff", // same as above
		},
		{
			name: "should hash with group by filter",
			query: QueryParams{
				From: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				FilterGroupBy: map[string][]string{
					"group1": {"value1.1", "value1.2"},
					"group2": {"value2.1", "value2.2"},
				},
			},
			want: "bc2088a1203d5ccb",
		},
		{
			name: "should hash with group by filter in different order",
			query: QueryParams{
				From: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				FilterGroupBy: map[string][]string{
					"group2": {"value2.2", "value2.1"},
					"group1": {"value1.2", "value1.1"},
				},
			},
			want: "bc2088a1203d5ccb", // same as above
		},
		{
			name: "should hash with group by",
			query: QueryParams{
				From:    lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				GroupBy: []string{"group1", "group2"},
			},
			want: "8cfe2f204fc2fd3f",
		},
		{
			name: "should hash with group by in different order",
			query: QueryParams{
				From:    lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				GroupBy: []string{"group2", "group1"},
			},
			want: "8cfe2f204fc2fd3f", // same as above
		},
		{
			name: "should hash with window time zone",
			query: QueryParams{
				From:           lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				WindowTimeZone: time.FixedZone("Europe/Budapest", 3600),
			},
			want: "8657c64dd4616908",
		},
		{
			name: "should hash with window time zone in different order",
			query: QueryParams{
				From:           lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				WindowTimeZone: time.FixedZone("Europe/Berlin", 3600),
			},
			want: "4c61eca541f3e645",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			key, err := tt.query.Hash()
			if err != nil {
				t.Fatal(fmt.Errorf("failed to hash: %w", err))
				return
			}

			assert.Equal(t, tt.want, fmt.Sprintf("%x", key))
		})
	}
}
