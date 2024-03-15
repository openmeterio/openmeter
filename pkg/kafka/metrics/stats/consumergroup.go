// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stats

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ConsumerGroupStats struct {
	// Local consumer group handler's state
	State ConsumerGroupState `json:"state"`
	// Time elapsed since last state change (milliseconds)
	StateAge int64 `json:"stateage"`
	// Local consumer group handler's join state
	JoinState ConsumerGroupJoinState `json:"join_state"`
	// Time elapsed since last rebalance (assign or revoke) (milliseconds)
	RebalanceAge int64 `json:"rebalance_age"`
	// Total number of rebalances (assign or revoke)
	RebalanceCount int64 `json:"rebalance_cnt"`
	// Last rebalance reason, or empty string
	RebalanceReason string `json:"rebalance_reason"`
	// Current assignment's partition count
	PartitionAssigned int64 `json:"assignment_size"`
}

const (
	ConsumerGroupStateUnknown ConsumerGroupState = "unknown"
	// ConsumerGroupStateInit means the consumer group is initialization state
	ConsumerGroupStateInit ConsumerGroupState = "init"
	// ConsumerGroupStateTerminated means the Consumer Group has been stopped. This is a final state.
	ConsumerGroupStateTerminated ConsumerGroupState = "term"
	// ConsumerGroupStateQueryCoordinator means the consumer group is querying for group coordinator
	ConsumerGroupStateQueryCoordinator ConsumerGroupState = "query-coord"
	// ConsumerGroupStateWaitCoordinator means that the client is awaiting response for outstanding query
	ConsumerGroupStateWaitCoordinator ConsumerGroupState = "wait-coord"
	// ConsumerGroupStateWaitBroker means that the client is waiting ack from assigned consumer group manager broker thread
	ConsumerGroupStateWaitBroker ConsumerGroupState = "wait-broker"
	// ConsumerGroupStateWaitBrokerTransport means that the client is waiting for manager broker thread to connect to broker
	ConsumerGroupStateWaitBrokerTransport ConsumerGroupState = "wait-broker-transport"
	// ConsumerGroupStateUp means that the coordinator is up and manager is assigned
	ConsumerGroupStateUp ConsumerGroupState = "up"
)

type ConsumerGroupState string

func (s *ConsumerGroupState) UnmarshalJSON(data []byte) error {
	var value string

	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("failed to unmarshal value: %s", data)
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "init":
		*s = ConsumerGroupStateInit
	case "term":
		*s = ConsumerGroupStateTerminated
	case "query-coord":
		*s = ConsumerGroupStateQueryCoordinator
	case "wait-coord":
		*s = ConsumerGroupStateWaitCoordinator
	case "wait-broker":
		*s = ConsumerGroupStateWaitBroker
	case "wait-broker-transport":
		*s = ConsumerGroupStateWaitBrokerTransport
	case "up":
		*s = ConsumerGroupStateUp
	default:
		*s = ConsumerGroupStateUnknown
	}

	return nil
}

func (s ConsumerGroupState) Int64() int64 {
	var i int64

	switch s {
	case ConsumerGroupStateInit:
		i = 0
	case ConsumerGroupStateTerminated:
		i = 1
	case ConsumerGroupStateQueryCoordinator:
		i = 2
	case ConsumerGroupStateWaitCoordinator:
		i = 3
	case ConsumerGroupStateWaitBroker:
		i = 4
	case ConsumerGroupStateWaitBrokerTransport:
		i = 5
	case ConsumerGroupStateUp:
		i = 6
	default:
		i = -1
	}

	return i
}

const (
	ConsumerGroupJoinStateUnknown ConsumerGroupJoinState = "unknown"
	// ConsumerGroupJoinStateInit join or rejoin, possibly with an existing assignment
	ConsumerGroupJoinStateInit ConsumerGroupJoinState = "init"
	// ConsumerGroupJoinStateWaitJoin JoinGroupRequest sent, awaiting response
	ConsumerGroupJoinStateWaitJoin ConsumerGroupJoinState = "wait-join"
	// ConsumerGroupJoinStateWaitMetadata MetadataRequest sent, awaiting response.
	ConsumerGroupJoinStateWaitMetadata ConsumerGroupJoinState = "wait-metadata"
	// ConsumerGroupJoinStateWaitSync SyncGroupRequest sent, awaiting response. Follower only.
	ConsumerGroupJoinStateWaitSync ConsumerGroupJoinState = "wait-sync"
	// ConsumerGroupJoinStateWaitAssignCall waiting for application to call *_assign()
	ConsumerGroupJoinStateWaitAssignCall ConsumerGroupJoinState = "wait-assign-call"
	// ConsumerGroupJoinStateWaitUnassignCall waiting for application to call *_unassign()
	ConsumerGroupJoinStateWaitUnassignCall ConsumerGroupJoinState = "wait-unassign-call"
	// ConsumerGroupJoinStateWaitUnassignToComplete means that the coordinator is up and manager is assigned
	ConsumerGroupJoinStateWaitUnassignToComplete ConsumerGroupJoinState = "wait-unassign-to-complete"
	// ConsumerGroupJoinStateWaitIncrementalUnassignToComplete means that the coordinator is up and manager is assigned
	ConsumerGroupJoinStateWaitIncrementalUnassignToComplete ConsumerGroupJoinState = "wait-incr-unassign-to-complete"
	// ConsumerGroupJoinStateSteady means synchronized and assigned may be an empty assignment
	ConsumerGroupJoinStateSteady ConsumerGroupJoinState = "steady"
)

type ConsumerGroupJoinState string

func (s *ConsumerGroupJoinState) UnmarshalJSON(data []byte) error {
	var value string

	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("failed to unmarshal value: %s", data)
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "init":
		*s = ConsumerGroupJoinStateInit
	case "wait-join":
		*s = ConsumerGroupJoinStateWaitJoin
	case "wait-metadata":
		*s = ConsumerGroupJoinStateWaitMetadata
	case "wait-sync":
		*s = ConsumerGroupJoinStateWaitSync
	case "wait-assign-call":
		*s = ConsumerGroupJoinStateWaitAssignCall
	case "wait-unassign-call":
		*s = ConsumerGroupJoinStateWaitUnassignCall
	case "wait-unassign-to-complete":
		*s = ConsumerGroupJoinStateWaitUnassignToComplete
	case "wait-incr-unassign-to-complete":
		*s = ConsumerGroupJoinStateWaitIncrementalUnassignToComplete
	case "steady":
		*s = ConsumerGroupJoinStateSteady
	default:
		*s = ConsumerGroupJoinStateUnknown
	}

	return nil
}

func (s ConsumerGroupJoinState) Int64() int64 {
	var i int64

	switch s {
	case ConsumerGroupJoinStateInit:
		i = 0
	case ConsumerGroupJoinStateWaitJoin:
		i = 1
	case ConsumerGroupJoinStateWaitMetadata:
		i = 2
	case ConsumerGroupJoinStateWaitSync:
		i = 3
	case ConsumerGroupJoinStateWaitAssignCall:
		i = 4
	case ConsumerGroupJoinStateWaitUnassignCall:
		i = 5
	case ConsumerGroupJoinStateWaitUnassignToComplete:
		i = 6
	case ConsumerGroupJoinStateWaitIncrementalUnassignToComplete:
		i = 7
	case ConsumerGroupJoinStateSteady:
		i = 8
	default:
		i = -1
	}

	return i
}
