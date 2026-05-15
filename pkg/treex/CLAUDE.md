# treex

<!-- archie:ai-start -->

> Generic typed tree data structure (Node[T], Tree[T]) providing DFS traversal, cycle detection, shallow/deep clone, and child-swap operations. Used by subscription and product-catalog domains to model hierarchical rate-card and field-descriptor trees where immutable-style node updates via pointer-identity swapping are required.

## Patterns

**Non-nil pointer values only** — NewNode[T](v T) panics at runtime if v is not a non-nil pointer. T must be a pointer type. This enforces the pointer-identity semantics used by SwapChild/RemoveChild comparisons. (`node := treex.NewNode(&myStruct{id: 1})  // T is *myStruct`)
**Seal the graph with NewTree after building** — Build the full node graph using AddChild, then call NewTree(root) once to validate acyclicity. NewTree returns ErrGraphHasCycle for cycles and ErrNodeGraphInvalid for nil children. Do not call NewTree incrementally. (`tr, err := treex.NewTree(rootNode); if errors.Is(err, treex.ErrGraphHasCycle) { ... }`)
**ShallowClone for immutable-style node replacement** — ShallowClone copies the node struct and first-level children slice, reattaching existing children (updating child.parent) to the clone. Use to 'rename' or 'update' a node value without rebuilding the subtree; then call SwapChild or SwapNode on the parent. (`updated := &wrapper{name: "new"}; updated.node = old.node.ShallowClone(); updated.node.SetValue(updated); root.node.SwapChild(old.node, updated.node)`)
**Self-referential node embedding** — Domain structs embed *Node[*Self] and set it in their constructor. The node's value points back to the owning struct, enabling tree traversal to recover the domain object via n.Value(). (`type FieldDescriptor struct { name string; node *treex.Node[*FieldDescriptor] }
func newFD(name string) *FieldDescriptor { fd := &FieldDescriptor{name: name}; fd.node = treex.NewNode(fd); return fd }`)
**DFS stop=true prunes subtree only** — Returning (true, nil) from the DFS callback halts descent into that node's children but sibling subtrees continue walking. Returning a non-nil error aborts the entire walk immediately. (`tr.DFS(func(n *treex.Node[*T]) (bool, error) { if skip(n) { return true, nil }; return false, process(n) })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `node.go` | Node[T] struct with parent/children management; AddChild, RemoveChild, SwapChild, ShallowClone, DeepClone. | SwapChild and RemoveChild use pointer equality (c == child), not value equality — always pass the exact *Node pointer. ShallowClone reattaches first-level children to the clone, mutating child.parent on those immediate children. |
| `tree.go` | Tree[T] wrapper: NewTree validates acyclicity via DFS with visited+onStack maps; exposes DFS, Leafs, SwapNode. | SwapNode on the root replaces t.root directly without re-validation — caller is responsible for the new node being non-nil. Mid-DFS SwapNode requires returning stop=true immediately to avoid following the detached subtree. |
| `errors.go` | Sentinel error vars: ErrRootNodeIsNil, ErrGraphHasCycle, ErrNodeHasNoParentButNotRoot, ErrNodeGraphInvalid. | Use errors.Is for matching — all four are package-level var values, not typed error structs. |

## Anti-Patterns

- Passing value types (non-pointers) to NewNode — panics immediately at construction
- Modifying node.children slice directly instead of using AddChild/RemoveChild/SwapChild — bypasses parent-pointer bookkeeping, corrupts tree invariants
- Continuing DFS traversal after SwapNode inside the callback without returning stop=true — the callback will follow the now-detached old subtree
- Calling NewTree repeatedly as nodes are added rather than building the full graph first — unnecessary O(n) validation on each call

## Decisions

- **Node values are constrained to pointer types, enforced at runtime via reflect.ValueOf().Kind()** — Tree mutations (SwapChild, RemoveChild) rely on pointer identity comparison (c == child); value types would make sibling lookup ambiguous and break the self-referential embedding pattern.
- **Cycle detection runs at NewTree construction time, not incrementally on AddChild** — Allows building node graphs freely and attaching subtrees in arbitrary order before sealing them into a validated Tree; the O(n) DFS validation is paid only once.

## Example: Self-referential embedding + immutable update via ShallowClone + SwapChild

```
import "github.com/openmeterio/openmeter/pkg/treex"

type RateCard struct {
	name string
	node *treex.Node[*RateCard]
}

func newRateCard(name string) *RateCard {
	rc := &RateCard{name: name}
	rc.node = treex.NewNode(rc) // T must be *RateCard
	return rc
}

// Rename a node without rebuilding its subtree:
func renameCard(root *RateCard, old *RateCard, newName string) error {
// ...
```

<!-- archie:ai-end -->
