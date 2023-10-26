package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWindowSizeFromDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  WindowSize
		error error
	}{
		{
			input: time.Minute,
			want:  WindowSizeMinute,
			error: nil,
		},
		{
			input: time.Hour,
			want:  WindowSizeHour,
			error: nil,
		},
		{
			input: 24 * time.Hour,
			want:  WindowSizeDay,
			error: nil,
		},
		{
			input: 2 * time.Minute,
			want:  "",
			error: fmt.Errorf("invalid window size duration: %s", 2*time.Minute),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := WindowSizeFromDuration(tt.input)
			if err != nil {
				if tt.error == nil {
					t.Error(err)
				}

				assert.Equal(t, tt.error, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
