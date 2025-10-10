package treex

import (
	"errors"
	"reflect"

	"github.com/samber/lo"
)

func NewNode[T any](value T) *Node[T] {
	if reflect.ValueOf(value).Kind() != reflect.Ptr {
		panic("Node value has to be a pointer")
	}

	if reflect.ValueOf(value).IsNil() {
		panic("Node value has to be a non-nil pointer")
	}

	return &Node[T]{value: value}
}

type Node[T any] struct {
	value    T
	parent   *Node[T]
	children []*Node[T]
}

// ShallowClone creates a new node with the same value and parent and a copied
// first-level children slice. It reattaches the existing first-level children
// to the cloned node (i.e., updates child.parent to point to the clone) but
// does not traverse deeper. This is useful for immutable-style updates when
// replacing a node while keeping its immediate subtree.
func (n *Node[T]) ShallowClone() *Node[T] {
	children := make([]*Node[T], len(n.children))
	copy(children, n.children)

	clone := &Node[T]{
		value:    n.value,
		parent:   n.parent,
		children: children,
	}

	for _, child := range children {
		if child != nil {
			child.parent = clone
		}
	}

	return clone
}

// DeepClone creates a deep copy of the node and all its descendants.
// The returned clone is fully detached (parent pointers set appropriately
// within the cloned subtree, with the top-level node having nil parent).
func (n *Node[T]) DeepClone() *Node[T] {
	if n == nil {
		return nil
	}

	// clone the current node without parent and without children for now
	clone := &Node[T]{
		value:  n.value,
		parent: nil,
	}
	// recursively clone children and attach
	for _, child := range n.children {
		if child == nil {
			continue
		}
		childClone := child.DeepClone()
		clone.AddChild(childClone)
	}
	return clone
}

func (n *Node[T]) SetValue(value T) {
	n.value = value
}

func (n *Node[T]) Value() T {
	return n.value
}

func (n *Node[T]) Parent() *Node[T] {
	return n.parent
}

func (n *Node[T]) Children() []*Node[T] {
	return n.children
}

func (n *Node[T]) AddChild(child *Node[T]) {
	n.children = append(n.children, child)
	child.parent = n
}

func (n *Node[T]) RemoveChild(child *Node[T]) error {
	_, ok := lo.Find(n.children, func(c *Node[T]) bool {
		return c == child
	})

	if !ok {
		return errors.New("child not found")
	}

	n.children = lo.Filter(n.children, func(c *Node[T], _ int) bool {
		return c != child
	})

	child.parent = nil

	return nil
}

func (n *Node[T]) SwapChild(old *Node[T], new *Node[T]) error {
	_, idx, ok := lo.FindIndexOf(n.children, func(c *Node[T]) bool {
		return c == old
	})

	if !ok {
		return errors.New("child not found")
	}

	n.children[idx] = new
	old.parent = nil
	new.parent = n

	return nil
}

func (n *Node[T]) IsLeaf() bool {
	return len(n.children) == 0
}

func (n *Node[T]) IsRoot() bool {
	return n.parent == nil
}
