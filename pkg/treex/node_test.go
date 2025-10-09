package treex

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nodeVal struct{ id int }

func TestNewNode_RequiresNonNilPointer(t *testing.T) {
	// Non-pointer should panic
	assert.Panics(t, func() { NewNode(nodeVal{id: 1}) })
	// Nil pointer should panic
	var nilPtr *nodeVal
	assert.Panics(t, func() { NewNode(nilPtr) })
	// Valid non-nil pointer should not panic
	assert.NotPanics(t, func() { _ = NewNode(&nodeVal{id: 1}) })
}

func TestNode_ValueAndSetValue(t *testing.T) {
	v := &nodeVal{id: 1}
	n := NewNode(v)
	assert.Equal(t, v, n.Value())

	v2 := &nodeVal{id: 2}
	n.SetValue(v2)
	assert.Equal(t, v2, n.Value())
}

func TestNode_AddChild_SetsParentAndChildren(t *testing.T) {
	p := NewNode(&nodeVal{id: 1})
	c := NewNode(&nodeVal{id: 2})

	assert.Nil(t, c.Parent())
	assert.Empty(t, p.Children())

	p.AddChild(c)

	require.Len(t, p.Children(), 1)
	assert.Equal(t, c, p.Children()[0])
	assert.Equal(t, p, c.Parent())
}

func TestNode_RemoveChild(t *testing.T) {
	p := NewNode(&nodeVal{id: 1})
	c1 := NewNode(&nodeVal{id: 2})
	c2 := NewNode(&nodeVal{id: 3})

	p.AddChild(c1)
	p.AddChild(c2)
	require.Len(t, p.Children(), 2)

	err := p.RemoveChild(c1)
	require.NoError(t, err)
	require.Len(t, p.Children(), 1)
	assert.Equal(t, c2, p.Children()[0])
	assert.Nil(t, c1.Parent())

	// removing non-child returns error
	nc := NewNode(&nodeVal{id: 4})
	err = p.RemoveChild(nc)
	require.Error(t, err)
}

func TestNode_SwapChild(t *testing.T) {
	p := NewNode(&nodeVal{id: 1})
	c1 := NewNode(&nodeVal{id: 2})
	c2 := NewNode(&nodeVal{id: 3})

	p.AddChild(c1)
	require.Len(t, p.Children(), 1)
	assert.Equal(t, p, c1.Parent())
	assert.Nil(t, c2.Parent())

	err := p.SwapChild(c1, c2)
	require.NoError(t, err)
	require.Len(t, p.Children(), 1)
	assert.Equal(t, c2, p.Children()[0])
	assert.Equal(t, p, c2.Parent())
	assert.Nil(t, c1.Parent())

	// swapping non-child returns error
	err = p.SwapChild(c1, NewNode(&nodeVal{id: 4}))
	require.Error(t, err)
}

func TestNode_IsLeafAndIsRoot(t *testing.T) {
	p := NewNode(&nodeVal{id: 1})
	c := NewNode(&nodeVal{id: 2})

	assert.True(t, p.IsLeaf())
	assert.True(t, p.IsRoot())

	p.AddChild(c)
	assert.False(t, p.IsLeaf())
	assert.True(t, p.IsRoot())
	assert.True(t, c.IsLeaf())
	assert.False(t, c.IsRoot())
}

func TestNode_ShallowClone(t *testing.T) {
	p := NewNode(&nodeVal{id: 1})
	c1 := NewNode(&nodeVal{id: 2})
	c2 := NewNode(&nodeVal{id: 3})

	p.AddChild(c1)
	p.AddChild(c2)

	clone := p.ShallowClone()
	// Different pointer
	assert.NotSame(t, p, clone)
	// Value pointer equal
	assert.Equal(t, p.Value(), clone.Value())
	// Parent pointer equal
	assert.Equal(t, p.Parent(), clone.Parent())
	// Children slice content equal but not the same slice
	require.Len(t, clone.Children(), 2)
	assert.Equal(t, p.Children(), clone.Children())
	if len(p.Children()) > 0 {
		assert.NotSame(t, &p.children[0], &clone.children[0])
	}
	// Child parent pointers should be updated to the clone
	assert.Equal(t, clone, c1.Parent())
	assert.Equal(t, clone, c2.Parent())
}

func TestNode_DeepClone(t *testing.T) {
	// Build a small tree p -> (c1, c2), and c1 -> (g1)
	p := NewNode(&nodeVal{id: 1})
	c1 := NewNode(&nodeVal{id: 2})
	c2 := NewNode(&nodeVal{id: 3})
	g1 := NewNode(&nodeVal{id: 4})
	p.AddChild(c1)
	p.AddChild(c2)
	c1.AddChild(g1)

	clone := p.DeepClone()
	// root clone is detached
	assert.Nil(t, clone.Parent())
	assert.NotSame(t, p, clone)
	assert.Equal(t, p.Value(), clone.Value())

	// structure preserved
	require.Len(t, clone.Children(), 2)
	cc1 := clone.Children()[0]
	cc2 := clone.Children()[1]
	assert.Equal(t, clone, cc1.Parent())
	assert.Equal(t, clone, cc2.Parent())
	assert.Equal(t, c1.Value(), cc1.Value())
	assert.Equal(t, c2.Value(), cc2.Value())

	require.Len(t, cc1.Children(), 1)
	cg1 := cc1.Children()[0]
	assert.Equal(t, cc1, cg1.Parent())
	assert.Equal(t, g1.Value(), cg1.Value())

	// originals remain unchanged
	assert.Equal(t, p, c1.Parent())
	assert.Equal(t, p, c2.Parent())
	assert.Equal(t, c1, g1.Parent())
}
