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

package grant

import (
	"errors"

	"github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/event/spec"
)

const (
	EventSubsystem spec.EventSubsystem = "credit"
)

const (
	grantCreatedEventName spec.EventName = "grant.created"
	grantVoidedEventName  spec.EventName = "grant.voided"
)

type grantEvent struct {
	Grant

	Subject models.SubjectKeyAndID `json:"subject"`
	// Namespace from Grant cannot be used as it will never be serialized
	Namespace models.NamespaceID `json:"namespace"`
}

func (g grantEvent) Validate() error {
	// Basic sanity on grant
	if g.Grant.ID == "" {
		return errors.New("GrantID must be set")
	}

	if g.Grant.OwnerID == "" {
		return errors.New("GrantOwnerID must be set")
	}

	if err := g.Subject.Validate(); err != nil {
		return err
	}

	if err := g.Namespace.Validate(); err != nil {
		return err
	}

	return nil
}

type CreatedEvent grantEvent

var grantCreatedEventSpec = spec.EventTypeSpec{
	Subsystem: EventSubsystem,
	Name:      grantCreatedEventName,
	Version:   "v1",
}

func (e CreatedEvent) Spec() *spec.EventTypeSpec {
	return &grantCreatedEventSpec
}

func (e CreatedEvent) Validate() error {
	return grantEvent(e).Validate()
}

type VoidedEvent grantEvent

var grantVoidedEventSpec = spec.EventTypeSpec{
	Subsystem: EventSubsystem,
	Name:      grantVoidedEventName,
	Version:   "v1",
}

func (e VoidedEvent) Spec() *spec.EventTypeSpec {
	return &grantVoidedEventSpec
}

func (e VoidedEvent) Validate() error {
	return grantEvent(e).Validate()
}
