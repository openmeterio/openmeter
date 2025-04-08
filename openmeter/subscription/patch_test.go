package subscription_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/subscription"
)

func TestPathRelations(t *testing.T) {
	t.Run("Should determine identity as parent path", func(t *testing.T) {
		path := subscription.SpecPath("/phases/0/items/0")
		assert.True(t, path.IsParentOf(path))
	})

	t.Run("Should find parent path", func(t *testing.T) {
		path := subscription.SpecPath("/phases/0")
		child := subscription.SpecPath("/phases/0/items/0")
		assert.True(t, path.IsParentOf(child))
	})

	t.Run("Should not find parent path when child", func(t *testing.T) {
		path := subscription.SpecPath("/phases/0/items/0")
		invalidChild := subscription.SpecPath("/phases/0")
		assert.False(t, path.IsParentOf(invalidChild))
	})

	t.Run("Should not find parent if completely different", func(t *testing.T) {
		path := subscription.SpecPath("/phases/0/items/0")
		invalidChild := subscription.SpecPath("/phases/1/items/0")
		assert.False(t, path.IsParentOf(invalidChild))
	})

	t.Run("Should not find parent if completely different", func(t *testing.T) {
		path := subscription.SpecPath("/phases/0")
		invalidChild := subscription.SpecPath("/phases/1/items/0")
		assert.False(t, path.IsParentOf(invalidChild))
	})
}
