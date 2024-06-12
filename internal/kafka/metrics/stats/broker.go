package stats

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	BrokerSourceUnknown    BrokerSource = "unknown"
	BrokerSourceLearned    BrokerSource = "learned"
	BrokerSourceConfigured BrokerSource = "configured"
	BrokerSourceInternal   BrokerSource = "internal"
	BrokerSourceLogical    BrokerSource = "logical"
)

type BrokerSource string

func (s *BrokerSource) UnmarshalJSON(data []byte) error {
	var value string

	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("failed to unmarshal value: %s", data)
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "learned":
		*s = BrokerSourceLearned
	case "configured":
		*s = BrokerSourceConfigured
	case "internal":
		*s = BrokerSourceInternal
	case "logical":
		*s = BrokerSourceLogical
	default:
		*s = BrokerSourceUnknown
	}

	return nil
}

func (s BrokerSource) Int64() int64 {
	var i int64
	switch s {
	case BrokerSourceLearned:
		i = 0
	case BrokerSourceConfigured:
		i = 1
	case BrokerSourceInternal:
		i = 2
	case BrokerSourceLogical:
		i = 3
	default:
		i = -1
	}

	return i
}

const (
	BrokerStateUnknown         BrokerState = "unknown"
	BrokerStateInit            BrokerState = "init"
	BrokerStateDown            BrokerState = "down"
	BrokerStateConnect         BrokerState = "connect"
	BrokerStateAuth            BrokerState = "auth"
	BrokerStateApiVersionQuery BrokerState = "apiversion-query"
	BrokerStateAuthHandshake   BrokerState = "auth-handshake"
	BrokerStateUp              BrokerState = "up"
	BrokerStateUpdate          BrokerState = "update"
)

type BrokerState string

func (s *BrokerState) UnmarshalJSON(data []byte) error {
	var value string

	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("failed to unmarshal value: %s", data)
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "init":
		*s = BrokerStateInit
	case "down":
		*s = BrokerStateDown
	case "connect":
		*s = BrokerStateConnect
	case "auth":
		*s = BrokerStateAuth
	case "apiversion-query":
		*s = BrokerStateApiVersionQuery
	case "auth-handshake":
		*s = BrokerStateAuthHandshake
	case "up":
		*s = BrokerStateUp
	case "update":
		*s = BrokerStateUpdate
	default:
		*s = BrokerStateUnknown
	}

	return nil
}

func (s BrokerState) Int64() int64 {
	var i int64
	switch s {
	case BrokerStateInit:
		i = 0
	case BrokerStateDown:
		i = 1
	case BrokerStateConnect:
		i = 2
	case BrokerStateAuth:
		i = 3
	case BrokerStateApiVersionQuery:
		i = 4
	case BrokerStateAuthHandshake:
		i = 5
	case BrokerStateUp:
		i = 6
	case BrokerStateUpdate:
		i = 7
	case BrokerStateUnknown:
		fallthrough
	default:
		i = -1
	}

	return i
}

type BrokerTopicPartition struct {
	Topic     string `json:"topic"`
	Partition int64  `json:"partition"`
}

type BrokerStats struct {
	// Broker hostname, port and broker id
	Name string `json:"name"`
	// Broker id (-1 for bootstraps)
	NodeID int64 `json:"nodeid"`
	// Broker hostname
	NodeName string `json:"nodename"`
	// Broker source (learned, configured, internal, logical)
	Source BrokerSource `json:"source"`
	// Broker state (INIT, DOWN, CONNECT, AUTH, APIVERSION_QUERY, AUTH_HANDSHAKE, UP, UPDATE)
	State BrokerState `json:"state"`
	// Time since last broker state change (microseconds)
	StateAge int64 `json:"stateage"`
	// Number of requests awaiting transmission to broker
	RequestsAwaitingTransmission int64 `json:"outbuf_cnt"`
	// Number of messages awaiting transmission to broker
	MessagesAwaitingTransmission int64 `json:"outbuf_msg_cnt"`
	// Number of requests in-flight to broker awaiting response
	InflightRequestsAwaitingResponse int64 `json:"waitresp_cnt"`
	// Number of messages in-flight to broker awaiting response
	InflightMessagesAwaitingResponse int64 `json:"waitresp_msg_cnt"`
	// Total number of requests sent
	RequestsSent int64 `json:"tx"`
	// Total number of bytes sent
	RequestBytesSent int64 `json:"txbytes"`
	// Total number of transmission errors
	RequestErrors int64 `json:"txerrs"`
	// Total number of request retries
	RequestRetries int64 `json:"txretries"`
	// Microseconds since last socket send (or -1 if no sends yet for current connection).
	LastSocketSend int64 `json:"txidle"`
	// Total number of requests timed out
	RequestTimeouts int64 `json:"req_timeouts"`
	// Total number of responses received
	ResponsesReceived int64 `json:"rx"`
	// Total number of bytes received
	ResponseBytesReceived int64 `json:"rxbytes"`
	// Total number of receive errors
	ResponseErrors int64 `json:"rxerrs"`
	// Microseconds since last socket receive (or -1 if no receives yet for current connection).
	LastSocketReceive int64 `json:"rxidle"`
	// Broker thread poll loop wakeups
	Wakeups int64 `json:"wakeups"`
	// Number of connection attempts, including successful and failed, and name resolution failures.
	Connects int64 `json:"connects"`
	// Number of disconnects (triggered by broker, network, load-balancer, etc.).
	Disconnects     int64                           `json:"disconnects"`
	Latency         WindowStats                     `json:"rtt"`
	Throttle        WindowStats                     `json:"throttle"`
	TopicPartitions map[string]BrokerTopicPartition `json:"toppars"`
}
