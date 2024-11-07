package plan

type Validator interface {
	// Validate returns an error if the instance of the Validator is invalid.
	Validate() error
}

type Equaler[T any] interface {
	// StrictEqual must return true in case all attributes of T are strictly equal.
	StrictEqual(T) bool

	// Equal is a relaxed version of StrictEqual where it is allowed to return true in case a subset of the attributes are equal.
	// Typically, you want to use Equal to compare two instances of T where one or both missing some managed fields which are
	// not required for comparison. Like comparing two instances of T before and after stored in data layer which assigns
	// managed fields like timestamps, unique identifiers, etc.
	Equal(T) bool
}
