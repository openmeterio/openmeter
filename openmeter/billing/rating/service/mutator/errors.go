package mutator

import "errors"

var (
	// ErrUnitConfigDisabled is returned by ForbidUnitConfig when a line carries a
	// unit_config while the unitConfig feature is disabled. Rating the raw quantity
	// would silently under/over-bill, so the inconsistency is surfaced instead.
	ErrUnitConfigDisabled = errors.New("unit_config is set on the line but the unitConfig feature is disabled")

	// ErrUnitConfigUnsupportedPrice is returned by UnitConfig when a unit_config rides
	// on a price type that cannot convert (flat/package/dynamic) or on a line with no
	// price. The authoring validator blocks this, so reaching rating means inconsistent
	// data that must not be billed as raw.
	ErrUnitConfigUnsupportedPrice = errors.New("unit_config is set on a price that does not support unit conversion")
)
