package subscriptionitem_test

import (
	"testing"

	subscriptionitem "github.com/openmeterio/openmeter/openmeter/subscription/item"
	"github.com/stretchr/testify/assert"
)

func TestShouldParseKeyFromValue(t *testing.T) {
	t.Run("should parse price key", func(t *testing.T) {
		parsed, err := subscriptionitem.NewItemKeyFromValue("price:123")
		assert.Nil(t, err)
		val, typ := parsed.ByType()
		assert.Equal(t, "123", val)
		assert.Equal(t, subscriptionitem.ContentKeyPrice, typ)
	})

	t.Run("should parse feature key", func(t *testing.T) {
		parsed, err := subscriptionitem.NewItemKeyFromValue("feature:id:abc")
		assert.Nil(t, err)
		val, typ := parsed.ByType()
		assert.Equal(t, "id:abc", val)
		assert.Equal(t, subscriptionitem.ContentKeyFeature, typ)
	})

	t.Run("should fail to parse invalid key for invalid feature", func(t *testing.T) {
		_, err := subscriptionitem.NewItemKeyFromValue("feature:abc")
		assert.NotNil(t, err)
	})

	t.Run("should fail to parse invalid key for invalid price", func(t *testing.T) {
		_, err := subscriptionitem.NewItemKeyFromValue("price")
		assert.NotNil(t, err)
		_, err = subscriptionitem.NewItemKeyFromValue("price:")
		assert.NotNil(t, err)
	})

	t.Run("should fail to parse invalid key for invalid type", func(t *testing.T) {
		_, err := subscriptionitem.NewItemKeyFromValue("abc:123")
		assert.NotNil(t, err)
	})
}
