package models

import (
	"encoding/json"
	"strings"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/treex"
)

type FieldDescriptor struct {
	field string
	exp   FieldExpression
	attrs Attributes

	node *treex.Node[*FieldDescriptor]
}

func (s FieldDescriptor) Clone() *FieldDescriptor {
	desc := &FieldDescriptor{
		field: s.field,
		exp:   s.exp,
		attrs: s.attrs.Clone(),
		node:  s.node.ShallowClone(),
	}
	desc.node.SetValue(desc)

	return desc
}

func (s FieldDescriptor) WithExpression(exp FieldExpression) *FieldDescriptor {
	s.exp = exp
	s.node = s.node.ShallowClone()
	s.node.SetValue(&s)

	return &s
}

func (s FieldDescriptor) WithAttributes(attrs Attributes) *FieldDescriptor {
	curr := s.attrs

	if curr == nil {
		curr = make(Attributes)
	}

	s.attrs = curr.Merge(attrs)
	s.node = s.node.ShallowClone()
	s.node.SetValue(&s)

	return &s
}

func (p FieldDescriptor) GetAttributes() Attributes {
	return p.attrs
}

func (p FieldDescriptor) WithPrefix(_prefix *FieldDescriptor) *FieldDescriptor {
	var prefix *FieldDescriptor
	if _prefix != nil {
		prefix = _prefix.Clone()
	}

	p.node = p.node.ShallowClone()
	p.node.SetValue(&p)

	if prefix == nil {
		return NewFieldSelectorGroup(&p)
	}

	return NewFieldSelectorGroup(prefix, &p)
}

func (p *FieldDescriptor) MarshalJSON() ([]byte, error) {
	if p == nil {
		return json.Marshal("")
	}

	return json.Marshal(p.JSONPath())
}

func (p *FieldDescriptor) String() string {
	if p == nil {
		return ""
	}

	b := strings.Builder{}

	// We'll use a DFS traversal to build the string
	if err := p.Tree(func(t *FieldDescriptorTree) error {
		leafCount := 0

		return t.DFS(func(n *treex.Node[*FieldDescriptor]) (bool, error) {
			// Only leaf nodes (childrenless segments with field names) make up the path, the rest of the graph is just hierarchical information
			if n.IsLeaf() {
				desc := n.Value()

				if leafCount > 0 {
					b.WriteString(".")
				}

				b.WriteString(n.Value().field)

				if desc.exp != nil {
					if exp := desc.exp.String(); exp != "" {
						b.WriteString("[")
						b.WriteString(exp)
						b.WriteString("]")
					}
				}

				leafCount++
			}

			return false, nil
		})
	}); err != nil {
		return ""
	}

	return b.String()
}

func (p *FieldDescriptor) JSONPath() string {
	if p == nil {
		return ""
	}

	b := strings.Builder{}

	// Tree.Root().IsRoot() is always true so we have to make this assertion here
	if p.node.IsRoot() {
		b.WriteString("$")
		b.WriteString(".")
	}

	if err := p.Tree(func(t *FieldDescriptorTree) error {
		leafCount := 0

		return t.DFS(func(n *treex.Node[*FieldDescriptor]) (bool, error) {
			// Only leaf nodes (childrenless segments with field names) make up the path, the rest of the graph is just hierarchical information
			if n.IsLeaf() {
				desc := n.Value()

				if leafCount > 0 {
					b.WriteString(".")
				}

				b.WriteString(desc.field)

				if desc.exp != nil {
					expOpen := "["
					expClose := "]"

					if desc.exp.IsCondition() {
						expOpen = "[?("
						expClose = ")]"
					}

					if exp := desc.exp.JSONPathExpression(); exp != "" {
						b.WriteString(expOpen)
						b.WriteString(exp)
						b.WriteString(expClose)
					}
				}

				leafCount++
			}

			return false, nil
		})
	}); err != nil {
		return ""
	}

	return b.String()
}

// FieldDescriptorTree is a wrapper around treex.Tree[*FieldDescriptor]
// with methods meaningful for a FieldDescriptor
type FieldDescriptorTree struct {
	*treex.Tree[*FieldDescriptor]
}

// If called while walking the tree, you MUST start backtracking (return true)
// otherwise you'll keep walking the detached subtree!
func (t *FieldDescriptorTree) Swap(old, new *FieldDescriptor) error {
	return t.Tree.SwapNode(old.node, new.node)
}

func (t *FieldDescriptorTree) Leafs() []*FieldDescriptor {
	return lo.Map(t.Tree.Leafs(), func(n *treex.Node[*FieldDescriptor], _ int) *FieldDescriptor {
		return n.Value()
	})
}

// Tree returns a treex.Tree[*FieldDescriptor] from the FieldDescriptor so it can be traversed
func (s FieldDescriptor) Tree(cb func(t *FieldDescriptorTree) error) error {
	tree, err := treex.NewTree(s.node)
	if err != nil {
		return err
	}

	return cb(&FieldDescriptorTree{Tree: tree})
}

func newFieldDescriptor() *FieldDescriptor {
	desc := &FieldDescriptor{}
	desc.node = treex.NewNode(desc)
	return desc
}

func NewFieldSelector(field string) *FieldDescriptor {
	desc := newFieldDescriptor()
	desc.field = field

	return desc
}

func NewFieldSelectorGroup(selectors ...*FieldDescriptor) *FieldDescriptor {
	selectors = lo.Filter(selectors, func(item *FieldDescriptor, _ int) bool {
		return item != nil
	})

	if len(selectors) == 0 {
		return nil
	}

	desc := newFieldDescriptor()
	for i := range selectors {
		nd := selectors[i].node
		desc.node.AddChild(nd)
	}

	return desc
}
