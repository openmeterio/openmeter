package datetime

const (
	// layoutTZName is a placeholder timezone name used in RFC9557 format layouts.
	// This gets replaced with the actual timezone name during formatting.
	layoutTZName = "Europe/Budapest"
)

// RFC9557 format layouts support timezone information in square brackets.
const (
	RFC9557Layout      = "2006-01-02T15:04:05Z07:00[" + layoutTZName + "]"
	RFC9557MilliLayout = "2006-01-02T15:04:05.999Z07:00[" + layoutTZName + "]"
	RFC9557MicroLayout = "2006-01-02T15:04:05.999999Z07:00[" + layoutTZName + "]"
	RFC9557NanoLayout  = "2006-01-02T15:04:05.999999999Z07:00[" + layoutTZName + "]"
)

// ISO8601 format layouts for standard timestamp parsing.
const (
	ISO8601Layout      = "2006-01-02T15:04:05-07:00"
	ISO8601MilliLayout = "2006-01-02T15:04:05.999-07:00"
	ISO8601MicroLayout = "2006-01-02T15:04:05.999999-07:00"
	ISO8601NanoLayout  = "2006-01-02T15:04:05.999999999-07:00"
)

// ISO8601 Zulu (UTC) format layouts.
const (
	ISO8601ZuluLayout      = "2006-01-02T15:04:05Z"
	ISO8601ZuluMilliLayout = "2006-01-02T15:04:05.999Z"
	ISO8601ZuluMicroLayout = "2006-01-02T15:04:05.999999Z"
	ISO8601ZuluNanoLayout  = "2006-01-02T15:04:05.999999999Z"
)
