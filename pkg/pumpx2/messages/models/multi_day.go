package models

import "strconv"

// MultiDay is a day of the week as one bit of the pump's day bitmasks.
type MultiDay int32

// The days, by their bits.
const (
	Monday    MultiDay = 1
	Tuesday   MultiDay = 2
	Wednesday MultiDay = 4
	Thursday  MultiDay = 8
	Friday    MultiDay = 16
	Saturday  MultiDay = 32
	Sunday    MultiDay = 64
)

// MultiDays are the days in pumpX2's declaration order.
var MultiDays = []MultiDay{Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday}

// MultiDayFromBitmask is pumpX2's MultiDay.fromBitmask, which keeps each day
// whose bit divides the bitmask rather than each day whose bit is set: 3
// gives only Monday, and 6 Monday and Tuesday.
func MultiDayFromBitmask(bitmask int32) []MultiDay {
	days := []MultiDay{}
	for _, d := range MultiDays {
		if bitmask%int32(d) == 0 {
			days = append(days, d)
		}
	}
	return days
}

// MultiDayToBitmask sums the days' bits.
func MultiDayToBitmask(days ...MultiDay) int32 {
	var mask int32
	for _, d := range days {
		mask += int32(d)
	}
	return mask
}

func (d MultiDay) String() string {
	switch d {
	case Monday:
		return "MONDAY"
	case Tuesday:
		return "TUESDAY"
	case Wednesday:
		return "WEDNESDAY"
	case Thursday:
		return "THURSDAY"
	case Friday:
		return "FRIDAY"
	case Saturday:
		return "SATURDAY"
	case Sunday:
		return "SUNDAY"
	}
	return "MultiDay(" + strconv.Itoa(int(d)) + ")"
}
