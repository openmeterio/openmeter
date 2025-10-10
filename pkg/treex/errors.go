package treex

import "errors"

var (
	ErrRootNodeIsNil             = errors.New("root node cannot be nil")
	ErrGraphHasCycle             = errors.New("graph cannot contain cycles")
	ErrNodeHasNoParentButNotRoot = errors.New("node has no parent but is not the root")

	// catch-all error for invalid nodes or graph structure
	ErrNodeGraphInvalid = errors.New("invalid graph structure")
)
