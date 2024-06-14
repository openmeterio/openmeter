package kafka

import (
	"encoding"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type configValue interface {
	fmt.Stringer
	encoding.TextUnmarshaler
	json.Unmarshaler
}

const (
	BrokerAddressFamilyAny  BrokerAddressFamily = "any"
	BrokerAddressFamilyIPv4 BrokerAddressFamily = "v4"
	BrokerAddressFamilyIPv6 BrokerAddressFamily = "v6"
)

var _ configValue = (*BrokerAddressFamily)(nil)

type BrokerAddressFamily string

func (s *BrokerAddressFamily) UnmarshalText(text []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(text))) {
	case "v4":
		*s = BrokerAddressFamilyIPv4
	case "v6":
		*s = BrokerAddressFamilyIPv6
	case "any":
		*s = BrokerAddressFamilyAny
	default:
		return fmt.Errorf("invalid value broker family address: %s", text)
	}

	return nil
}

func (s *BrokerAddressFamily) UnmarshalJSON(data []byte) error {
	return s.UnmarshalText(data)
}

func (s BrokerAddressFamily) String() string {
	return string(s)
}

var _ configValue = (*TimeDurationMilliSeconds)(nil)

type TimeDurationMilliSeconds time.Duration

func (d *TimeDurationMilliSeconds) UnmarshalText(text []byte) error {
	v, err := time.ParseDuration(strings.TrimSpace(string(text)))
	if err != nil {
		return fmt.Errorf("failed to parse time duration: %w", err)
	}

	*d = TimeDurationMilliSeconds(v)

	return nil
}

func (d *TimeDurationMilliSeconds) UnmarshalJSON(data []byte) error {
	return d.UnmarshalText(data)
}

func (d TimeDurationMilliSeconds) Duration() time.Duration {
	return time.Duration(d)
}

func (d TimeDurationMilliSeconds) String() string {
	return strconv.Itoa(int(time.Duration(d).Milliseconds()))
}
