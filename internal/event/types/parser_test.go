package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/event/types"
)

type event struct {
	Namespace string
}

func (e event) Spec() *types.EventTypeSpec {
	return &types.EventTypeSpec{
		Subsystem:   "subsys",
		Name:        "test",
		SpecVersion: "1.0",
		Version:     "v1",
		SubjectKind: "testentity",
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
	cloudEvent, err := types.NewCloudEvent(
		types.EventSpec{
			ID:     "test",
			Source: "somesource",

			Namespace: "test",
			SubjectID: "subjectentityid",
		},
		event{
			Namespace: "test",
		})

	assert.NoError(t, err)
	assert.Equal(t, "openmeter.subsys.v1.test", cloudEvent.Type())
	assert.Equal(t, "/namespace/test/testentity/subjectentityid", cloudEvent.Subject())

	// parsing
	parsedEvent, err := types.ParseCloudEvent[event](cloudEvent)
	assert.NoError(t, err)
	assert.Equal(t, "test", parsedEvent.Namespace)

	// validation support
	_, err = types.NewCloudEvent(
		types.EventSpec{
			ID:     "test",
			Source: "somesource",

			Namespace: "test",
			SubjectID: "subjectentityid",
		},
		event{},
	)

	assert.Error(t, err)
	assert.Equal(t, errNamespaceIsRequired, err)
}
