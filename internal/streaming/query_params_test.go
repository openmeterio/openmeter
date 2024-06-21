package streaming

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestQueryParamsValidate(t *testing.T) {
	queryWindowSizeMinute := models.WindowSizeMinute
	queryWindowSizeHour := models.WindowSizeHour
	queryWindowSizeDay := models.WindowSizeDay

	tests := []struct {
		name                string
		paramFrom           string
		paramTo             string
		paramWindowTimeZone string
		paramWindowSize     *models.WindowSize
		meterWindowSize     models.WindowSize
		want                error
	}{
		{
			name:            "should fail when from and to are equal",
			paramFrom:       "2023-01-01T00:00:00Z",
			paramTo:         "2023-01-01T00:00:00Z",
			paramWindowSize: &queryWindowSizeMinute,
			meterWindowSize: models.WindowSizeMinute,
			want:            fmt.Errorf("to must be after from"),
		},
		{
			name:            "should fail when from is before to",
			paramFrom:       "2023-01-02T00:00:00Z",
			paramTo:         "2023-01-01T00:00:00Z",
			paramWindowSize: &queryWindowSizeMinute,
			meterWindowSize: models.WindowSizeMinute,
			want:            fmt.Errorf("to must be after from"),
		},
		{
			name:            "should fail when querying on minute but meter is hour",
			paramFrom:       "2023-01-01T00:00:00Z",
			paramTo:         "2023-01-01T00:01:00Z",
			paramWindowSize: &queryWindowSizeMinute,
			meterWindowSize: models.WindowSizeHour,
			want:            fmt.Errorf("cannot query meter with window size HOUR on window size MINUTE"),
		},
		{
			name:            "should fail when querying on minute but meter is day",
			paramFrom:       "2023-01-01T00:00:00Z",
			paramTo:         "2023-01-01T00:01:00Z",
			paramWindowSize: &queryWindowSizeMinute,
			meterWindowSize: models.WindowSizeDay,
			want:            fmt.Errorf("cannot query meter with window size DAY on window size MINUTE"),
		},
		{
			name:            "should fail when querying on hour but meter is day",
			paramFrom:       "2023-01-01T00:00:00Z",
			paramTo:         "2023-01-01T01:00:00Z",
			paramWindowSize: &queryWindowSizeHour,
			meterWindowSize: models.WindowSizeDay,
			want:            fmt.Errorf("cannot query meter with window size DAY on window size HOUR"),
		},
		{
			name:            "should be ok to query per hour on minute meter",
			paramFrom:       "2023-01-01T00:00:00Z",
			paramTo:         "2023-01-01T01:00:00Z",
			paramWindowSize: &queryWindowSizeHour,
			meterWindowSize: models.WindowSizeMinute,
			want:            nil,
		},
		{
			name:            "should be ok to query per day on minute meter",
			paramFrom:       "2023-01-01T00:00:00Z",
			paramTo:         "2023-01-02T00:00:00Z",
			paramWindowSize: &queryWindowSizeDay,
			meterWindowSize: models.WindowSizeMinute,
			want:            nil,
		},
		{
			name:            "should be ok to query per day on hour meter",
			paramFrom:       "2023-01-01T00:00:00Z",
			paramTo:         "2023-01-02T00:00:00Z",
			paramWindowSize: &queryWindowSizeDay,
			meterWindowSize: models.WindowSizeMinute,
			want:            nil,
		},
		{
			name:            "should be ok with rounded to minute",
			paramFrom:       "2023-01-01T00:00:00Z",
			paramTo:         "2023-01-01T00:01:00Z",
			paramWindowSize: &queryWindowSizeMinute,
			meterWindowSize: models.WindowSizeMinute,
			want:            nil,
		},
		{
			name:            "should be with rounded to hour",
			paramFrom:       "2023-01-01T00:00:00Z",
			paramTo:         "2023-01-01T01:00:00Z",
			paramWindowSize: &queryWindowSizeMinute,
			meterWindowSize: models.WindowSizeMinute,
			want:            nil,
		},
		{
			name:            "should be with rounded to day",
			paramFrom:       "2023-01-01T00:00:00Z",
			paramTo:         "2023-01-02T00:01:00Z",
			paramWindowSize: &queryWindowSizeMinute,
			meterWindowSize: models.WindowSizeMinute,
			want:            nil,
		},
		{
			name:            "should fail with not rounded to minute",
			paramFrom:       "2023-01-01T00:00:01Z",
			paramTo:         "2023-01-01T00:01:00Z",
			paramWindowSize: nil,
			meterWindowSize: models.WindowSizeMinute,
			want:            fmt.Errorf("cannot query meter aggregating on MINUTE window size: from must be rounded to MINUTE like YYYY-MM-DDTHH:mm:00"),
		},
		{
			name:            "should fail with not rounded to hour",
			paramFrom:       "2023-01-01T00:00:00Z",
			paramTo:         "2023-01-01T01:01:00Z",
			paramWindowSize: nil,
			meterWindowSize: models.WindowSizeHour,
			want:            fmt.Errorf("cannot query meter aggregating on HOUR window size: to must be rounded to HOUR like YYYY-MM-DDTHH:00:00"),
		},
		{
			name:            "should fail with not rounded to day",
			paramFrom:       "2023-01-01T00:00:00Z",
			paramTo:         "2023-01-01T01:00:00Z",
			paramWindowSize: nil,
			meterWindowSize: models.WindowSizeDay,
			want:            fmt.Errorf("cannot query meter aggregating on DAY window size: to must be rounded to DAY like YYYY-MM-DDT00:00:00"),
		},
	}

	for _, tt := range tests {
		tt := tt
		paramWindowSize := "none"
		if tt.paramWindowSize != nil {
			paramWindowSize = string(*tt.paramWindowSize)
		}
		name := fmt.Sprintf("%s/%s/%s", tt.meterWindowSize, paramWindowSize, tt.name)
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

			got := p.Validate(models.Meter{WindowSize: tt.meterWindowSize})
			if tt.want == nil {
				assert.NoError(t, got)
			} else {
				assert.EqualError(t, got, tt.want.Error())
			}
		})
	}
}
