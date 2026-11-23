package entitlement

// CurrentValueOperation distinguishes a caller's read from a worker recalculation
// for performance comparisons. Both evaluate the entitlement at the supplied time.
type CurrentValueOperation uint8

const (
	CurrentValueOperationRead CurrentValueOperation = iota + 1
	CurrentValueOperationCalculate
)
