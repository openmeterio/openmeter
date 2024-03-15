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

package spec

import (
	"fmt"
	"time"
)

type (
	EventSubsystem   string
	EventName        string
	EventVersion     string
	EventSubjectKind string
	EventSpecVersion string
)

type EventTypeSpec struct {
	// Subsystem defines which connector/component is responsible for the event (e.g. ingest, entitlements, etc)
	Subsystem EventSubsystem

	// Type is the actual event type (e.g. ingestion, flush, etc)
	Name EventName

	// Version is the version of the event (e.g. v1, v2, etc)
	Version EventVersion

	// cloudEventType is the actual cloud event type, so that we don't have the calculate it
	// for each message
	cloudEventType string
}

func (s *EventTypeSpec) Type() string {
	if s.cloudEventType != "" {
		return s.cloudEventType
	}
	s.cloudEventType = fmt.Sprintf("io.openmeter.%s.%s.%s", s.Subsystem, s.Version, s.Name)
	return s.cloudEventType
}

type EventSpec struct {
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
