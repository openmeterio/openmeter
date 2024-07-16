package kafka

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBrokerAddressFamily(t *testing.T) {
	tests := []struct {
		Name string

		Value          string
		ExpectedError  error
		ExplectedValue BrokerAddressFamily
	}{
		{
			Name:           "Any",
			Value:          "any",
			ExpectedError:  nil,
			ExplectedValue: BrokerAddressFamilyAny,
		},
		{
			Name:           "IPv4",
			Value:          "v4",
			ExpectedError:  nil,
			ExplectedValue: BrokerAddressFamilyIPv4,
		},
		{
			Name:           "IPv6",
			Value:          "v6",
			ExpectedError:  nil,
			ExplectedValue: BrokerAddressFamilyIPv6,
		},
		{
			Name:          "Invalid",
			Value:         "invalid",
			ExpectedError: errors.New("invalid value broker family address: invalid"),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var family BrokerAddressFamily

			err := family.UnmarshalText([]byte(test.Value))
			assert.Equal(t, test.ExpectedError, err)
			if err == nil {
				assert.Equal(t, test.ExplectedValue, family)
			}

			err = family.UnmarshalJSON([]byte(test.Value))
			assert.Equal(t, test.ExpectedError, err)
			if err == nil {
				assert.Equal(t, test.ExplectedValue, family)
			}
		})
	}
}

func TestTimeDurationMilliSeconds(t *testing.T) {
	tests := []struct {
		Name string

		Value            string
		ExpectedError    error
		ExplectedValue   TimeDurationMilliSeconds
		ExpectedString   string
		ExpectedDuration time.Duration
	}{
		{
			Name:             "Duration",
			Value:            "6s",
			ExpectedError:    nil,
			ExplectedValue:   TimeDurationMilliSeconds(6 * time.Second),
			ExpectedString:   "6000",
			ExpectedDuration: 6 * time.Second,
		},
		{
			Name:          "Invalid",
			Value:         "10000",
			ExpectedError: fmt.Errorf("failed to parse time duration: %w", errors.New("time: missing unit in duration \"10000\"")),
		},
		{
			Name:          "Invalid",
			Value:         "invalid",
			ExpectedError: fmt.Errorf("failed to parse time duration: %w", errors.New("time: invalid duration \"invalid\"")),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var timeMs TimeDurationMilliSeconds

			err := timeMs.UnmarshalText([]byte(test.Value))
			assert.Equal(t, test.ExpectedError, err)
			if err == nil {
				assert.Equal(t, test.ExplectedValue, timeMs)
				assert.Equal(t, test.ExpectedString, timeMs.String())
				assert.Equal(t, test.ExpectedDuration, timeMs.Duration())
			}

			err = timeMs.UnmarshalJSON([]byte(test.Value))
			assert.Equal(t, test.ExpectedError, err)
			if err == nil {
				assert.Equal(t, test.ExplectedValue, timeMs)
				assert.Equal(t, test.ExpectedString, timeMs.String())
				assert.Equal(t, test.ExpectedDuration, timeMs.Duration())
			}
		})
	}
}
