package override

// UniquelyComparable means the interface can be compared, diffed, etc... by a unique self-referential identifier
// returned by UniqBy.
//
// For example, if you were to override a RateCard in a subscription, both the old version of the RateCard and the
// new version of the RateCard would have the same UniqBy() value, while their other properties would differ.
type UniquelyComparable interface {
	// UniqBy returns a unique identifier for the interface by which it can be diffed
	UniqBy() string
}

type OverrideAction string

const (
	OverrideActionAdd    OverrideAction = "add"
	OverrideActionRemove OverrideAction = "remove"
)

type Override[T UniquelyComparable] struct {
	// Action is the action to take on the value
	Action OverrideAction

	// Value is the value to apply the action to.
	// Value.UniqBy() is used to determine which value it should be applied to.
	Value T
}

func ApplyOverrides[T UniquelyComparable](base []T, overrides []Override[T]) []T {
	// Create a map of the base values by their UniqBy value
	baseMap := make(map[string]T)
	for _, v := range base {
		baseMap[v.UniqBy()] = v
	}

	// Apply the overrides
	for _, o := range overrides {
		switch o.Action {
		case OverrideActionAdd:
			baseMap[o.Value.UniqBy()] = o.Value
		case OverrideActionRemove:
			delete(baseMap, o.Value.UniqBy())
		}
	}

	// Convert the map back to a slice
	result := make([]T, 0, len(baseMap))
	for _, v := range baseMap {
		result = append(result, v)
	}

	return result
}
