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

package entitlement

import (
	"errors"

	"github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/event/spec"
)

const (
	EventSubsystem spec.EventSubsystem = "entitlement"
)

const (
	entitlementCreatedEventName spec.EventName = "entitlement.created"
	entitlementDeletedEventName spec.EventName = "entitlement.deleted"
)

type entitlementEvent struct {
	Entitlement
	Namespace models.NamespaceID `json:"namespace"`
}

func (e entitlementEvent) Validate() error {
	if e.ID == "" {
		return errors.New("ID must not be empty")
	}

	if e.SubjectKey == "" {
		return errors.New("subjectKey must not be empty")
	}

	if err := e.Namespace.Validate(); err != nil {
		return err
	}

	return nil
}

type EntitlementCreatedEvent entitlementEvent

var entitlementCreatedEventSpec = spec.EventTypeSpec{
	Subsystem: EventSubsystem,
	Name:      entitlementCreatedEventName,
	Version:   "v1",
}

func (e EntitlementCreatedEvent) Spec() *spec.EventTypeSpec {
	return &entitlementCreatedEventSpec
}

func (e EntitlementCreatedEvent) Validate() error {
	return entitlementEvent(e).Validate()
}

type EntitlementDeletedEvent entitlementEvent

var entitlementDeletedEventSpec = spec.EventTypeSpec{
	Subsystem: EventSubsystem,
	Name:      entitlementDeletedEventName,
	Version:   "v1",
}

func (e EntitlementDeletedEvent) Spec() *spec.EventTypeSpec {
	return &entitlementDeletedEventSpec
}

func (e EntitlementDeletedEvent) Validate() error {
	return entitlementEvent(e).Validate()
}
