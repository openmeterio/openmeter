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

package meteredentitlement

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystem metadata.EventSubsystem = "meteredEntitlement"
)

type EntitlementResetEvent struct {
	EntitlementID string                 `json:"entitlementId"`
	Namespace     models.NamespaceID     `json:"namespace"`
	Subject       models.SubjectKeyAndID `json:"subject"`
	ResetAt       time.Time              `json:"resetAt"`
	RetainAnchor  bool                   `json:"retainAnchor"`
}

var (
	_ marshaler.Event = EntitlementResetEvent{}

	resetEntitlementEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "entitlement.reset",
		Version:   "v1",
	})
)

func (e EntitlementResetEvent) EventName() string {
	return resetEntitlementEventName
}

func (e EntitlementResetEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntityEntitlement, e.EntitlementID),
		Subject: metadata.ComposeResourcePath(e.Namespace.ID, metadata.EntitySubjectKey, e.Subject.Key),
	}
}

func (e EntitlementResetEvent) Validate() error {
	if e.EntitlementID == "" {
		return errors.New("entitlementID must be set")
	}

	if err := e.Namespace.Validate(); err != nil {
		return err
	}

	if err := e.Subject.Validate(); err != nil {
		return err
	}

	if e.ResetAt.IsZero() {
		return errors.New("resetAt must be set")
	}
	return nil
}
