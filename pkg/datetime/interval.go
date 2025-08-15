package datetime

var (
	DurationSecond ISODuration = NewISODuration(0, 0, 0, 0, 0, 0, 1)
	DurationMinute ISODuration = NewISODuration(0, 0, 0, 0, 0, 1, 0)
	DurationHour   ISODuration = NewISODuration(0, 0, 0, 0, 1, 0, 0)
	DurationDay    ISODuration = NewISODuration(0, 0, 0, 1, 0, 0, 0)
	DurationWeek   ISODuration = NewISODuration(0, 0, 1, 0, 0, 0, 0)
	DurationMonth  ISODuration = NewISODuration(0, 1, 0, 0, 0, 0, 0)
	DurationYear   ISODuration = NewISODuration(1, 0, 0, 0, 0, 0, 0)
)
