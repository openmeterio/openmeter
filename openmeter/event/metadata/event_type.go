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

package metadata

import (
	"fmt"
	"time"
)

type (
	EventSubsystem string
	EventName      string
	EventVersion   string
)

type EventType struct {
	// Subsystem defines which connector/component is responsible for the event (e.g. ingest, entitlements, etc)
	Subsystem EventSubsystem

	// Type is the actual event type (e.g. ingestion, flush, etc)
	Name EventName

	// Version is the version of the event (e.g. v1, v2, etc)
	Version EventVersion
}

func (s *EventType) EventName() string {
	return fmt.Sprintf("io.openmeter.%s.%s.%s", s.Subsystem, s.Version, s.Name)
}

func (s *EventType) VersionSubsystem() string {
	return fmt.Sprintf("io.openmeter.%s.%s", s.Subsystem, s.Version)
}

func GetEventName(spec EventType) string {
	return spec.EventName()
}

type EventMetadata struct {
	// ID of the event
	ID string

	// Time specifies when the event occurred
	Time time.Time

	// Subject meta
	// Examples for source and subject pairs
	//  grant:
	//      source: //openmeter.io/namespace/<id>/entitlement/<id>/grant/<id>
	//      subject: //openmeter.io/namespace/<id>/subject/<subjectID>
	//
	//  entitlement:
	//      source: //openmeter.io/namespace/<id>/entitlement/<id>
	//      subject: //openmeter.io/namespace/<id>/subject/<subjectID>
	//
	//  ingest:
	//      source: //openmeter.io/namespace/<id>/event
	//      subject: //openmeter.io/namespace/<id>/subject/<subjectID>
	Subject string
	Source  string
}
