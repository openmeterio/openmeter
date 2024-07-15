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

var _ configValue = (*DebugContext)(nil)

type DebugContext string

func (c DebugContext) String() string {
	return string(c)
}

func (c *DebugContext) UnmarshalText(text []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(text))) {
	case "generic":
		*c = DebugContextGeneric
	case "broker":
		*c = DebugContextBroker
	case "topic":
		*c = DebugContextTopic
	case "metadata":
		*c = DebugContextMetadata
	case "feature":
		*c = DebugContextFeature
	case "queue":
		*c = DebugContextQueue
	case "msg":
		*c = DebugContextMessage
	case "protocol":
		*c = DebugContextProtocol
	case "cgrp":
		*c = DebugContextConsumerGroup
	case "security":
		*c = DebugContextSecurity
	case "fetch":
		*c = DebugContextFetch
	case "interceptor":
		*c = DebugContextInterceptor
	case "plugin":
		*c = DebugContextPlugin
	case "consumer":
		*c = DebugContextConsumer
	case "admin":
		*c = DebugContextAdmin
	case "eos":
		*c = DebugContextIdempotentProducer
	case "mock":
		*c = DebugContextMock
	case "assignor":
		*c = DebugContextAssignor
	case "conf":
		*c = DebugContextConfig
	case "all":
		*c = DebugContextAll
	default:
		return fmt.Errorf("invalid debug context: %s", text)
	}

	return nil
}

func (c *DebugContext) UnmarshalJSON(data []byte) error {
	return c.UnmarshalText(data)
}

const (
	// DebugContextGeneric enables generic client instance level debugging.
	// Includes initialization and termination debugging.
	// Client Type: producer, consumer
	DebugContextGeneric DebugContext = "generic"
	// DebugContextBroker enables broker and connection state debugging.
	// Client Type: producer, consumer
	DebugContextBroker DebugContext = "broker"
	// DebugContextTopic enables topic and partition state debugging. Includes leader changes.
	// Client Type: producer, consumer
	DebugContextTopic DebugContext = "topic"
	// DebugContextMetadata enables cluster and topic metadata retrieval debugging.
	// Client Type: producer, consumer
	DebugContextMetadata DebugContext = "metadata"
	// DebugContextFeature enables Kafka protocol feature support as negotiated with the broker.
	// Client Type: producer, consumer
	DebugContextFeature DebugContext = "feature"
	// DebugContextQueue enables message queue debugging.
	// Client Type: producer
	DebugContextQueue DebugContext = "queue"
	// DebugContextMessage enables message debugging. Includes information about batching, compression, sizes, etc.
	// Client Type: producer, consumer
	DebugContextMessage DebugContext = "msg"
	// DebugContextProtocol enables Kafka protocol request/response debugging. Includes latency (rtt) printouts.
	// Client Type: producer, consumer
	DebugContextProtocol DebugContext = "protocol"
	// DebugContextConsumerGroup enables low-level consumer group state debugging.
	// Client Type: consumer
	DebugContextConsumerGroup DebugContext = "cgrp"
	// DebugContextSecurity enables security and authentication debugging.
	// Client Type: producer, consumer
	DebugContextSecurity DebugContext = "security"
	// DebugContextFetch enables consumer message fetch debugging. Includes decision when and why messages are fetched.
	// Client Type: consumer
	DebugContextFetch DebugContext = "fetch"
	// DebugContextInterceptor enables interceptor interface debugging.
	// Client Type: producer, consumer
	DebugContextInterceptor DebugContext = "interceptor"
	// DebugContextPlugin enables plugin loading debugging.
	// Client Type: producer, consumer
	DebugContextPlugin DebugContext = "plugin"
	// DebugContextConsumer enables high-level consumer debugging.
	// Client Type: consumer
	DebugContextConsumer DebugContext = "consumer"
	// DebugContextAdmin enables admin API debugging.
	// Client Type: admin
	DebugContextAdmin DebugContext = "admin"
	// DebugContextIdempotentProducer enables idempotent Producer debugging.
	// Client Type: producer
	DebugContextIdempotentProducer DebugContext = "eos"
	// DebugContextMock enables mock cluster functionality debugging.
	// Client Type: producer, consumer
	DebugContextMock DebugContext = "mock"
	// DebugContextAssignor enables detailed consumer group partition assignor debugging.
	// Client Type: consumer
	DebugContextAssignor DebugContext = "assignor"
	// DebugContextConfig enables displaying set configuration properties on startup.
	// Client Type: producer, consumer
	DebugContextConfig DebugContext = "conf"
	// DebugContextAll enables all of the above.
	// Client Type: producer, consumer
	DebugContextAll DebugContext = "all"
)

var _ fmt.Stringer = (DebugContexts)(nil)

type DebugContexts []DebugContext

func (d DebugContexts) String() string {
	if len(d) > 0 {
		dd := make([]string, len(d))
		for idx, v := range d {
			dd[idx] = v.String()
		}

		return strings.Join(dd, ",")
	}

	return ""
}
