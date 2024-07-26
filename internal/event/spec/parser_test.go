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
		Subsystem:   "subsys",
		Name:        "test",
		SpecVersion: "1.0",
		Version:     "v1",
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
	assert.Equal(t, "openmeter.subsys.v1.test", cloudEvent.Type())
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
