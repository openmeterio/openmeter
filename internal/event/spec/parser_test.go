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

package spec_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/event/spec"
)

type event struct {
	Namespace string
}

func (e event) Spec() *spec.EventTypeSpec {
	return &spec.EventTypeSpec{
		Subsystem: "subsys",
		Name:      "test",
		Version:   "v1",
	}
}

var errNamespaceIsRequired = errors.New("namespace is required")

func (e event) Validate() error {
	if e.Namespace == "" {
		return errNamespaceIsRequired
	}
	return nil
}

func TestParserSanity(t *testing.T) {
	cloudEvent, err := spec.NewCloudEvent(
		spec.EventSpec{
			ID:     "test",
			Source: "somesource",

			Subject: spec.ComposeResourcePath("default", "subject", "ID"),
		},
		event{
			Namespace: "test",
		})

	assert.NoError(t, err)
	assert.Equal(t, "io.openmeter.subsys.v1.test", cloudEvent.Type())
	assert.Equal(t, "//openmeter.io/namespace/default/subject/ID", cloudEvent.Subject())
	assert.Equal(t, "somesource", cloudEvent.Source())

	// parsing
	parsedEvent, err := spec.ParseCloudEvent[event](cloudEvent)
	assert.NoError(t, err)
	assert.Equal(t, "test", parsedEvent.Namespace)

	// validation support
	_, err = spec.NewCloudEvent(
		spec.EventSpec{
			ID:     "test",
			Source: "somesource",

			Subject: spec.ComposeResourcePath("default", "subject", "ID"),
		},
		event{},
	)

	assert.Error(t, err)
	assert.Equal(t, errNamespaceIsRequired, err)

	// ID autogeneration
	cloudEvent, err = spec.NewCloudEvent(
		spec.EventSpec{
			Source: "somesource",
		},
		event{
			Namespace: "test",
		})

	assert.NoError(t, err)
	assert.NotEmpty(t, cloudEvent.ID())
}
