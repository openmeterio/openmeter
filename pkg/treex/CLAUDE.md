# treex

<!-- archie:ai-start -->

> Generic typed tree (Node[T], Tree[T]) with DFS traversal, cycle detection, shallow/deep clone, and child-swap. Used by subscription and product-catalog to model hierarchical rate-card and field-descriptor trees where immutable-style updates via pointer-identity swapping are required.

## Patterns

**Non-nil pointer values only** — NewNode[T](v) panics if v is not a non-nil pointer; T must be a pointer type. This enforces the pointer-identity semantics used by SwapChild/RemoveChild. (`node := treex.NewNode(&myStruct{id: 1}) // T is *myStruct`)
**Seal the graph with NewTree after building** — Build the full graph with AddChild, then call NewTree(root) once to validate acyclicity (ErrGraphHasCycle / ErrNodeGraphInvalid). Do not call NewTree incrementally. (`tr, err := treex.NewTree(rootNode); if errors.Is(err, treex.ErrGraphHasCycle) { ... }`)
**ShallowClone for immutable-style node replacement** — ShallowClone copies the node struct and first-level children, reattaching them to the clone; use to 'update' a node value then SwapChild on the parent. (`updated := &wrapper{name:"new"}; updated.node = old.node.ShallowClone(); updated.node.SetValue(updated); root.node.SwapChild(old.node, updated.node)`)
**Self-referential node embedding** — Domain structs embed *Node[*Self] set in their constructor; the node's value points back so traversal can recover the domain object via n.Value(). (`func newFD(name string) *FieldDescriptor { fd := &FieldDescriptor{name:name}; fd.node = treex.NewNode(fd); return fd }`)
**DFS stop=true prunes subtree only** — Returning (true, nil) halts descent into that node's children but siblings continue; a non-nil error aborts the whole walk. (`tr.DFS(func(n *treex.Node[*T]) (bool, error) { if skip(n) { return true, nil }; return false, process(n) })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `node.go` | Node[T] with parent/children management: AddChild, RemoveChild, SwapChild, ShallowClone, DeepClone. | SwapChild/RemoveChild use pointer equality (c == child) — pass the exact *Node. ShallowClone reattaches first-level children, mutating their parent pointers. |
| `tree.go` | Tree[T] wrapper: NewTree validates acyclicity via DFS with visited+onStack; exposes DFS, Leafs, SwapNode. | SwapNode on root replaces t.root without re-validation; mid-DFS SwapNode requires returning stop=true to avoid following the detached subtree. |
| `errors.go` | Sentinel error vars: ErrRootNodeIsNil, ErrGraphHasCycle, ErrNodeHasNoParentButNotRoot, ErrNodeGraphInvalid. | Match with errors.Is — these are package-level var values, not typed error structs. |

## Anti-Patterns

- Passing value (non-pointer) types to NewNode — panics at construction.
- Mutating node.children directly instead of AddChild/RemoveChild/SwapChild — corrupts parent-pointer invariants.
- Continuing DFS after SwapNode inside the callback without returning stop=true — follows the detached old subtree.
- Calling NewTree repeatedly while adding nodes instead of building the full graph first.

## Decisions

- **Node values constrained to pointer types, enforced at runtime via reflect.Kind()** — Tree mutations rely on pointer-identity comparison; value types make sibling lookup ambiguous and break self-referential embedding.
- **Cycle detection runs at NewTree, not incrementally on AddChild** — Lets subtrees be attached in arbitrary order before sealing; the O(n) validation is paid once.

## Example: Self-referential embedding + immutable update via ShallowClone + SwapChild

```
import "github.com/openmeterio/openmeter/pkg/treex"

type RateCard struct { name string; node *treex.Node[*RateCard] }
func newRateCard(name string) *RateCard { rc := &RateCard{name:name}; rc.node = treex.NewNode(rc); return rc }
// Rename without rebuilding subtree: ShallowClone old.node, SetValue, then parent.node.SwapChild(old.node, clone)
```

<!-- archie:ai-end -->
