package productcatalog

import "github.com/openmeterio/openmeter/openmeter/productcatalog/unitconfig"

// The UnitConfig domain type lives in the leaf package
// openmeter/productcatalog/unitconfig so that packages below productcatalog in the
// import graph (entitlement, credit/grant, credit/balance) can use the typed value
// without an import cycle. These aliases keep productcatalog.UnitConfig and its
// enums working unchanged for every existing caller; the type alias carries all
// methods (Apply, Validate, Equal, Clone), so only the enum constants need
// re-exporting.
type (
	UnitConfig             = unitconfig.UnitConfig
	UnitConfigOperation    = unitconfig.UnitConfigOperation
	UnitConfigRoundingMode = unitconfig.UnitConfigRoundingMode
)

const (
	UnitConfigOperationMultiply = unitconfig.UnitConfigOperationMultiply
	UnitConfigOperationDivide   = unitconfig.UnitConfigOperationDivide

	UnitConfigRoundingModeNone    = unitconfig.UnitConfigRoundingModeNone
	UnitConfigRoundingModeCeiling = unitconfig.UnitConfigRoundingModeCeiling
	UnitConfigRoundingModeFloor   = unitconfig.UnitConfigRoundingModeFloor
	UnitConfigRoundingModeHalfUp  = unitconfig.UnitConfigRoundingModeHalfUp
)
